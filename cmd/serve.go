package cmd

import (
	"context"
	"crypto/sha512"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"

	"github.com/jackc/envconf"
	"github.com/jackc/web-starter-app/httpz"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

var shutdownSignals = []os.Signal{os.Interrupt}
var serveEnvconf = envconf.New()

// serveCmd represents the serve command.
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the server",

	Run: func(cmd *cobra.Command, args []string) {
		startHTTPServer, _ := cmd.Flags().GetBool("http")

		// Get config from the environment.
		databaseURL := serveEnvconf.Value("DATABASE_URL")
		listenAddress := serveEnvconf.Value("LISTEN_ADDRESS")
		logFormat := serveEnvconf.Value("LOG_FORMAT")

		digestKey := func(keyName string, minInputLen, outputLen int) []byte {
			str := serveEnvconf.Value(keyName)
			if len(str) < minInputLen {
				fmt.Fprintf(os.Stderr, "%s not set or too short.\n", keyName)
				os.Exit(1)
			}

			h := sha512.Sum512([]byte(str))
			return h[:outputLen]
		}

		csrfKey := digestKey("CSRF_KEY", 64, 64)
		cookieAuthenticationKey := digestKey("COOKIE_AUTHENTICATION_KEY", 64, 64)
		cookieEncryptionKey := digestKey("COOKIE_ENCRYPTION_KEY", 64, 32)

		cookieSecure, err := strconv.ParseBool(serveEnvconf.Value("COOKIE_SECURE"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "COOKIE_SECURE must be true or false.\n")
			os.Exit(1)
		}

		// processCtx and processCancel are used to signal when the process is shutting down.
		processCtx, processCancel := context.WithCancel(context.Background())

		logger := setupLogger(logFormat)
		dbpool := setupPGXConnPool(processCtx, databaseURL, logger)

		loadManifest := func(path string) (map[string]string, error) {
			manifestBytes, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("LoadManifest: %w", err)
			}

			var manifest map[string]any
			err = json.Unmarshal(manifestBytes, &manifest)
			if err != nil {
				return nil, fmt.Errorf("LoadManifest %s: %w", path, err)
			}

			assetMap := make(map[string]string, len(manifest))
			for k, v := range manifest {
				assetMap[k] = v.(map[string]any)["file"].(string)
			}

			return assetMap, nil
		}

		var assetManifest map[string]string
		if assetManifestPath := serveEnvconf.Value("ASSET_MANIFEST"); assetManifestPath != "" {
			assetManifest, err = loadManifest(assetManifestPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Could not load asset manifest: %s\n", err)
				os.Exit(1)
			}
		}

		// Listen for shutdown signals. When a signal is received, cancel the processCtx.
		interruptChan := make(chan os.Signal, 1)
		signal.Notify(interruptChan, shutdownSignals...)
		go func() {
			s := <-interruptChan
			signal.Reset() // Only listen for one interrupt. If another interrupt signal is received allow it to terminate the program.
			zerolog.Ctx(processCtx).Info().Str("signal", s.String()).Msg("shutdown signal received")
			processCancel()
		}()

		// The program can run in more than one worker at a time. For example, it may run an HTTP server and a job worker.
		// Use a WaitGroup to wait for all workers to finish before exiting.
		wg := &sync.WaitGroup{}

		if startHTTPServer {
			handler, err := httpz.NewHandler(
				dbpool,
				zerolog.Ctx(processCtx),
				csrfKey,
				cookieSecure,
				cookieAuthenticationKey,
				cookieEncryptionKey,
				assetManifest,
			)
			if err != nil {
				zerolog.Ctx(processCtx).Fatal().Err(err).Msg("Could not create HTTP app handler")
			}

			server := &http.Server{
				Addr:    listenAddress,
				Handler: handler,
			}

			wg.Add(1)
			go func() {
				defer wg.Done()
				zerolog.Ctx(processCtx).Info().Str("listen_address", listenAddress).Msg("Starting HTTP server")

				err := server.ListenAndServe()
				if err != http.ErrServerClosed {
					zerolog.Ctx(processCtx).Fatal().Err(err).Msg("HTTP server failed to start")
				}
			}()

			wg.Add(1)
			go func() {
				defer wg.Done()
				<-processCtx.Done()
				zerolog.Ctx(processCtx).Info().Msg("Stopping HTTP server")
				server.SetKeepAlivesEnabled(false)
				err := server.Shutdown(context.Background())
				if err != nil {
					zerolog.Ctx(processCtx).Error().Err(err).Msg("HTTP server failed to cleanly shutdown")
				}
			}()
		}

		wg.Wait()
	},
}

func init() {
	serveEnvconf.Register(envconf.Item{Name: "DATABASE_URL", Default: "", Description: "The PostgreSQL connection string"})
	serveEnvconf.Register(envconf.Item{Name: "LISTEN_ADDRESS", Default: "127.0.0.1:8080", Description: "The address to listen on for HTTP requests"})
	serveEnvconf.Register(envconf.Item{Name: "LOG_FORMAT", Default: "json", Description: "Log format (json or console)"})
	serveEnvconf.Register(envconf.Item{Name: "CSRF_KEY", Default: "", Description: "Key for CSRF protection"})
	serveEnvconf.Register(envconf.Item{Name: "COOKIE_SECURE", Default: "true", Description: "Set the Secure flag on cookies"})
	serveEnvconf.Register(envconf.Item{Name: "COOKIE_AUTHENTICATION_KEY", Default: "", Description: "Key to protect cookies from tampering"})
	serveEnvconf.Register(envconf.Item{Name: "COOKIE_ENCRYPTION_KEY", Default: "", Description: "Key to protect cookies from being readable by the client"})
	serveEnvconf.Register(envconf.Item{Name: "ASSET_MANIFEST", Default: "", Description: "Path to the asset manifest file"})

	long := &strings.Builder{}
	long.WriteString("Run the server.\n\nConfigure with the following environment variables:\n\n")
	for _, item := range serveEnvconf.Items() {
		long.WriteString(fmt.Sprintf("  %s\n    Default: %s\n    %s\n\n", item.Name, item.Default, item.Description))
	}
	serveCmd.Long = long.String()

	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().Bool("http", true, "Serve HTTP requests.")
}
