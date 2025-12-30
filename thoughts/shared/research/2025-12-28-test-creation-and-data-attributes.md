---
date: 2025-12-29T03:27:38Z
researcher: Claude
git_commit: 9f651e2d23fd5bc0e12cef5d7f092b869120d9f4
branch: feat/use-data-attributes
repository: headline-goat
topic: "Test Creation Methods: Explicit Init vs Implicit Data Attributes"
tags: [research, codebase, init, data-attributes, snippets, beacon]
status: complete
last_updated: 2025-12-28
last_updated_by: Claude
---

# Research: Test Creation Methods - Explicit Init vs Implicit Data Attributes

**Date**: 2025-12-29T03:27:38Z
**Researcher**: Claude
**Git Commit**: 9f651e2d23fd5bc0e12cef5d7f092b869120d9f4
**Branch**: feat/use-data-attributes
**Repository**: headline-goat

## Research Question

How are headline tests currently created explicitly with `./headline-goat init`, and what exists to support implicit test creation using HTML data attributes?

## Summary

headline-goat currently supports **explicit test creation** via the `init` CLI command, which:
1. Validates test name and variants
2. Persists to SQLite with optional weights and conversion goal
3. Generates framework-specific code snippets

The system also has **partial data-attribute support** in its runtime JavaScript (`/t/<test>.js`):
- `[data-ht="<testname>"]` - Selects headline element for text swap
- `[data-ht-convert="<testname>"]` - Identifies conversion trigger elements

However, **tests must exist in the database before the runtime JS works**. The current flow requires:
1. Run `headline-goat init` to create test in database
2. Include `<script src="/t/testname.js">` in HTML
3. Add data attributes to HTML elements

## Detailed Findings

### 1. Explicit Test Creation (`init` Command)

#### Entry Point
- `internal/cli/init.go:22-35` - Command definition
- `internal/cli/init.go:44-143` - `runInit()` execution

#### Parameters Accepted
| Flag | Type | Description |
|------|------|-------------|
| `[name]` | positional | Test name (alphanumeric + hyphens) |
| `-v, --variants` | []string | Variant text options |
| `-w, --weights` | []float64 | Optional traffic weights (must sum to 1.0) |
| `-g, --goal` | string | Conversion goal description |

#### Validation Rules (`init.go:58-93`)
- Name: regex `^[a-zA-Z0-9-]+$` (line 203)
- Variants: minimum 2 required (line 72)
- Weights: must match variant count, each 0-1, sum to 1.0 within 0.001 tolerance (lines 78-92)

#### Database Persistence (`store/sqlite.go:77-116`)
```sql
INSERT INTO tests (name, variants, weights, conversion_goal, state, created_at, updated_at)
VALUES (?, ?, ?, ?, 'running', ?, ?)
```
- Variants and weights stored as JSON strings
- State defaults to 'running'
- Returns auto-incremented ID

### 2. Test Data Model

#### Schema (`store/sqlite.go:21-31`)
```sql
CREATE TABLE IF NOT EXISTS tests (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    variants TEXT NOT NULL,           -- JSON array
    weights TEXT,                      -- JSON array, nullable
    conversion_goal TEXT,
    state TEXT NOT NULL DEFAULT 'running',
    winner_variant INTEGER,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);
```

#### Go Struct (`store/models.go:13-23`)
```go
type Test struct {
    ID             int64
    Name           string
    Variants       []string
    Weights        []float64
    ConversionGoal string
    State          TestState  // running, paused, completed
    WinnerVariant  *int
    CreatedAt      time.Time
    UpdatedAt      time.Time
}
```

### 3. Current Runtime JavaScript (`/t/<test>.js`)

#### Handler (`server/handlers.go:132-173`)
- Parses test name from URL path
- Retrieves test from database (**test must exist**)
- Generates minified JavaScript with embedded variants

#### Generated Script (`server/handlers.go:175-200`)
```javascript
(function(){
  var T="testname",V=["Variant A","Variant B"],K='ht_'+T,S="http://server";
  var l=localStorage,i=l[K],v=l['ht_vid']||(l['ht_vid']=Math.random().toString(36).slice(2));
  if(i==null){i=Math.floor(Math.random()*V.length);l[K]=i}
  var h=document.querySelector('[data-ht="'+T+'"]');
  if(h)h.textContent=V[i];
  document.querySelectorAll('[data-ht-convert="'+T+'"]').forEach(function(e){
    e.addEventListener('click',C)
  });
  function B(e){navigator.sendBeacon(S+'/b',JSON.stringify({t:T,v:+i,e:e,vid:v}))}
  function C(){B('convert')}
  B('view');
  window.HT=window.HT||{};window.HT[T]={convert:C}
})();
```

#### Current Data Attributes
| Attribute | Purpose | Behavior |
|-----------|---------|----------|
| `data-ht="testname"` | Headline element | Text swapped to selected variant |
| `data-ht-convert="testname"` | Conversion trigger | Click sends conversion beacon |

### 4. Beacon Endpoint (`/b`)

#### Handler (`server/handlers.go:74-130`)
- Accepts POST with JSON body
- Validates test exists in database
- Records event with deduplication

#### Request Format
```json
{
  "t": "testname",    // Test name
  "v": 0,             // Variant index
  "e": "view",        // Event type: "view" or "convert"
  "vid": "abc123"     // Visitor ID
}
```

#### Deduplication (`store/sqlite.go:49`)
```sql
CREATE UNIQUE INDEX idx_events_dedup ON events(test_name, visitor_id, event_type)
```
- Uses `INSERT OR IGNORE` for idempotent event recording

### 5. Gap Analysis: What's Missing for Implicit Creation

The user's proposed syntax:
```html
<h1 data-ht-name="hero" data-ht-variants='["Ship Faster","Build Better"]'>
  Ship Faster
</h1>
<button data-ht-convert="hero">Sign Up</button>
```

#### Current Limitations
1. **Test Must Pre-exist**: The `/t/<test>.js` endpoint returns 404 if test doesn't exist (`handlers.go:153-155`)
2. **Variants Server-Side Only**: Variants are embedded in server-generated JS, not read from HTML
3. **No Auto-Creation Endpoint**: No mechanism to create tests from client-side data

#### Two Architecture Options

**Option A: Smart Global Script**
- Single script: `<script src="/headline-goat.js">`
- Client-side JS discovers all `[data-ht-name]` elements
- Extracts variants from `data-ht-variants` attribute
- Beacons include variant text (not index)
- Server auto-creates tests on first beacon

**Option B: Test Creation API**
- New endpoint: `POST /tests` creates test
- Client script calls API before first view
- Maintains current indexed variant approach
- More network overhead

## Code References

- `internal/cli/init.go:44-143` - Init command implementation
- `internal/store/sqlite.go:77-116` - CreateTest database method
- `internal/store/sqlite.go:21-31` - Tests table schema
- `internal/store/models.go:13-23` - Test struct definition
- `internal/server/handlers.go:132-200` - Client JS generation
- `internal/server/handlers.go:74-130` - Beacon endpoint
- `internal/snippets/generator.go:140-158` - HTML snippet with data attributes

## Architecture Documentation

### Current Data Flow
```
1. CLI: headline-goat init hero -v "A" -v "B"
      ↓
2. SQLite: INSERT INTO tests (name, variants, ...)
      ↓
3. HTML: <script src="/t/hero.js">
      ↓
4. Server: GET /t/hero.js → generates JS with embedded variants
      ↓
5. Browser: JS swaps text, sends beacon
      ↓
6. Server: POST /b → validates test exists → records event
```

### Proposed Data Flow (Implicit)
```
1. HTML: <h1 data-ht-name="hero" data-ht-variants='["A","B"]'>
      ↓
2. Global Script: discovers elements, assigns variants
      ↓
3. Browser: swaps text, sends beacon with variant TEXT
      ↓
4. Server: POST /b → auto-creates test if missing → records event
```

## Open Questions

1. Should implicit tests support weights, or always use equal distribution?
2. How to handle conflicting definitions (same test name, different variants)?
3. Should the global script support animations like framework snippets do?
4. How to prevent test pollution from invalid/malicious data attributes?
