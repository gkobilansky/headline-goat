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

	// GetOrCreateTest returns existing test or creates new one with source="client"
	// Used for auto-creating tests from client data attributes
	// Returns: test, wasCreated, error
	GetOrCreateTest(ctx context.Context, name string, variants []string) (*Test, bool, error)

	// SetSourceConflict marks a test as having a source conflict
	SetSourceConflict(ctx context.Context, name string, hasConflict bool) error

	// GetTestsByURL returns all running tests matching a URL
	GetTestsByURL(ctx context.Context, url string) ([]*Test, error)

	// SetTestURLFields sets URL-related fields on a test
	SetTestURLFields(ctx context.Context, name, url, target, ctaTarget, conversionURL string) error

	// Event operations
	RecordEvent(ctx context.Context, testName string, variant int, eventType string, visitorID string) error
	GetVariantStats(ctx context.Context, testName string) ([]VariantStats, error)
	GetEvents(ctx context.Context, testName string) ([]*Event, error)

	// Lifecycle
	Close() error
}
