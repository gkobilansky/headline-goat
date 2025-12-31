package store_test

import (
	"context"
	"testing"

	"github.com/headline-goat/headline-goat/internal/store"
)

func TestGetOrCreateTest_CreatesNew(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	s, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	// GetOrCreate should create new test
	test, created, err := s.GetOrCreateTest(ctx, "hero", []string{"Ship Faster", "Build Better"})
	if err != nil {
		t.Fatalf("GetOrCreateTest failed: %v", err)
	}

	if !created {
		t.Error("expected test to be created")
	}

	if test.Name != "hero" {
		t.Errorf("expected name 'hero', got %s", test.Name)
	}

	if test.Source != "client" {
		t.Errorf("expected source 'client', got %s", test.Source)
	}

	if len(test.Variants) != 2 {
		t.Errorf("expected 2 variants, got %d", len(test.Variants))
	}
}

func TestGetOrCreateTest_ReturnsExisting(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	s, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	// Create test first via CLI (source=server)
	_, err = s.CreateTest(ctx, "hero", []string{"A", "B"}, nil, "")
	if err != nil {
		t.Fatalf("CreateTest failed: %v", err)
	}

	// GetOrCreate should return existing
	test, created, err := s.GetOrCreateTest(ctx, "hero", []string{"X", "Y"})
	if err != nil {
		t.Fatalf("GetOrCreateTest failed: %v", err)
	}

	if created {
		t.Error("expected test NOT to be created (already exists)")
	}

	// Should have original variants, not the new ones
	if test.Variants[0] != "A" {
		t.Errorf("expected original variant 'A', got %s", test.Variants[0])
	}

	// Should have source=server (from CLI creation)
	if test.Source != "server" {
		t.Errorf("expected source 'server', got %s", test.Source)
	}
}

func TestCreateTest_HasServerSource(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	s, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	// Create via CLI (CreateTest)
	test, err := s.CreateTest(ctx, "hero", []string{"A", "B"}, nil, "")
	if err != nil {
		t.Fatalf("CreateTest failed: %v", err)
	}

	if test.Source != "server" {
		t.Errorf("expected source 'server', got %s", test.Source)
	}

	// Verify persisted
	test, err = s.GetTest(ctx, "hero")
	if err != nil {
		t.Fatalf("GetTest failed: %v", err)
	}

	if test.Source != "server" {
		t.Errorf("expected persisted source 'server', got %s", test.Source)
	}
}

func TestSetSourceConflict(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	s, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	// Create test
	_, err = s.CreateTest(ctx, "hero", []string{"A", "B"}, nil, "")
	if err != nil {
		t.Fatalf("CreateTest failed: %v", err)
	}

	// Set conflict
	err = s.SetSourceConflict(ctx, "hero", true)
	if err != nil {
		t.Fatalf("SetSourceConflict failed: %v", err)
	}

	// Verify
	test, err := s.GetTest(ctx, "hero")
	if err != nil {
		t.Fatalf("GetTest failed: %v", err)
	}

	if !test.HasSourceConflict {
		t.Error("expected HasSourceConflict to be true")
	}

	// Clear conflict
	err = s.SetSourceConflict(ctx, "hero", false)
	if err != nil {
		t.Fatalf("SetSourceConflict failed: %v", err)
	}

	test, _ = s.GetTest(ctx, "hero")
	if test.HasSourceConflict {
		t.Error("expected HasSourceConflict to be false")
	}
}

func TestNewColumnsDefault(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	s, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	// Create test
	test, err := s.CreateTest(ctx, "hero", []string{"A", "B"}, nil, "")
	if err != nil {
		t.Fatalf("CreateTest failed: %v", err)
	}

	// Verify new fields have sensible defaults
	if test.URL != "" {
		t.Errorf("expected URL to be empty, got %s", test.URL)
	}
	if test.ConversionURL != "" {
		t.Errorf("expected ConversionURL to be empty, got %s", test.ConversionURL)
	}
	if test.Target != "" {
		t.Errorf("expected Target to be empty, got %s", test.Target)
	}
	if test.CTATarget != "" {
		t.Errorf("expected CTATarget to be empty, got %s", test.CTATarget)
	}
	if test.HasSourceConflict {
		t.Error("expected HasSourceConflict to be false")
	}
}
