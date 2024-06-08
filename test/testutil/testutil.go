package testutil

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/testdb"
)

// InitTestDBManager performs the standard initialization of a *testdb.Manager for ISO Amp. It requires a *testing.M to
// ensure it is only called by TestMain. If something fails it calls os.Exit(1).
func InitTestDBManager(*testing.M) *testdb.Manager {
	manager := &testdb.Manager{
		ResetDB: func(ctx context.Context, conn *pgx.Conn) error {
			_, err := conn.Exec(ctx, `select pgundolog.undo()`)
			return err
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	err := manager.Connect(ctx, "")
	if err != nil {
		fmt.Println("failed to init testdb.Manager:", err)
		os.Exit(1)
	}

	return manager
}

// CopyTestPGEnvironmentVariables copies all environment variables that start with "TEST_PG" to "PG". This allows using
// the standard PG environment variables to configure development and test databases.
func CopyTestPGEnvironmentVariables() error {
	envvars := os.Environ()
	for _, envvar := range envvars {
		if strings.HasPrefix(envvar, "TEST_PG") {
			parts := strings.SplitN(envvar, "=", 2)
			err := os.Setenv(parts[0][5:], parts[1])
			if err != nil {
				return fmt.Errorf("CopyTestPGEnvironmentVariables: %w", err)
			}
		} else if strings.HasPrefix(envvar, "TEST_DATABASE_URL=") {
			err := os.Setenv("DATABASE_URL", envvar[18:])
			if err != nil {
				return fmt.Errorf("CopyTestPGEnvironmentVariables: %w", err)
			}
		}
	}

	return nil
}
