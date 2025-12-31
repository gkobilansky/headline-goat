package stats

import "math"

// WilsonInterval calculates the Wilson score confidence interval
// for a binomial proportion. It's more accurate for small samples
// than the normal approximation.
func WilsonInterval(successes, trials int, confidence float64) (lower, upper float64) {
	if trials == 0 {
		return 0, 0
	}

	z := ZScore(confidence)
	p := float64(successes) / float64(trials)
	n := float64(trials)

	denominator := 1 + z*z/n
	center := (p + z*z/(2*n)) / denominator
	spread := (z / denominator) * math.Sqrt(p*(1-p)/n+z*z/(4*n*n))

	lower = center - spread
	upper = center + spread

	// Clamp to [0, 1]
	if lower < 0 {
		lower = 0
	}
	if upper > 1 {
		upper = 1
	}

	return lower, upper
}

// ZScore returns the z-score for a given confidence level.
// Common values:
//   - 0.90 -> 1.645
//   - 0.95 -> 1.96
//   - 0.99 -> 2.576
func ZScore(confidence float64) float64 {
	// Use inverse of standard normal CDF
	// For common values, we use precomputed z-scores
	switch {
	case confidence >= 0.99:
		return 2.576
	case confidence >= 0.95:
		return 1.96
	case confidence >= 0.90:
		return 1.645
	case confidence >= 0.85:
		return 1.44
	case confidence >= 0.80:
		return 1.28
	default:
		// Approximate using Abramowitz and Stegun formula
		// for inverse normal CDF
		return approximateZScore(confidence)
	}
}

// approximateZScore uses a rational approximation for the inverse
// of the standard normal CDF
func approximateZScore(confidence float64) float64 {
	// Convert confidence to one-tailed probability
	p := (1 + confidence) / 2

	// Rational approximation coefficients
	a := []float64{-3.969683028665376e+01, 2.209460984245205e+02,
		-2.759285104469687e+02, 1.383577518672690e+02,
		-3.066479806614716e+01, 2.506628277459239e+00}
	b := []float64{-5.447609879822406e+01, 1.615858368580409e+02,
		-1.556989798598866e+02, 6.680131188771972e+01,
		-1.328068155288572e+01}
	c := []float64{-7.784894002430293e-03, -3.223964580411365e-01,
		-2.400758277161838e+00, -2.549732539343734e+00,
		4.374664141464968e+00, 2.938163982698783e+00}
	d := []float64{7.784695709041462e-03, 3.224671290700398e-01,
		2.445134137142996e+00, 3.754408661907416e+00}

	pLow := 0.02425
	pHigh := 1 - pLow

	var q, r float64

	if p < pLow {
		q = math.Sqrt(-2 * math.Log(p))
		return (((((c[0]*q+c[1])*q+c[2])*q+c[3])*q+c[4])*q + c[5]) /
			((((d[0]*q+d[1])*q+d[2])*q+d[3])*q + 1)
	} else if p <= pHigh {
		q = p - 0.5
		r = q * q
		return (((((a[0]*r+a[1])*r+a[2])*r+a[3])*r+a[4])*r + a[5]) * q /
			(((((b[0]*r+b[1])*r+b[2])*r+b[3])*r+b[4])*r + 1)
	} else {
		q = math.Sqrt(-2 * math.Log(1-p))
		return -(((((c[0]*q+c[1])*q+c[2])*q+c[3])*q+c[4])*q + c[5]) /
			((((d[0]*q+d[1])*q+d[2])*q+d[3])*q + 1)
	}
}
