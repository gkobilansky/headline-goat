package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gkobilansky/headline-goat/internal/server"
	"github.com/gkobilansky/headline-goat/internal/store"
)

func TestTestsAPI_ReturnsTestsByURL(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	s, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer s.Close()

	// Create a test with URL
	ctx := context.Background()
	_, err = s.CreateTest(ctx, "hero", []string{"A", "B"}, nil, "")
	if err != nil {
		t.Fatalf("failed to create test: %v", err)
	}
	err = s.SetTestURLFields(ctx, "hero", "/", "h1", "", "")
	if err != nil {
		t.Fatalf("failed to set URL fields: %v", err)
	}

	// Start server
	srv := server.New(s, 0, "")

	// Test API endpoint
	req := httptest.NewRequest(http.MethodGet, "/api/tests?url=/", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Parse response
	var tests []struct {
		Name     string   `json:"name"`
		Variants []string `json:"variants"`
		Target   string   `json:"target"`
	}
	if err := json.NewDecoder(w.Body).Decode(&tests); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(tests) != 1 {
		t.Fatalf("expected 1 test, got %d", len(tests))
	}

	if tests[0].Name != "hero" {
		t.Errorf("expected test name 'hero', got %s", tests[0].Name)
	}

	if tests[0].Target != "h1" {
		t.Errorf("expected target 'h1', got %s", tests[0].Target)
	}
}

func TestTestsAPI_ReturnsEmptyForNoMatch(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	s, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer s.Close()

	srv := server.New(s, 0, "")

	req := httptest.NewRequest(http.MethodGet, "/api/tests?url=/nonexistent", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var tests []interface{}
	if err := json.NewDecoder(w.Body).Decode(&tests); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(tests) != 0 {
		t.Errorf("expected 0 tests, got %d", len(tests))
	}
}

func TestTestsAPI_RequiresURLParam(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	s, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer s.Close()

	srv := server.New(s, 0, "")

	req := httptest.NewRequest(http.MethodGet, "/api/tests", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestTestsAPI_CORS(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	s, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer s.Close()

	srv := server.New(s, 0, "")

	req := httptest.NewRequest(http.MethodGet, "/api/tests?url=/", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("expected CORS header to be set")
	}
}
