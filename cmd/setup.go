package cmd

import (
	"context"
	"io"
	"os"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

func setupPGXConnPool(ctx context.Context, connString string, logger *zerolog.Logger) *pgxpool.Pool {
	dbconfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to parse database connection string")
	}

	// Log PostgreSQL notices.
	dbconfig.ConnConfig.OnNotice = func(conn *pgconn.PgConn, n *pgconn.Notice) {
		var event *zerolog.Event
		switch n.Severity {
		case "DEBUG":
			event = logger.Debug()
		case "LOG", "INFO", "NOTICE":
			event = logger.Info()
		case "WARNING":
			event = logger.Warn()
		case "EXCEPTION":
			event = logger.Error()
		}

		event.Str("msg", n.Message).Uint32("PID", conn.PID()).Msg("PostgreSQL OnNotice")
	}

	dbpool, err := pgxpool.NewWithConfig(ctx, dbconfig)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create connection pool")
	}

	return dbpool
}

func setupLogger(logFormat string) *zerolog.Logger {
	var logWriter io.Writer
	if logFormat == "json" {
		logWriter = os.Stdout
	} else {
		logWriter = zerolog.ConsoleWriter{Out: os.Stdout}
	}

	logger := zerolog.New(logWriter).With().Timestamp().Logger()
	zerolog.DefaultContextLogger = &logger

	return &logger
}
