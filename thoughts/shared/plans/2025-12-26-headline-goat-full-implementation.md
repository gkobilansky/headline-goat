# headline-goat Full Implementation Plan

## Overview

This plan covers the complete implementation of headline-goat, a minimal self-hosted A/B testing tool for headlines. The implementation follows strict TDD principles and is broken into 5 independently deployable phases.

## Current State Analysis

**What exists:**
- `SPEC.md` - Complete feature specification
- `CLAUDE.md` - Development guidelines
- Git repository initialized

**What's missing:**
- All source code
- Go module configuration
- Tests
- Build configuration

## Desired End State

A fully functional CLI tool that:
1. Creates and manages A/B tests for headlines
2. Serves a beacon endpoint to track views/conversions
3. Provides statistical analysis with confidence intervals
4. Displays results in a web dashboard
5. Generates framework-specific code snippets

**Verification:** All commands from SPEC.md work as documented, all tests pass with `go test ./... -v -race`.

## Core Concepts

### Events

The tool tracks two event types per test:

| Event | Meaning | When to Fire |
|-------|---------|--------------|
| **Impression (view)** | User saw a variant | On page load / component mount |
| **Conversion** | User did the thing you care about | On CTA click, form submit, etc. |

The tool doesn't care *what* the conversion is - it just needs to know that test X, variant Y, got a conversion. Users describe their goal via `--goal "Signup button click"` for their own reference.

### Two Integration Patterns

The beacon endpoint (`POST /b`) is the same for everyone. What differs is how you integrate:

| Pattern | Best For | Variant Storage | How It Works |
|---------|----------|-----------------|--------------|
| **Script-based** | Vanilla HTML, WordPress, static sites | localStorage | Drop in `<script>`, everything automatic |
| **Component-based** | React, Next.js, Vue, Laravel, Django | Cookies | Use middleware + components, explicit control |

#### Pattern 1: Script-Based (Vanilla JS)

```html
<!-- Add test attribute where headline goes -->
<h1 data-ht="hero">Loading...</h1>

<!-- Add convert attribute to CTA -->
<button data-ht-convert="hero">Sign Up</button>

<!-- Include script (handles everything) -->
<script src="https://your-server.com/t/hero.js"></script>
```

The script:
- Picks/stores variant in localStorage
- Injects text into `[data-ht="hero"]`
- Fires impression beacon automatically
- Binds conversion to `[data-ht-convert="hero"]` clicks
- Exposes `HT.hero()` for programmatic conversion

#### Pattern 2: Component-Based (React/Next.js Example)

```tsx
// middleware.ts - Pick variant server-side
export function middleware(request: NextRequest) {
  const response = NextResponse.next();
  if (!request.cookies.has('ht_hero')) {
    response.cookies.set('ht_hero', String(Math.floor(Math.random() * 3)));
  }
  return response;
}

// page.tsx - Render correct variant (SSR, no flash)
export default async function Home() {
  const cookieStore = await cookies();
  const variant = Number(cookieStore.get('ht_hero')?.value ?? 0);
  const variants = ["Ship Faster", "Build Better", "Scale Smart"];

  return (
    <>
      <TrackImpression test="hero" variant={variant} />
      <h1>{variants[variant]}</h1>
      <ConvertButton test="hero" variant={variant}>
        Sign Up
      </ConvertButton>
    </>
  );
}

// components/TrackImpression.tsx - Fire view beacon
'use client';
import { useEffect } from 'react';
import { useVisitorId } from './useVisitorId';

export function TrackImpression({ test, variant }: { test: string; variant: number }) {
  const vid = useVisitorId();

  useEffect(() => {
    if (vid) {
      navigator.sendBeacon(
        process.env.NEXT_PUBLIC_HT_URL + '/b',
        JSON.stringify({ t: test, v: variant, e: 'view', vid })
      );
    }
  }, [test, variant, vid]);

  return null;
}

// components/ConvertButton.tsx - Fire convert beacon on click
'use client';
import { useVisitorId } from './useVisitorId';

export function ConvertButton({
  test,
  variant,
  children,
  ...props
}: { test: string; variant: number; children: React.ReactNode } & React.ButtonHTMLAttributes<HTMLButtonElement>) {
  const vid = useVisitorId();

  const handleClick = (e: React.MouseEvent<HTMLButtonElement>) => {
    if (vid) {
      navigator.sendBeacon(
        process.env.NEXT_PUBLIC_HT_URL + '/b',
        JSON.stringify({ t: test, v: variant, e: 'convert', vid })
      );
    }
    props.onClick?.(e);
  };

  return <button {...props} onClick={handleClick}>{children}</button>;
}

// hooks/useConvert.ts - Programmatic conversion
'use client';
import { useCallback } from 'react';
import { useVisitorId } from './useVisitorId';

export function useConvert(test: string, variant: number) {
  const vid = useVisitorId();

  return useCallback(() => {
    if (vid) {
      navigator.sendBeacon(
        process.env.NEXT_PUBLIC_HT_URL + '/b',
        JSON.stringify({ t: test, v: variant, e: 'convert', vid })
      );
    }
  }, [test, variant, vid]);
}
```

#### Pattern 2: Component-Based (Laravel Example)

```php
// app/Http/Middleware/HeadlineTest.php
public function handle(Request $request, Closure $next) {
    if (!$request->cookie('ht_hero')) {
        $variant = random_int(0, 2);
        Cookie::queue('ht_hero', $variant, 60 * 24 * 365);
        $request->attributes->set('ht_hero', $variant);
    } else {
        $request->attributes->set('ht_hero', (int) $request->cookie('ht_hero'));
    }
    return $next($request);
}
```

```blade
{{-- resources/views/home.blade.php --}}
@php
  $variants = ["Ship Faster", "Build Better", "Scale Smart"];
  $variant = request()->attributes->get('ht_hero') ?? 0;
@endphp

<h1>{{ $variants[$variant] }}</h1>

<x-ht-track-impression test="hero" :variant="$variant" />
<x-ht-convert-button test="hero" :variant="$variant">
  Sign Up
</x-ht-convert-button>
```

### Beacon Protocol

All integrations send the same payload to `POST /b`:

```json
{
  "t": "hero",        // test name
  "v": 1,             // variant index
  "e": "view",        // event: "view" or "convert"
  "vid": "abc123xyz"  // visitor ID (for deduplication)
}
```

Response: `204 No Content`

## What We're NOT Doing

- No containerization/Docker (single binary philosophy)
- No external database support (SQLite only)
- No user accounts/multi-tenancy (single-user tool)
- No real-time updates/WebSockets (simple HTTP polling)
- No CI/CD pipeline configuration (out of scope)

## Technology Decisions

| Component | Choice | Rationale |
|-----------|--------|-----------|
| SQLite Driver | `modernc.org/sqlite` | Pure Go, easy cross-compilation |
| CLI Framework | `github.com/spf13/cobra` | Industry standard, good UX |
| Interactive Prompts | `github.com/manifoldco/promptui` | Clean selection UI for snippets |
| HTTP Router | `net/http` (stdlib) | No external dependency needed |
| Templates | `html/template` (stdlib) | Embedded, secure |

## Phase Overview

```
Phase 1: Foundation & Core Data Flow     [CLI skeleton, DB, init, list, serve --health]
    |
    v
Phase 2: Beacon & Event Tracking         [/b endpoint, /t/<test>.js, event storage]
    |
    v
Phase 3: Results & Statistics            [Wilson intervals, z-test, results, export]
    |
    v
Phase 4: Dashboard UI                    [Auth, templates, /dashboard endpoints, otp]
    |
    v
Phase 5: Snippets & Winner               [7 framework templates, snippet cmd, winner cmd]
```

Each phase produces a working, testable increment.

---

# Phase 1: Foundation & Core Data Flow

## Overview

Set up the Go module, database layer, CLI skeleton, and implement the `init`, `list`, and basic `serve` commands. By the end, users can create tests and list them.

## Changes Required

### 1. Project Initialization

**File**: `go.mod`

```go
module github.com/headline-goat/headline-goat

go 1.21

require (
    github.com/spf13/cobra v1.8.0
    modernc.org/sqlite v1.28.0
)
```

**File**: `.gitignore`

```
# Binaries
headline-goat
headline-goat-*
*.exe

# Database
*.db
*.db-wal
*.db-shm

# IDE
.idea/
.vscode/
*.swp

# OS
.DS_Store
Thumbs.db

# Test coverage
coverage.out
```

### 2. Database Layer

**File**: `internal/store/models.go`

```go
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
```

**File**: `internal/store/store.go`

```go
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
```

**File**: `internal/store/sqlite.go`

Implements the Store interface with:
- `Open(dbPath string) (*SQLiteStore, error)` - Opens/creates DB with WAL mode
- Schema migration on first open
- All CRUD operations with proper error handling

**Schema** (embedded in sqlite.go):
```sql
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
```

### 3. CLI Skeleton

**File**: `cmd/headline-goat/main.go`

```go
package main

import (
    "os"
    "github.com/headline-goat/headline-goat/internal/cli"
)

func main() {
    if err := cli.Execute(); err != nil {
        os.Exit(1)
    }
}
```

**File**: `internal/cli/root.go`

Root command with:
- `--db` flag (default: `./headline-goat.db` or `HG_DB_PATH` env)
- Subcommand registration

**File**: `internal/cli/init.go`

`headline-goat init <name> --variants "A" "B" [--weights 0.5 0.5] [--goal "Signup button click"]`
- Validates name (alphanumeric + hyphens)
- Validates variants (at least 2)
- Validates weights sum to 1.0 if provided
- Stores optional conversion goal description
- Creates test in database
- Prints success message

**File**: `internal/cli/list.go`

`headline-goat list`
- Lists all tests in table format
- Shows: NAME, STATE, VARIANTS, VIEWS, CONVERSIONS, CREATED

**File**: `internal/cli/serve.go` (minimal)

`headline-goat serve --port 8080`
- Starts HTTP server
- Only `/health` endpoint for now
- Prints startup message with port

### 4. Server Foundation

**File**: `internal/server/server.go`

```go
package server

type Server struct {
    store  store.Store
    port   int
    token  string // Generated on startup
    router *http.ServeMux
}

func New(store store.Store, port int) *Server
func (s *Server) Start() error
func (s *Server) Token() string
```

**File**: `internal/server/handlers.go`

```go
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request)
```

Returns JSON:
```json
{"status": "ok", "tests_count": 5, "db_size_bytes": 102400, "uptime_seconds": 3600}
```

## Test Structure

```
tests/
├── unit/
│   └── store/
│       └── models_test.go      # Test JSON encoding/decoding
├── integration/
│   └── store/
│       └── sqlite_test.go      # Test all store operations
└── e2e/
    └── cli_test.go             # Test init, list commands
```

## Success Criteria

### Automated Verification:
- [x] `go mod tidy` succeeds
- [x] `go build ./cmd/headline-goat` produces binary
- [x] `go test ./... -v -race` passes all tests
- [x] `go vet ./...` reports no issues
- [x] `./headline-goat init test1 --variants "A" "B"` creates test
- [x] `./headline-goat init test2 --variants "A" "B" --goal "Button click"` stores goal
- [x] `./headline-goat list` shows the tests
- [x] `./headline-goat serve` starts and `/health` returns 200

### Manual Verification:
- [x] Binary runs on target platform without errors
- [x] Database file created in expected location
- [x] Health endpoint returns accurate test count

---

# Phase 2: Beacon & Event Tracking

## Overview

Implement the core tracking infrastructure that serves BOTH integration patterns:
1. **Beacon endpoint** (`POST /b`) - receives events from any client
2. **Client script** (`GET /t/<test>.js`) - for script-based integration (vanilla HTML)

The beacon endpoint is the foundation - it works the same whether events come from our generated script, React components, Laravel Blade, or any other source.

## Changes Required

### 1. Beacon Endpoint (Framework-Agnostic)

**File**: `internal/server/handlers.go` (add)

```go
func (s *Server) handleBeacon(w http.ResponseWriter, r *http.Request)
```

- Accepts POST to `/b`
- Parses JSON body: `{t, v, e, vid}`
- Validates test exists
- Validates variant in range
- Records event (with deduplication)
- Returns 204 No Content
- CORS headers for all origins

### 2. Client JavaScript Endpoint

**File**: `internal/server/handlers.go` (add)

```go
func (s *Server) handleClientJS(w http.ResponseWriter, r *http.Request)
```

- Serves GET `/t/<test>.js`
- Returns JavaScript from SPEC.md
- Sets proper Content-Type and Cache-Control headers
- Returns 404 if test doesn't exist

**File**: `internal/server/templates/client.js.tmpl`

Embedded template for client JavaScript:

```javascript
(function(){
  var T='{{.TestName}}',V={{.VariantsJSON}},K='ht_'+T;
  var d=localStorage,i=d[K],vid=d['ht_vid'];
  if(!vid){vid=Math.random().toString(36).slice(2);d['ht_vid']=vid}
  if(i==null){i=Math.random()*V.length|0;d[K]=i}else{i=+i}

  // Inject variant text into [data-ht="testname"] elements
  var el=document.querySelector('[data-ht="'+T+'"]');
  if(el)el.textContent=V[i];

  // Declarative convert bindings for [data-ht-convert="testname"]
  document.querySelectorAll('[data-ht-convert="'+T+'"]').forEach(function(e){
    e.addEventListener('click',function(){C()});
  });

  // Beacon helpers
  var S='{{.ServerURL}}';
  function B(e){navigator.sendBeacon(S+'/b',JSON.stringify({t:T,v:i,e:e,vid:vid}))}
  function C(){B('convert')}

  // Fire impression on load
  B('view');

  // Expose convert API:
  // - HT.hero() - shorthand
  // - HT.convert.hero() - explicit
  window.HT=window.HT||{};
  window.HT.convert=window.HT.convert||{};
  window.HT[T]=C;
  window.HT.convert[T]=C;
})();
```

~500 bytes minified. User gets:
- `HT.hero()` or `HT.convert.hero()` to fire conversion programmatically
- `data-ht-convert="hero"` for declarative click tracking
- Automatic impression beacon on script load

### 3. Event Deduplication

Update `RecordEvent` in sqlite.go:
- Check if (visitor_id, test_name, event_type) already exists
- Only insert if not duplicate
- Use unique index for efficiency

### 4. Server Routing

Update `internal/server/server.go`:
- Add routes for `/b` and `/t/`
- Add CORS middleware for beacon endpoint

**File**: `internal/server/middleware.go`

```go
func corsMiddleware(next http.Handler) http.Handler
func loggingMiddleware(next http.Handler) http.Handler
```

## Test Structure

```
tests/
├── integration/
│   └── server/
│       ├── beacon_test.go      # Test beacon endpoint
│       └── clientjs_test.go    # Test JS generation
```

## Success Criteria

### Automated Verification:
- [x] `go test ./... -v -race` passes
- [x] POST to `/b` with valid payload returns 204
- [x] POST to `/b` with invalid test returns 400
- [x] GET `/t/testname.js` returns valid JavaScript
- [x] GET `/t/nonexistent.js` returns 404
- [x] Duplicate events are not double-counted

### Manual Verification:
- [x] JavaScript loads in browser without errors
- [x] Beacon sends successfully from browser
- [x] Events appear in database after page view

### Integration Pattern Support

Phase 2 establishes the beacon endpoint that supports ALL frameworks. The key insight:

| Framework | Variant Selection | Impression | Conversion |
|-----------|-------------------|------------|------------|
| Vanilla HTML | `/t/hero.js` (localStorage) | Auto on script load | `data-ht-convert` or `HT.hero()` |
| React/Next.js | Middleware (cookie) | `<TrackImpression>` | `<ConvertButton>` or `useConvert()` |
| Laravel | Middleware (cookie) | `<x-ht-track-impression>` | `<x-ht-convert-button>` |
| Vue/Svelte | Component (localStorage or cookie) | Component onMount | Component onClick or function |

**All paths lead to the same beacon:**
```
POST /b
{"t": "hero", "v": 1, "e": "view", "vid": "abc123"}
```

The `/t/<test>.js` endpoint is a convenience for vanilla HTML users - component-based frameworks don't use it at all.

---

# Phase 3: Results & Statistics

## Overview

Implement statistical calculations (Wilson score intervals, significance testing) and the `results` and `export` CLI commands.

## Changes Required

### 1. Statistics Package

**File**: `internal/stats/wilson.go`

```go
package stats

// WilsonInterval calculates the Wilson score confidence interval
func WilsonInterval(successes, trials int, confidence float64) (lower, upper float64)

// ZScore returns the z-score for a given confidence level
func ZScore(confidence float64) float64
```

**File**: `internal/stats/significance.go`

```go
// SignificanceTest performs a two-proportion z-test
// Returns confidence level (0-1) that variant A beats variant B
func SignificanceTest(aConv, aViews, bConv, bViews int) float64

// Result represents statistical analysis of a test
type Result struct {
    Variants      []VariantResult
    Confident     bool    // >= 95% confidence
    ConfidenceLevel float64
    LeadingVariant  int
}

type VariantResult struct {
    Index       int
    Name        string
    Views       int
    Conversions int
    Rate        float64
    CILower     float64
    CIUpper     float64
}

// Analyze calculates full statistics for a test
func Analyze(test *store.Test, stats []store.VariantStats) *Result
```

### 2. Results Command

**File**: `internal/cli/results.go`

`headline-goat results <name>`
- Fetches test and stats from store
- Calls stats.Analyze
- Formats output as shown in SPEC.md:
  ```
  TEST: hero
  STATE: running
  GOAL: Signup button click    ← Shows conversion goal if set
  CREATED: 2024-01-15

  VARIANT           VIEWS    CONVERSIONS  RATE     95% CI
  ─────────────────────────────────────────────────────────
  Ship Faster       412      32           7.77%    [5.2%, 10.3%]
  Build Better      398      41           10.30%   [7.4%, 13.2%]  ← LEADING
  ...
  ```
- Shows leading variant indicator
- Shows statistical significance message

### 3. Export Command

**File**: `internal/cli/export.go`

`headline-goat export <name> --format csv|json`
- Fetches all events for test
- Outputs in requested format
- Writes to stdout (pipeable)

CSV format:
```csv
timestamp,variant,event_type,visitor_id
1705312800,0,view,abc123
```

JSON format:
```json
{"events": [{"timestamp": 1705312800, "variant": 0, "event_type": "view", "visitor_id": "abc123"}]}
```

## Test Structure

```
tests/
├── unit/
│   └── stats/
│       ├── wilson_test.go        # Test interval calculations
│       └── significance_test.go  # Test z-test
├── integration/
│   └── cli/
│       ├── results_test.go
│       └── export_test.go
```

## Success Criteria

### Automated Verification:
- [x] `go test ./... -v -race` passes
- [x] Wilson interval matches expected values for known inputs
- [x] Significance test correctly identifies winner at 95%
- [x] `results` command outputs formatted table
- [x] `export --format csv` produces valid CSV
- [x] `export --format json` produces valid JSON

### Manual Verification:
- [ ] Results display is readable and correctly formatted
- [ ] Confidence intervals look reasonable for sample data
- [ ] Export files open correctly in spreadsheet/JSON tools

---

# Phase 4: Dashboard UI

## Overview

Implement the web dashboard with authentication, HTML templates, and API endpoints.

## Changes Required

### 1. Authentication

**File**: `internal/server/auth.go`

```go
// GenerateToken creates a random OTP token
func GenerateToken() string

// AuthMiddleware checks for valid token in query param or cookie
func (s *Server) authMiddleware(next http.Handler) http.Handler
```

Token validation:
- Check `?token=` query param first
- If valid, set `ht_token` cookie and redirect without param
- Check `ht_token` cookie
- Return 401 if neither valid

### 2. OTP Command

**File**: `internal/cli/otp.go`

`headline-goat otp`
- Reads token from a shared location (file or prints instruction)
- Note: Token is generated per-server-instance, stored in memory
- For CLI access, we need a token file approach

Alternative: Store token in DB or file when server starts.

### 3. Dashboard Templates

**File**: `internal/dashboard/assets/style.css`

Minimal CSS (~5KB):
- System font stack
- Dark mode via `prefers-color-scheme`
- Progress bar styles
- Mobile-friendly

**File**: `internal/dashboard/templates/layout.html`
**File**: `internal/dashboard/templates/list.html`
**File**: `internal/dashboard/templates/detail.html`

Using Go's `html/template` with embedded files.

### 4. Dashboard Endpoints

**File**: `internal/server/handlers.go` (add)

```go
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request)
func (s *Server) handleDashboardTest(w http.ResponseWriter, r *http.Request)
func (s *Server) handleDashboardAPI(w http.ResponseWriter, r *http.Request)
```

- `/dashboard` - Test list page (shows test name, state, goal, variant count)
- `/dashboard/test/<name>` - Test detail page (shows conversion goal, full stats)
- `/dashboard/api/tests` - JSON API (includes `conversion_goal` field)

### 5. Embed Assets

**File**: `internal/dashboard/embed.go`

```go
package dashboard

import "embed"

//go:embed assets/*
var Assets embed.FS

//go:embed templates/*
var Templates embed.FS
```

## Test Structure

```
tests/
├── integration/
│   └── server/
│       ├── auth_test.go         # Test token validation
│       └── dashboard_test.go    # Test dashboard endpoints
```

## Success Criteria

### Automated Verification:
- [ ] `go test ./... -v -race` passes
- [ ] `/dashboard` without token returns 401
- [ ] `/dashboard?token=valid` returns 200 and sets cookie
- [ ] `/dashboard` with valid cookie returns 200
- [ ] `/dashboard/api/tests` returns valid JSON
- [ ] Assets are properly embedded in binary

### Manual Verification:
- [ ] Dashboard displays correctly in browser
- [ ] Dark mode works with system preference
- [ ] Mobile layout is usable
- [ ] Test detail page shows stats correctly
- [ ] Progress bars render properly

---

# Phase 5: Snippets & Winner

## Overview

Implement the interactive snippet generator that produces **complete, copy-paste-ready integration code** for each framework, plus the winner command to finalize tests.

## Changes Required

### 1. Snippet Output Per Framework

Each framework gets complete integration code, not just a template fragment:

| Framework | What the Snippet Includes |
|-----------|--------------------------|
| **HTML** | Script tag + data attributes |
| **Next.js** | middleware.ts + TrackImpression.tsx + ConvertButton.tsx + useConvert.ts + useVisitorId.ts + page.tsx example |
| **React** | TrackImpression.tsx + ConvertButton.tsx + useConvert.ts + useVisitorId.ts + usage example |
| **Vue** | HeadlineTest.vue component with tracking |
| **Svelte** | HeadlineTest.svelte component with tracking |
| **Laravel** | Middleware + Blade components (track-impression, convert-button) |
| **Django** | Middleware + template tags |

### 2. Snippet Templates

**File**: `internal/snippets/templates/` (directory)

```
templates/
├── html.tmpl           # Simple script + attributes
├── nextjs/             # Multi-file output
│   ├── middleware.tmpl
│   ├── TrackImpression.tmpl
│   ├── ConvertButton.tmpl
│   ├── useConvert.tmpl
│   ├── useVisitorId.tmpl
│   └── page.tmpl
├── react/              # Client-only components
│   ├── TrackImpression.tmpl
│   ├── ConvertButton.tmpl
│   ├── useConvert.tmpl
│   ├── useVisitorId.tmpl
│   └── usage.tmpl
├── vue.tmpl            # Single component
├── svelte.tmpl         # Single component
├── laravel/            # Multi-file output
│   ├── middleware.tmpl
│   ├── track-impression.tmpl
│   └── convert-button.tmpl
└── django/             # Multi-file output
    ├── middleware.tmpl
    └── templatetags.tmpl
```

Template variables:
- `{{.TestName}}` - e.g., "hero"
- `{{.TestNamePascal}}` - e.g., "Hero"
- `{{.VariantsJSON}}` - e.g., `["Ship Faster", "Build Better"]`
- `{{.VariantCount}}` - e.g., 2
- `{{.ServerURL}}` - e.g., "https://ht.example.com"
- `{{.Animation}}` - e.g., "scramble" (React/Vue/Svelte only)

### 3. Snippet Generator

**File**: `internal/snippets/generator.go`

```go
package snippets

type Framework string

const (
    FrameworkHTML    Framework = "html"
    FrameworkNextJS  Framework = "nextjs"
    FrameworkReact   Framework = "react"
    FrameworkVue     Framework = "vue"
    FrameworkSvelte  Framework = "svelte"
    FrameworkLaravel Framework = "laravel"
    FrameworkDjango  Framework = "django"
)

type Animation string

const (
    AnimationScramble   Animation = "scramble"
    AnimationPixel      Animation = "pixel"
    AnimationTypewriter Animation = "typewriter"
    AnimationNone       Animation = "none"
)

type Config struct {
    TestName  string
    Variants  []string
    ServerURL string
    Animation Animation
}

// SnippetFile represents one file in the output
type SnippetFile struct {
    Filename string // e.g., "middleware.ts" or "components/TrackImpression.tsx"
    Content  string
}

// Generate returns one or more files depending on framework
func Generate(framework Framework, config Config) ([]SnippetFile, error)
```

### 4. Snippet Command

**File**: `internal/cli/snippet.go`

`headline-goat snippet <name>`
- Uses promptui for interactive selection
- Framework selection (7 options)
- Animation selection (for React/Vue/Svelte only, defaults to scramble)
- Server URL prompt (with default from env or localhost:8080)
- Generates and prints snippet code with file headers

Output format for multi-file snippets:
```
══════════════════════════════════════════════════════════════
📄 middleware.ts
══════════════════════════════════════════════════════════════

[middleware code here]

══════════════════════════════════════════════════════════════
📄 components/TrackImpression.tsx
══════════════════════════════════════════════════════════════

[component code here]

...
```

Also supports flags for non-interactive:
- `--framework nextjs`
- `--animation scramble`
- `--server-url https://ht.example.com`

### 5. Animation Variants (React/Vue/Svelte)

All three animations implemented for React, Vue, and Svelte:

| Animation | Effect |
|-----------|--------|
| `scramble` | Letters randomize then resolve left-to-right |
| `pixel` | Characters fade in with pixelated effect |
| `typewriter` | Characters appear one at a time |
| `none` | Instant display, no animation |

### 6. Winner Command

**File**: `internal/cli/winner.go`

`headline-goat winner <name> --variant <index>`
- Validates test exists and is running
- Validates variant index
- Updates test state to "completed"
- Sets winner_variant
- Prints success message
- Notes that snippet will now return static version

### 5. Static Winner Snippet

Update snippet generator:
- If test.State == "completed" and test.WinnerVariant != nil
- Generate simplified static snippet (no A/B logic, just winner text)

## Test Structure

```
tests/
├── unit/
│   └── snippets/
│       └── generator_test.go    # Test template rendering
├── integration/
│   └── cli/
│       ├── snippet_test.go
│       └── winner_test.go
```

## Success Criteria

### Automated Verification:
- [ ] `go test ./... -v -race` passes
- [ ] All 7 framework templates render without errors
- [ ] Multi-file snippets (Next.js, React, Laravel, Django) output all required files
- [ ] Single-file snippets (HTML, Vue, Svelte) output correctly
- [ ] All 3 animation variants render for React/Vue/Svelte
- [ ] Snippet includes correct test name, variants, and server URL
- [ ] Winner command updates test state to "completed"
- [ ] Snippet for completed test shows static winner content (no A/B logic)

### Manual Verification:
- [ ] Interactive prompts work correctly in terminal
- [ ] Generated Next.js snippet works in a real Next.js app
- [ ] Generated React snippet works in a Create React App
- [ ] Generated Laravel snippet works in a Laravel app
- [ ] HTML snippet shows variant in browser
- [ ] Conversion tracking fires correctly and appears in dashboard
- [ ] All 3 animations display correctly

---

# Testing Strategy

## Test Categories

### Unit Tests (`tests/unit/`)
- Pure function testing
- No database or network
- Fast execution
- Focus on: stats calculations, JSON encoding, template rendering

### Integration Tests (`tests/integration/`)
- Database operations
- HTTP handlers
- Uses real SQLite (in-memory or temp file)
- Focus on: store operations, endpoint behavior, auth flow

### E2E Tests (`tests/e2e/`)
- Full CLI command execution
- Spawns actual binary
- Tests user-visible behavior
- Focus on: complete workflows, error messages

## Test Commands

```bash
# Run all tests
go test ./... -v -race

# Run specific category
go test ./tests/unit/... -v
go test ./tests/integration/... -v
go test ./tests/e2e/... -v

# Run with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## TDD Workflow Reminder

For each feature:
1. **RED**: Write failing test first
2. **GREEN**: Minimal code to pass
3. **REFACTOR**: Clean up, tests stay green
4. **COMMIT**: Only after `go test ./... -v -race` passes

---

# Implementation Order

Within each phase, implement in this order:

1. **Models/Types** - Define data structures
2. **Store/Database** - Implement persistence
3. **Business Logic** - Core functionality (stats, generators)
4. **HTTP Handlers** - API endpoints
5. **CLI Commands** - User interface
6. **Templates/Assets** - UI components

Always TDD: test file first, then implementation.

---

# Dependencies

```go
require (
    github.com/spf13/cobra v1.8.0
    github.com/manifoldco/promptui v0.9.0
    modernc.org/sqlite v1.28.0
)
```

No other external dependencies. Use standard library for:
- HTTP server (`net/http`)
- JSON (`encoding/json`)
- Templates (`html/template`, `text/template`)
- Embedding (`embed`)
- Testing (`testing`)

---

# References

- Specification: `SPEC.md`
- Development guidelines: `CLAUDE.md`
- Wilson score interval: https://en.wikipedia.org/wiki/Binomial_proportion_confidence_interval#Wilson_score_interval
- Two-proportion z-test: https://en.wikipedia.org/wiki/Statistical_hypothesis_testing
