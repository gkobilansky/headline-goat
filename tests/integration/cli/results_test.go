package cli_test

import (
	"context"
	"testing"

	"github.com/gkobilansky/headline-goat/internal/stats"
	"github.com/gkobilansky/headline-goat/tests/testutil"
)

// Test the status estimate integration with results data
func TestStatusEstimate_Integration(t *testing.T) {
	s := testutil.SetupTestStore(t)
	ctx := context.Background()

	// Create test with variants
	test, err := s.CreateTest(ctx, "hero", []string{"A", "B"}, nil, "")
	if err != nil {
		t.Fatalf("failed to create test: %v", err)
	}

	// Add some events
	_ = s.RecordEvent(ctx, "hero", 0, "view", "v1")
	_ = s.RecordEvent(ctx, "hero", 0, "view", "v2")
	_ = s.RecordEvent(ctx, "hero", 1, "view", "v3")
	_ = s.RecordEvent(ctx, "hero", 0, "convert", "v1")

	// Get stats
	variantStats, err := s.GetVariantStats(ctx, "hero")
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}

	// Analyze
	result := stats.Analyze(test, variantStats)

	// Get traffic rate
	viewCount, firstEvent, err := s.GetRecentViewCount(ctx, "hero", 24)
	if err != nil {
		t.Fatalf("failed to get view count: %v", err)
	}
	trafficRate := stats.CalculateTrafficRate(viewCount, firstEvent, 24)

	// Calculate sample size needed (using baseline rate from control)
	baselineRate := 0.0
	if result.Variants[0].Views > 0 {
		baselineRate = result.Variants[0].Rate
	}
	viewsNeeded := stats.RequiredSampleSize(baselineRate, 0.20)

	// Get status estimate
	status := stats.EstimateStatus(result, viewsNeeded, trafficRate)

	// Verify status fields
	if status.CurrentMinViews != 1 {
		t.Errorf("expected CurrentMinViews=1 (min of 2 and 1), got %d", status.CurrentMinViews)
	}

	if status.Ready {
		t.Error("expected Ready=false with minimal data")
	}

	if status.Message == "" {
		t.Error("expected non-empty status message")
	}

	if trafficRate <= 0 {
		t.Error("expected positive traffic rate")
	}
}
