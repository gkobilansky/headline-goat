package store

import "context"

// Store defines the interface for test storage operations
type Store interface {
	// Test operations
	CreateTest(ctx context.Context, name string, variants []string, weights []float64, conversionGoal string) (*Test, error)
	GetTest(ctx context.Context, name string) (*Test, error)
	ListTests(ctx context.Context) ([]*Test, error)
	UpdateTestState(ctx context.Context, name string, state TestState, winnerVariant *int) error
	DeleteTest(ctx context.Context, name string) error

	// Event operations
	RecordEvent(ctx context.Context, testName string, variant int, eventType string, visitorID string) error
	GetVariantStats(ctx context.Context, testName string) ([]VariantStats, error)
	GetEvents(ctx context.Context, testName string) ([]*Event, error)

	// Lifecycle
	Close() error
}
