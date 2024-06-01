/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// serveCmd represents the serve command.
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the server",

	Run: func(cmd *cobra.Command, args []string) {
		listenAddress, _ := cmd.Flags().GetString("listen-address")
		fmt.Println("serve called: " + listenAddress)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().StringP("listen-address", "l", "127.0.0.1:8080", "The address to listen on for HTTP requests.")
}
