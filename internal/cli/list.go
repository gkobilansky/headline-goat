package cli

import (
	"context"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/headline-goat/headline-goat/internal/store"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tests",
	Long:  `List all A/B tests with their status and statistics.`,
	RunE:  runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	// Open database
	s, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer s.Close()

	ctx := context.Background()

	// Get all tests
	tests, err := s.ListTests(ctx)
	if err != nil {
		return fmt.Errorf("failed to list tests: %w", err)
	}

	if len(tests) == 0 {
		fmt.Println("No tests yet.")
		fmt.Println()
		fmt.Println("Tests auto-create when visitors arrive. Add the script to your site:")
		fmt.Println("  <script src=\"YOUR_SERVER/hlg.js\" defer></script>")
		return nil
	}

	// Print table
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSOURCE\tSTATE\tVARIANTS\tVIEWS\tCONVERSIONS\tCREATED")

	for _, test := range tests {
		// Get stats for this test
		stats, err := s.GetVariantStats(ctx, test.Name)
		if err != nil {
			return fmt.Errorf("failed to get stats for test %s: %w", test.Name, err)
		}

		totalViews := 0
		totalConversions := 0
		for _, stat := range stats {
			totalViews += stat.Views
			totalConversions += stat.Conversions
		}

		// Format source with conflict indicator
		source := test.Source
		if test.HasSourceConflict {
			source += " (!)"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\t%s\n",
			test.Name,
			source,
			strings.ToUpper(string(test.State)),
			len(test.Variants),
			formatNumber(totalViews),
			formatNumber(totalConversions),
			test.CreatedAt.Format("2006-01-02"),
		)
	}

	w.Flush()
	return nil
}

func formatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1000000 {
		return fmt.Sprintf("%d,%03d", n/1000, n%1000)
	}
	return fmt.Sprintf("%d,%03d,%03d", n/1000000, (n/1000)%1000, n%1000)
}
