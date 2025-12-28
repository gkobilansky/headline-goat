package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGlobalJS_ReturnsJavaScript(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/ht.js", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Should return non-empty content
	if w.Body.Len() == 0 {
		t.Error("expected non-empty body")
	}
}

func TestGlobalJS_HasCorrectContentType(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/ht.js", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "javascript") {
		t.Errorf("expected javascript content type, got %s", contentType)
	}
}

func TestGlobalJS_HasCacheHeaders(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/ht.js", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	cacheControl := w.Header().Get("Cache-Control")
	if !strings.Contains(cacheControl, "max-age") {
		t.Errorf("expected cache-control with max-age, got %s", cacheControl)
	}
}

func TestGlobalJS_ContainsServerURL(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/ht.js", nil)
	req.Host = "localhost:8080"
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	body := w.Body.String()

	// Should contain the server URL (derived from request)
	if !strings.Contains(body, "localhost:8080") {
		t.Error("expected body to contain server URL")
	}
}

func TestGlobalJS_ContainsDataAttributeSelectors(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/ht.js", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	body := w.Body.String()

	// Should contain data attribute selectors
	if !strings.Contains(body, "data-ht-name") {
		t.Error("expected body to contain 'data-ht-name' selector")
	}

	if !strings.Contains(body, "data-ht-convert") {
		t.Error("expected body to contain 'data-ht-convert' selector")
	}
}

func TestGlobalJS_ContainsBeaconLogic(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/ht.js", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	body := w.Body.String()

	// Should contain beacon sending
	if !strings.Contains(body, "sendBeacon") {
		t.Error("expected body to contain 'sendBeacon'")
	}

	// Should contain beacon endpoint
	if !strings.Contains(body, "/b") {
		t.Error("expected body to contain beacon endpoint '/b'")
	}
}

func TestGlobalJS_MethodNotAllowed(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/ht.js", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}
