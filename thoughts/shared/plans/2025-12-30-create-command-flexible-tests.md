# Create Command & Flexible Test Creation Implementation Plan

## Overview

Add a `create` CLI command for explicit test creation and enable flexible test definition via either CLI or HTML data attributes. Tests can be created upfront via CLI (server source) or auto-created when beacons arrive from data attributes (client source). The system tracks source type and detects conflicts when both methods are used for the same test.

## Current State Analysis

**What exists:**
- CLI commands: `init`, `list`, `results`, `winner`, `otp`, `export` (`internal/cli/`)
- Global script `/ht.js` processes `data-ht-*` attributes (`internal/server/globaljs.go:30-99`)
- Beacon endpoint `/b` validates test exists before recording (`internal/server/handlers.go:109-113`)
- `store.CreateTest()` exists but no CLI exposes it (`internal/store/sqlite.go:77-116`)

**What's missing:**
- No CLI command to create tests
- Beacon rejects unknown tests (no auto-create)
- No source tracking (client vs server)
- No URL-based test matching
- No `/api/tests?url=` endpoint for server-side tests

### Key Discoveries:
- CLI uses Cobra with factory pattern for commands with flags (`internal/cli/winner.go:15-73`)
- Schema uses `CREATE TABLE IF NOT EXISTS` - add `ALTER TABLE` for migrations (`internal/store/sqlite.go:64-68`)
- Beacon payload is minimal JSON: `{t, v, e, vid}` (`internal/server/handlers.go:65-70`)
- Global script assigns variants client-side via localStorage (`internal/server/globaljs.go:55-62`)

## Desired End State

After implementation:
1. Users can create tests via CLI: `./headline-goat create "hero" --variants "A,B"`
2. Users can optionally specify URL targeting: `--url "/" --target "h1"`
3. Data-attribute tests auto-create on first beacon (backward compatible)
4. Dashboard shows source type and warns on conflicts
5. Global script fetches server-side tests for URL matching

### Verification:
```bash
# Create a test
./headline-goat create "hero" --variants "Ship Faster,Build Better"

# List shows test with source=server
./headline-goat list

# Data-attribute test auto-creates
# (visit page with data-ht-name, check list shows source=client)
```

## What We're NOT Doing

- **Multi-site/multi-tenant support** - Single headline-goat instance per site (see Design Decisions)
- **Pattern-based URL matching** (e.g., `/blog/*`) - Exact match only for v1
- **Export/import for environment migration** - Users re-run CLI commands or use data attributes
- **Weighted traffic distribution via CLI** - Can add later, focus on core functionality
- **Server-side variant assignment** - Keep client-side localStorage pattern

## Design Decisions

### Single-Tenant Model
One headline-goat instance serves one site. Each site gets its own instance with its own database. This keeps the tool minimal and avoids:
- Cross-site data mixing
- Token/API key management
- Origin validation complexity

If a user has multiple sites, they deploy multiple headline-goat instances.

### Source Tracking
- `source = "client"` - Test auto-created from data attributes
- `source = "server"` - Test created via CLI

Conflict occurs when a server-created test receives client-sourced beacons (user has both CLI test AND data attributes for same name).

### CLI-First Approach
Primary flow is: create test via CLI → add script to site → test runs.
Data attributes still work (auto-create) but are secondary.

## Implementation Approach

Five phases, each independently testable:
1. **CLI create command** - Core functionality, no schema changes yet
2. **Schema + auto-create** - Add columns, enable beacon auto-creation
3. **Source tracking** - Track source, detect conflicts, dashboard warnings
4. **URL-based API** - Add endpoint for fetching tests by URL
5. **Global script updates** - Fetch and apply server-side tests

---

## Phase 1: CLI `create` Command

### Overview
Add minimal `create` command that accepts test name and variants. Uses existing `store.CreateTest()` method.

### Changes Required:

#### 1. New CLI Command
**File**: `internal/cli/create.go` (new file)

```go
package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/lancekey/headline-goat/internal/store"
	"github.com/spf13/cobra"
)

func newCreateCmd() *cobra.Command {
	var variants string

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new A/B test",
		Long: `Create a new A/B test with the specified name and variants.

Examples:
  headline-goat create "hero" --variants "Ship Faster,Build Better"
  headline-goat create "cta" --variants "Sign Up,Get Started,Try Free"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			testName := args[0]

			// Parse variants
			variantList := strings.Split(variants, ",")
			for i := range variantList {
				variantList[i] = strings.TrimSpace(variantList[i])
			}

			if len(variantList) < 2 {
				return fmt.Errorf("at least 2 variants required")
			}

			// Open database
			dbPath, _ := cmd.Flags().GetString("db")
			s, err := store.Open(dbPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer s.Close()

			ctx := context.Background()

			// Create test
			test, err := s.CreateTest(ctx, testName, variantList, nil, "")
			if err != nil {
				return fmt.Errorf("failed to create test: %w", err)
			}

			fmt.Printf("Created test '%s' with %d variants:\n", test.Name, len(test.Variants))
			for i, v := range test.Variants {
				fmt.Printf("  %d: %s\n", i, v)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&variants, "variants", "v", "", "comma-separated variant names (required)")
	cmd.MarkFlagRequired("variants")

	return cmd
}

func init() {
	rootCmd.AddCommand(newCreateCmd())
}
```

#### 2. Test File
**File**: `tests/integration/cli/create_test.go` (new file)

```go
package cli_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateCommand_Success(t *testing.T) {
	// Setup temp database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Run create command
	cmd := exec.Command("go", "run", "../../cmd/headline-goat",
		"--db", dbPath,
		"create", "hero",
		"--variants", "Ship Faster,Build Better")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("create command failed: %v\nOutput: %s", err, output)
	}

	// Verify output
	if !strings.Contains(string(output), "Created test 'hero'") {
		t.Errorf("expected success message, got: %s", output)
	}
	if !strings.Contains(string(output), "Ship Faster") {
		t.Errorf("expected variant in output, got: %s", output)
	}
}

func TestCreateCommand_RequiresVariants(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	cmd := exec.Command("go", "run", "../../cmd/headline-goat",
		"--db", dbPath,
		"create", "hero")
	output, err := cmd.CombinedOutput()

	// Should fail - variants required
	if err == nil {
		t.Fatalf("expected error for missing variants, got success: %s", output)
	}
}

func TestCreateCommand_MinimumTwoVariants(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	cmd := exec.Command("go", "run", "../../cmd/headline-goat",
		"--db", dbPath,
		"create", "hero",
		"--variants", "Only One")
	output, err := cmd.CombinedOutput()

	if err == nil {
		t.Fatalf("expected error for single variant, got success: %s", output)
	}
	if !strings.Contains(string(output), "at least 2 variants") {
		t.Errorf("expected minimum variants error, got: %s", output)
	}
}

func TestCreateCommand_DuplicateName(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create first test
	cmd := exec.Command("go", "run", "../../cmd/headline-goat",
		"--db", dbPath,
		"create", "hero",
		"--variants", "A,B")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("first create failed: %v\nOutput: %s", err, output)
	}

	// Try to create duplicate
	cmd = exec.Command("go", "run", "../../cmd/headline-goat",
		"--db", dbPath,
		"create", "hero",
		"--variants", "C,D")
	output, err := cmd.CombinedOutput()

	if err == nil {
		t.Fatalf("expected error for duplicate name, got success: %s", output)
	}
}
```

### Success Criteria:

#### Automated Verification:
- [x] All tests pass: `go test ./... -v -race`
- [x] Build succeeds: `go build -o headline-goat ./cmd/headline-goat`
- [x] Create command works: `./headline-goat create "test" --variants "A,B"`
- [x] Help shows create: `./headline-goat --help`

#### Manual Verification:
- [x] Created test appears in `./headline-goat list`
- [x] Error message is clear when variants missing
- [x] Error message is clear for duplicate test name

**Implementation Note**: After completing this phase and all automated verification passes, pause here for manual confirmation before proceeding to Phase 2.

---

## Phase 2: Schema Changes + Auto-Create

### Overview
Add new columns to tests table and enable beacon handler to auto-create tests from client data attributes.

### Changes Required:

#### 1. Schema Migration
**File**: `internal/store/sqlite.go`

Add migration after schema creation (around line 68):

```go
// Apply migrations for new columns
migrations := []string{
	"ALTER TABLE tests ADD COLUMN source TEXT NOT NULL DEFAULT 'client'",
	"ALTER TABLE tests ADD COLUMN has_source_conflict INTEGER NOT NULL DEFAULT 0",
	"ALTER TABLE tests ADD COLUMN url TEXT",
	"ALTER TABLE tests ADD COLUMN conversion_url TEXT",
	"ALTER TABLE tests ADD COLUMN target TEXT",
	"ALTER TABLE tests ADD COLUMN cta_target TEXT",
}

for _, m := range migrations {
	// Ignore errors - column may already exist
	db.Exec(m)
}
```

#### 2. Update Test Model
**File**: `internal/store/models.go`

Add fields to Test struct:

```go
type Test struct {
	ID               int64
	Name             string
	Variants         []string
	Weights          []float64
	ConversionGoal   string
	State            TestState
	WinnerVariant    *int
	Source           string  // "client" or "server"
	HasSourceConflict bool
	URL              string  // For URL-based matching
	ConversionURL    string  // URL-based conversion
	Target           string  // CSS selector for headline
	CTATarget        string  // CSS selector for CTA
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
```

#### 3. Update Store Interface
**File**: `internal/store/store.go`

Add new method:

```go
type Store interface {
	// ... existing methods ...

	// GetOrCreateTest returns existing test or creates new one with given variants
	// Used for auto-creating tests from client data attributes
	GetOrCreateTest(ctx context.Context, name string, variants []string) (*Test, bool, error)
}
```

#### 4. Implement GetOrCreateTest
**File**: `internal/store/sqlite.go`

```go
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
	test, err = s.CreateTest(ctx, name, variants, nil, "")
	if err != nil {
		// Handle race condition - another request may have created it
		if strings.Contains(err.Error(), "UNIQUE constraint") {
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
```

#### 5. Update Beacon Payload
**File**: `internal/server/handlers.go`

Update BeaconRequest struct:

```go
type BeaconRequest struct {
	TestName  string   `json:"t"`
	Variant   int      `json:"v"`
	EventType string   `json:"e"`
	VisitorID string   `json:"vid"`
	Source    string   `json:"src"`      // "client" or "server"
	Variants  []string `json:"variants"` // For auto-creation
}
```

#### 6. Update Beacon Handler
**File**: `internal/server/handlers.go`

Replace test existence check with auto-create logic:

```go
// Get or create test
var test *store.Test
if len(req.Variants) > 0 && req.Source == "client" {
	// Auto-create from client data attributes
	var created bool
	test, created, err = s.store.GetOrCreateTest(ctx, req.TestName, req.Variants)
	if err != nil {
		http.Error(w, "Failed to get or create test", http.StatusInternalServerError)
		return
	}
	if created {
		// Log for debugging
		log.Printf("Auto-created test '%s' with %d variants", test.Name, len(test.Variants))
	}
} else {
	// Existing behavior - test must exist
	test, err = s.store.GetTest(ctx, req.TestName)
	if err != nil {
		http.Error(w, "Test not found", http.StatusBadRequest)
		return
	}
}
```

#### 7. Update Global Script
**File**: `internal/server/globaljs.go`

Update beacon function to include source and variants:

```javascript
function beacon(t,v,e,variants){
  var payload={t:t,v:v,e:e,vid:vid,src:'client'};
  if(variants)payload.variants=variants;
  navigator.sendBeacon(S+'/b',JSON.stringify(payload));
}
```

Update test element processing to pass variants:

```javascript
// In test element processing
beacon(name,v,'view',variants);
```

### Success Criteria:

#### Automated Verification:
- [x] All tests pass: `go test ./... -v -race`
- [x] Migration adds columns without error
- [x] Beacon auto-creates test when variants provided
- [x] Existing tests work without variants in beacon

#### Manual Verification:
- [x] Create test via data attributes (no CLI), verify it appears in list
- [x] CLI-created test still works
- [x] Page with data-ht-* attributes works end-to-end

**Implementation Note**: After completing this phase, pause for manual verification before proceeding to Phase 3.

---

## Phase 3: Source Tracking & Conflict Detection

### Overview
Track test source, detect conflicts when both CLI and data attributes target same test, show warnings in dashboard.

### Changes Required:

#### 1. Update CreateTest for Source
**File**: `internal/store/sqlite.go`

Modify CreateTest to accept source parameter or create separate method:

```go
func (s *SQLiteStore) CreateTestWithSource(ctx context.Context, name string, variants []string, weights []float64, conversionGoal string, source string) (*Test, error)
```

#### 2. Conflict Detection in Beacon Handler
**File**: `internal/server/handlers.go`

After getting test, check for source conflict:

```go
// Check for source conflict
if test.Source != req.Source && !test.HasSourceConflict {
	// Mark conflict
	err = s.store.SetSourceConflict(ctx, test.Name, true)
	if err != nil {
		log.Printf("Failed to set source conflict for test '%s': %v", test.Name, err)
	}
}
```

#### 3. Add SetSourceConflict Method
**File**: `internal/store/sqlite.go`

```go
func (s *SQLiteStore) SetSourceConflict(ctx context.Context, name string, hasConflict bool) error {
	conflict := 0
	if hasConflict {
		conflict = 1
	}
	_, err := s.db.ExecContext(ctx,
		"UPDATE tests SET has_source_conflict = ?, updated_at = ? WHERE name = ?",
		conflict, time.Now().Unix(), name)
	return err
}
```

#### 4. Dashboard Warning
**File**: `internal/dashboard/templates/test_detail.html`

Add warning section:

```html
{{if .Test.HasSourceConflict}}
<div class="warning">
  <strong>Source Conflict Detected</strong>
  <p>This test was created via CLI but is receiving beacons from HTML data attributes.
     This may cause inconsistent variant assignments.</p>
  <p>Resolution options:</p>
  <ul>
    <li>Remove data-ht-* attributes from HTML (use server-side test only)</li>
    <li>Delete CLI test and rely on client-side data attributes</li>
  </ul>
</div>
{{end}}
```

#### 5. Update List Command Output
**File**: `internal/cli/list.go`

Show source in list output:

```go
fmt.Printf("  %s [%s] - %d variants, %d views, %d conversions\n",
	test.Name, test.Source, len(test.Variants), totalViews, totalConversions)
```

### Success Criteria:

#### Automated Verification:
- [x] All tests pass: `go test ./... -v -race`
- [x] CLI-created test has source="server"
- [x] Auto-created test has source="client"
- [x] Conflict detected when sources differ

#### Manual Verification:
- [x] List command shows source type
- [x] Dashboard shows conflict warning when applicable
- [x] Conflict warning is clear and actionable

**Implementation Note**: Pause for manual verification before Phase 4.

---

## Phase 4: URL-Based Test API

### Overview
Add `/api/tests?url=` endpoint for global script to fetch server-side tests matching current URL.

### Changes Required:

#### 1. Add GetTestsByURL Method
**File**: `internal/store/store.go`

```go
GetTestsByURL(ctx context.Context, url string) ([]*Test, error)
```

#### 2. Implement GetTestsByURL
**File**: `internal/store/sqlite.go`

```go
func (s *SQLiteStore) GetTestsByURL(ctx context.Context, url string) ([]*Test, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, variants, weights, conversion_goal, state, winner_variant,
		        source, has_source_conflict, url, conversion_url, target, cta_target,
		        created_at, updated_at
		 FROM tests
		 WHERE url = ? AND state = 'running'`,
		url)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tests []*Test
	for rows.Next() {
		// ... scan into Test struct
		tests = append(tests, test)
	}
	return tests, rows.Err()
}
```

#### 3. Add API Endpoint
**File**: `internal/server/handlers.go`

```go
func (s *Server) handleTestsAPI(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		http.Error(w, "url parameter required", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	tests, err := s.store.GetTestsByURL(ctx, url)
	if err != nil {
		http.Error(w, "Failed to fetch tests", http.StatusInternalServerError)
		return
	}

	// Return minimal test data for client
	type TestResponse struct {
		Name          string   `json:"name"`
		Variants      []string `json:"variants"`
		Target        string   `json:"target,omitempty"`
		CTATarget     string   `json:"cta_target,omitempty"`
		ConversionURL string   `json:"conversion_url,omitempty"`
	}

	var response []TestResponse
	for _, t := range tests {
		response = append(response, TestResponse{
			Name:          t.Name,
			Variants:      t.Variants,
			Target:        t.Target,
			CTATarget:     t.CTATarget,
			ConversionURL: t.ConversionURL,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(response)
}
```

#### 4. Register Endpoint
**File**: `internal/server/server.go`

```go
mux.HandleFunc("/api/tests", s.handleTestsAPI)
```

#### 5. Update Create Command with URL Flags
**File**: `internal/cli/create.go`

Add optional flags:

```go
var (
	variants      string
	url           string
	target        string
	ctaTarget     string
	conversionURL string
)

// In command setup
cmd.Flags().StringVar(&url, "url", "", "URL to match for this test (optional)")
cmd.Flags().StringVar(&target, "target", "", "CSS selector for headline element (optional)")
cmd.Flags().StringVar(&ctaTarget, "cta-target", "", "CSS selector for CTA element (optional)")
cmd.Flags().StringVar(&conversionURL, "conversion-url", "", "URL for page-load conversion (optional)")

// Validation: cta-target and conversion-url are mutually exclusive
if ctaTarget != "" && conversionURL != "" {
	return fmt.Errorf("specify either --cta-target or --conversion-url, not both")
}
```

### Success Criteria:

#### Automated Verification:
- [x] All tests pass: `go test ./... -v -race`
- [x] `/api/tests?url=/` returns tests matching that URL
- [x] Empty array returned when no tests match
- [x] CORS headers present on response

#### Manual Verification:
- [x] Create test with --url, verify it's returned by API
- [x] API returns proper JSON structure
- [x] Create command rejects both --cta-target and --conversion-url

**Implementation Note**: Pause for manual verification before Phase 5.

---

## Phase 5: Global Script Server Test Support

### Overview
Update global script to fetch server-side tests and apply variants to specified selectors. Cache full test config in localStorage to eliminate flash on repeat visits.

### Changes Required:

#### 1. Update Global Script with Caching
**File**: `internal/server/globaljs.go`

Add server test fetching with localStorage cache:

```javascript
// Server-side test handling with cache
(function(){
  var cacheKey='ht_tests_'+location.pathname;
  var cached=localStorage.getItem(cacheKey);

  // Apply cached tests immediately (no flash on repeat visits)
  if(cached){
    try{
      applyServerTests(JSON.parse(cached));
    }catch(e){}
  }

  // Fetch fresh config in background, update cache
  fetch(S+'/api/tests?url='+encodeURIComponent(location.href))
    .then(function(r){return r.json()})
    .then(function(tests){
      // Update cache for next visit
      localStorage.setItem(cacheKey,JSON.stringify(tests));
      // Apply if not already applied from cache
      if(!cached)applyServerTests(tests);
    })
    .catch(function(){});
})();

function applyServerTests(tests){
  tests.forEach(function(test){
    // Skip if already processed via data attributes
    if(document.querySelector('[data-ht-name="'+test.name+'"]'))return;

    // Find target element
    var el=document.querySelector(test.target);
    if(!el)return;

    // Assign variant (same localStorage pattern)
    var key='ht_'+test.name;
    var v=localStorage.getItem(key);
    if(v===null){
      v=Math.floor(Math.random()*test.variants.length);
      localStorage.setItem(key,v);
    }else{
      v=parseInt(v);
    }

    // Apply variant
    el.textContent=test.variants[v];
    beacon(test.name,v,'view',null,'server');

    // Setup conversion tracking
    if(test.cta_target){
      var cta=document.querySelector(test.cta_target);
      if(cta){
        cta.addEventListener('click',function(){
          beacon(test.name,v,'convert',null,'server');
        });
      }
    }
    if(test.conversion_url && location.href.indexOf(test.conversion_url)!==-1){
      beacon(test.name,v,'convert',null,'server');
    }
  });
}
```

**Caching Strategy:**
- Cache key: `ht_tests_{pathname}` - separate cache per URL path
- First visit: Flash occurs (no cache), then cache populated
- Repeat visits: Apply from cache immediately (no flash), fetch in background updates cache
- Config changes: Apply on next page load after fresh fetch

#### 2. Update Beacon Function
**File**: `internal/server/globaljs.go`

Update beacon to accept source:

```javascript
function beacon(t,v,e,variants,src){
  var payload={t:t,v:v,e:e,vid:vid,src:src||'client'};
  if(variants)payload.variants=variants;
  navigator.sendBeacon(S+'/b',JSON.stringify(payload));
}
```

### Success Criteria:

#### Automated Verification:
- [x] All tests pass: `go test ./... -v -race`
- [x] Global script contains fetch logic
- [x] Global script contains cache logic
- [x] Script handles API errors gracefully

#### Manual Verification:
- [ ] Create test with `--url "/" --target "h1"`
- [ ] First visit: verify variant applied (may flash)
- [ ] Second visit: verify no flash (applied from cache)
- [ ] Verify beacon sent with src="server"
- [ ] Data attribute tests still work alongside server tests
- [ ] Update test variants via CLI, verify new variants appear on next visit

**Implementation Note**: Full end-to-end testing recommended after Phase 5.

---

## Testing Strategy

### Unit Tests:
- `tests/unit/store/models_test.go` - Test new fields serialize correctly
- Validate CSS selector format (basic validation)
- Validate mutually exclusive flags

### Integration Tests:
- `tests/integration/cli/create_test.go` - Create command variations
- `tests/integration/server/beacon_test.go` - Auto-create, conflict detection
- `tests/integration/server/api_test.go` - Tests API endpoint

### Manual Testing Steps:
1. Create test via CLI with all flags
2. Visit page, verify variant applied
3. Click CTA, verify conversion recorded
4. Create conflicting data-attribute test, verify warning shows
5. Test with multiple tests on same URL

## Performance Considerations

- API endpoint should be fast (simple SELECT with index on url)
- Add index: `CREATE INDEX idx_tests_url ON tests(url)`
- Global script fetch is async, doesn't block page render
- Full test config cached in localStorage per URL path
  - First visit: may flash (fetching from API)
  - Repeat visits: no flash (applied from cache instantly)
  - Background fetch updates cache for eventual consistency

## Migration Notes

- Schema migrations use ALTER TABLE with ignored errors (column exists)
- No data migration needed - new columns have sensible defaults
- Existing tests get source="client" by default

## References

- Research: `thoughts/shared/research/2025-12-29-create-command-and-global-script.md`
- v2 Refactor: `thoughts/shared/plans/2025-12-28-headline-goat-v2-refactor.md`
- CLI patterns: `internal/cli/winner.go:15-73`
- Beacon handler: `internal/server/handlers.go:72-128`
- Global script: `internal/server/globaljs.go:30-99`
