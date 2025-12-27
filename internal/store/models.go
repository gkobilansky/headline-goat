package store

import "time"

type TestState string

const (
	StateRunning   TestState = "running"
	StatePaused    TestState = "paused"
	StateCompleted TestState = "completed"
)

type Test struct {
	ID             int64
	Name           string
	Variants       []string  // Decoded from JSON
	Weights        []float64 // Optional, decoded from JSON
	ConversionGoal string    // Optional description of what conversion means
	State          TestState
	WinnerVariant  *int
	CreatedAt      time.Time
	UpdatedAt      time.Time
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
