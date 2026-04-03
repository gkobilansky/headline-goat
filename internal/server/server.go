package server

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"time"

	"github.com/gkobilansky/headline-goat/internal/dashboard"
	"github.com/gkobilansky/headline-goat/internal/store"
)

type Server struct {
	store     *store.SQLiteStore
	port      int
	token     string
	tokenFile string
	router    *http.ServeMux
	startTime time.Time
	css       template.CSS
	templates map[string]*template.Template
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

	srv.parseTemplates()
	srv.setupRoutes()
	return srv
}

func (s *Server) setupRoutes() {
	// Public endpoints
	s.router.HandleFunc("/health", s.handleHealth)
	s.router.HandleFunc("/b", s.handleBeacon)
	s.router.HandleFunc("/hlg.js", s.handleGlobalJS)
	s.router.HandleFunc("/api/tests", s.handleTestsAPI)

	// Dashboard endpoints (protected)
	s.router.Handle("/dashboard", s.authMiddleware(http.HandlerFunc(s.handleDashboard)))
	s.router.Handle("/dashboard/test/", s.authMiddleware(http.HandlerFunc(s.handleDashboardTest)))
	s.router.Handle("/dashboard/api/tests", s.authMiddleware(http.HandlerFunc(s.handleDashboardAPI)))
}

// Start writes the token file if configured and begins serving HTTP traffic.
func (s *Server) Start() error {
	// Write token to file for OTP command
	if s.tokenFile != "" {
		if err := os.WriteFile(s.tokenFile, []byte(s.token), 0600); err != nil {
			fmt.Printf("Warning: failed to write token file: %v\n", err)
		}
	}

	addr := fmt.Sprintf(":%d", s.port)
	return http.ListenAndServe(addr, s.router)
}

func (s *Server) Token() string {
	return s.token
}

func (s *Server) Handler() http.Handler {
	return s.router
}

func (s *Server) parseTemplates() {
	cssBytes, _ := dashboard.Assets.ReadFile("assets/style.css")
	s.css = template.CSS(cssBytes)

	layoutBytes, _ := dashboard.Templates.ReadFile("templates/layout.html")

	s.templates = make(map[string]*template.Template)
	for _, name := range []string{"list.html", "detail.html"} {
		contentBytes, _ := dashboard.Templates.ReadFile("templates/" + name)
		tmpl := template.Must(template.New("layout").Parse(string(layoutBytes)))
		template.Must(tmpl.New("content").Parse(string(contentBytes)))
		s.templates[name] = tmpl
	}
}

func generateToken() string {
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to a simple token if crypto/rand fails
		return "a1b2c3d4"
	}
	return hex.EncodeToString(bytes)
}
