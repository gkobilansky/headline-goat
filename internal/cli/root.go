package cli

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	dbPath string
)

var rootCmd = &cobra.Command{
	Use:   "headline-goat",
	Short: "A minimal, self-hosted A/B testing tool for headlines",
	Long: `headline-goat is a minimal, self-hosted A/B testing tool for headlines.
Single Go binary, embedded SQLite, no external dependencies.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", getEnvOrDefault("HG_DB_PATH", "./headline-goat.db"), "database path")
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
