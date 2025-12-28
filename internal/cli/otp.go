package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var otpCmd = &cobra.Command{
	Use:   "otp",
	Short: "Show current dashboard token",
	Long:  `Show the current dashboard access token (for when you've scrolled past it).`,
	RunE:  runOTP,
}

func init() {
	rootCmd.AddCommand(otpCmd)
}

func runOTP(cmd *cobra.Command, args []string) error {
	tokenFile := getTokenFilePath()

	data, err := os.ReadFile(tokenFile)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no server running (token file not found)\nStart the server with: headline-goat serve")
		}
		return fmt.Errorf("failed to read token file: %w", err)
	}

	token := string(data)
	if token == "" {
		return fmt.Errorf("token file is empty")
	}

	fmt.Printf("Current dashboard token: %s\n", token)
	return nil
}

// getTokenFilePath returns the path to the token file
func getTokenFilePath() string {
	// Store token file alongside the database
	dir := filepath.Dir(dbPath)
	return filepath.Join(dir, ".headline-goat-token")
}
