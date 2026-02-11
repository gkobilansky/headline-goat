package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gkobilansky/headline-goat/internal/server"
	"github.com/gkobilansky/headline-goat/internal/store"
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

// --- GET query param beacon tests (eliminates CORS preflight) ---

func TestBeacon_GET_ValidRequest(t *testing.T) {
	srv, s, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	_, err := s.CreateTest(ctx, "hero", []string{"A", "B", "C"}, nil, "")
	if err != nil {
		t.Fatalf("failed to create test: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/b?t=hero&v=1&e=view&vid=visitor123", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
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

func TestBeacon_GET_ConversionEvent(t *testing.T) {
	srv, s, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	_, err := s.CreateTest(ctx, "hero", []string{"A", "B"}, nil, "")
	if err != nil {
		t.Fatalf("failed to create test: %v", err)
	}

	// Send view via GET
	req := httptest.NewRequest(http.MethodGet, "/b?t=hero&v=0&e=view&vid=visitor456", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	// Send conversion via GET
	req = httptest.NewRequest(http.MethodGet, "/b?t=hero&v=0&e=convert&vid=visitor456", nil)
	w = httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	stats, err := s.GetVariantStats(ctx, "hero")
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}

	if len(stats) == 0 {
		t.Fatal("expected at least 1 variant stat, got 0")
	}
	if stats[0].Views != 1 {
		t.Errorf("expected 1 view, got %d", stats[0].Views)
	}
	if stats[0].Conversions != 1 {
		t.Errorf("expected 1 conversion, got %d", stats[0].Conversions)
	}
}

func TestBeacon_GET_MissingFields(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Missing test name and visitor ID
	req := httptest.NewRequest(http.MethodGet, "/b?e=view", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestBeacon_GET_AutoCreateTest(t *testing.T) {
	srv, s, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Send GET beacon with variants param to auto-create test
	req := httptest.NewRequest(http.MethodGet,
		`/b?t=new-test&v=0&e=view&vid=visitor789&src=client&variants=["X","Y"]`, nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify test was auto-created
	test, err := s.GetTest(ctx, "new-test")
	if err != nil {
		t.Fatalf("expected auto-created test to exist: %v", err)
	}
	if len(test.Variants) != 2 {
		t.Errorf("expected 2 variants, got %d", len(test.Variants))
	}
}

func TestBeacon_GET_ReturnsTransparentPixel(t *testing.T) {
	srv, s, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	_, _ = s.CreateTest(ctx, "hero", []string{"A", "B"}, nil, "")

	req := httptest.NewRequest(http.MethodGet, "/b?t=hero&v=0&e=view&vid=visitor123", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	// GET beacon should return a 1x1 transparent GIF for image pixel compatibility
	contentType := w.Header().Get("Content-Type")
	if contentType != "image/gif" {
		t.Errorf("expected Content-Type image/gif, got %s", contentType)
	}
}

// --- Bot detection tests ---

func TestBeacon_BotUserAgent_Rejected(t *testing.T) {
	srv, s, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	_, _ = s.CreateTest(ctx, "hero", []string{"A", "B"}, nil, "")

	botUAs := []string{
		"Googlebot/2.1 (+http://www.google.com/bot.html)",
		"Mozilla/5.0 (compatible; bingbot/2.0; +http://www.bing.com/bingbot.htm)",
		"Mozilla/5.0 (compatible; YandexBot/3.0; +http://yandex.com/bots)",
		"facebookexternalhit/1.1",
		"Twitterbot/1.0",
		"Mozilla/5.0 AppleWebKit/537.36 (KHTML, like Gecko; compatible; HeadlessChrome/120.0)",
	}

	for _, ua := range botUAs {
		req := httptest.NewRequest(http.MethodGet, "/b?t=hero&v=0&e=view&vid=bot-visitor", nil)
		req.Header.Set("User-Agent", ua)
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, req)

		// Bot requests should be silently accepted but not recorded
		if w.Code != http.StatusNoContent && w.Code != http.StatusOK {
			t.Errorf("bot UA %q: expected 204 or 200, got %d", ua, w.Code)
		}
	}

	// Verify no events were recorded for bot traffic
	stats, err := s.GetVariantStats(ctx, "hero")
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}
	if len(stats) != 0 {
		t.Errorf("expected 0 stats (bot traffic filtered), got %d", len(stats))
	}
}

func TestBeacon_RealUserAgent_Accepted(t *testing.T) {
	srv, s, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	_, _ = s.CreateTest(ctx, "hero", []string{"A", "B"}, nil, "")

	realUAs := []string{
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 17_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Mobile/15E148 Safari/604.1",
		"", // Empty UA should still be accepted (sendBeacon may not set UA)
	}

	for i, ua := range realUAs {
		vid := fmt.Sprintf("real-visitor-%d", i)
		req := httptest.NewRequest(http.MethodGet, "/b?t=hero&v=0&e=view&vid="+vid, nil)
		if ua != "" {
			req.Header.Set("User-Agent", ua)
		}
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusNoContent && w.Code != http.StatusOK {
			t.Errorf("real UA %q: expected 204 or 200, got %d", ua, w.Code)
		}
	}

	// Verify events were recorded
	stats, err := s.GetVariantStats(ctx, "hero")
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}
	if len(stats) == 0 {
		t.Error("expected stats for real user agents")
	}
	if stats[0].Views != 3 {
		t.Errorf("expected 3 views from real users, got %d", stats[0].Views)
	}
}

func TestBeacon_BotDetection_POST(t *testing.T) {
	srv, s, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	_, _ = s.CreateTest(ctx, "hero", []string{"A", "B"}, nil, "")

	// Bot detection should also work for POST requests
	payload := map[string]interface{}{
		"t":   "hero",
		"v":   0,
		"e":   "view",
		"vid": "bot-visitor",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/b", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Googlebot/2.1 (+http://www.google.com/bot.html)")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	// Verify no events were recorded
	stats, err := s.GetVariantStats(ctx, "hero")
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}
	if len(stats) != 0 {
		t.Errorf("expected 0 stats (bot filtered), got %d", len(stats))
	}
}
