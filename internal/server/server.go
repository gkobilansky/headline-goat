package server

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/headline-goat/headline-goat/internal/store"
)

type Server struct {
	store     *store.SQLiteStore
	port      int
	token     string
	router    *http.ServeMux
	startTime time.Time
}

func New(s *store.SQLiteStore, port int) *Server {
	srv := &Server{
		store:     s,
		port:      port,
		token:     generateToken(),
		router:    http.NewServeMux(),
		startTime: time.Now(),
	}

	srv.setupRoutes()
	return srv
}

func (s *Server) setupRoutes() {
	s.router.HandleFunc("/health", s.handleHealth)
	s.router.HandleFunc("/b", s.handleBeacon)
	s.router.HandleFunc("/t/", s.handleClientJS)
}

func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	fmt.Printf("headline-goat running on %s\n", addr)
	fmt.Printf("Dashboard: http://localhost:%d/dashboard\n", s.port)
	fmt.Printf("Dashboard token: %s\n", s.token)
	fmt.Println("\nPress Ctrl+C to stop")

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
