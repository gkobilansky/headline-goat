package server

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"
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
	TestName  string `json:"t"`
	Variant   int    `json:"v"`
	EventType string `json:"e"`
	VisitorID string `json:"vid"`
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

	// Validate test exists
	test, err := s.store.GetTest(ctx, req.TestName)
	if err != nil {
		http.Error(w, "Test not found", http.StatusBadRequest)
		return
	}

	// Validate variant in range
	if req.Variant < 0 || req.Variant >= len(test.Variants) {
		http.Error(w, "Invalid variant", http.StatusBadRequest)
		return
	}

	// Record event (deduplication handled by store)
	if err := s.store.RecordEvent(ctx, req.TestName, req.Variant, req.EventType, req.VisitorID); err != nil {
		http.Error(w, "Failed to record event", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

