package server_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDashboard_Unauthorized(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestDashboard_ValidToken(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	// First, access with token in query param
	req := httptest.NewRequest(http.MethodGet, "/dashboard?token="+srv.Token(), nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	// Should redirect and set cookie
	if w.Code != http.StatusFound {
		t.Errorf("expected status 302 (redirect), got %d", w.Code)
	}

	// Check that cookie was set
	cookies := w.Result().Cookies()
	var tokenCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "ht_token" {
			tokenCookie = c
			break
		}
	}

	if tokenCookie == nil {
		t.Error("expected ht_token cookie to be set")
	}
}

func TestDashboard_WithCookie(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.AddCookie(&http.Cookie{
		Name:  "ht_token",
		Value: srv.Token(),
	})
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Check that HTML is returned
	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("expected HTML content type, got %s", contentType)
	}
}

func TestDashboard_InvalidToken(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/dashboard?token=wrongtoken", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestDashboardAPI_Tests(t *testing.T) {
	srv, s, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	_, _ = s.CreateTest(ctx, "hero", []string{"A", "B"}, nil, "Click button")
	_, _ = s.CreateTest(ctx, "pricing", []string{"X", "Y", "Z"}, nil, "")

	req := httptest.NewRequest(http.MethodGet, "/dashboard/api/tests", nil)
	req.AddCookie(&http.Cookie{
		Name:  "ht_token",
		Value: srv.Token(),
	})
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Check JSON content type
	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("expected JSON content type, got %s", contentType)
	}

	// Check body contains test names
	body := w.Body.String()
	if !strings.Contains(body, "hero") {
		t.Error("expected body to contain 'hero'")
	}
	if !strings.Contains(body, "pricing") {
		t.Error("expected body to contain 'pricing'")
	}
	if !strings.Contains(body, "Click button") {
		t.Error("expected body to contain conversion goal")
	}
}

func TestDashboardTest_Detail(t *testing.T) {
	srv, s, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	_, _ = s.CreateTest(ctx, "hero", []string{"Ship Faster", "Build Better"}, nil, "")

	req := httptest.NewRequest(http.MethodGet, "/dashboard/test/hero", nil)
	req.AddCookie(&http.Cookie{
		Name:  "ht_token",
		Value: srv.Token(),
	})
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	body := w.Body.String()

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d, body: %s", w.Code, body)
	}

	if !strings.Contains(body, "hero") {
		t.Error("expected body to contain test name")
	}
	if !strings.Contains(body, "Ship Faster") {
		t.Error("expected body to contain variant name")
	}
}

func TestDashboardTest_NotFound(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/dashboard/test/nonexistent", nil)
	req.AddCookie(&http.Cookie{
		Name:  "ht_token",
		Value: srv.Token(),
	})
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}
