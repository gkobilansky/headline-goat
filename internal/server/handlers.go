package server

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gkobilansky/headline-goat/internal/store"
)

// Beacon event types.
const (
	EventView    = "view"
	EventConvert = "convert"
)


// transparentGIF is a 1x1 transparent GIF pixel (43 bytes).
var transparentGIF = []byte{
	0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00,
	0x01, 0x00, 0x80, 0x00, 0x00, 0xff, 0xff, 0xff,
	0x00, 0x00, 0x00, 0x21, 0xf9, 0x04, 0x01, 0x00,
	0x00, 0x00, 0x00, 0x2c, 0x00, 0x00, 0x00, 0x00,
	0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02, 0x44,
	0x01, 0x00, 0x3b,
}

// writePixel responds with a 1x1 transparent GIF for image pixel compatibility.
func writePixel(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "image/gif")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(transparentGIF)
}

// setCORS sets standard CORS headers for cross-origin requests.
func setCORS(w http.ResponseWriter, methods string) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", methods)
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

// handlePreflight handles OPTIONS preflight requests and returns true if handled.
func handlePreflight(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return true
	}
	return false
}

// requireMethod checks if the request method matches and sends 405 if not.
// Returns true if the method matches, false if an error response was sent.
func requireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method != method {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return false
	}
	return true
}

type HealthResponse struct {
	Status        string `json:"status"`
	TestsCount    int    `json:"tests_count"`
	DBSizeBytes   int64  `json:"db_size_bytes"`
	UptimeSeconds int64  `json:"uptime_seconds"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
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
	row := s.store.DB().QueryRow("SELECT page_count * page_size FROM pragma_page_count(), pragma_page_size()")
	if err := row.Scan(&dbSize); err != nil {
		// Try to get file size as fallback
		if info, statErr := os.Stat(s.store.DBPath()); statErr == nil {
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
	setCORS(w, "GET, POST, OPTIONS")

	if handlePreflight(w, r) {
		return
	}

	// Bot detection: silently drop requests from known bots
	if IsBot(r.UserAgent()) {
		if r.Method == http.MethodGet {
			writePixel(w)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
		return
	}

	var req BeaconRequest

	switch r.Method {
	case http.MethodGet:
		// Parse from query parameters (no CORS preflight needed)
		q := r.URL.Query()
		req.TestName = q.Get("t")
		v, _ := strconv.Atoi(q.Get("v"))
		req.Variant = v
		req.EventType = q.Get("e")
		req.VisitorID = q.Get("vid")
		req.Source = q.Get("src")
		if variants := q.Get("variants"); variants != "" {
			json.Unmarshal([]byte(variants), &req.Variants)
		}
	case http.MethodPost:
		// Existing JSON body parsing (backward compatible)
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Validate required fields
	if req.TestName == "" || req.VisitorID == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	if req.EventType != EventView && req.EventType != EventConvert {
		http.Error(w, "Invalid event type", http.StatusBadRequest)
		return
	}

	ctx := context.Background()

	// Get or create test
	var test *store.Test
	var err error

	// Default source to client if not specified
	if req.Source == "" {
		req.Source = store.SourceClient
	}

	if len(req.Variants) > 0 && req.Source == store.SourceClient {
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
	if test.Source == store.SourceServer && req.Source == store.SourceClient && !test.HasSourceConflict {
		// Mark conflict (ignore error, non-critical)
		_ = s.store.SetSourceConflict(ctx, test.Name, true)
	}

	// Record event (deduplication handled by store)
	if err := s.store.RecordEvent(ctx, req.TestName, req.Variant, req.EventType, req.VisitorID); err != nil {
		http.Error(w, "Failed to record event", http.StatusInternalServerError)
		return
	}

	// GET requests return a transparent 1x1 GIF (image pixel compatibility)
	if r.Method == http.MethodGet {
		writePixel(w)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleTestsAPI returns tests matching a URL for the global script
func (s *Server) handleTestsAPI(w http.ResponseWriter, r *http.Request) {
	setCORS(w, "GET, OPTIONS")

	if handlePreflight(w, r) {
		return
	}

	if !requireMethod(w, r, http.MethodGet) {
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

