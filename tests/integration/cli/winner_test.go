package cli_test

import (
	"context"
	"testing"

	"github.com/gkobilansky/headline-goat/internal/store"
	"github.com/gkobilansky/headline-goat/tests/testutil"
)

func TestSetWinner_Success(t *testing.T) {
	s := testutil.SetupTestStore(t)

	ctx := context.Background()
	_, err := s.CreateTest(ctx, "hero", []string{"Ship Faster", "Build Better"}, nil, "")
	if err != nil {
		t.Fatalf("failed to create test: %v", err)
	}

	// Set winner
	err = s.SetWinner(ctx, "hero", 1)
	if err != nil {
		t.Fatalf("SetWinner failed: %v", err)
	}

	// Verify state
	test, err := s.GetTest(ctx, "hero")
	if err != nil {
		t.Fatalf("failed to get test: %v", err)
	}

	if test.State != store.StateCompleted {
		t.Errorf("expected state to be 'completed', got %s", test.State)
	}

	if test.WinnerVariant == nil {
		t.Fatal("expected winner variant to be set")
	}

	if *test.WinnerVariant != 1 {
		t.Errorf("expected winner variant to be 1, got %d", *test.WinnerVariant)
	}
}

func TestSetWinner_TestNotFound(t *testing.T) {
	s := testutil.SetupTestStore(t)

	ctx := context.Background()

	// Try to set winner for non-existent test
	err := s.SetWinner(ctx, "nonexistent", 0)
	if err == nil {
		t.Error("expected SetWinner to fail for non-existent test")
	}

	if err != store.ErrNotFound {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestSetWinner_MultipleTimes(t *testing.T) {
	s := testutil.SetupTestStore(t)

	ctx := context.Background()
	_, err := s.CreateTest(ctx, "hero", []string{"A", "B", "C"}, nil, "")
	if err != nil {
		t.Fatalf("failed to create test: %v", err)
	}

	// Set winner first time
	err = s.SetWinner(ctx, "hero", 0)
	if err != nil {
		t.Fatalf("first SetWinner failed: %v", err)
	}

	// Try to set winner again (this should still work at the store level)
	err = s.SetWinner(ctx, "hero", 2)
	if err != nil {
		t.Fatalf("second SetWinner failed: %v", err)
	}

	// Verify final state
	test, _ := s.GetTest(ctx, "hero")
	if test.WinnerVariant == nil || *test.WinnerVariant != 2 {
		t.Error("expected winner variant to be updated to 2")
	}
}

func TestWinnerStateTransition(t *testing.T) {
	s := testutil.SetupTestStore(t)

	ctx := context.Background()
	test, err := s.CreateTest(ctx, "hero", []string{"A", "B"}, nil, "")
	if err != nil {
		t.Fatalf("failed to create test: %v", err)
	}

	// Verify initial state is running
	if test.State != store.StateRunning {
		t.Errorf("expected initial state to be 'running', got %s", test.State)
	}

	// Set winner
	err = s.SetWinner(ctx, "hero", 0)
	if err != nil {
		t.Fatalf("SetWinner failed: %v", err)
	}

	// Verify state transitioned to completed
	test, _ = s.GetTest(ctx, "hero")
	if test.State != store.StateCompleted {
		t.Errorf("expected state to be 'completed', got %s", test.State)
	}
}

func TestWinnerVariantPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	// Create and set winner
	s, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}

	ctx := context.Background()
	_, err = s.CreateTest(ctx, "hero", []string{"Option A", "Option B", "Option C"}, nil, "")
	if err != nil {
		t.Fatalf("failed to create test: %v", err)
	}
	err = s.SetWinner(ctx, "hero", 1)
	if err != nil {
		t.Fatalf("SetWinner failed: %v", err)
	}
	s.Close()

	// Reopen database and verify persistence
	s2, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to reopen store: %v", err)
	}
	defer s2.Close()

	test, err := s2.GetTest(ctx, "hero")
	if err != nil {
		t.Fatalf("failed to get test: %v", err)
	}

	if test.State != store.StateCompleted {
		t.Errorf("expected persisted state to be 'completed', got %s", test.State)
	}

	if test.WinnerVariant == nil || *test.WinnerVariant != 1 {
		t.Error("expected persisted winner variant to be 1")
	}
}
