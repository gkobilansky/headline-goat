package server

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/headline-goat/headline-goat/internal/store"
)

type Server struct {
	store     *store.SQLiteStore
	port      int
	token     string
	tokenFile string
	router    *http.ServeMux
	startTime time.Time
}

func New(s *store.SQLiteStore, port int, tokenFile string) *Server {
	srv := &Server{
		store:     s,
		port:      port,
		token:     generateToken(),
		tokenFile: tokenFile,
		router:    http.NewServeMux(),
		startTime: time.Now(),
	}

	srv.setupRoutes()
	return srv
}

func (s *Server) setupRoutes() {
	// Public endpoints
	s.router.HandleFunc("/health", s.handleHealth)
	s.router.HandleFunc("/b", s.handleBeacon)
	s.router.HandleFunc("/ht.js", s.handleGlobalJS)
	s.router.HandleFunc("/api/tests", s.handleTestsAPI)

	// Dashboard endpoints (protected)
	s.router.Handle("/dashboard", s.authMiddleware(http.HandlerFunc(s.handleDashboard)))
	s.router.Handle("/dashboard/test/", s.authMiddleware(http.HandlerFunc(s.handleDashboardTest)))
	s.router.Handle("/dashboard/api/tests", s.authMiddleware(http.HandlerFunc(s.handleDashboardAPI)))
}

func (s *Server) Start() error {
	return s.StartWithOptions(true)
}

// StartQuiet starts the server without printing startup messages
func (s *Server) StartQuiet() error {
	return s.StartWithOptions(false)
}

func (s *Server) StartWithOptions(printMessages bool) error {
	// Write token to file for OTP command
	if s.tokenFile != "" {
		if err := os.WriteFile(s.tokenFile, []byte(s.token), 0600); err != nil {
			fmt.Printf("Warning: failed to write token file: %v\n", err)
		}
	}

	addr := fmt.Sprintf(":%d", s.port)

	if printMessages {
		fmt.Println()
		fmt.Printf("headline-goat running on http://localhost:%d\n", s.port)
		fmt.Printf("Dashboard: http://localhost:%d/dashboard?token=%s\n", s.port, s.token)
		fmt.Println()
		fmt.Println("Press Ctrl+C to stop")
	}

	return http.ListenAndServe(addr, s.router)
}

func (s *Server) Token() string {
	return s.token
}

func (s *Server) Store() *store.SQLiteStore {
	return s.store
}

func (s *Server) StartTime() time.Time {
	return s.startTime
}

func (s *Server) Handler() http.Handler {
	return s.router
}

func generateToken() string {
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to a simple token if crypto/rand fails
		return "a1b2c3d4"
	}
	return hex.EncodeToString(bytes)
}
