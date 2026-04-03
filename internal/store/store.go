package store

import (
	"context"
	"time"
)

// Store defines the interface for test and event persistence.
type Store interface {
	Close() error

	// Test CRUD
	CreateTest(ctx context.Context, name string, variants []string, weights []float64, conversionGoal string) (*Test, error)
	GetTest(ctx context.Context, name string) (*Test, error)
	ListTests(ctx context.Context) ([]*Test, error)
	UpdateTestState(ctx context.Context, name string, state TestState, winnerVariant *int) error
	DeleteTest(ctx context.Context, name string) error
	SetWinner(ctx context.Context, testName string, variantIndex int) error
	GetOrCreateTest(ctx context.Context, name string, variants []string) (*Test, bool, error)
	GetTestsByURL(ctx context.Context, url string) ([]*Test, error)
	SetTestURLFields(ctx context.Context, name, url, target, ctaTarget, conversionURL string) error
	SetSourceConflict(ctx context.Context, name string, hasConflict bool) error
	CountTests(ctx context.Context) (int, error)

	// Events
	RecordEvent(ctx context.Context, testName string, variant int, eventType string, visitorID string) error
	GetEvents(ctx context.Context, testName string) ([]*Event, error)

	// Stats
	GetVariantStats(ctx context.Context, testName string) ([]VariantStats, error)
	GetAllVariantStats(ctx context.Context) (map[string][]VariantStats, error)
	GetRecentViewCount(ctx context.Context, testName string, hours int) (int, time.Time, error)

	// Settings
	SetSetting(ctx context.Context, key, value string) error
	GetSetting(ctx context.Context, key string) (string, error)

	// Health
	DBSizeBytes(ctx context.Context) (int64, error)
}

// Compile-time check that SQLiteStore implements Store.
var _ Store = (*SQLiteStore)(nil)
