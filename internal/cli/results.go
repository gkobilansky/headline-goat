package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/gkobilansky/headline-goat/internal/stats"
	"github.com/gkobilansky/headline-goat/internal/store"
	"github.com/spf13/cobra"
)

var (
	resultsJSON bool
)

var resultsCmd = &cobra.Command{
	Use:   "results <name>",
	Short: "Show detailed results for a test",
	Long:  `Show detailed results including conversion rates and confidence intervals.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runResults,
}

func init() {
	resultsCmd.Flags().BoolVar(&resultsJSON, "json", false, "output results as JSON")
	rootCmd.AddCommand(resultsCmd)
}

func runResults(cmd *cobra.Command, args []string) error {
	name := args[0]

	return withStore(func(s store.Store) error {
		ctx := context.Background()

		// JSON output mode
		if resultsJSON {
			output, err := FormatResultsJSON(ctx, s, name)
			if err != nil {
				if err == store.ErrNotFound {
					return fmt.Errorf("test '%s' not found", name)
				}
				return err
			}
			fmt.Println(output)
			return nil
		}

		// Get test
		test, err := s.GetTest(ctx, name)
		if err != nil {
			if err == store.ErrNotFound {
				return fmt.Errorf("test '%s' not found. Run 'hlg list' to see available tests", name)
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
			variantName := v.Name
			if len(variantName) > 16 {
				variantName = variantName[:13] + "..."
			}

			fmt.Printf("%-16s  %-7d  %-11d  %-7s  %s%s\n",
				variantName,
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
			leadingName := result.LeadingVariantName()
			confPct := result.ConfidenceLevel * 100

			if result.Confident {
				fmt.Printf("Statistical significance: %.1f%% confident \"%s\" is the winner\n", confPct, leadingName)
			} else if confPct >= 90 {
				fmt.Printf("Statistical significance: %.1f%% confident \"%s\" beats control (not yet significant)\n", confPct, leadingName)
			} else {
				fmt.Println("Statistical significance: Not enough data to determine a winner")
			}
		}

		// Get traffic rate for estimates
		viewCount, firstEvent, err := s.GetRecentViewCount(ctx, name, 24)
		if err != nil {
			return fmt.Errorf("failed to get traffic data: %w", err)
		}
		trafficRate := stats.CalculateTrafficRate(viewCount, firstEvent, 24)

		// Calculate sample size needed
		baselineRate := 0.0
		if len(result.Variants) > 0 && result.Variants[0].Views > 0 {
			baselineRate = result.Variants[0].Rate
		}
		viewsNeeded := stats.RequiredSampleSize(baselineRate, 0.20) // 20% MDE default

		// Get status estimate
		status := stats.EstimateStatus(result, viewsNeeded, trafficRate)

		// Print STATUS section
		fmt.Println()
		fmt.Println("STATUS")
		fmt.Println(strings.Repeat("─", 6))

		if status.Ready {
			fmt.Println("✓ Ready to declare winner")
			fmt.Printf("Run: hlg winner %s --variant %d\n", name, status.RecommendedVariant)
		} else {
			// Progress line
			totalViews := 0
			for _, v := range result.Variants {
				totalViews += v.Views
			}
			totalNeeded := viewsNeeded * len(result.Variants)
			progressPct := 0
			if totalNeeded > 0 {
				progressPct = (totalViews * 100) / totalNeeded
			}
			fmt.Printf("Progress: %d / %d views (%d%%)\n", totalViews, totalNeeded, progressPct)

			// Traffic rate
			if trafficRate > 0 {
				fmt.Printf("Traffic: %.0f views/hour\n", trafficRate)
			}

			// Check back time
			if status.EstimatedHours > 0 {
				hours := int(status.EstimatedHours + 0.5) // Round
				fmt.Printf("Check back in: ~%d hours\n", hours)
			}
		}

		return nil
	})
}

func formatPercent(rate float64) string {
	if rate == 0 {
		return "0%"
	}
	return fmt.Sprintf("%.2f%%", rate*100)
}
