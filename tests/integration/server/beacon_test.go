package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/headline-goat/headline-goat/internal/server"
	"github.com/headline-goat/headline-goat/internal/store"
)

func setupTestServer(t *testing.T) (*server.Server, *store.SQLiteStore, func()) {
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

	srv := server.New(s, 8080, "")

	cleanup := func() {
		s.Close()
		os.RemoveAll(tmpDir)
	}

	return srv, s, cleanup
}

func TestBeacon_ValidRequest(t *testing.T) {
	srv, s, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a test first
	ctx := context.Background()
	_, err := s.CreateTest(ctx, "hero", []string{"A", "B", "C"}, nil, "")
	if err != nil {
		t.Fatalf("failed to create test: %v", err)
	}

	// Send beacon
	payload := map[string]interface{}{
		"t":   "hero",
		"v":   1,
		"e":   "view",
		"vid": "visitor123",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/b", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d: %s", w.Code, w.Body.String())
	}

	// Verify event was recorded
	stats, err := s.GetVariantStats(ctx, "hero")
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}

	if len(stats) != 1 {
		t.Fatalf("expected 1 variant stat, got %d", len(stats))
	}
	if stats[0].Views != 1 {
		t.Errorf("expected 1 view, got %d", stats[0].Views)
	}
}

func TestBeacon_ConversionEvent(t *testing.T) {
	srv, s, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	_, err := s.CreateTest(ctx, "hero", []string{"A", "B"}, nil, "")
	if err != nil {
		t.Fatalf("failed to create test: %v", err)
	}

	// Send view first
	viewPayload := map[string]interface{}{
		"t":   "hero",
		"v":   0,
		"e":   "view",
		"vid": "visitor123",
	}
	body, _ := json.Marshal(viewPayload)
	req := httptest.NewRequest(http.MethodPost, "/b", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	// Send conversion
	convertPayload := map[string]interface{}{
		"t":   "hero",
		"v":   0,
		"e":   "convert",
		"vid": "visitor123",
	}
	body, _ = json.Marshal(convertPayload)
	req = httptest.NewRequest(http.MethodPost, "/b", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", w.Code)
	}

	// Verify both events were recorded
	stats, err := s.GetVariantStats(ctx, "hero")
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}

	if stats[0].Views != 1 {
		t.Errorf("expected 1 view, got %d", stats[0].Views)
	}
	if stats[0].Conversions != 1 {
		t.Errorf("expected 1 conversion, got %d", stats[0].Conversions)
	}
}

func TestBeacon_InvalidTest(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	payload := map[string]interface{}{
		"t":   "nonexistent",
		"v":   0,
		"e":   "view",
		"vid": "visitor123",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/b", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestBeacon_InvalidVariant(t *testing.T) {
	srv, s, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	_, err := s.CreateTest(ctx, "hero", []string{"A", "B"}, nil, "")
	if err != nil {
		t.Fatalf("failed to create test: %v", err)
	}

	payload := map[string]interface{}{
		"t":   "hero",
		"v":   5, // Out of range
		"e":   "view",
		"vid": "visitor123",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/b", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestBeacon_Deduplication(t *testing.T) {
	srv, s, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	_, err := s.CreateTest(ctx, "hero", []string{"A", "B"}, nil, "")
	if err != nil {
		t.Fatalf("failed to create test: %v", err)
	}

	payload := map[string]interface{}{
		"t":   "hero",
		"v":   0,
		"e":   "view",
		"vid": "visitor123",
	}
	body, _ := json.Marshal(payload)

	// Send same beacon twice
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/b", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("request %d: expected status 204, got %d", i+1, w.Code)
		}
	}

	// Verify only one event recorded
	stats, err := s.GetVariantStats(ctx, "hero")
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}

	if stats[0].Views != 1 {
		t.Errorf("expected 1 view (deduplication), got %d", stats[0].Views)
	}
}

func TestBeacon_CORS(t *testing.T) {
	srv, s, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	_, _ = s.CreateTest(ctx, "hero", []string{"A", "B"}, nil, "")

	payload := map[string]interface{}{
		"t":   "hero",
		"v":   0,
		"e":   "view",
		"vid": "visitor123",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/b", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	// Check CORS headers
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected CORS header *, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestBeacon_OptionsRequest(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodOptions, "/b", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNoContent && w.Code != http.StatusOK {
		t.Errorf("expected status 200 or 204, got %d", w.Code)
	}

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected CORS header *, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
}
