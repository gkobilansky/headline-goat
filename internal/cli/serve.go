package cli

import (
	"fmt"
	"os"
	"strconv"

	"github.com/headline-goat/headline-goat/internal/server"
	"github.com/headline-goat/headline-goat/internal/store"
	"github.com/spf13/cobra"
)

var port int

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP server",
	Long: `Start the headline-goat HTTP server.

The server provides:
  - Beacon endpoint for tracking events
  - Dashboard for viewing results
  - Health check endpoint

Example:
  headline-goat serve --port 8080`,
	RunE: runServe,
}

func init() {
	defaultPort := 8080
	if p := os.Getenv("HG_PORT"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil {
			defaultPort = parsed
		}
	}

	serveCmd.Flags().IntVarP(&port, "port", "p", defaultPort, "port to listen on")
	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
	// Open database
	s, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer s.Close()

	// Create and start server
	srv := server.New(s, port)
	return srv.Start()
}
