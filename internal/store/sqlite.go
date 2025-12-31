package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var ErrNotFound = errors.New("not found")

type SQLiteStore struct {
	db *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS tests (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    variants TEXT NOT NULL,
    weights TEXT,
    conversion_goal TEXT,
    state TEXT NOT NULL DEFAULT 'running',
    winner_variant INTEGER,
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE INDEX IF NOT EXISTS idx_tests_name ON tests(name);
CREATE INDEX IF NOT EXISTS idx_tests_state ON tests(state);

CREATE TABLE IF NOT EXISTS events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    test_name TEXT NOT NULL,
    variant INTEGER NOT NULL,
    event_type TEXT NOT NULL,
    visitor_id TEXT NOT NULL,
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    FOREIGN KEY (test_name) REFERENCES tests(name)
);

CREATE INDEX IF NOT EXISTS idx_events_test ON events(test_name);
CREATE INDEX IF NOT EXISTS idx_events_test_event ON events(test_name, event_type);
CREATE INDEX IF NOT EXISTS idx_events_visitor ON events(test_name, visitor_id, event_type);
CREATE UNIQUE INDEX IF NOT EXISTS idx_events_dedup ON events(test_name, visitor_id, event_type);

CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
`

func Open(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Apply schema
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to apply schema: %w", err)
	}

	// Apply migrations for new columns (ignore errors - column may already exist)
	migrations := []string{
		"ALTER TABLE tests ADD COLUMN source TEXT NOT NULL DEFAULT 'client'",
		"ALTER TABLE tests ADD COLUMN has_source_conflict INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE tests ADD COLUMN url TEXT",
		"ALTER TABLE tests ADD COLUMN conversion_url TEXT",
		"ALTER TABLE tests ADD COLUMN target TEXT",
		"ALTER TABLE tests ADD COLUMN cta_target TEXT",
	}
	for _, m := range migrations {
		db.Exec(m) // Ignore errors - column may already exist
	}

	// Add index for URL lookups
	db.Exec("CREATE INDEX IF NOT EXISTS idx_tests_url ON tests(url)")

	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) CreateTest(ctx context.Context, name string, variants []string, weights []float64, conversionGoal string) (*Test, error) {
	return s.createTestWithSource(ctx, name, variants, weights, conversionGoal, "server")
}

func (s *SQLiteStore) createTestWithSource(ctx context.Context, name string, variants []string, weights []float64, conversionGoal string, source string) (*Test, error) {
	variantsJSON, err := json.Marshal(variants)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal variants: %w", err)
	}

	var weightsJSON []byte
	if len(weights) > 0 {
		weightsJSON, err = json.Marshal(weights)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal weights: %w", err)
		}
	}

	now := time.Now().Unix()
	result, err := s.db.ExecContext(ctx,
		`INSERT INTO tests (name, variants, weights, conversion_goal, state, source, created_at, updated_at)
		 VALUES (?, ?, ?, ?, 'running', ?, ?, ?)`,
		name, string(variantsJSON), nullableString(weightsJSON), conversionGoal, source, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert test: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return &Test{
		ID:             id,
		Name:           name,
		Variants:       variants,
		Weights:        weights,
		ConversionGoal: conversionGoal,
		State:          StateRunning,
		Source:         source,
		CreatedAt:      time.Unix(now, 0),
		UpdatedAt:      time.Unix(now, 0),
	}, nil
}

func (s *SQLiteStore) GetTest(ctx context.Context, name string) (*Test, error) {
	var test Test
	var variantsJSON string
	var weightsJSON sql.NullString
	var winnerVariant sql.NullInt64
	var hasSourceConflict int64
	var url, conversionURL, target, ctaTarget sql.NullString
	var createdAt, updatedAt int64

	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, variants, weights, conversion_goal, state, winner_variant,
		        source, has_source_conflict, url, conversion_url, target, cta_target,
		        created_at, updated_at
		 FROM tests WHERE name = ?`, name,
	).Scan(&test.ID, &test.Name, &variantsJSON, &weightsJSON, &test.ConversionGoal, &test.State, &winnerVariant,
		&test.Source, &hasSourceConflict, &url, &conversionURL, &target, &ctaTarget,
		&createdAt, &updatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get test: %w", err)
	}

	if err := json.Unmarshal([]byte(variantsJSON), &test.Variants); err != nil {
		return nil, fmt.Errorf("failed to unmarshal variants: %w", err)
	}

	if weightsJSON.Valid && weightsJSON.String != "" {
		if err := json.Unmarshal([]byte(weightsJSON.String), &test.Weights); err != nil {
			return nil, fmt.Errorf("failed to unmarshal weights: %w", err)
		}
	}

	if winnerVariant.Valid {
		w := int(winnerVariant.Int64)
		test.WinnerVariant = &w
	}

	test.HasSourceConflict = hasSourceConflict != 0
	if url.Valid {
		test.URL = url.String
	}
	if conversionURL.Valid {
		test.ConversionURL = conversionURL.String
	}
	if target.Valid {
		test.Target = target.String
	}
	if ctaTarget.Valid {
		test.CTATarget = ctaTarget.String
	}

	test.CreatedAt = time.Unix(createdAt, 0)
	test.UpdatedAt = time.Unix(updatedAt, 0)

	return &test, nil
}

func (s *SQLiteStore) ListTests(ctx context.Context) ([]*Test, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, variants, weights, conversion_goal, state, winner_variant,
		        source, has_source_conflict, url, conversion_url, target, cta_target,
		        created_at, updated_at
		 FROM tests ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list tests: %w", err)
	}
	defer rows.Close()

	var tests []*Test
	for rows.Next() {
		var test Test
		var variantsJSON string
		var weightsJSON sql.NullString
		var winnerVariant sql.NullInt64
		var hasSourceConflict int64
		var url, conversionURL, target, ctaTarget sql.NullString
		var createdAt, updatedAt int64

		err := rows.Scan(&test.ID, &test.Name, &variantsJSON, &weightsJSON, &test.ConversionGoal, &test.State, &winnerVariant,
			&test.Source, &hasSourceConflict, &url, &conversionURL, &target, &ctaTarget,
			&createdAt, &updatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan test: %w", err)
		}

		if err := json.Unmarshal([]byte(variantsJSON), &test.Variants); err != nil {
			return nil, fmt.Errorf("failed to unmarshal variants: %w", err)
		}

		if weightsJSON.Valid && weightsJSON.String != "" {
			if err := json.Unmarshal([]byte(weightsJSON.String), &test.Weights); err != nil {
				return nil, fmt.Errorf("failed to unmarshal weights: %w", err)
			}
		}

		if winnerVariant.Valid {
			w := int(winnerVariant.Int64)
			test.WinnerVariant = &w
		}

		test.HasSourceConflict = hasSourceConflict != 0
		if url.Valid {
			test.URL = url.String
		}
		if conversionURL.Valid {
			test.ConversionURL = conversionURL.String
		}
		if target.Valid {
			test.Target = target.String
		}
		if ctaTarget.Valid {
			test.CTATarget = ctaTarget.String
		}

		test.CreatedAt = time.Unix(createdAt, 0)
		test.UpdatedAt = time.Unix(updatedAt, 0)

		tests = append(tests, &test)
	}

	return tests, nil
}

func (s *SQLiteStore) UpdateTestState(ctx context.Context, name string, state TestState, winnerVariant *int) error {
	now := time.Now().Unix()

	var result sql.Result
	var err error

	if winnerVariant != nil {
		result, err = s.db.ExecContext(ctx,
			`UPDATE tests SET state = ?, winner_variant = ?, updated_at = ? WHERE name = ?`,
			string(state), *winnerVariant, now, name,
		)
	} else {
		result, err = s.db.ExecContext(ctx,
			`UPDATE tests SET state = ?, updated_at = ? WHERE name = ?`,
			string(state), now, name,
		)
	}

	if err != nil {
		return fmt.Errorf("failed to update test state: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *SQLiteStore) DeleteTest(ctx context.Context, name string) error {
	// First delete related events
	_, err := s.db.ExecContext(ctx, `DELETE FROM events WHERE test_name = ?`, name)
	if err != nil {
		return fmt.Errorf("failed to delete events: %w", err)
	}

	result, err := s.db.ExecContext(ctx, `DELETE FROM tests WHERE name = ?`, name)
	if err != nil {
		return fmt.Errorf("failed to delete test: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *SQLiteStore) RecordEvent(ctx context.Context, testName string, variant int, eventType string, visitorID string) error {
	now := time.Now().Unix()

	// Use INSERT OR IGNORE for deduplication via unique index
	_, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO events (test_name, variant, event_type, visitor_id, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		testName, variant, eventType, visitorID, now,
	)
	if err != nil {
		return fmt.Errorf("failed to record event: %w", err)
	}

	return nil
}

func (s *SQLiteStore) GetVariantStats(ctx context.Context, testName string) ([]VariantStats, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			variant,
			COUNT(DISTINCT CASE WHEN event_type = 'view' THEN visitor_id END) as views,
			COUNT(DISTINCT CASE WHEN event_type = 'convert' THEN visitor_id END) as conversions
		FROM events
		WHERE test_name = ?
		GROUP BY variant
		ORDER BY variant
	`, testName)
	if err != nil {
		return nil, fmt.Errorf("failed to get variant stats: %w", err)
	}
	defer rows.Close()

	var stats []VariantStats
	for rows.Next() {
		var s VariantStats
		if err := rows.Scan(&s.Variant, &s.Views, &s.Conversions); err != nil {
			return nil, fmt.Errorf("failed to scan stats: %w", err)
		}
		stats = append(stats, s)
	}

	return stats, nil
}

func (s *SQLiteStore) GetEvents(ctx context.Context, testName string) ([]*Event, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, test_name, variant, event_type, visitor_id, created_at
		 FROM events WHERE test_name = ? ORDER BY created_at DESC`,
		testName,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}
	defer rows.Close()

	var events []*Event
	for rows.Next() {
		var e Event
		var createdAt int64
		if err := rows.Scan(&e.ID, &e.TestName, &e.Variant, &e.EventType, &e.VisitorID, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		e.CreatedAt = time.Unix(createdAt, 0)
		events = append(events, &e)
	}

	return events, nil
}

// DB returns the underlying database connection for health checks
func (s *SQLiteStore) DB() *sql.DB {
	return s.db
}

// SetWinner marks a test as completed with the specified winning variant
func (s *SQLiteStore) SetWinner(ctx context.Context, testName string, variantIndex int) error {
	return s.UpdateTestState(ctx, testName, StateCompleted, &variantIndex)
}

// GetOrCreateTest returns existing test or creates new one with source="client"
// Used for auto-creating tests from client data attributes
// Returns: test, wasCreated, error
func (s *SQLiteStore) GetOrCreateTest(ctx context.Context, name string, variants []string) (*Test, bool, error) {
	// Try to get existing test first
	test, err := s.GetTest(ctx, name)
	if err == nil {
		return test, false, nil // exists, not created
	}
	if err != ErrNotFound {
		return nil, false, err // real error
	}

	// Create new test with source=client
	test, err = s.createTestWithSource(ctx, name, variants, nil, "", "client")
	if err != nil {
		// Handle race condition - another request may have created it
		if containsUniqueConstraint(err) {
			test, err = s.GetTest(ctx, name)
			if err != nil {
				return nil, false, err
			}
			return test, false, nil
		}
		return nil, false, err
	}

	return test, true, nil // created
}

// containsUniqueConstraint checks if error is a UNIQUE constraint violation
func containsUniqueConstraint(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "UNIQUE constraint") || strings.Contains(errStr, "unique constraint")
}

// SetSourceConflict marks a test as having a source conflict
func (s *SQLiteStore) SetSourceConflict(ctx context.Context, name string, hasConflict bool) error {
	conflict := 0
	if hasConflict {
		conflict = 1
	}
	now := time.Now().Unix()
	result, err := s.db.ExecContext(ctx,
		"UPDATE tests SET has_source_conflict = ?, updated_at = ? WHERE name = ?",
		conflict, now, name)
	if err != nil {
		return fmt.Errorf("failed to set source conflict: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// GetTestsByURL returns all running tests matching a URL
func (s *SQLiteStore) GetTestsByURL(ctx context.Context, url string) ([]*Test, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, variants, weights, conversion_goal, state, winner_variant,
		        source, has_source_conflict, url, conversion_url, target, cta_target,
		        created_at, updated_at
		 FROM tests
		 WHERE url = ? AND state = 'running'`,
		url)
	if err != nil {
		return nil, fmt.Errorf("failed to query tests by URL: %w", err)
	}
	defer rows.Close()

	var tests []*Test
	for rows.Next() {
		var test Test
		var variantsJSON string
		var weightsJSON sql.NullString
		var winnerVariant sql.NullInt64
		var hasSourceConflict int64
		var urlVal, conversionURL, target, ctaTarget sql.NullString
		var createdAt, updatedAt int64

		err := rows.Scan(&test.ID, &test.Name, &variantsJSON, &weightsJSON, &test.ConversionGoal, &test.State, &winnerVariant,
			&test.Source, &hasSourceConflict, &urlVal, &conversionURL, &target, &ctaTarget,
			&createdAt, &updatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan test: %w", err)
		}

		if err := json.Unmarshal([]byte(variantsJSON), &test.Variants); err != nil {
			return nil, fmt.Errorf("failed to unmarshal variants: %w", err)
		}

		if weightsJSON.Valid && weightsJSON.String != "" {
			if err := json.Unmarshal([]byte(weightsJSON.String), &test.Weights); err != nil {
				return nil, fmt.Errorf("failed to unmarshal weights: %w", err)
			}
		}

		if winnerVariant.Valid {
			w := int(winnerVariant.Int64)
			test.WinnerVariant = &w
		}

		test.HasSourceConflict = hasSourceConflict != 0
		if urlVal.Valid {
			test.URL = urlVal.String
		}
		if conversionURL.Valid {
			test.ConversionURL = conversionURL.String
		}
		if target.Valid {
			test.Target = target.String
		}
		if ctaTarget.Valid {
			test.CTATarget = ctaTarget.String
		}

		test.CreatedAt = time.Unix(createdAt, 0)
		test.UpdatedAt = time.Unix(updatedAt, 0)

		tests = append(tests, &test)
	}

	return tests, rows.Err()
}

// SetTestURLFields sets URL-related fields on a test
func (s *SQLiteStore) SetTestURLFields(ctx context.Context, name, url, target, ctaTarget, conversionURL string) error {
	now := time.Now().Unix()
	result, err := s.db.ExecContext(ctx,
		`UPDATE tests SET url = ?, target = ?, cta_target = ?, conversion_url = ?, updated_at = ? WHERE name = ?`,
		nullableStringPtr(url), nullableStringPtr(target), nullableStringPtr(ctaTarget), nullableStringPtr(conversionURL), now, name)
	if err != nil {
		return fmt.Errorf("failed to set URL fields: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func nullableStringPtr(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullableString(b []byte) sql.NullString {
	if len(b) == 0 {
		return sql.NullString{}
	}
	return sql.NullString{String: string(b), Valid: true}
}

// SetSetting stores a key-value setting (upserts)
func (s *SQLiteStore) SetSetting(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO settings (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value)
	if err != nil {
		return fmt.Errorf("failed to set setting: %w", err)
	}
	return nil
}

// GetSetting retrieves a setting by key
func (s *SQLiteStore) GetSetting(ctx context.Context, key string) (string, error) {
	var value string
	err := s.db.QueryRowContext(ctx,
		`SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("failed to get setting: %w", err)
	}
	return value, nil
}
