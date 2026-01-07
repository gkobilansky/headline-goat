package stats

import (
	"math"

	"github.com/gkobilansky/headline-goat/internal/store"
)

// Result represents statistical analysis of a test
type Result struct {
	Variants        []VariantResult
	Confident       bool    // >= 95% confidence
	ConfidenceLevel float64 // 0-1
	LeadingVariant  int
}

// VariantResult contains statistics for a single variant
type VariantResult struct {
	Index       int
	Name        string
	Views       int
	Conversions int
	Rate        float64
	CILower     float64
	CIUpper     float64
}

// SignificanceTest performs a two-proportion z-test.
// Returns confidence level (0-1) that variant A beats variant B.
func SignificanceTest(aConv, aViews, bConv, bViews int) float64 {
	// Handle edge cases
	if aViews == 0 && bViews == 0 {
		return 0.5 // No data, can't determine
	}
	if aViews == 0 || bViews == 0 {
		return 0.5 // Need data from both variants
	}

	// Calculate proportions
	pA := float64(aConv) / float64(aViews)
	pB := float64(bConv) / float64(bViews)

	// Pooled proportion under null hypothesis (pA = pB)
	pooledP := float64(aConv+bConv) / float64(aViews+bViews)

	// Standard error of the difference
	se := math.Sqrt(pooledP * (1 - pooledP) * (1/float64(aViews) + 1/float64(bViews)))

	if se == 0 {
		if pA > pB {
			return 1.0
		} else if pA < pB {
			return 0.0
		}
		return 0.5
	}

	// Z-statistic
	z := (pA - pB) / se

	// Convert to confidence level using standard normal CDF
	// P(Z < z) gives us confidence that A > B
	confidence := normalCDF(z)

	return confidence
}

// normalCDF approximates the cumulative distribution function
// of the standard normal distribution
func normalCDF(x float64) float64 {
	// Use the approximation from Abramowitz and Stegun
	// Handbook of Mathematical Functions, formula 7.1.26
	a1 := 0.254829592
	a2 := -0.284496736
	a3 := 1.421413741
	a4 := -1.453152027
	a5 := 1.061405429
	p := 0.3275911

	sign := 1.0
	if x < 0 {
		sign = -1.0
	}
	x = math.Abs(x) / math.Sqrt(2)

	t := 1.0 / (1.0 + p*x)
	y := 1.0 - (((((a5*t+a4)*t)+a3)*t+a2)*t+a1)*t*math.Exp(-x*x)

	return 0.5 * (1.0 + sign*y)
}

// Analyze calculates full statistics for a test
func Analyze(test *store.Test, variantStats []store.VariantStats) *Result {
	// Create a map for quick lookup
	statsMap := make(map[int]store.VariantStats)
	for _, s := range variantStats {
		statsMap[s.Variant] = s
	}

	// Build variant results
	variants := make([]VariantResult, len(test.Variants))
	maxRate := 0.0
	leadingVariant := 0

	for i, name := range test.Variants {
		stat := statsMap[i] // Will be zero-valued if not present

		rate := 0.0
		if stat.Views > 0 {
			rate = float64(stat.Conversions) / float64(stat.Views)
		}

		ciLower, ciUpper := WilsonInterval(stat.Conversions, stat.Views, 0.95)

		variants[i] = VariantResult{
			Index:       i,
			Name:        name,
			Views:       stat.Views,
			Conversions: stat.Conversions,
			Rate:        rate,
			CILower:     ciLower,
			CIUpper:     ciUpper,
		}

		if rate > maxRate {
			maxRate = rate
			leadingVariant = i
		}
	}

	// Calculate significance between leading variant and control (variant 0)
	var confidenceLevel float64
	if len(variants) >= 2 {
		// Compare leading variant against control (variant 0)
		if leadingVariant == 0 {
			// Control is leading, compare against best challenger
			bestChallenger := 1
			bestRate := 0.0
			for i := 1; i < len(variants); i++ {
				if variants[i].Rate > bestRate {
					bestRate = variants[i].Rate
					bestChallenger = i
				}
			}
			confidenceLevel = SignificanceTest(
				variants[0].Conversions, variants[0].Views,
				variants[bestChallenger].Conversions, variants[bestChallenger].Views,
			)
		} else {
			// Challenger is leading, compare against control
			confidenceLevel = SignificanceTest(
				variants[leadingVariant].Conversions, variants[leadingVariant].Views,
				variants[0].Conversions, variants[0].Views,
			)
		}
	}

	return &Result{
		Variants:        variants,
		Confident:       confidenceLevel >= 0.95,
		ConfidenceLevel: confidenceLevel,
		LeadingVariant:  leadingVariant,
	}
}
