package testutil

import (
	"testing"

	"github.com/headline-goat/headline-goat/internal/store"
)

// SetupTestStore creates a test database and returns the store with a cleanup function.
// Uses t.TempDir() for automatic cleanup on test completion.
func SetupTestStore(t *testing.T) *store.SQLiteStore {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	s, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}

	t.Cleanup(func() {
		s.Close()
	})

	return s
}
