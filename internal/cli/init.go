package cli

import (
	"context"
	"fmt"
	"math"
	"regexp"

	"github.com/headline-goat/headline-goat/internal/store"
	"github.com/spf13/cobra"
)

var (
	variants []string
	weights  []float64
	goal     string
)

var initCmd = &cobra.Command{
	Use:   "init <name>",
	Short: "Create a new A/B test",
	Long: `Create a new A/B test with the specified name and variants.

Examples:
  headline-goat init hero --variants "Ship Faster" "Build Better" "Scale Smart"
  headline-goat init hero --variants "A" "B" --weights 0.9 0.1
  headline-goat init hero --variants "A" "B" --goal "Signup button click"`,
	Args: cobra.ExactArgs(1),
	RunE: runInit,
}

func init() {
	initCmd.Flags().StringSliceVarP(&variants, "variants", "v", nil, "variant names (at least 2 required)")
	initCmd.Flags().Float64SliceVarP(&weights, "weights", "w", nil, "variant weights (must sum to 1.0)")
	initCmd.Flags().StringVarP(&goal, "goal", "g", "", "conversion goal description")
	initCmd.MarkFlagRequired("variants")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Validate name (alphanumeric + hyphens)
	if !isValidName(name) {
		return fmt.Errorf("invalid test name: must be alphanumeric with hyphens only")
	}

	// Validate variants (at least 2)
	if len(variants) < 2 {
		return fmt.Errorf("at least 2 variants required")
	}

	// Validate weights if provided
	if len(weights) > 0 {
		if len(weights) != len(variants) {
			return fmt.Errorf("number of weights must match number of variants")
		}

		sum := 0.0
		for _, w := range weights {
			if w < 0 || w > 1 {
				return fmt.Errorf("weights must be between 0 and 1")
			}
			sum += w
		}

		if math.Abs(sum-1.0) > 0.001 {
			return fmt.Errorf("weights must sum to 1.0 (got %.3f)", sum)
		}
	}

	// Open database
	s, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer s.Close()

	// Create test
	ctx := context.Background()
	test, err := s.CreateTest(ctx, name, variants, weights, goal)
	if err != nil {
		return fmt.Errorf("failed to create test: %w", err)
	}

	fmt.Printf("Created test '%s' with %d variants\n", test.Name, len(test.Variants))
	if goal != "" {
		fmt.Printf("Conversion goal: %s\n", goal)
	}

	return nil
}

func isValidName(name string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9-]+$`, name)
	return matched && len(name) > 0
}
