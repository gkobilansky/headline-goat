package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/headline-goat/headline-goat/internal/stats"
	"github.com/headline-goat/headline-goat/internal/store"
	"github.com/spf13/cobra"
)

var resultsCmd = &cobra.Command{
	Use:   "results <name>",
	Short: "Show detailed results for a test",
	Long:  `Show detailed results including conversion rates and confidence intervals.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runResults,
}

func init() {
	rootCmd.AddCommand(resultsCmd)
}

func runResults(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Open database
	s, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer s.Close()

	ctx := context.Background()

	// Get test
	test, err := s.GetTest(ctx, name)
	if err != nil {
		if err == store.ErrNotFound {
			return fmt.Errorf("test '%s' not found", name)
		}
		return fmt.Errorf("failed to get test: %w", err)
	}

	// Get stats
	variantStats, err := s.GetVariantStats(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to get stats: %w", err)
	}

	// Analyze
	result := stats.Analyze(test, variantStats)

	// Print header
	fmt.Printf("TEST: %s\n", test.Name)
	fmt.Printf("STATE: %s\n", test.State)
	if test.ConversionGoal != "" {
		fmt.Printf("GOAL: %s\n", test.ConversionGoal)
	}
	fmt.Printf("CREATED: %s\n", test.CreatedAt.Format("2006-01-02"))
	fmt.Println()

	// Print table header
	fmt.Println("VARIANT           VIEWS    CONVERSIONS  RATE     95% CI")
	fmt.Println(strings.Repeat("─", 60))

	// Print each variant
	for _, v := range result.Variants {
		indicator := ""
		if v.Index == result.LeadingVariant && len(result.Variants) > 1 {
			indicator = " ← LEADING"
		}

		ciStr := fmt.Sprintf("[%.1f%%, %.1f%%]", v.CILower*100, v.CIUpper*100)
		if v.Views == 0 {
			ciStr = "N/A"
		}

		// Truncate name if too long
		name := v.Name
		if len(name) > 16 {
			name = name[:13] + "..."
		}

		fmt.Printf("%-16s  %-7d  %-11d  %-7s  %s%s\n",
			name,
			v.Views,
			v.Conversions,
			formatPercent(v.Rate),
			ciStr,
			indicator,
		)
	}

	fmt.Println()

	// Print significance message
	if len(result.Variants) > 1 {
		leadingName := result.Variants[result.LeadingVariant].Name
		confPct := result.ConfidenceLevel * 100

		if result.Confident {
			fmt.Printf("Statistical significance: %.1f%% confident \"%s\" is the winner\n", confPct, leadingName)
		} else if confPct >= 90 {
			fmt.Printf("Statistical significance: %.1f%% confident \"%s\" beats control (not yet significant)\n", confPct, leadingName)
		} else {
			fmt.Println("Statistical significance: Not enough data to determine a winner")
		}
	}

	return nil
}

func formatPercent(rate float64) string {
	if rate == 0 {
		return "0%"
	}
	return fmt.Sprintf("%.2f%%", rate*100)
}
