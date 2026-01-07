package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gkobilansky/headline-goat/internal/store"
	"github.com/spf13/cobra"
)

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Show dashboard URL with access token",
	Long: `Show the dashboard URL with your access token.

Use this when you've scrolled past the startup message or need to
share the dashboard link.

Example:
  hlg token`,
	RunE: runToken,
}

func init() {
	rootCmd.AddCommand(tokenCmd)
}

func runToken(cmd *cobra.Command, args []string) error {
	tokenFile := getTokenFilePath()

	data, err := os.ReadFile(tokenFile)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no server running. Start with: hlg")
		}
		return fmt.Errorf("failed to read token file: %w", err)
	}

	token := string(data)
	if token == "" {
		return fmt.Errorf("token file is empty. Restart the server with: hlg")
	}

	// Try to get the server URL from settings
	serverURL := "http://localhost:8080"
	s, err := store.Open(dbPath)
	if err == nil {
		defer s.Close()
		if url, err := s.GetSetting(context.Background(), "server_url"); err == nil && url != "" {
			serverURL = url
		}
	}

	fmt.Printf("Dashboard: %s/dashboard?token=%s\n", serverURL, token)
	fmt.Println()
	fmt.Println("Tip: Bookmark this URL or run 'hlg token' anytime.")
	return nil
}

// getTokenFilePath returns the path to the token file
func getTokenFilePath() string {
	// Store token file alongside the database
	dir := filepath.Dir(dbPath)
	return filepath.Join(dir, ".hlg-token")
}
