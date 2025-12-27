package server_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientJS_ValidTest(t *testing.T) {
	srv, s, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	_, err := s.CreateTest(ctx, "hero", []string{"Ship Faster", "Build Better", "Scale Smart"}, nil, "")
	if err != nil {
		t.Fatalf("failed to create test: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/t/hero.js", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Check content type
	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "javascript") {
		t.Errorf("expected javascript content type, got %s", contentType)
	}

	// Check cache control
	cacheControl := w.Header().Get("Cache-Control")
	if !strings.Contains(cacheControl, "max-age") {
		t.Errorf("expected cache-control with max-age, got %s", cacheControl)
	}

	// Check body contains test name and variants
	body := w.Body.String()
	if !strings.Contains(body, "hero") {
		t.Error("expected body to contain test name 'hero'")
	}
	if !strings.Contains(body, "Ship Faster") {
		t.Error("expected body to contain variant 'Ship Faster'")
	}
	if !strings.Contains(body, "Build Better") {
		t.Error("expected body to contain variant 'Build Better'")
	}
	if !strings.Contains(body, "/b") {
		t.Error("expected body to contain beacon endpoint '/b'")
	}
}

func TestClientJS_NonexistentTest(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/t/nonexistent.js", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestClientJS_InvalidPath(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Missing .js extension
	req := httptest.NewRequest(http.MethodGet, "/t/hero", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestClientJS_ContainsRequiredFunctions(t *testing.T) {
	srv, s, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	_, err := s.CreateTest(ctx, "hero", []string{"A", "B"}, nil, "")
	if err != nil {
		t.Fatalf("failed to create test: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/t/hero.js", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	body := w.Body.String()

	// Check for required functionality
	requiredPatterns := []string{
		"localStorage",       // Variant storage
		"ht_vid",             // Visitor ID key
		"sendBeacon",         // Beacon sending
		"data-ht",            // Element selector
		"data-ht-convert",    // Convert button selector
		"window.HT",          // Global API
	}

	for _, pattern := range requiredPatterns {
		if !strings.Contains(body, pattern) {
			t.Errorf("expected body to contain '%s'", pattern)
		}
	}
}
