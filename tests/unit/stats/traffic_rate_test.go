package stats_test

import (
	"testing"
	"time"

	"github.com/gkobilansky/headline-goat/internal/stats"
)

func TestCalculateTrafficRate_NormalCase(t *testing.T) {
	// 100 views in 2 hours = 50 views/hour
	firstEvent := time.Now().Add(-2 * time.Hour)
	rate := stats.CalculateTrafficRate(100, firstEvent, 24)

	if rate < 49 || rate > 51 {
		t.Errorf("expected ~50 views/hour, got %f", rate)
	}
}

func TestCalculateTrafficRate_MinimumOneHour(t *testing.T) {
	// 100 views in 10 minutes should use minimum 1 hour
	firstEvent := time.Now().Add(-10 * time.Minute)
	rate := stats.CalculateTrafficRate(100, firstEvent, 24)

	if rate != 100 {
		t.Errorf("expected 100 views/hour (minimum 1 hour), got %f", rate)
	}
}

func TestCalculateTrafficRate_CappedAtWindow(t *testing.T) {
	// 100 views, but window is 24 hours and data is older
	firstEvent := time.Now().Add(-48 * time.Hour)
	rate := stats.CalculateTrafficRate(100, firstEvent, 24)

	// Should cap elapsed time at 24 hours
	expected := 100.0 / 24.0 // ~4.17
	if rate < 4 || rate > 5 {
		t.Errorf("expected ~4.17 views/hour, got %f (expected %f)", rate, expected)
	}
}

func TestCalculateTrafficRate_ZeroViews(t *testing.T) {
	firstEvent := time.Now().Add(-2 * time.Hour)
	rate := stats.CalculateTrafficRate(0, firstEvent, 24)

	if rate != 0 {
		t.Errorf("expected 0 views/hour, got %f", rate)
	}
}

func TestCalculateTrafficRate_ZeroTime(t *testing.T) {
	// Zero time indicates no events
	rate := stats.CalculateTrafficRate(0, time.Time{}, 24)

	if rate != 0 {
		t.Errorf("expected 0 views/hour for zero time, got %f", rate)
	}
}
