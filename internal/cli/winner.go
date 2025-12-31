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
		Long: `Declare a winning variant for an A/B test and complete it.

After declaring a winner, the snippet command will generate static code
showing only the winning variant (no A/B testing logic).

Example:
  headline-goat winner hero --variant 0`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			testName := args[0]

			dbPath, _ := cmd.Flags().GetString("db")
			s, err := store.Open(dbPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer s.Close()

			ctx := context.Background()
			test, err := s.GetTest(ctx, testName)
			if err != nil {
				return fmt.Errorf("test not found: %s", testName)
			}

			// Validate test is running
			if test.State != store.StateRunning {
				return fmt.Errorf("test is not running (current state: %s)", test.State)
			}

			// Validate variant index
			if variantIndex < 0 || variantIndex >= len(test.Variants) {
				return fmt.Errorf("invalid variant index: %d (test has %d variants: 0-%d)", variantIndex, len(test.Variants), len(test.Variants)-1)
			}

			// Set winner
			err = s.SetWinner(ctx, testName, variantIndex)
			if err != nil {
				return fmt.Errorf("failed to set winner: %w", err)
			}

			fmt.Printf("Declared winner for test '%s': variant %d (\"%s\")\n", testName, variantIndex, test.Variants[variantIndex])
			fmt.Println("Test has been marked as completed.")
			fmt.Println("\nNote: Running 'snippet' will now generate static code with the winning variant only.")

			return nil
		},
	}

	cmd.Flags().IntVarP(&variantIndex, "variant", "v", -1, "winning variant index (required)")
	cmd.MarkFlagRequired("variant")

	return cmd
}
