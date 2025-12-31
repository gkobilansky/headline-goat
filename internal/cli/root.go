package cli

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	dbPath string
)

var rootCmd = &cobra.Command{
	Use:   "hlg",
	Short: "Headline Goat - A minimal, self-hosted A/B testing tool for headlines",
	Long: `🐐 Headline Goat is a minimal, self-hosted A/B testing tool for headlines.
Single Go binary, embedded SQLite, no external dependencies.

Running without a subcommand starts the server (same as 'hlg init').`,
	RunE: runInit, // Default action is to start server
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", getEnvOrDefault("HG_DB_PATH", "./hlg.db"), "database path")
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
