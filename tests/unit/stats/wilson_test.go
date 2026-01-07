package stats_test

import (
	"math"
	"testing"

	"github.com/gkobilansky/headline-goat/internal/stats"
)

func TestWilsonInterval_50PercentConversion(t *testing.T) {
	// 50 successes out of 100 trials
	lower, upper := stats.WilsonInterval(50, 100, 0.95)

	// Expected: approximately [0.40, 0.60] with some tolerance
	if lower < 0.38 || lower > 0.42 {
		t.Errorf("lower bound %f not in expected range [0.38, 0.42]", lower)
	}
	if upper < 0.58 || upper > 0.62 {
		t.Errorf("upper bound %f not in expected range [0.58, 0.62]", upper)
	}
}

func TestWilsonInterval_LowConversion(t *testing.T) {
	// 5 successes out of 100 trials (5% conversion)
	lower, upper := stats.WilsonInterval(5, 100, 0.95)

	// Should be roughly [0.02, 0.11]
	if lower < 0.01 || lower > 0.03 {
		t.Errorf("lower bound %f not in expected range [0.01, 0.03]", lower)
	}
	if upper < 0.09 || upper > 0.13 {
		t.Errorf("upper bound %f not in expected range [0.09, 0.13]", upper)
	}
}

func TestWilsonInterval_HighConversion(t *testing.T) {
	// 95 successes out of 100 trials (95% conversion)
	lower, upper := stats.WilsonInterval(95, 100, 0.95)

	// Should be roughly [0.89, 0.98]
	if lower < 0.87 || lower > 0.91 {
		t.Errorf("lower bound %f not in expected range [0.87, 0.91]", lower)
	}
	if upper < 0.97 || upper > 0.99 {
		t.Errorf("upper bound %f not in expected range [0.97, 0.99]", upper)
	}
}

func TestWilsonInterval_ZeroTrials(t *testing.T) {
	lower, upper := stats.WilsonInterval(0, 0, 0.95)

	if lower != 0 || upper != 0 {
		t.Errorf("expected (0, 0) for zero trials, got (%f, %f)", lower, upper)
	}
}

func TestWilsonInterval_ZeroSuccesses(t *testing.T) {
	lower, upper := stats.WilsonInterval(0, 100, 0.95)

	if lower != 0 {
		t.Errorf("expected lower bound 0, got %f", lower)
	}
	if upper < 0.01 || upper > 0.05 {
		t.Errorf("upper bound %f not in expected range [0.01, 0.05]", upper)
	}
}

func TestWilsonInterval_AllSuccesses(t *testing.T) {
	lower, upper := stats.WilsonInterval(100, 100, 0.95)

	if lower < 0.95 || lower > 0.99 {
		t.Errorf("lower bound %f not in expected range [0.95, 0.99]", lower)
	}
	if upper < 0.99 || upper > 1.0 {
		t.Errorf("upper bound %f not in expected range [0.99, 1.0]", upper)
	}
}

func TestWilsonInterval_SmallSample(t *testing.T) {
	// Small sample size should have wider interval
	lower, upper := stats.WilsonInterval(5, 10, 0.95)

	// Width should be significant for small samples
	width := upper - lower
	if width < 0.3 {
		t.Errorf("interval width %f too narrow for small sample", width)
	}
}

func TestZScore(t *testing.T) {
	tests := []struct {
		confidence float64
		expected   float64
		tolerance  float64
	}{
		{0.90, 1.645, 0.01},
		{0.95, 1.96, 0.01},
		{0.99, 2.576, 0.01},
	}

	for _, tt := range tests {
		z := stats.ZScore(tt.confidence)
		if math.Abs(z-tt.expected) > tt.tolerance {
			t.Errorf("ZScore(%f) = %f, want %f (tolerance %f)", tt.confidence, z, tt.expected, tt.tolerance)
		}
	}
}
