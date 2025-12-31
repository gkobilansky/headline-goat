package cli

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/headline-goat/headline-goat/internal/store"
	"github.com/spf13/cobra"
)

var exportFormat string

var exportCmd = &cobra.Command{
	Use:   "export <name>",
	Short: "Export raw event data",
	Long: `Export raw event data in CSV or JSON format.

Examples:
  headline-goat export hero --format csv > hero-data.csv
  headline-goat export hero --format json > hero-data.json`,
	Args: cobra.ExactArgs(1),
	RunE: runExport,
}

func init() {
	exportCmd.Flags().StringVarP(&exportFormat, "format", "f", "csv", "output format (csv or json)")
	rootCmd.AddCommand(exportCmd)
}

func runExport(cmd *cobra.Command, args []string) error {
	name := args[0]

	if exportFormat != "csv" && exportFormat != "json" {
		return fmt.Errorf("invalid format: must be 'csv' or 'json'")
	}

	// Open database
	s, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer s.Close()

	ctx := context.Background()

	// Verify test exists
	_, err = s.GetTest(ctx, name)
	if err != nil {
		if err == store.ErrNotFound {
			return fmt.Errorf("test '%s' not found", name)
		}
		return fmt.Errorf("failed to get test: %w", err)
	}

	// Get events
	events, err := s.GetEvents(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to get events: %w", err)
	}

	if exportFormat == "csv" {
		return exportCSV(events)
	}
	return exportJSON(events)
}

func exportCSV(events []*store.Event) error {
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()

	// Write header
	if err := w.Write([]string{"timestamp", "variant", "event_type", "visitor_id"}); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write rows
	for _, e := range events {
		row := []string{
			strconv.FormatInt(e.CreatedAt.Unix(), 10),
			strconv.Itoa(e.Variant),
			e.EventType,
			e.VisitorID,
		}
		if err := w.Write(row); err != nil {
			return fmt.Errorf("failed to write row: %w", err)
		}
	}

	return nil
}

type jsonExport struct {
	Events []jsonEvent `json:"events"`
}

type jsonEvent struct {
	Timestamp int64  `json:"timestamp"`
	Variant   int    `json:"variant"`
	EventType string `json:"event_type"`
	VisitorID string `json:"visitor_id"`
}

func exportJSON(events []*store.Event) error {
	export := jsonExport{
		Events: make([]jsonEvent, len(events)),
	}

	for i, e := range events {
		export.Events[i] = jsonEvent{
			Timestamp: e.CreatedAt.Unix(),
			Variant:   e.Variant,
			EventType: e.EventType,
			VisitorID: e.VisitorID,
		}
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(export)
}
