package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"

	"github.com/jackc/envconf"
	"github.com/jackc/web-starter-app/db"
	"github.com/jackc/web-starter-app/server"
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

		databaseURL := serveEnvconf.Value("DATABASE_URL")
		listenAddress := serveEnvconf.Value("LISTEN_ADDRESS")
		logFormat := serveEnvconf.Value("LOG_FORMAT")

		processCtx, processCancel := context.WithCancel(context.Background())

		logger := setupLogger(logFormat)
		dbpool := setupPGXConnPool(processCtx, databaseURL, logger)
		dbsession := db.NewSession(dbpool)

		interruptChan := make(chan os.Signal, 1)
		signal.Notify(interruptChan, shutdownSignals...)
		go func() {
			s := <-interruptChan
			signal.Reset() // Only listen for one interrupt. If another interrupt signal is received allow it to terminate the program.
			zerolog.Ctx(processCtx).Info().Str("signal", s.String()).Msg("shutdown signal received")
			processCancel()
		}()

		wg := &sync.WaitGroup{}
		if startHTTPServer {
			server, err := server.NewServer(
				listenAddress,
				dbsession,
				zerolog.Ctx(processCtx),
			)
			if err != nil {
				zerolog.Ctx(processCtx).Fatal().Err(err).Msg("Could not create web server")
			}

			wg.Add(1)
			go func() {
				defer wg.Done()
				err := server.Serve()
				if err != nil {
					zerolog.Ctx(processCtx).Fatal().Err(err).Msg("HTTP server failed to start")
				}
			}()

			wg.Add(1)
			go func() {
				defer wg.Done()
				<-processCtx.Done()
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
	long := &strings.Builder{}
	long.WriteString("Run the server.\n\nConfigure with the following environment variables:\n\n")
	for _, item := range serveEnvconf.Items() {
		long.WriteString(fmt.Sprintf("  %s\n    Default: %s\n    %s\n\n", item.Name, item.Default, item.Description))
	}
	serveCmd.Long = long.String()

	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().Bool("http", true, "Serve HTTP requests.")
}
