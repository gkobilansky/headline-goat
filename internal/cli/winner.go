package cli

import (
	"context"
	"fmt"

	"github.com/headline-goat/headline-goat/internal/store"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newWinnerCmd())
}

func newWinnerCmd() *cobra.Command {
	var variantIndex int

	cmd := &cobra.Command{
		Use:   "winner <name>",
		Short: "Declare a winner for a test",
		Long: `Declare a winning variant for an A/B test and mark it complete.

Example:
  hlg winner hero --variant 0
  hlg winner pricing --variant 1`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			testName := args[0]

			return withStore(func(s *store.SQLiteStore) error {
				ctx := context.Background()
				test, err := s.GetTest(ctx, testName)
				if err != nil {
					return fmt.Errorf("test '%s' not found. Run 'hlg list' to see available tests", testName)
				}

				// Validate test is running
				if test.State != store.StateRunning {
					return fmt.Errorf("test '%s' is already %s", testName, test.State)
				}

				// Validate variant index
				if variantIndex < 0 || variantIndex >= len(test.Variants) {
					return fmt.Errorf("variant %d doesn't exist. Test '%s' has variants 0-%d", variantIndex, testName, len(test.Variants)-1)
				}

				// Set winner
				err = s.SetWinner(ctx, testName, variantIndex)
				if err != nil {
					return fmt.Errorf("failed to set winner: %w", err)
				}

				fmt.Printf("Winner declared: \"%s\" (variant %d)\n", test.Variants[variantIndex], variantIndex)
				fmt.Printf("Test '%s' is now complete.\n", testName)
				fmt.Println()
				fmt.Println("You can now update your HTML to use the winning text directly:")
				fmt.Printf("  <h1>%s</h1>\n", test.Variants[variantIndex])

				return nil
			})
		},
	}

	cmd.Flags().IntVarP(&variantIndex, "variant", "v", -1, "winning variant index (required)")
	cmd.MarkFlagRequired("variant")

	return cmd
}
