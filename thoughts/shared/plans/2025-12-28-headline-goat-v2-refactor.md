# headline-goat v2 Refactor Plan

## Overview

This plan refactors headline-goat from per-test scripts to a global script approach with HTML data attributes. The goal is a simpler, more flexible tool where configuration lives in HTML and tests auto-create.

## TDD Methodology

Every change follows this strict order:

1. **Identify affected tests** - Which existing tests will break?
2. **Update/delete affected tests** - Modify expectations or remove obsolete tests
3. **Write new failing tests** - Define new behavior
4. **Implement code** - Minimal code to pass tests
5. **Run full suite** - `go test ./... -v -race` must pass
6. **Refactor** - Improve code, keep tests green

## Existing Test Inventory

| Test File | What It Tests | Impact |
|-----------|--------------|--------|
| `tests/unit/store/models_test.go` | Model structs with Variants, Weights, ConversionGoal | **MODIFY** - Remove variant/weight/goal fields |
| `tests/integration/store/sqlite_test.go` | CreateTest with variants/weights, RecordEvent, GetVariantStats | **MODIFY** - Simplify CreateTest, add GetOrCreateTest |
| `tests/integration/server/beacon_test.go` | Beacon endpoint, requires test to exist first | **MODIFY** - Test auto-creation, remove variant validation |
| `tests/integration/server/clientjs_test.go` | Per-test `/t/<test>.js` endpoint | **DELETE** - Replaced by global script |
| `tests/integration/server/dashboard_test.go` | Dashboard auth, API, detail page with variant names | **MODIFY** - Show variants by index |
| `tests/unit/snippets/generator_test.go` | Snippet generation for all frameworks | **DELETE** - Snippets package removed |
| `tests/integration/cli/snippet_test.go` | CLI snippet command | **DELETE** - Snippet command removed |
| `tests/integration/cli/winner_test.go` | SetWinner, state transitions | **KEEP** - Minor updates for schema |
| `tests/unit/stats/significance_test.go` | Statistical calculations | **KEEP** - No changes needed |
| `tests/unit/stats/wilson_test.go` | Wilson score intervals | **KEEP** - No changes needed |

## Key Changes from v1

| Aspect | v1 (Current) | v2 (New) |
|--------|--------------|----------|
| Test config | Server-side via `init` command | HTML data attributes |
| Script | Per-test (`/t/hero.js`) | Global (`/ht.js`) |
| Test creation | Explicit `init` command | Auto-create on first beacon |
| CLI startup | `serve` command | `init` (or just `headline-goat`) starts server |
| Snippets | Generated per framework | Instructions shown at startup |
| Variants storage | Stored in database | Not stored (come from HTML) |

## What We Keep

- SQLite database with WAL mode
- Beacon endpoint (`POST /b`)
- Dashboard UI and auth
- Stats calculation (Wilson intervals, z-test)
- `results`, `winner`, `list`, `export`, `otp` commands
- TDD approach

## What We Remove/Change

- Remove `serve` command (merged into default/init)
- Remove `snippet` command (replaced by startup instructions)
- Remove `/t/<test>.js` endpoint (replaced by `/ht.js`)
- Remove variants from database schema
- Simplify `init` command (no variants input)

---

## Phase 1: Global Script & Simplified CLI

### Goal
Replace per-test scripts with global script. Combine `init` and `serve` into single command.

### Step 1.1: Delete Obsolete Tests (RED→GREEN by removal)

**Files to delete:**
- `tests/integration/server/clientjs_test.go` - Tests `/t/<test>.js` which we're removing
- `tests/unit/snippets/generator_test.go` - Tests snippets package which we're removing
- `tests/integration/cli/snippet_test.go` - Tests snippet CLI which we're removing

**Verify:**
```bash
go test ./... -v -race  # Should pass (fewer tests)
```

### Step 1.2: Write Global Script Tests (RED)

**Create:** `tests/unit/globaljs/script_test.go`

```go
func TestGenerateGlobalScript_ReturnsValidJS(t *testing.T)
func TestGenerateGlobalScript_ContainsBeaconEndpoint(t *testing.T)
func TestGenerateGlobalScript_ContainsLocalStorageLogic(t *testing.T)
func TestGenerateGlobalScript_ContainsDataAttributeSelectors(t *testing.T)
func TestGenerateGlobalScript_ContainsVariantAssignment(t *testing.T)
```

**Create:** `tests/integration/server/globaljs_test.go`

```go
func TestGlobalJS_ReturnsJavaScript(t *testing.T)
func TestGlobalJS_HasCorrectContentType(t *testing.T)
func TestGlobalJS_HasCacheHeaders(t *testing.T)
func TestGlobalJS_ContainsServerURL(t *testing.T)
```

**Verify tests fail:**
```bash
go test ./tests/unit/globaljs/... -v  # Should fail (no implementation)
go test ./tests/integration/server/... -v  # Should fail
```

### Step 1.3: Implement Global Script (GREEN)

**Create:** `internal/server/globaljs.go`

```go
func (s *Server) handleGlobalJS(w http.ResponseWriter, r *http.Request) {
    // Return the global ht.js script
    // Template with server URL
}
```

The script:
- Finds `[data-ht-name]` elements, assigns variants, swaps text, beacons view
- Finds `[data-ht-convert]` elements, handles click conversions
- Uses localStorage for visitor ID and variant persistence
- Templates server URL from request origin (for deployment flexibility)

**Update:** `internal/server/server.go`

```go
func (s *Server) setupRoutes() {
    s.router.HandleFunc("/ht.js", s.handleGlobalJS)      // NEW
    s.router.HandleFunc("/health", s.handleHealth)
    s.router.HandleFunc("/b", s.handleBeacon)
    // Remove: s.router.HandleFunc("/t/", s.handleClientJS)
    // Dashboard routes unchanged
}
```

**Verify:**
```bash
go test ./... -v -race  # All tests should pass
```

### Step 1.4: Remove Old Client JS Handler

**Delete:** `handleClientJS` function from `internal/server/handlers.go`

**Verify:**
```bash
go test ./... -v -race  # Should pass
```

### Step 1.5: Update CLI - Combine init + serve

**Update affected tests first:**

No existing CLI tests for init/serve workflow, but update `internal/cli/init.go`:
- Remove variants input prompts
- Remove snippet generation call
- Start server instead

**Update:** `internal/cli/root.go`

```go
var rootCmd = &cobra.Command{
    Use:   "headline-goat",
    Short: "A minimal, self-hosted A/B testing tool",
    RunE:  runInit, // Default action is to start server
}
```

**Delete:** `internal/cli/serve.go`, `internal/cli/snippet.go`

**Verify:**
```bash
go test ./... -v -race
go build -o headline-goat ./cmd/headline-goat
./headline-goat --help  # Verify no serve/snippet commands
```

### Step 1.6: Delete Snippets Package

**Delete:** `internal/snippets/` directory

**Verify:**
```bash
go test ./... -v -race  # Should pass
go build -o headline-goat ./cmd/headline-goat
```

### Phase 1 Success Criteria

- [ ] `go test ./... -v -race` passes
- [ ] `/ht.js` returns valid JavaScript
- [ ] Script correctly finds and processes data attributes
- [ ] Script sends view beacons
- [ ] Script handles convert clicks
- [ ] `./headline-goat` starts server and shows instructions
- [ ] Old `/t/<test>.js` endpoint removed
- [ ] `snippet` command removed
- [ ] `serve` command removed

---

## Phase 2: Auto-Create Tests & Schema Simplification

### Goal
Auto-create tests on first beacon. Remove variants from database schema.

### Step 2.1: Update Model Tests (RED→GREEN)

**Modify:** `tests/unit/store/models_test.go`

```go
// BEFORE
func TestTest_Struct(t *testing.T) {
    test := store.Test{
        Variants:       []string{"A", "B", "C"},
        Weights:        []float64{0.5, 0.3, 0.2},
        ConversionGoal: "Signup button click",
        // ...
    }
}

// AFTER
func TestTest_Struct(t *testing.T) {
    test := store.Test{
        ID:            1,
        Name:          "hero",
        State:         store.StateRunning,
        WinnerVariant: nil,
        // Remove: Variants, Weights, ConversionGoal
    }
}
```

**Remove tests:**
- `TestVariants_JSONEncoding`
- `TestWeights_JSONEncoding`

**Verify tests fail (model still has old fields):**
```bash
go test ./tests/unit/store/... -v  # Should fail
```

### Step 2.2: Update Store Tests (RED)

**Modify:** `tests/integration/store/sqlite_test.go`

```go
// BEFORE
func TestCreateTest(t *testing.T) {
    test, err := s.CreateTest(ctx, "hero", []string{"A", "B", "C"}, nil, "")
    if len(test.Variants) != 3 { ... }
}

// AFTER
func TestCreateTest(t *testing.T) {
    test, err := s.CreateTest(ctx, "hero")
    if test.Name != "hero" { ... }
    if test.State != store.StateRunning { ... }
}
```

**Add new tests:**
```go
func TestGetOrCreateTest_CreatesNew(t *testing.T) {
    test, err := s.GetOrCreateTest(ctx, "newtest")
    // Should create and return new test
}

func TestGetOrCreateTest_ReturnsExisting(t *testing.T) {
    s.CreateTest(ctx, "existing")
    test, err := s.GetOrCreateTest(ctx, "existing")
    // Should return existing test, not create duplicate
}
```

**Remove tests:**
- `TestCreateTest_WithWeightsAndGoal`

**Modify tests:**
- `TestGetTest` - Remove ConversionGoal assertion
- `TestRecordEvent` - Should work same (variant index comes from beacon)
- All tests creating tests with variants - simplify to just name

**Verify tests fail:**
```bash
go test ./tests/integration/store/... -v  # Should fail
```

### Step 2.3: Update Models (GREEN)

**Modify:** `internal/store/models.go`

```go
// BEFORE
type Test struct {
    ID             int64
    Name           string
    Variants       []string
    Weights        []float64
    ConversionGoal string
    State          TestState
    WinnerVariant  *int
    CreatedAt      time.Time
    UpdatedAt      time.Time
}

// AFTER
type Test struct {
    ID            int64
    Name          string
    State         TestState
    WinnerVariant *int
    CreatedAt     time.Time
    UpdatedAt     time.Time
}
```

**Verify:**
```bash
go test ./tests/unit/store/... -v  # Should pass now
```

### Step 2.4: Update Store Implementation (GREEN)

**Modify:** `internal/store/sqlite.go`

```sql
-- New schema
CREATE TABLE tests (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    state TEXT NOT NULL DEFAULT 'running',
    winner_variant INTEGER,
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at INTEGER NOT NULL DEFAULT (unixepoch())
);
```

```go
// Update CreateTest signature
func (s *SQLiteStore) CreateTest(ctx context.Context, name string) (*Test, error)

// Add GetOrCreateTest
func (s *SQLiteStore) GetOrCreateTest(ctx context.Context, name string) (*Test, error)
```

**Verify:**
```bash
go test ./tests/integration/store/... -v  # Should pass now
```

### Step 2.5: Update Beacon Handler Tests (RED)

**Modify:** `tests/integration/server/beacon_test.go`

```go
// BEFORE - Requires test to exist first
func TestBeacon_ValidRequest(t *testing.T) {
    _, err := s.CreateTest(ctx, "hero", []string{"A", "B", "C"}, nil, "")
    // Send beacon
}

// AFTER - Test auto-creates
func TestBeacon_AutoCreatesTest(t *testing.T) {
    // Send beacon for non-existent test
    payload := map[string]interface{}{
        "t":   "newtest",  // Does not exist
        "v":   0,
        "e":   "view",
        "vid": "visitor123",
    }
    // Should succeed and create test
}

// BEFORE - Invalid test returns 400
func TestBeacon_InvalidTest(t *testing.T) {
    // Expects 400 for non-existent test
}

// AFTER - DELETE THIS TEST (tests auto-create now)

// BEFORE - Validates variant index
func TestBeacon_InvalidVariant(t *testing.T) {
    _, err := s.CreateTest(ctx, "hero", []string{"A", "B"}, nil, "")
    payload := map[string]interface{}{"v": 5} // Out of range
    // Expects 400
}

// AFTER - DELETE THIS TEST (variants not stored, accept any index)
```

**Verify tests fail:**
```bash
go test ./tests/integration/server/... -v  # Should fail
```

### Step 2.6: Update Beacon Handler (GREEN)

**Modify:** `internal/server/handlers.go`

```go
func (s *Server) handleBeacon(w http.ResponseWriter, r *http.Request) {
    // Parse beacon
    // Auto-create test if not exists (GetOrCreateTest)
    // Record event (existing logic, no variant validation)
}
```

**Verify:**
```bash
go test ./tests/integration/server/... -v  # Should pass
go test ./... -v -race  # Full suite should pass
```

### Phase 2 Success Criteria

- [ ] Tests auto-create on first beacon
- [ ] Database schema simplified (no variants, weights, goal)
- [ ] GetOrCreateTest method works
- [ ] All store tests pass with new schema
- [ ] Beacon endpoint accepts any variant index
- [ ] `list` command shows auto-created tests

---

## Phase 3: Dashboard & Results Updates

### Goal
Update dashboard and CLI to work without stored variants.

### Step 3.1: Update Dashboard Tests (RED)

**Modify:** `tests/integration/server/dashboard_test.go`

```go
// BEFORE
func TestDashboardAPI_Tests(t *testing.T) {
    _, _ = s.CreateTest(ctx, "hero", []string{"A", "B"}, nil, "Click button")
    // Check body contains "Click button"
}

// AFTER
func TestDashboardAPI_Tests(t *testing.T) {
    _, _ = s.CreateTest(ctx, "hero")
    // Check body contains test name, state
    // Remove: conversion goal assertion
}

// BEFORE
func TestDashboardTest_Detail(t *testing.T) {
    _, _ = s.CreateTest(ctx, "hero", []string{"Ship Faster", "Build Better"}, nil, "")
    // Check body contains "Ship Faster"
}

// AFTER
func TestDashboardTest_Detail(t *testing.T) {
    _, _ = s.CreateTest(ctx, "hero")
    // Record some events to create variant data
    _ = s.RecordEvent(ctx, "hero", 0, "view", "v1")
    _ = s.RecordEvent(ctx, "hero", 1, "view", "v2")
    // Check body contains "Variant 0", "Variant 1"
}
```

**Verify tests fail:**
```bash
go test ./tests/integration/server/dashboard_test.go -v  # Should fail
```

### Step 3.2: Update Dashboard Templates (GREEN)

**Modify:** `internal/dashboard/templates/detail.html`

Show variants as "Variant 0", "Variant 1", etc. (since names aren't stored).

**Modify:** `internal/dashboard/templates/index.html`

Remove conversion goal column.

**Verify:**
```bash
go test ./tests/integration/server/dashboard_test.go -v  # Should pass
```

### Step 3.3: Update Stats Calculation

**Tests:** `tests/unit/stats/significance_test.go` and `wilson_test.go` should remain unchanged - they work with raw numbers.

Stats now work purely from event data:
- Get unique variant indices from events
- Calculate rates per variant

No test changes needed for stats.

### Step 3.4: Update Results Command

**Modify:** `internal/cli/results.go`

Show variants by index:

```
VARIANT      VIEWS    CONVERSIONS  RATE     95% CI
─────────────────────────────────────────────────────
Variant 0    412      32           7.77%    [5.2%, 10.3%]
Variant 1    398      41           10.30%   [7.4%, 13.2%]  ← LEADING
```

No existing tests for results command output format - manual verification.

### Step 3.5: Update Winner Command Tests

**Modify:** `tests/integration/cli/winner_test.go`

```go
// BEFORE
func TestSetWinner_Success(t *testing.T) {
    _, err = s.CreateTest(ctx, "hero", []string{"Ship Faster", "Build Better"}, nil, "")
}

// AFTER
func TestSetWinner_Success(t *testing.T) {
    _, err = s.CreateTest(ctx, "hero")
}
```

Similar updates for all tests in file.

**Verify:**
```bash
go test ./tests/integration/cli/winner_test.go -v  # Should pass after store changes
```

### Phase 3 Success Criteria

- [ ] Dashboard displays variants by index
- [ ] Dashboard API returns tests without variants/goal
- [ ] Results command shows variant indices
- [ ] Winner command works with indices
- [ ] All tests pass

---

## Phase 4: URL Conversion & Polish

### Goal
Add URL-based conversion tracking. Polish the UX.

### Step 4.1: Write URL Conversion Tests (RED)

**Add to:** `tests/unit/globaljs/script_test.go`

```go
func TestGlobalScript_URLConversion_ContainsLogic(t *testing.T)
func TestGlobalScript_ConvertVariants_ContainsLogic(t *testing.T)
func TestGlobalScript_SSRSelected_ContainsLogic(t *testing.T)
```

**Verify tests fail:**
```bash
go test ./tests/unit/globaljs/... -v  # Should fail
```

### Step 4.2: Update Global Script (GREEN)

**Modify:** `internal/server/globaljs.go`

Add URL conversion handling:
```javascript
if (el.dataset.htConvertType === 'url') {
    beacon(name, v, 'convert');
    return;
}
```

Add convert button variants:
```javascript
var variants = el.dataset.htConvertVariants;
if (variants) {
    variants = JSON.parse(variants);
    if (variants[v]) el.textContent = variants[v];
}
```

Add SSR support:
```javascript
if (el.dataset.htSelected !== undefined) {
    beacon(name, parseInt(el.dataset.htSelected), 'view');
    return;
}
```

**Verify:**
```bash
go test ./tests/unit/globaljs/... -v  # Should pass
```

### Step 4.3: Update Startup Instructions

**Modify:** `internal/cli/init.go`

Add framework-specific instructions shown at startup:

```go
switch framework {
case "html":
    printHTMLInstructions()
case "react", "nextjs":
    printReactInstructions()
case "vue":
    printVueInstructions()
}
```

**Important:** Instructions should use deployment-oriented URLs:
- Show `https://YOUR-DEPLOYED-URL/ht.js` instead of `localhost`
- Include link to deployment documentation
- Remind user to deploy before adding script to production site

Manual testing only - no automated tests for console output.

### Phase 4 Success Criteria

- [ ] URL-based conversion works (script contains logic)
- [ ] Convert button text variants work
- [ ] SSR `data-ht-selected` prevents flash
- [ ] Framework instructions are helpful
- [ ] All tests pass

---

## Phase 5: Cleanup & Documentation

### Goal
Remove unused code, update documentation, final testing.

### Step 5.1: Remove Unused Code

- Verify no dangling imports
- Clean up any dead code paths
- Remove unused helper functions

```bash
go vet ./...
go test ./... -v -race
```

### Step 5.2: Update CLAUDE.md

Update project documentation for new architecture:
- Update architecture section
- Update command list
- Update data flow

### Step 5.3: Update README

Create user-facing documentation with:
- Quick start (deployment-first approach)
- Deployment guide (Cloudflare Tunnel, Fly.io, VPS)
- Data attributes reference
- Framework integration examples
- FAQ

**Important:** README should lead with deployment since users need a URL before adding the script to their site. Examples should use `https://ht.example.com` or `https://YOUR-DEPLOYED-URL`.

### Step 5.4: Final Test Pass

```bash
go test ./... -v -race
go build -o headline-goat ./cmd/headline-goat
./headline-goat  # Manual testing
```

### Phase 5 Success Criteria

- [ ] No unused code
- [ ] `go vet ./...` passes
- [ ] Documentation complete
- [ ] All tests pass
- [ ] Binary builds successfully
- [ ] Manual testing passes

---

## Test File Summary After Refactor

| Test File | Status |
|-----------|--------|
| `tests/unit/store/models_test.go` | Modified (fewer fields) |
| `tests/integration/store/sqlite_test.go` | Modified (simpler CreateTest, new GetOrCreateTest) |
| `tests/integration/server/beacon_test.go` | Modified (auto-creation, no variant validation) |
| `tests/integration/server/clientjs_test.go` | **DELETED** |
| `tests/integration/server/dashboard_test.go` | Modified (variants by index) |
| `tests/unit/snippets/generator_test.go` | **DELETED** |
| `tests/integration/cli/snippet_test.go` | **DELETED** |
| `tests/integration/cli/winner_test.go` | Modified (simpler test creation) |
| `tests/unit/stats/significance_test.go` | Unchanged |
| `tests/unit/stats/wilson_test.go` | Unchanged |
| `tests/unit/globaljs/script_test.go` | **NEW** |
| `tests/integration/server/globaljs_test.go` | **NEW** |

---

## Migration Notes

### For existing v1 users

1. v2 uses a new database schema (variants not stored)
2. Export data before upgrading if needed
3. Replace per-test scripts with global script
4. Update HTML to use data attributes

### Breaking changes

- `/t/<test>.js` endpoint removed
- `init` command no longer accepts variants
- `snippet` command removed
- Database schema changed (fresh DB recommended)

---

## Phase Order Summary

| Phase | TDD Focus | Key Test Changes |
|-------|-----------|------------------|
| 1 | Delete obsolete tests, write new globaljs tests | -3 files, +2 files |
| 2 | Update store/model tests, beacon tests | Modify signatures, remove variant validation |
| 3 | Update dashboard tests | Remove variant names, use indices |
| 4 | Add URL conversion tests | Extend globaljs tests |
| 5 | Final verification | No test changes, just cleanup |

Each phase produces a working, testable increment following strict TDD principles.
