package stats_test

import (
	"strings"
	"testing"
	"time"

	"github.com/gkobilansky/headline-goat/internal/stats"
)

func TestEstimateStatus_ReadyWhenSignificant(t *testing.T) {
	result := &stats.Result{
		Confident:       true,
		ConfidenceLevel: 0.97,
		LeadingVariant:  1,
	}

	status := stats.EstimateStatus(result, 390, 50.0)

	if !status.Ready {
		t.Error("expected Ready=true when significant")
	}
	if status.RecommendedVariant != 1 {
		t.Errorf("expected RecommendedVariant=1, got %d", status.RecommendedVariant)
	}
}

func TestEstimateStatus_CalculatesHoursCorrectly(t *testing.T) {
	result := &stats.Result{
		Confident: false,
		Variants: []stats.VariantResult{
			{Views: 100}, {Views: 100},
		},
	}

	// Need 390 per variant, have 100 (min), traffic 50/hour
	// Need 290 more, at 50/hour = 5.8 hours
	status := stats.EstimateStatus(result, 390, 50.0)

	if status.EstimatedHours < 5 || status.EstimatedHours > 7 {
		t.Errorf("expected ~5.8 hours, got %f", status.EstimatedHours)
	}
}

func TestEstimateStatus_ZeroTrafficRate(t *testing.T) {
	result := &stats.Result{
		Confident: false,
		Variants:  []stats.VariantResult{{Views: 10}, {Views: 10}},
	}

	status := stats.EstimateStatus(result, 390, 0)

	if status.EstimatedHours != 0 {
		t.Errorf("expected 0 hours when no traffic, got %f", status.EstimatedHours)
	}
	if status.Message == "" {
		t.Error("expected non-empty message")
	}
}

func TestEstimateStatus_TracksCurrentMinViews(t *testing.T) {
	result := &stats.Result{
		Confident: false,
		Variants: []stats.VariantResult{
			{Views: 100},
			{Views: 50}, // minimum
		},
	}

	status := stats.EstimateStatus(result, 390, 50.0)

	if status.CurrentMinViews != 50 {
		t.Errorf("expected CurrentMinViews=50, got %d", status.CurrentMinViews)
	}
}

func TestEstimateStatus_ViewsRemaining(t *testing.T) {
	result := &stats.Result{
		Confident: false,
		Variants: []stats.VariantResult{
			{Views: 100},
			{Views: 100},
		},
	}

	status := stats.EstimateStatus(result, 390, 50.0)

	// Need 390, have 100, remaining = 290
	if status.ViewsRemaining != 290 {
		t.Errorf("expected ViewsRemaining=290, got %d", status.ViewsRemaining)
	}
}

func TestEstimateStatus_CheckBackTimeSet(t *testing.T) {
	result := &stats.Result{
		Confident: false,
		Variants: []stats.VariantResult{
			{Views: 100}, {Views: 100},
		},
	}

	status := stats.EstimateStatus(result, 390, 50.0)

	if status.CheckBackTime.IsZero() {
		t.Error("expected CheckBackTime to be set when traffic rate available")
	}

	// Should be approximately 5-6 hours from now
	hoursUntil := time.Until(status.CheckBackTime).Hours()
	if hoursUntil < 4 || hoursUntil > 8 {
		t.Errorf("expected CheckBackTime ~6 hours from now, got %f", hoursUntil)
	}
}

func TestEstimateStatus_MessageForReady(t *testing.T) {
	result := &stats.Result{
		Confident:       true,
		ConfidenceLevel: 0.97,
		LeadingVariant:  1,
	}

	status := stats.EstimateStatus(result, 390, 50.0)

	if !strings.Contains(status.Message, "winner") {
		t.Errorf("expected message to mention winner, got: %s", status.Message)
	}
}

func TestEstimateStatus_MessageForProgress(t *testing.T) {
	result := &stats.Result{
		Confident:       false,
		ConfidenceLevel: 0.87,
		LeadingVariant:  1,
		Variants: []stats.VariantResult{
			{Views: 100}, {Views: 100},
		},
	}

	status := stats.EstimateStatus(result, 390, 50.0)

	// Should mention the confidence level or check back time
	if !strings.Contains(status.Message, "87") && !strings.Contains(status.Message, "Check back") {
		t.Errorf("expected message to show progress, got: %s", status.Message)
	}
}

func TestEstimateStatus_RecommendedVariantNegativeWhenNotReady(t *testing.T) {
	result := &stats.Result{
		Confident:      false,
		LeadingVariant: 1,
	}

	status := stats.EstimateStatus(result, 390, 50.0)

	if status.RecommendedVariant != -1 {
		t.Errorf("expected RecommendedVariant=-1 when not ready, got %d", status.RecommendedVariant)
	}
}
