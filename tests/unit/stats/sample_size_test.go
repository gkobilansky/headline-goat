package stats_test

import (
	"testing"

	"github.com/gkobilansky/headline-goat/internal/stats"
)

func TestRequiredSampleSize_FivePercentBaseline(t *testing.T) {
	// 5% baseline, 20% relative MDE (1% absolute)
	// Standard statistical formula yields ~8000 per variant
	n := stats.RequiredSampleSize(0.05, 0.20)
	if n < 7000 || n > 9000 {
		t.Errorf("expected ~8000, got %d", n)
	}
}

func TestRequiredSampleSize_TenPercentBaseline(t *testing.T) {
	// 10% baseline, 20% relative MDE (2% absolute)
	// Higher baseline with larger absolute difference needs fewer samples
	n := stats.RequiredSampleSize(0.10, 0.20)
	if n < 3000 || n > 5000 {
		t.Errorf("expected ~4000, got %d", n)
	}
}

func TestRequiredSampleSize_ZeroBaseline(t *testing.T) {
	// Zero baseline should use default assumption (5%)
	n := stats.RequiredSampleSize(0, 0.20)
	zeroDefault := stats.RequiredSampleSize(0.05, 0.20)
	if n != zeroDefault {
		t.Errorf("expected zero baseline to match 5%% default, got %d vs %d", n, zeroDefault)
	}
}

func TestRequiredSampleSize_LowBaseline(t *testing.T) {
	// 2% baseline needs more samples (smaller absolute difference)
	n := stats.RequiredSampleSize(0.02, 0.20)
	if n < 18000 || n > 24000 {
		t.Errorf("expected ~21000, got %d", n)
	}
}

func TestRequiredSampleSize_HigherBaselineNeedsFewerSamples(t *testing.T) {
	// With same MDE, higher baseline needs fewer samples
	// because absolute effect is larger
	n5 := stats.RequiredSampleSize(0.05, 0.20)
	n10 := stats.RequiredSampleSize(0.10, 0.20)
	n20 := stats.RequiredSampleSize(0.20, 0.20)

	if n10 >= n5 {
		t.Errorf("10%% baseline should need fewer samples than 5%%: %d vs %d", n10, n5)
	}
	if n20 >= n10 {
		t.Errorf("20%% baseline should need fewer samples than 10%%: %d vs %d", n20, n10)
	}
}

func TestRequiredSampleSize_RoundsUpToNearest50(t *testing.T) {
	// Result should be rounded up to nearest 50
	n := stats.RequiredSampleSize(0.05, 0.20)
	if n%50 != 0 {
		t.Errorf("expected result to be multiple of 50, got %d", n)
	}
}

func TestRequiredSampleSize_LargerMDENeedsFewer(t *testing.T) {
	// Larger MDE (bigger effect to detect) needs fewer samples
	nSmall := stats.RequiredSampleSize(0.10, 0.10) // 10% lift
	nLarge := stats.RequiredSampleSize(0.10, 0.50) // 50% lift

	if nLarge >= nSmall {
		t.Errorf("larger MDE should need fewer samples: %d vs %d", nLarge, nSmall)
	}
}
