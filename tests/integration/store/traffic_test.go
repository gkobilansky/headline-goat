package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/gkobilansky/headline-goat/tests/testutil"
)

func TestGetRecentViewCount_WithEvents(t *testing.T) {
	s := testutil.SetupTestStore(t)
	ctx := context.Background()

	_, _ = s.CreateTest(ctx, "hero", []string{"A", "B"}, nil, "")
	_ = s.RecordEvent(ctx, "hero", 0, "view", "v1")
	_ = s.RecordEvent(ctx, "hero", 0, "view", "v2")
	_ = s.RecordEvent(ctx, "hero", 1, "view", "v3")

	count, _, err := s.GetRecentViewCount(ctx, "hero", 24)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 views, got %d", count)
	}
}

func TestGetRecentViewCount_NoEvents(t *testing.T) {
	s := testutil.SetupTestStore(t)
	ctx := context.Background()

	_, _ = s.CreateTest(ctx, "hero", []string{"A", "B"}, nil, "")

	count, _, err := s.GetRecentViewCount(ctx, "hero", 24)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 views, got %d", count)
	}
}

func TestGetRecentViewCount_IgnoresConversions(t *testing.T) {
	s := testutil.SetupTestStore(t)
	ctx := context.Background()

	_, _ = s.CreateTest(ctx, "hero", []string{"A", "B"}, nil, "")
	_ = s.RecordEvent(ctx, "hero", 0, "view", "v1")
	_ = s.RecordEvent(ctx, "hero", 0, "convert", "v1")

	count, _, err := s.GetRecentViewCount(ctx, "hero", 24)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 view (not conversion), got %d", count)
	}
}

func TestGetRecentViewCount_ReturnsFirstEventTime(t *testing.T) {
	s := testutil.SetupTestStore(t)
	ctx := context.Background()

	_, _ = s.CreateTest(ctx, "hero", []string{"A", "B"}, nil, "")
	_ = s.RecordEvent(ctx, "hero", 0, "view", "v1")
	time.Sleep(10 * time.Millisecond) // Small delay to ensure different timestamps
	_ = s.RecordEvent(ctx, "hero", 0, "view", "v2")

	count, firstEvent, err := s.GetRecentViewCount(ctx, "hero", 24)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 views, got %d", count)
	}

	// First event time should be recent (within last minute)
	if firstEvent.IsZero() {
		t.Error("expected non-zero first event time")
	}
	if time.Since(firstEvent) > time.Minute {
		t.Errorf("first event time too old: %v", firstEvent)
	}
}

func TestGetRecentViewCount_NoEventsReturnsZeroTime(t *testing.T) {
	s := testutil.SetupTestStore(t)
	ctx := context.Background()

	_, _ = s.CreateTest(ctx, "hero", []string{"A", "B"}, nil, "")

	count, firstEvent, err := s.GetRecentViewCount(ctx, "hero", 24)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 views, got %d", count)
	}
	if !firstEvent.IsZero() {
		t.Errorf("expected zero time when no events, got %v", firstEvent)
	}
}
