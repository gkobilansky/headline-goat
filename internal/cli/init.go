package cli

import (
	"context"
	"fmt"
	"math"
	"os"
	"regexp"
	"strings"

	"github.com/headline-goat/headline-goat/internal/store"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var (
	variants []string
	weights  []float64
	goal     string
)

var initCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Create a new A/B test",
	Long: `Create a new A/B test interactively or with flags.

Interactive mode (recommended):
  headline-goat init

With flags:
  headline-goat init hero -v "Ship Faster" -v "Build Better"
  headline-goat init hero -v "A" -v "B" --goal "Signup button click"`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInit,
}

func init() {
	initCmd.Flags().StringSliceVarP(&variants, "variants", "v", nil, "variant names (at least 2 required)")
	initCmd.Flags().Float64SliceVarP(&weights, "weights", "w", nil, "variant weights (must sum to 1.0)")
	initCmd.Flags().StringVarP(&goal, "goal", "g", "", "conversion goal description")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	var name string
	var err error

	// Interactive mode if no name provided
	if len(args) == 0 {
		name, err = promptTestName()
		if err != nil {
			return err
		}
	} else {
		name = args[0]
	}

	// Validate name
	if !isValidName(name) {
		return fmt.Errorf("invalid test name: must be alphanumeric with hyphens only")
	}

	// Interactive variant input if not provided via flags
	if len(variants) == 0 {
		variants, err = promptVariants()
		if err != nil {
			return err
		}
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

	// Success message
	fmt.Println()
	fmt.Printf("Created test '%s' with %d variants:\n", test.Name, len(test.Variants))
	for i, v := range test.Variants {
		fmt.Printf("  %d: \"%s\"\n", i, v)
	}

	// Explain next steps
	fmt.Println()
	fmt.Println("How it works:")
	fmt.Println("  1. Add the snippet below to your site")
	fmt.Println("  2. The script randomly shows one variant to each visitor")
	fmt.Println("  3. Track conversions with the convert button/function")
	fmt.Println("  4. Check results in the dashboard or with 'headline-goat results'")
	fmt.Println()

	// Generate snippet (always)
	err = RunSnippetFlow(test)
	if err != nil {
		return err
	}

	// Final guidance
	fmt.Println()
	fmt.Println("What's next:")
	fmt.Printf("  - View results:    headline-goat results %s\n", test.Name)
	fmt.Printf("  - Open dashboard:  headline-goat otp\n")
	fmt.Printf("  - Declare winner:  headline-goat winner %s -v <index>\n", test.Name)
	fmt.Println()
	fmt.Println("Once you declare a winner, running 'snippet' will generate")
	fmt.Println("static code with just the winning variant (no A/B testing logic).")
	fmt.Println()

	return nil
}

func promptTestName() (string, error) {
	prompt := promptui.Prompt{
		Label: "Test name (e.g., hero, pricing-cta)",
		Validate: func(input string) error {
			if !isValidName(input) {
				return fmt.Errorf("must be alphanumeric with hyphens only")
			}
			return nil
		},
	}

	result, err := prompt.Run()
	if err != nil {
		if err == promptui.ErrInterrupt {
			os.Exit(0)
		}
		return "", err
	}

	return result, nil
}

func promptVariants() ([]string, error) {
	prompt := promptui.Prompt{
		Label: "Variants (comma-separated, e.g., Ship Faster, Build Better)",
		Validate: func(input string) error {
			parts := parseVariants(input)
			if len(parts) < 2 {
				return fmt.Errorf("enter at least 2 variants separated by commas")
			}
			return nil
		},
	}

	result, err := prompt.Run()
	if err != nil {
		if err == promptui.ErrInterrupt {
			os.Exit(0)
		}
		return nil, err
	}

	return parseVariants(result), nil
}

func parseVariants(input string) []string {
	parts := strings.Split(input, ",")
	var result []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func isValidName(name string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9-]+$`, name)
	return matched && len(name) > 0
}
