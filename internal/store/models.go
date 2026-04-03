package store

import "time"

type TestState string

const (
	StateRunning   TestState = "running"
	StateCompleted TestState = "completed"
)

// Source identifiers for how a test was created.
const (
	SourceClient = "client"
	SourceServer = "server"
)

type Test struct {
	ID                int64
	Name              string
	Variants          []string  // Decoded from JSON
	Weights           []float64 // Optional, decoded from JSON
	ConversionGoal    string    // Optional description of what conversion means
	State             TestState
	WinnerVariant     *int
	Source            string // SourceClient or SourceServer
	HasSourceConflict bool
	URL               string // For URL-based matching
	ConversionURL     string // URL-based conversion
	Target            string // CSS selector for headline
	CTATarget         string // CSS selector for CTA
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type Event struct {
	ID        int64
	TestName  string
	Variant   int
	EventType string // "view" or "convert"
	VisitorID string
	CreatedAt time.Time
}

type VariantStats struct {
	Variant     int
	Views       int
	Conversions int
}
