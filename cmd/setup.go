package cmd

import (
	"io"
	"os"

	"github.com/rs/zerolog"
)

func setupLogger(logFormat string) {
	var logWriter io.Writer
	if logFormat == "json" {
		logWriter = os.Stdout
	} else {
		logWriter = zerolog.ConsoleWriter{Out: os.Stdout}
	}

	logger := zerolog.New(logWriter).With().Timestamp().Logger()
	zerolog.DefaultContextLogger = &logger
}
