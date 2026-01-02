package store_test

import (
	"context"
	"testing"

	"github.com/headline-goat/headline-goat/internal/store"
)

func TestSetSetting(t *testing.T) {
	s, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	err := s.SetSetting(ctx, "server_url", "https://ab.example.com")
	if err != nil {
		t.Fatalf("failed to set setting: %v", err)
	}
}

func TestGetSetting(t *testing.T) {
	s, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Set a value
	err := s.SetSetting(ctx, "server_url", "https://ab.example.com")
	if err != nil {
		t.Fatalf("failed to set setting: %v", err)
	}

	// Get it back
	value, err := s.GetSetting(ctx, "server_url")
	if err != nil {
		t.Fatalf("failed to get setting: %v", err)
	}

	if value != "https://ab.example.com" {
		t.Errorf("got %q, want %q", value, "https://ab.example.com")
	}
}

func TestGetSetting_NotFound(t *testing.T) {
	s, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	_, err := s.GetSetting(ctx, "nonexistent")
	if err != store.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestSetSetting_Update(t *testing.T) {
	s, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Set initial value
	err := s.SetSetting(ctx, "framework", "react")
	if err != nil {
		t.Fatalf("failed to set setting: %v", err)
	}

	// Update it
	err = s.SetSetting(ctx, "framework", "vue")
	if err != nil {
		t.Fatalf("failed to update setting: %v", err)
	}

	// Verify update
	value, err := s.GetSetting(ctx, "framework")
	if err != nil {
		t.Fatalf("failed to get setting: %v", err)
	}

	if value != "vue" {
		t.Errorf("got %q, want %q", value, "vue")
	}
}

func TestGetSettings_Multiple(t *testing.T) {
	s, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Set multiple settings
	_ = s.SetSetting(ctx, "server_url", "https://ab.example.com")
	_ = s.SetSetting(ctx, "framework", "react")

	// Verify both
	serverURL, _ := s.GetSetting(ctx, "server_url")
	framework, _ := s.GetSetting(ctx, "framework")

	if serverURL != "https://ab.example.com" {
		t.Errorf("server_url: got %q, want %q", serverURL, "https://ab.example.com")
	}
	if framework != "react" {
		t.Errorf("framework: got %q, want %q", framework, "react")
	}
}
