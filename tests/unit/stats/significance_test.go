package stats_test

import (
	"testing"

	"github.com/headline-goat/headline-goat/internal/stats"
	"github.com/headline-goat/headline-goat/internal/store"
)

func TestSignificanceTest_ClearWinner(t *testing.T) {
	// Variant A: 10% conversion (100/1000)
	// Variant B: 5% conversion (50/1000)
	// Should be very confident A beats B
	confidence := stats.SignificanceTest(100, 1000, 50, 1000)

	if confidence < 0.95 {
		t.Errorf("expected high confidence (>0.95), got %f", confidence)
	}
}

func TestSignificanceTest_NoSignificance(t *testing.T) {
	// Both variants have same conversion rate
	// Should not be confident either wins
	confidence := stats.SignificanceTest(50, 1000, 50, 1000)

	if confidence > 0.60 {
		t.Errorf("expected low confidence (<0.60) for equal rates, got %f", confidence)
	}
}

func TestSignificanceTest_SmallSample(t *testing.T) {
	// Small samples should not show significance even with different rates
	confidence := stats.SignificanceTest(5, 20, 2, 20)

	if confidence > 0.95 {
		t.Errorf("expected lower confidence for small sample, got %f", confidence)
	}
}

func TestSignificanceTest_ZeroViews(t *testing.T) {
	// Should handle zero views gracefully
	confidence := stats.SignificanceTest(0, 0, 0, 0)

	if confidence != 0.5 {
		t.Errorf("expected 0.5 for zero views, got %f", confidence)
	}
}

func TestSignificanceTest_OnlyOneVariantHasViews(t *testing.T) {
	confidence := stats.SignificanceTest(10, 100, 0, 0)

	// Can't determine significance with only one variant
	if confidence > 0.6 || confidence < 0.4 {
		t.Errorf("expected ~0.5 when only one variant has data, got %f", confidence)
	}
}

func TestAnalyze_BasicResults(t *testing.T) {
	test := &store.Test{
		Name:     "hero",
		Variants: []string{"A", "B"},
		State:    store.StateRunning,
	}

	variantStats := []store.VariantStats{
		{Variant: 0, Views: 100, Conversions: 10},
		{Variant: 1, Views: 100, Conversions: 20},
	}

	result := stats.Analyze(test, variantStats)

	if len(result.Variants) != 2 {
		t.Fatalf("expected 2 variants, got %d", len(result.Variants))
	}

	// Check rates
	if result.Variants[0].Rate < 0.09 || result.Variants[0].Rate > 0.11 {
		t.Errorf("variant 0 rate %f not ~0.10", result.Variants[0].Rate)
	}
	if result.Variants[1].Rate < 0.19 || result.Variants[1].Rate > 0.21 {
		t.Errorf("variant 1 rate %f not ~0.20", result.Variants[1].Rate)
	}

	// Variant 1 should be leading
	if result.LeadingVariant != 1 {
		t.Errorf("expected variant 1 to be leading, got %d", result.LeadingVariant)
	}
}

func TestAnalyze_WithConfidenceIntervals(t *testing.T) {
	test := &store.Test{
		Name:     "hero",
		Variants: []string{"A", "B"},
		State:    store.StateRunning,
	}

	variantStats := []store.VariantStats{
		{Variant: 0, Views: 1000, Conversions: 100},
		{Variant: 1, Views: 1000, Conversions: 150},
	}

	result := stats.Analyze(test, variantStats)

	// Check confidence intervals exist and are reasonable
	for i, v := range result.Variants {
		if v.CILower >= v.Rate {
			t.Errorf("variant %d: CI lower %f should be < rate %f", i, v.CILower, v.Rate)
		}
		if v.CIUpper <= v.Rate {
			t.Errorf("variant %d: CI upper %f should be > rate %f", i, v.CIUpper, v.Rate)
		}
		if v.CILower < 0 || v.CIUpper > 1 {
			t.Errorf("variant %d: CI [%f, %f] out of bounds", i, v.CILower, v.CIUpper)
		}
	}
}

func TestAnalyze_EmptyStats(t *testing.T) {
	test := &store.Test{
		Name:     "hero",
		Variants: []string{"A", "B"},
		State:    store.StateRunning,
	}

	var variantStats []store.VariantStats

	result := stats.Analyze(test, variantStats)

	// Should return results with zero stats for all variants
	if len(result.Variants) != 2 {
		t.Fatalf("expected 2 variants even with empty stats, got %d", len(result.Variants))
	}

	for _, v := range result.Variants {
		if v.Views != 0 || v.Conversions != 0 {
			t.Errorf("expected zero views/conversions for empty stats")
		}
	}
}

func TestAnalyze_VariantNames(t *testing.T) {
	test := &store.Test{
		Name:     "hero",
		Variants: []string{"Ship Faster", "Build Better"},
		State:    store.StateRunning,
	}

	variantStats := []store.VariantStats{
		{Variant: 0, Views: 100, Conversions: 10},
		{Variant: 1, Views: 100, Conversions: 20},
	}

	result := stats.Analyze(test, variantStats)

	if result.Variants[0].Name != "Ship Faster" {
		t.Errorf("expected variant 0 name 'Ship Faster', got '%s'", result.Variants[0].Name)
	}
	if result.Variants[1].Name != "Build Better" {
		t.Errorf("expected variant 1 name 'Build Better', got '%s'", result.Variants[1].Name)
	}
}
