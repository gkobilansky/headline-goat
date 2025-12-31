package server

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/headline-goat/headline-goat/internal/store"
)

type HealthResponse struct {
	Status        string `json:"status"`
	TestsCount    int    `json:"tests_count"`
	DBSizeBytes   int64  `json:"db_size_bytes"`
	UptimeSeconds int64  `json:"uptime_seconds"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := context.Background()

	// Get test count
	tests, err := s.store.ListTests(ctx)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Get database size
	var dbSize int64
	db := s.store.DB()
	row := db.QueryRow("SELECT page_count * page_size FROM pragma_page_count(), pragma_page_size()")
	if err := row.Scan(&dbSize); err != nil {
		// Try to get file size as fallback
		if info, statErr := os.Stat(getDBPath(db)); statErr == nil {
			dbSize = info.Size()
		}
	}

	// Calculate uptime
	uptime := int64(time.Since(s.startTime).Seconds())

	response := HealthResponse{
		Status:        "ok",
		TestsCount:    len(tests),
		DBSizeBytes:   dbSize,
		UptimeSeconds: uptime,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getDBPath attempts to get the database file path
func getDBPath(db interface{}) string {
	// This is a simplified version - in practice you'd track the path
	return "./headline-goat.db"
}

// BeaconRequest represents an incoming beacon event
type BeaconRequest struct {
	TestName  string   `json:"t"`
	Variant   int      `json:"v"`
	EventType string   `json:"e"`
	VisitorID string   `json:"vid"`
	Source    string   `json:"src"`      // "client" or "server"
	Variants  []string `json:"variants"` // For auto-creation
}

func (s *Server) handleBeacon(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers for all responses
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Handle preflight
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req BeaconRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.TestName == "" || req.VisitorID == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	if req.EventType != "view" && req.EventType != "convert" {
		http.Error(w, "Invalid event type", http.StatusBadRequest)
		return
	}

	ctx := context.Background()

	// Get or create test
	var test *store.Test
	var err error

	// Default source to "client" if not specified
	if req.Source == "" {
		req.Source = "client"
	}

	if len(req.Variants) > 0 && req.Source == "client" {
		// Auto-create from client data attributes
		var created bool
		test, created, err = s.store.GetOrCreateTest(ctx, req.TestName, req.Variants)
		if err != nil {
			http.Error(w, "Failed to get or create test", http.StatusInternalServerError)
			return
		}
		_ = created // Could log if needed
	} else {
		// Existing behavior - test must exist
		test, err = s.store.GetTest(ctx, req.TestName)
		if err != nil {
			http.Error(w, "Test not found", http.StatusBadRequest)
			return
		}
	}

	// Validate variant in range
	if req.Variant < 0 || req.Variant >= len(test.Variants) {
		http.Error(w, "Invalid variant", http.StatusBadRequest)
		return
	}

	// Check for source conflict (server-created test receiving client beacons)
	if test.Source == "server" && req.Source == "client" && !test.HasSourceConflict {
		// Mark conflict (ignore error, non-critical)
		_ = s.store.SetSourceConflict(ctx, test.Name, true)
	}

	// Record event (deduplication handled by store)
	if err := s.store.RecordEvent(ctx, req.TestName, req.Variant, req.EventType, req.VisitorID); err != nil {
		http.Error(w, "Failed to record event", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleTestsAPI returns tests matching a URL for the global script
func (s *Server) handleTestsAPI(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	url := r.URL.Query().Get("url")
	if url == "" {
		http.Error(w, "url parameter required", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	tests, err := s.store.GetTestsByURL(ctx, url)
	if err != nil {
		http.Error(w, "Failed to fetch tests", http.StatusInternalServerError)
		return
	}

	// Return minimal test data for client
	type TestResponse struct {
		Name          string   `json:"name"`
		Variants      []string `json:"variants"`
		Target        string   `json:"target,omitempty"`
		CTATarget     string   `json:"cta_target,omitempty"`
		ConversionURL string   `json:"conversion_url,omitempty"`
	}

	var response []TestResponse
	for _, t := range tests {
		response = append(response, TestResponse{
			Name:          t.Name,
			Variants:      t.Variants,
			Target:        t.Target,
			CTATarget:     t.CTATarget,
			ConversionURL: t.ConversionURL,
		})
	}

	// Return empty array instead of null
	if response == nil {
		response = []TestResponse{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

