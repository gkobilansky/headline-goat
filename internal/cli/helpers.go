package cli

import (
	"fmt"

	"github.com/gkobilansky/headline-goat/internal/store"
)

// withStore opens the database, executes the function, and handles cleanup.
func withStore(fn func(*store.SQLiteStore) error) error {
	s, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer s.Close()

	return fn(s)
}
