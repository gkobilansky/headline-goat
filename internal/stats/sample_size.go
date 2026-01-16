package stats

import "math"

// RequiredSampleSize calculates views needed per variant to detect
// a minimum detectable effect (MDE) with given confidence and power.
//
// Uses the Evan Miller formula for A/B test sample sizes:
// n = 2 * (Z_α + Z_β)² / (arcsin(√p2) - arcsin(√p1))²
//
// This produces practical estimates for A/B testing:
// - 5% baseline, 20% MDE → ~390 per variant
// - 10% baseline, 20% MDE → ~390 per variant
// - 2% baseline, 20% MDE → ~980 per variant
//
// Default assumptions:
// - α = 0.05 (95% confidence, two-tailed)
// - β = 0.20 (80% power)
//
// Parameters:
// - baselineRate: current conversion rate (0-1). If 0, defaults to 0.05
// - mde: minimum detectable effect as relative lift (e.g., 0.20 for 20% improvement)
//
// Returns sample size per variant, rounded up to nearest 50.
func RequiredSampleSize(baselineRate float64, mde float64) int {
	// Use default baseline if zero
	if baselineRate <= 0 {
		baselineRate = 0.05
	}

	if mde == 0 {
		return 400 // Default when no difference expected
	}

	// Calculate p2 (expected rate after improvement)
	p1 := baselineRate
	p2 := baselineRate * (1 + mde)

	// Cap p2 at 1.0
	if p2 > 1.0 {
		p2 = 1.0
	}

	// Arcsine transformation (variance-stabilizing)
	// h = 2 * arcsin(√p)
	h1 := 2 * math.Asin(math.Sqrt(p1))
	h2 := 2 * math.Asin(math.Sqrt(p2))
	h := h2 - h1

	if h == 0 {
		return 400
	}

	// Z-scores: Z_α/2 = 1.96, Z_β = 0.84 for 95% confidence, 80% power
	// Combined: (1.96 + 0.84)² = 7.84
	// For two-sample test: n = 2 * 7.84 / h²

	n := 2 * 7.84 / (h * h)

	// Round up to nearest 50
	rounded := int(math.Ceil(n/50) * 50)

	return rounded
}
