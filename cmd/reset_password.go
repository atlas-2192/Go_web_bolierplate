package cmd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/envconf"
	"github.com/jackc/web-starter-app/db"
	"github.com/spf13/cobra"
)

var resetPasswordEnvconf = envconf.New()

// resetPasswordCmd represents the reset-password command.
var resetPasswordCmd = &cobra.Command{
	Use:   "reset-password username",
	Short: "Reset a user's password",
	Args:  cobra.ExactArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		// Get config from the environment.
		databaseURL := resetPasswordEnvconf.Value("DATABASE_URL")

		logger := setupLogger("console")
		dbpool := setupPGXConnPool(context.Background(), databaseURL, logger)

		var userID uuid.UUID
		username := args[0]

		err := dbpool.QueryRow(context.Background(), "select id from users where username = $1", username).Scan(&userID)
		if err != nil {
			logger.Fatal().Err(err).Msg("Failed to find user")
		}

		randBytes := make([]byte, 16)
		_, err = rand.Read(randBytes)
		if err != nil {
			logger.Fatal().Err(err).Msg("Failed to generate random bytes")
		}

		password := hex.EncodeToString(randBytes)

		err = db.SetUserPassword(context.Background(), dbpool, userID, password)
		if err != nil {
			logger.Fatal().Err(err).Msg("Failed to set user password")
		}

		fmt.Printf("Password reset for user %s: %s\n", username, password)
	},
}

func init() {
	resetPasswordEnvconf.Register(envconf.Item{Name: "DATABASE_URL", Default: "", Description: "The PostgreSQL connection string"})

	long := &strings.Builder{}
	long.WriteString("Reset a users password.\n\nConfigure with the following environment variables:\n\n")
	for _, item := range serveEnvconf.Items() {
		long.WriteString(fmt.Sprintf("  %s\n    Default: %s\n    %s\n\n", item.Name, item.Default, item.Description))
	}
	resetPasswordCmd.Long = long.String()

	rootCmd.AddCommand(resetPasswordCmd)
}
