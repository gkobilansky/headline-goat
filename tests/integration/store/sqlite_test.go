package store_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/headline-goat/headline-goat/internal/store"
)

func setupTestDB(t *testing.T) (*store.SQLiteStore, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "headline-goat-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	dbPath := filepath.Join(tmpDir, "test.db")

	s, err := store.Open(dbPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to open store: %v", err)
	}

	cleanup := func() {
		s.Close()
		os.RemoveAll(tmpDir)
	}

	return s, cleanup
}

func TestOpen(t *testing.T) {
	s, cleanup := setupTestDB(t)
	defer cleanup()

	if s == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestCreateTest(t *testing.T) {
	s, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	test, err := s.CreateTest(ctx, "hero", []string{"A", "B", "C"}, nil, "")
	if err != nil {
		t.Fatalf("failed to create test: %v", err)
	}

	if test.Name != "hero" {
		t.Errorf("got Name %s, want hero", test.Name)
	}
	if len(test.Variants) != 3 {
		t.Errorf("got %d variants, want 3", len(test.Variants))
	}
	if test.State != store.StateRunning {
		t.Errorf("got State %s, want running", test.State)
	}
	if test.ID == 0 {
		t.Error("expected non-zero ID")
	}
}

func TestCreateTest_WithWeightsAndGoal(t *testing.T) {
	s, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	weights := []float64{0.5, 0.3, 0.2}
	test, err := s.CreateTest(ctx, "pricing", []string{"A", "B", "C"}, weights, "Signup button click")
	if err != nil {
		t.Fatalf("failed to create test: %v", err)
	}

	if len(test.Weights) != 3 {
		t.Errorf("got %d weights, want 3", len(test.Weights))
	}
	if test.Weights[0] != 0.5 {
		t.Errorf("got first weight %f, want 0.5", test.Weights[0])
	}
	if test.ConversionGoal != "Signup button click" {
		t.Errorf("got ConversionGoal %s, want 'Signup button click'", test.ConversionGoal)
	}
}

func TestCreateTest_DuplicateName(t *testing.T) {
	s, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	_, err := s.CreateTest(ctx, "hero", []string{"A", "B"}, nil, "")
	if err != nil {
		t.Fatalf("failed to create first test: %v", err)
	}

	_, err = s.CreateTest(ctx, "hero", []string{"X", "Y"}, nil, "")
	if err == nil {
		t.Fatal("expected error for duplicate name")
	}
}

func TestGetTest(t *testing.T) {
	s, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	_, err := s.CreateTest(ctx, "hero", []string{"A", "B"}, nil, "Click the button")
	if err != nil {
		t.Fatalf("failed to create test: %v", err)
	}

	test, err := s.GetTest(ctx, "hero")
	if err != nil {
		t.Fatalf("failed to get test: %v", err)
	}

	if test.Name != "hero" {
		t.Errorf("got Name %s, want hero", test.Name)
	}
	if test.ConversionGoal != "Click the button" {
		t.Errorf("got ConversionGoal %s, want 'Click the button'", test.ConversionGoal)
	}
}

func TestGetTest_NotFound(t *testing.T) {
	s, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	_, err := s.GetTest(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent test")
	}
}

func TestListTests(t *testing.T) {
	s, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	_, _ = s.CreateTest(ctx, "hero", []string{"A", "B"}, nil, "")
	_, _ = s.CreateTest(ctx, "pricing", []string{"X", "Y", "Z"}, nil, "")

	tests, err := s.ListTests(ctx)
	if err != nil {
		t.Fatalf("failed to list tests: %v", err)
	}

	if len(tests) != 2 {
		t.Errorf("got %d tests, want 2", len(tests))
	}
}

func TestUpdateTestState(t *testing.T) {
	s, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	_, err := s.CreateTest(ctx, "hero", []string{"A", "B"}, nil, "")
	if err != nil {
		t.Fatalf("failed to create test: %v", err)
	}

	winner := 1
	err = s.UpdateTestState(ctx, "hero", store.StateCompleted, &winner)
	if err != nil {
		t.Fatalf("failed to update test state: %v", err)
	}

	test, err := s.GetTest(ctx, "hero")
	if err != nil {
		t.Fatalf("failed to get test: %v", err)
	}

	if test.State != store.StateCompleted {
		t.Errorf("got State %s, want completed", test.State)
	}
	if test.WinnerVariant == nil || *test.WinnerVariant != 1 {
		t.Error("expected WinnerVariant to be 1")
	}
}

func TestDeleteTest(t *testing.T) {
	s, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	_, err := s.CreateTest(ctx, "hero", []string{"A", "B"}, nil, "")
	if err != nil {
		t.Fatalf("failed to create test: %v", err)
	}

	err = s.DeleteTest(ctx, "hero")
	if err != nil {
		t.Fatalf("failed to delete test: %v", err)
	}

	_, err = s.GetTest(ctx, "hero")
	if err == nil {
		t.Fatal("expected error for deleted test")
	}
}

func TestRecordEvent(t *testing.T) {
	s, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	_, err := s.CreateTest(ctx, "hero", []string{"A", "B"}, nil, "")
	if err != nil {
		t.Fatalf("failed to create test: %v", err)
	}

	err = s.RecordEvent(ctx, "hero", 0, "view", "visitor1")
	if err != nil {
		t.Fatalf("failed to record event: %v", err)
	}

	err = s.RecordEvent(ctx, "hero", 0, "convert", "visitor1")
	if err != nil {
		t.Fatalf("failed to record event: %v", err)
	}
}

func TestRecordEvent_Deduplication(t *testing.T) {
	s, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	_, err := s.CreateTest(ctx, "hero", []string{"A", "B"}, nil, "")
	if err != nil {
		t.Fatalf("failed to create test: %v", err)
	}

	// Record same event twice
	_ = s.RecordEvent(ctx, "hero", 0, "view", "visitor1")
	_ = s.RecordEvent(ctx, "hero", 0, "view", "visitor1")

	stats, err := s.GetVariantStats(ctx, "hero")
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}

	if len(stats) != 1 {
		t.Fatalf("expected 1 variant stat, got %d", len(stats))
	}
	if stats[0].Views != 1 {
		t.Errorf("got Views %d, want 1 (deduplication failed)", stats[0].Views)
	}
}

func TestGetVariantStats(t *testing.T) {
	s, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	_, err := s.CreateTest(ctx, "hero", []string{"A", "B"}, nil, "")
	if err != nil {
		t.Fatalf("failed to create test: %v", err)
	}

	// Record events for variant 0
	_ = s.RecordEvent(ctx, "hero", 0, "view", "v1")
	_ = s.RecordEvent(ctx, "hero", 0, "view", "v2")
	_ = s.RecordEvent(ctx, "hero", 0, "convert", "v1")

	// Record events for variant 1
	_ = s.RecordEvent(ctx, "hero", 1, "view", "v3")
	_ = s.RecordEvent(ctx, "hero", 1, "convert", "v3")

	stats, err := s.GetVariantStats(ctx, "hero")
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}

	if len(stats) != 2 {
		t.Fatalf("expected 2 variant stats, got %d", len(stats))
	}

	// Find variant 0 stats
	var v0, v1 store.VariantStats
	for _, s := range stats {
		if s.Variant == 0 {
			v0 = s
		} else if s.Variant == 1 {
			v1 = s
		}
	}

	if v0.Views != 2 {
		t.Errorf("variant 0: got Views %d, want 2", v0.Views)
	}
	if v0.Conversions != 1 {
		t.Errorf("variant 0: got Conversions %d, want 1", v0.Conversions)
	}
	if v1.Views != 1 {
		t.Errorf("variant 1: got Views %d, want 1", v1.Views)
	}
	if v1.Conversions != 1 {
		t.Errorf("variant 1: got Conversions %d, want 1", v1.Conversions)
	}
}

func TestGetEvents(t *testing.T) {
	s, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	_, err := s.CreateTest(ctx, "hero", []string{"A", "B"}, nil, "")
	if err != nil {
		t.Fatalf("failed to create test: %v", err)
	}

	_ = s.RecordEvent(ctx, "hero", 0, "view", "v1")
	_ = s.RecordEvent(ctx, "hero", 1, "convert", "v2")

	events, err := s.GetEvents(ctx, "hero")
	if err != nil {
		t.Fatalf("failed to get events: %v", err)
	}

	if len(events) != 2 {
		t.Errorf("got %d events, want 2", len(events))
	}
}
