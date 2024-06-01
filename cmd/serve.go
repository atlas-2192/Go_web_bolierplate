package cmd

import (
	"context"
	"os"
	"os/signal"
	"sync"

	"github.com/jackc/web-starter-app/server"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

var shutdownSignals = []os.Signal{os.Interrupt}

// serveCmd represents the serve command.
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the server",

	Run: func(cmd *cobra.Command, args []string) {
		startHTTPServer, _ := cmd.Flags().GetBool("http")
		listenAddress, _ := cmd.Flags().GetString("listen-address")
		logFormat, _ := cmd.Flags().GetString("log-format")

		processCtx, processCancel := context.WithCancel(context.Background())

		setupLogger(logFormat)

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
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().Bool("http", true, "Serve HTTP requests.")
	serveCmd.Flags().StringP("listen-address", "l", "127.0.0.1:8080", "The address to listen on for HTTP requests.")
	serveCmd.Flags().String("log-format", "json", "Log format (json or console)")
}
