package cli_test

import (
	"context"
	"testing"

	"github.com/gkobilansky/headline-goat/internal/store"
	"github.com/gkobilansky/headline-goat/tests/testutil"
)

func TestCreateTest_Success(t *testing.T) {
	s := testutil.SetupTestStore(t)

	ctx := context.Background()

	// Create test with variants
	test, err := s.CreateTest(ctx, "hero", []string{"Ship Faster", "Build Better"}, nil, "")
	if err != nil {
		t.Fatalf("CreateTest failed: %v", err)
	}

	// Verify test properties
	if test.Name != "hero" {
		t.Errorf("expected name 'hero', got %s", test.Name)
	}

	if len(test.Variants) != 2 {
		t.Errorf("expected 2 variants, got %d", len(test.Variants))
	}

	if test.Variants[0] != "Ship Faster" {
		t.Errorf("expected first variant 'Ship Faster', got %s", test.Variants[0])
	}

	if test.Variants[1] != "Build Better" {
		t.Errorf("expected second variant 'Build Better', got %s", test.Variants[1])
	}

	if test.State != store.StateRunning {
		t.Errorf("expected state 'running', got %s", test.State)
	}
}

func TestCreateTest_DuplicateName(t *testing.T) {
	s := testutil.SetupTestStore(t)

	ctx := context.Background()

	// Create first test
	_, err := s.CreateTest(ctx, "hero", []string{"A", "B"}, nil, "")
	if err != nil {
		t.Fatalf("first CreateTest failed: %v", err)
	}

	// Try to create duplicate - should fail
	_, err = s.CreateTest(ctx, "hero", []string{"C", "D"}, nil, "")
	if err == nil {
		t.Fatal("expected error for duplicate name, got nil")
	}
}

func TestCreateTest_VerifyInList(t *testing.T) {
	s := testutil.SetupTestStore(t)

	ctx := context.Background()

	// Create test
	_, err := s.CreateTest(ctx, "hero", []string{"A", "B"}, nil, "")
	if err != nil {
		t.Fatalf("CreateTest failed: %v", err)
	}

	// List tests
	tests, err := s.ListTests(ctx)
	if err != nil {
		t.Fatalf("ListTests failed: %v", err)
	}

	if len(tests) != 1 {
		t.Fatalf("expected 1 test, got %d", len(tests))
	}

	if tests[0].Name != "hero" {
		t.Errorf("expected test name 'hero', got %s", tests[0].Name)
	}
}

func TestCreateTest_ThreeVariants(t *testing.T) {
	s := testutil.SetupTestStore(t)

	ctx := context.Background()

	// Create test with 3 variants
	test, err := s.CreateTest(ctx, "cta", []string{"Sign Up", "Get Started", "Try Free"}, nil, "")
	if err != nil {
		t.Fatalf("CreateTest failed: %v", err)
	}

	if len(test.Variants) != 3 {
		t.Errorf("expected 3 variants, got %d", len(test.Variants))
	}
}

func TestCreateTest_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	// Create test and close connection
	s, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}

	ctx := context.Background()
	_, err = s.CreateTest(ctx, "hero", []string{"A", "B"}, nil, "")
	if err != nil {
		t.Fatalf("CreateTest failed: %v", err)
	}
	s.Close()

	// Reopen and verify
	s2, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to reopen store: %v", err)
	}
	defer s2.Close()

	test, err := s2.GetTest(ctx, "hero")
	if err != nil {
		t.Fatalf("GetTest failed: %v", err)
	}

	if test.Name != "hero" {
		t.Errorf("expected name 'hero', got %s", test.Name)
	}
}
