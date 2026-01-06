package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/headline-goat/headline-goat/internal/store"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newCreateCmd())
}

func newCreateCmd() *cobra.Command {
	var (
		variants      string
		url           string
		target        string
		ctaTarget     string
		conversionURL string
	)

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new A/B test",
		Long: `Create a new A/B test with the specified name and variants.

Examples:
  hlg create hero --variants "Ship Faster,Build Better"
  hlg create cta --variants "Sign Up,Get Started,Try Free"
  hlg create hero --variants "A,B" --url "/" --target "h1"
  hlg create hero --variants "A,B" --url "/" --target "h1" --cta-target "button.signup"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			testName := args[0]

			// Parse variants
			variantList := strings.Split(variants, ",")
			for i := range variantList {
				variantList[i] = strings.TrimSpace(variantList[i])
			}

			if len(variantList) < 2 {
				return fmt.Errorf("need at least 2 variants. Example: --variants \"A,B\"")
			}

			// Validate mutually exclusive flags
			if ctaTarget != "" && conversionURL != "" {
				return fmt.Errorf("use --cta-target OR --conversion-url, not both")
			}

			return withStore(func(s *store.SQLiteStore) error {
				ctx := context.Background()

				// Create test
				test, err := s.CreateTest(ctx, testName, variantList, nil, "")
				if err != nil {
					return fmt.Errorf("failed to create test: %w", err)
				}

				// Set URL fields if provided
				if url != "" || target != "" || ctaTarget != "" || conversionURL != "" {
					err = s.SetTestURLFields(ctx, testName, url, target, ctaTarget, conversionURL)
					if err != nil {
						return fmt.Errorf("failed to set URL fields: %w", err)
					}
				}

				fmt.Printf("Created test '%s' with %d variants:\n", test.Name, len(test.Variants))
				for i, v := range test.Variants {
					fmt.Printf("  %d: %s\n", i, v)
				}
				if url != "" {
					fmt.Printf("  URL: %s\n", url)
				}
				if target != "" {
					fmt.Printf("  Target: %s\n", target)
				}
				if ctaTarget != "" {
					fmt.Printf("  CTA Target: %s\n", ctaTarget)
				}
				if conversionURL != "" {
					fmt.Printf("  Conversion URL: %s\n", conversionURL)
				}

				return nil
			})
		},
	}

	cmd.Flags().StringVarP(&variants, "variants", "v", "", "comma-separated variant names (required)")
	cmd.Flags().StringVar(&url, "url", "", "URL to match for this test (optional)")
	cmd.Flags().StringVar(&target, "target", "", "CSS selector for headline element (optional)")
	cmd.Flags().StringVar(&ctaTarget, "cta-target", "", "CSS selector for CTA element (optional)")
	cmd.Flags().StringVar(&conversionURL, "conversion-url", "", "URL for page-load conversion (optional)")
	cmd.MarkFlagRequired("variants")

	return cmd
}
