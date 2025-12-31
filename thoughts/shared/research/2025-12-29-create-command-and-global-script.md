---
date: 2025-12-29T17:44:28Z
researcher: Claude
git_commit: 26a671bab5a89320f06a908e4826582fa4925969
branch: refactor/global-script
repository: headline-goat
topic: "CLI create command and global script test discovery"
tags: [research, codebase, cli, global-script, test-creation, api]
status: complete
last_updated: 2025-12-30
last_updated_by: Claude
last_updated_note: "Added UX decisions: CLI-first approach, deferred migration tooling, optional URL targeting"
---

# Research: CLI Create Command and Global Script Test Discovery

**Date**: 2025-12-29T17:44:28Z
**Researcher**: Claude
**Git Commit**: 26a671bab5a89320f06a908e4826582fa4925969
**Branch**: refactor/global-script
**Repository**: headline-goat

## Research Question

How does the current codebase handle test creation, and what exists that would support:
1. A `./headline-goat create` CLI command for creating tests with name, variants, URL, and optional conversion URL
2. Global script checking API for tests matching current URL
3. Fallback to H1/H2/H3 elements when no `data-ht-name` elements exist
4. Preserving backward compatibility with data-attribute based test creation

## Summary

The codebase currently requires tests to be created explicitly via `store.CreateTest()` before events can be recorded. There is no CLI command for test creation - the existing commands are `init`, `list`, `results`, `winner`, `otp`, and `export`. The global JavaScript script (`/ht.js`) processes `data-ht-*` attributes on page load but does not fetch test configurations from an API. Tests are stored in SQLite with a `tests` table containing name, variants (JSON), weights, conversion_goal, and state fields. The beacon endpoint (`/b`) validates that tests exist before recording events.

## Detailed Findings

### Current CLI Structure

**Location**: `internal/cli/`

The CLI uses Cobra with a root command and subcommands that self-register via Go's `init()` functions.

**Existing Commands**:
| Command | File | Purpose |
|---------|------|---------|
| `init` | `internal/cli/init.go:18-34` | Start server and show integration instructions |
| `list` | `internal/cli/list.go:13-18` | List all tests with statistics |
| `results` | `internal/cli/results.go:13-19` | Show detailed results for a specific test |
| `winner` | `internal/cli/winner.go:15-73` | Declare winning variant |
| `otp` | `internal/cli/otp.go:11-16` | Show dashboard access token |
| `export` | `internal/cli/export.go:17-27` | Export event data as CSV/JSON |

**Command Registration Pattern** (`internal/cli/init.go:45`):
```go
func init() {
    rootCmd.AddCommand(initCmd)
}
```

**Global Database Path**: All commands access `dbPath` variable set by root's persistent `--db` flag (`internal/cli/root.go:29`).

**No `create` command exists** - tests must currently be created programmatically or via direct database interaction.

### Database Schema

**Location**: `internal/store/sqlite.go:21-49`

**Tests Table**:
```sql
CREATE TABLE IF NOT EXISTS tests (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    variants TEXT NOT NULL,           -- JSON array: ["A", "B", "C"]
    weights TEXT,                     -- Optional JSON array: [0.5, 0.3, 0.2]
    conversion_goal TEXT,             -- Optional description
    state TEXT NOT NULL DEFAULT 'running',
    winner_variant INTEGER,
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at INTEGER NOT NULL DEFAULT (unixepoch())
);
```

**Events Table**:
```sql
CREATE TABLE IF NOT EXISTS events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    test_name TEXT NOT NULL,
    variant INTEGER NOT NULL,
    event_type TEXT NOT NULL,         -- "view" or "convert"
    visitor_id TEXT NOT NULL,
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    FOREIGN KEY (test_name) REFERENCES tests(name)
);
```

**Key Constraint**: Unique index on `(test_name, visitor_id, event_type)` for deduplication (`internal/store/sqlite.go:49`).

**Note**: The schema does **not** currently have a `url` or `conversion_url` field. These would need to be added to support URL-based test matching.

### Test Creation Method

**Location**: `internal/store/sqlite.go:77-116`

```go
func (s *SQLiteStore) CreateTest(ctx context.Context, name string, variants []string, weights []float64, conversionGoal string) (*Test, error)
```

- Marshals variants to JSON string
- Optionally marshals weights to JSON
- Returns populated `Test` struct with generated ID
- Enforces UNIQUE constraint on name

**Current Usage**: Only called from integration tests (`tests/integration/store/sqlite_test.go:48`):
```go
test, err := s.CreateTest(ctx, "hero", []string{"A", "B", "C"}, nil, "")
```

### Store Interface

**Location**: `internal/store/store.go:5-21`

```go
type Store interface {
    CreateTest(ctx context.Context, name string, variants []string, weights []float64, conversionGoal string) (*Test, error)
    GetTest(ctx context.Context, name string) (*Test, error)
    ListTests(ctx context.Context) ([]*Test, error)
    UpdateTestState(ctx context.Context, name string, state TestState, winnerVariant *int) error
    DeleteTest(ctx context.Context, name string) error
    RecordEvent(ctx context.Context, testName string, variant int, eventType string, visitorID string) error
    GetVariantStats(ctx context.Context, testName string) ([]VariantStats, error)
    GetEvents(ctx context.Context, testName string) ([]*Event, error)
    Close() error
}
```

### Global JavaScript Script

**Location**: `internal/server/globaljs.go:30-99`

The script is generated server-side and served at `/ht.js`. It handles:

1. **Visitor ID Management** (lines 34-39):
   - Creates/retrieves UUID in localStorage as `ht_vid`

2. **Test Element Processing** (lines 42-69):
   - Finds elements with `data-ht-name` attribute
   - Extracts variants from `data-ht-variants` JSON
   - Assigns random variant stored in localStorage as `ht_{testname}`
   - Swaps element text content with selected variant
   - Sends view beacon

3. **Convert Element Processing** (lines 72-93):
   - Finds elements with `data-ht-convert` attribute
   - Supports `data-ht-convert-type="url"` for page-load conversion
   - Adds click handler for standard conversions

4. **Beacon Function** (lines 95-97):
   - Uses `navigator.sendBeacon()` to POST to `/b`

**Current Limitation**: The script does **not** fetch test configurations from the server. It relies entirely on `data-ht-*` attributes present in the HTML.

### Beacon Endpoint

**Location**: `internal/server/handlers.go:72-128`

**Method**: POST `/b`

**Request Validation** (lines 109-113):
```go
test, err := s.store.GetTest(ctx, req.TestName)
if err != nil {
    http.Error(w, "Test not found", http.StatusBadRequest)
    return
}
```

**Critical Finding**: The beacon endpoint requires tests to exist before recording events. If a test doesn't exist, the request fails with "Test not found" (400).

### Server Endpoints Summary

| Endpoint | Method | Auth | Purpose |
|----------|--------|------|---------|
| `/health` | GET | No | Health check with DB stats |
| `/b` | POST | No | Beacon tracking (view/convert) |
| `/ht.js` | GET | No | Global JavaScript delivery |
| `/dashboard` | GET | Yes | Dashboard list view |
| `/dashboard/test/{name}` | GET | Yes | Test detail page |
| `/dashboard/api/tests` | GET | Yes | JSON API for test data |

**Missing Endpoints**:
- No public API to list tests (for global script to fetch)
- No API to create tests
- No API to get tests by URL

### Data Attribute API (Current)

**Test Definition**:
```html
<h1 data-ht-name="hero"
    data-ht-variants='["Ship Faster","Build Better"]'>
  Ship Faster
</h1>
```

**Conversion Tracking**:
```html
<button data-ht-convert="hero">Sign Up</button>
```

**URL-based Conversion**:
```html
<span data-ht-convert="hero" data-ht-convert-type="url" hidden></span>
```

**SSR Optimization**:
```html
<h1 data-ht-name="hero"
    data-ht-variants='["A","B"]'
    data-ht-selected="1">B</h1>
```

## Code References

- `internal/cli/root.go:13-21` - Root command definition
- `internal/cli/init.go:18-34` - Init command (starts server)
- `internal/cli/list.go:13-18` - List command
- `internal/cli/results.go:13-19` - Results command
- `internal/cli/winner.go:15-73` - Winner command (factory pattern example)
- `internal/store/sqlite.go:21-49` - Database schema
- `internal/store/sqlite.go:77-116` - CreateTest implementation
- `internal/store/models.go:13-23` - Test struct
- `internal/server/globaljs.go:30-99` - Global script generation
- `internal/server/handlers.go:72-128` - Beacon endpoint

## Architecture Documentation

### Current Test Creation Flow

1. Tests must be created via `store.CreateTest()` (no CLI/API method exists)
2. Client HTML includes `data-ht-*` attributes
3. Global script processes attributes and sends beacons
4. Beacon endpoint validates test exists → records event

### Current Limitation

The `init` command documentation mentions "Tests auto-create when the first beacon arrives" (`internal/cli/init.go:28`), but this functionality is **not implemented**. The beacon handler explicitly rejects unknown tests.

### Patterns Used

- **Cobra CLI**: Commands registered via `init()` functions
- **Factory Pattern**: Winner command uses factory for flag scoping
- **JSON Storage**: Arrays stored as JSON strings in SQLite
- **Embedded Assets**: Dashboard CSS/templates via `//go:embed`
- **localStorage**: Client-side visitor ID and variant persistence

## What Would Need to Change

To implement the requested features:

### 1. Database Schema Changes
- Add `url TEXT` field to tests table for URL matching
- Add `conversion_url TEXT` field for conversion page matching

### 2. New CLI Command
- Add `internal/cli/create.go` with `create` command
- Accept: name, variants (comma-separated or JSON), url, optional conversion-url

### 3. New API Endpoint
- Add `GET /api/tests?url={url}` to return tests matching a URL
- Public endpoint (no auth) for global script to call

### 4. Global Script Changes
- On page load, fetch tests from `/api/tests?url={currentURL}`
- If tests returned: apply variants to matching `data-ht-name` elements or fallback to H1/H2/H3
- If no tests returned: fall back to current data-attribute behavior (for backward compatibility)

### 5. Auto-Create for Data Attributes
- Add `POST /api/tests` endpoint for auto-creating tests
- Global script calls this when it finds `data-ht-*` attributes for unknown tests
- Or: beacon endpoint auto-creates test if it doesn't exist

## Open Questions

1. Should URL matching be exact or pattern-based (e.g., `/blog/*`)?
2. How should multiple tests on the same URL be handled?
3. Should the global script cache test configurations in localStorage?

---

## Follow-up: Design Decision (2025-12-29)

### Client-First Approach with Conflict Detection

**Decision**: Default to client-side (HTML data attributes) since it avoids a round-trip API call. The beacon will indicate the test source type, and the dashboard will surface conflicts.

### Test Source Types

| Type | Source | Created By | API Call Needed |
|------|--------|------------|-----------------|
| `client` | HTML `data-ht-*` attributes | Auto-created on first beacon | No |
| `server` | CLI `./headline-goat create` | User via CLI | Yes (for URL-based tests) |

### Beacon Payload Changes

Current beacon payload:
```json
{"t": "hero", "v": 1, "e": "view", "vid": "uuid"}
```

New beacon payload with source type:
```json
{"t": "hero", "v": 1, "e": "view", "vid": "uuid", "src": "client"}
```

Where `src` is:
- `"client"` - Test defined via HTML data attributes
- `"server"` - Test fetched from API (CLI-created with URL matching)

### Conflict Detection

A **conflict** occurs when:
- A test exists in the database with `source = "server"` (created via CLI)
- AND beacons arrive with `src: "client"` for the same test name

This indicates the user created a test via CLI but the page still has `data-ht-*` attributes for the same test name.

### Database Schema Addition

Add `source` column to tests table:
```sql
ALTER TABLE tests ADD COLUMN source TEXT NOT NULL DEFAULT 'client';
-- Values: 'client' or 'server'
```

Add conflict tracking to events or tests:
```sql
ALTER TABLE tests ADD COLUMN has_source_conflict INTEGER NOT NULL DEFAULT 0;
```

### Beacon Handler Logic

```
1. Receive beacon with {t, v, e, vid, src}
2. Look up test by name
3. If test doesn't exist AND src == "client":
   → Auto-create test with source = "client"
4. If test exists:
   → If test.source != beacon.src:
      → Set test.has_source_conflict = 1
   → Record event normally
```

### Dashboard Warning

On test detail page, if `has_source_conflict = 1`:
```
⚠️ Source Conflict Detected

This test was created via CLI but is receiving beacons from HTML
data attributes. This may cause inconsistent variant assignments.

Resolution options:
- Remove data-ht-* attributes from HTML (use server-side test only)
- Delete CLI test and rely on client-side data attributes
```

### Global Script Behavior

**For client-side tests (no changes needed)**:
1. Find `data-ht-*` elements
2. Assign variant locally (localStorage)
3. Send beacon with `src: "client"`

**For server-side tests (new behavior)**:
1. Fetch `GET /api/tests?url={currentURL}`
2. If tests returned for this URL:
   - Apply variant to matching `data-ht-name` element OR first H1/H2/H3
   - Send beacon with `src: "server"`
3. If no server tests, fall back to client behavior

### Implementation Order

1. Add `source` column to tests table
2. Add `src` field to beacon payload
3. Update beacon handler to auto-create client tests
4. Update beacon handler to detect conflicts
5. Add conflict warning to dashboard
6. Add `create` CLI command (creates with `source = "server"`)
7. Add `/api/tests?url=` endpoint for server-side tests
8. Update global script to fetch server tests for URL matching

---

## Follow-up: Final Design Decisions (2025-12-29)

### Decision 1: Auto-Create with Variants in Beacon

Beacon sends variant names on first request to enable auto-creation:

```json
{
  "t": "hero",
  "v": 1,
  "e": "view",
  "vid": "uuid",
  "src": "client",
  "variants": ["Ship Faster", "Build Better"]
}
```

Beacon handler logic:
- If test doesn't exist AND `variants` array provided → auto-create test
- If test exists → ignore `variants` field (test already has its variants)

Global script change:
- Always include `variants` array from `data-ht-variants` attribute in beacon

### Decision 2: Client-Side Variant Assignment for Server Tests

Server-side tests (CLI-created) still use **client-side assignment** via localStorage.

Flow:
1. Script fetches test config from `/api/tests?url={currentURL}`
2. Script assigns variant locally using same localStorage pattern (`ht_{testname}`)
3. Beacon sent with assigned variant

Benefits:
- Consistent with client-side tests
- No server-side session/visitor tracking needed
- Simpler implementation

### Decision 3: Target Selectors for Headlines and CTAs

Add `target` and `cta_target` fields to tests - any valid `querySelector()` selector.

**Schema additions**:
```sql
ALTER TABLE tests ADD COLUMN url TEXT;
ALTER TABLE tests ADD COLUMN conversion_url TEXT;
ALTER TABLE tests ADD COLUMN target TEXT;      -- CSS selector for headline element
ALTER TABLE tests ADD COLUMN cta_target TEXT;  -- CSS selector for conversion element
```

**CLI usage**:
```bash
./headline-goat create "hero" \
  --variants "Ship Faster,Build Better" \
  --url "https://example.com/" \
  --target "h1"                         # or ".hero-title", "#main-heading"
  --cta-target "button.signup"          # or "#cta", ".cta-button"
  --conversion-url "https://example.com/thank-you"  # optional
```

**Global script behavior for server-side tests**:
```javascript
// Fetch test config
const tests = await fetch(S + '/api/tests?url=' + encodeURIComponent(location.href));

for (const test of tests) {
  // Apply variant to headline
  const headlineEl = document.querySelector(test.target);
  if (headlineEl) {
    const v = getOrAssignVariant(test.name, test.variants.length);
    headlineEl.textContent = test.variants[v];
    beacon(test.name, v, 'view', 'server');
  }

  // Attach conversion handler
  if (test.cta_target) {
    const ctaEl = document.querySelector(test.cta_target);
    if (ctaEl) {
      ctaEl.addEventListener('click', () => {
        const v = localStorage.getItem('ht_' + test.name);
        beacon(test.name, parseInt(v), 'convert', 'server');
      });
    }
  }
}
```

### Decision 4: Exact URL Matching

URL matching is **exact** (no patterns like `/blog/*`).

- Simpler implementation
- Can add pattern matching later if needed
- Users can create multiple tests for different URLs

### Decision 5: Mutually Exclusive Conversion Methods

`cta_target` and `conversion_url` are **mutually exclusive** - CLI enforces one or the other.

| Method | Use Case |
|--------|----------|
| `--cta-target` | Click conversion on same page (e.g., "Sign Up" button) |
| `--conversion-url` | Page-load conversion on different URL (e.g., "/thank-you") |

**CLI validation**:
```bash
# Valid: click conversion
./headline-goat create "hero" --url "..." --target "h1" --cta-target "button.signup"

# Valid: URL conversion
./headline-goat create "hero" --url "..." --target "h1" --conversion-url "/thank-you"

# Invalid: both specified
./headline-goat create "hero" --cta-target "..." --conversion-url "..."
# Error: specify either --cta-target or --conversion-url, not both
```

**Note**: If neither is provided, the test tracks views only (no conversion tracking).

### Final Schema

```sql
CREATE TABLE IF NOT EXISTS tests (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    variants TEXT NOT NULL,              -- JSON array
    weights TEXT,                        -- Optional JSON array
    conversion_goal TEXT,
    state TEXT NOT NULL DEFAULT 'running',
    winner_variant INTEGER,
    source TEXT NOT NULL DEFAULT 'client',  -- 'client' or 'server'
    has_source_conflict INTEGER NOT NULL DEFAULT 0,
    url TEXT,                            -- For server-side tests (exact match)
    conversion_url TEXT,                 -- Optional URL-based conversion
    target TEXT,                         -- CSS selector for headline
    cta_target TEXT,                     -- CSS selector for CTA
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at INTEGER NOT NULL DEFAULT (unixepoch())
);
```

### Final Beacon Payload

```json
{
  "t": "hero",
  "v": 1,
  "e": "view",
  "vid": "visitor-uuid",
  "src": "client",
  "variants": ["Ship Faster", "Build Better"]
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `t` | string | yes | Test name |
| `v` | int | yes | Variant index (0-based) |
| `e` | string | yes | Event type: "view" or "convert" |
| `vid` | string | yes | Visitor ID |
| `src` | string | yes | Source: "client" or "server" |
| `variants` | []string | no | Variant names (for auto-creation) |

---

## Follow-up: UX Design Decisions (2025-12-30)

### Decision 6: CLI-First Approach

**Decision**: The primary "first test" experience is CLI-first, not data-attribute-first.

**Recommended flow**:
```bash
./headline-goat init
./headline-goat create "hero" --variants "A,B"
# Add to site: <script src=".../ht.js"></script>
# Visit page → test applied
```

**Rationale**:
- Explicit test creation gives users control
- CLI creates the test before any beacons arrive
- Cleaner mental model: "create test, then use it"
- Data attributes still work (auto-create on beacon) but are secondary

### Decision 7: Local → Production Migration (Deferred)

**Decision**: No special tooling for local → production migration in v1.

**Current approach**: Users can:
- Re-run CLI commands on production server
- Or use data attributes (portable in HTML)

**Not building**:
- Export/import commands
- Config file for test definitions
- Sync between environments

**Rationale**: Keep it simple. Can add migration tooling later if users request it.

### Decision 8: Optional URL Targeting

**Decision**: `--url` and `--target` flags are **optional** for the `create` command.

**Minimal usage** (works with data attributes):
```bash
./headline-goat create "hero" --variants "A,B"
```

**Full usage** (URL matching + CSS selector targeting):
```bash
./headline-goat create "hero" --variants "A,B" --url "/" --target "h1"
```

**Behavior when optional flags omitted**:
- Without `--url`: Test is created but won't be returned by `/api/tests?url=` endpoint
- Without `--target`: Script won't auto-apply variant to any element
- Test can still be used via `data-ht-name="hero"` in HTML

**Rationale**:
- Preserves flexibility for different use cases
- Allows pre-creating a test via CLI, then using data attributes to target it
- Users can start simple and add URL/target later if needed

### Updated Implementation Order

Based on CLI-first approach:

1. Add `create` CLI command (minimal: name + variants)
2. Update beacon handler to auto-create client tests (for data-attribute flow)
3. Add schema columns: `source`, `url`, `target`, `cta_target`, `conversion_url`
4. Add `src` and `variants` fields to beacon payload
5. Update beacon handler to detect source conflicts
6. Add conflict warning to dashboard
7. Add `/api/tests?url=` endpoint for server-side tests
8. Update global script to fetch server tests for URL matching
