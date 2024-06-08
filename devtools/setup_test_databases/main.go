package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/web-starter-app/test/testutil"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "setup_test_databases",
	Short: "Creates test databases for web-starter-app",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		ctx := context.Background()

		devConn, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
		if err != nil {
			return fmt.Errorf("connect to development database: %w", err)
		}
		defer devConn.Close(ctx)

		err = testutil.CopyTestPGEnvironmentVariables()
		if err != nil {
			return err
		}

		testConnConfig, err := pgx.ParseConfig(os.Getenv("DATABASE_URL"))
		if err != nil {
			return fmt.Errorf("parse test database URL: %w", err)
		}

		// Ensure test database name ends with _test before dropping it.
		if !strings.HasSuffix(testConnConfig.Database, "_test") {
			return fmt.Errorf("test database name %q must end with _test", testConnConfig.Database)
		}

		_, err = devConn.Exec(ctx, fmt.Sprintf("drop database if exists %s", pgx.Identifier{testConnConfig.Database}.Sanitize()))
		if err != nil {
			return fmt.Errorf("drop test database %q: %w", testConnConfig.Database, err)
		}

		_, err = devConn.Exec(ctx, fmt.Sprintf("create database %s", pgx.Identifier{testConnConfig.Database}.Sanitize()))
		if err != nil {
			return fmt.Errorf("create test database %q: %w", testConnConfig.Database, err)
		}

		// Migrate the test database.
		err = exec.Command("tern", "migrate").Run()
		if err != nil {
			return fmt.Errorf("tern migrate test database: %w", err)
		}

		testConn, err := pgx.ConnectConfig(ctx, testConnConfig)
		if err != nil {
			return fmt.Errorf("connect to test database: %w", err)
		}

		pgundologSQL, err := os.ReadFile("test/testdata/pgundolog.sql")
		if err != nil {
			return fmt.Errorf("read pgundolog.sql: %w", err)
		}

		_, err = testConn.Exec(ctx, string(pgundologSQL))
		if err != nil {
			return fmt.Errorf("install pgundolog: %w", err)
		}

		_, err = testConn.Exec(ctx, `select pgundolog.create_trigger_for_all_tables_in_schema('public')`)
		if err != nil {
			return fmt.Errorf("create pgundolog triggers: %w", err)
		}

		_, err = testConn.Exec(ctx, `create schema testdb`)
		if err != nil {
			return fmt.Errorf("create testdb schema: %w", err)
		}

		_, err = testConn.Exec(ctx, `create table testdb.databases (name text primary key, acquirer_pid int)`)
		if err != nil {
			return fmt.Errorf("create testdb.databases table: %w", err)
		}

		var testDatabaseCount int
		if s := os.Getenv("TEST_DATABASE_COUNT"); s != "" {
			testDatabaseCount, err = strconv.Atoi(s)
			if err != nil {
				return fmt.Errorf("parse TEST_DATABASE_COUNT: %w", err)
			}
		} else {
			testDatabaseCount = runtime.NumCPU()
		}

		testDatabaseNames := make([]string, testDatabaseCount)
		for i := range testDatabaseNames {
			testDatabaseNames[i] = fmt.Sprintf("%s_%d", testConnConfig.Database, i)
		}

		for _, dbname := range testDatabaseNames {
			_, err = testConn.Exec(ctx, `insert into testdb.databases (name) values ($1)`, dbname)
			if err != nil {
				return fmt.Errorf("insert into testdb.databases: %w", err)
			}
		}

		err = testConn.Close(ctx)
		if err != nil {
			return fmt.Errorf("close test connection: %w", err)
		}

		for _, dbname := range testDatabaseNames {
			_, err = devConn.Exec(ctx, fmt.Sprintf("drop database if exists %s with (force)", pgx.Identifier{dbname}.Sanitize()))
			if err != nil {
				return fmt.Errorf("drop test database %q: %w", dbname, err)
			}

			_, err = devConn.Exec(ctx, fmt.Sprintf("create database %s template = %s", pgx.Identifier{dbname}.Sanitize(), pgx.Identifier{testConnConfig.Database}.Sanitize()))
			if err != nil {
				return fmt.Errorf("create test database %q: %w", dbname, err)
			}
		}

		return nil
	},
}

func main() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
