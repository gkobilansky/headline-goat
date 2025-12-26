# headline-goat 🐐

## Overview

A minimal, self-hosted A/B testing tool for headlines. Single Go binary, embedded SQLite, no external dependencies.

---

## Development Process: Strict TDD

**This project follows Test-Driven Development. No exceptions.**

### The TDD Cycle

```
┌─────────────────────────────────────────────────────────────┐
│  1. RED    → Write a failing test that defines behavior     │
│  2. GREEN  → Write minimal code to make the test pass       │
│  3. REFACTOR → Improve code quality, keep tests green       │
│  4. REPEAT                                                  │
└─────────────────────────────────────────────────────────────┘
```

### Mandatory Rules

1. **Write tests FIRST** - Define expected behavior before implementation
2. **Minimal code only** - Write just enough code to pass the failing test
3. **Run full test suite before EVERY commit** - No exceptions
4. **Tests must verify actual functionality** - Avoid over-mocking
5. **No production code without a failing test first**

### Test Categories

```
tests/
├── unit/           # Pure functions, isolated logic
├── integration/    # Database operations, HTTP handlers
└── e2e/            # Full user flows, CLI commands
```

### Before Each Commit

```bash
go test ./... -v -race
```

If any test fails, **do not commit**.

---

## Architecture

### Components

```
headline-goat/
├── cmd/
│   └── headline-goat/
│       └── main.go           # CLI entry point
├── internal/
│   ├── cli/                  # CLI commands (init, list, serve, etc.)
│   ├── server/               # HTTP server
│   │   ├── handlers.go       # Route handlers
│   │   ├── middleware.go     # Auth, CORS, logging
│   │   └── server.go         # Server setup
│   ├── store/                # Database layer
│   │   ├── store.go          # Interface definitions
│   │   ├── sqlite.go         # SQLite implementation
│   │   └── models.go         # Data structures
│   ├── stats/                # Statistical calculations
│   │   └── significance.go   # Confidence intervals, winner detection
│   ├── snippets/             # Framework snippet generators
│   │   ├── generator.go      # Template engine
│   │   └── templates/        # Per-framework templates
│   └── dashboard/            # Dashboard UI
│       ├── assets/           # Static files (compiled into binary)
│       └── templates/        # HTML templates
├── tests/
│   ├── unit/
│   ├── integration/
│   └── e2e/
└── go.mod
```

### Data Flow

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  Client JS   │────▶│  /b beacon   │────▶│   SQLite     │
│  (browser)   │     │  endpoint    │     │   storage    │
└──────────────┘     └──────────────┘     └──────────────┘
                                                 │
                                                 ▼
                     ┌──────────────┐     ┌──────────────┐
                     │  Dashboard   │◀────│  Stats       │
                     │  UI          │     │  Calculator  │
                     └──────────────┘     └──────────────┘
```

---

## CLI Commands

### `headline-goat init <name>`

Create a new test.

```bash
headline-goat init hero --variants "Ship Faster" "Build Better" "Scale Smart"

# Optional: weighted variants (must sum to 1.0)
headline-goat init hero --variants "A" "B" --weights 0.9 0.1
```

**Behavior:**
- Creates test record in database
- State: `running`
- Returns success message with test ID

### `headline-goat list`

List all tests.

```bash
headline-goat list

# Output:
# NAME        STATE     VARIANTS  VIEWS    CONVERSIONS  CREATED
# hero        running   3         1,234    89           2024-01-15
# pricing     paused    2         567      23           2024-01-10
# cta-button  completed 2         10,000   892          2024-01-01
```

### `headline-goat results <name>`

Show detailed results for a test.

```bash
headline-goat results hero

# Output:
# TEST: hero
# STATE: running
# CREATED: 2024-01-15
#
# VARIANT           VIEWS    CONVERSIONS  RATE     95% CI         
# ─────────────────────────────────────────────────────────────────
# Ship Faster       412      32           7.77%    [5.2%, 10.3%]  
# Build Better      398      41           10.30%   [7.4%, 13.2%]  ← LEADING
# Scale Smart       424      16           3.77%    [2.0%, 5.6%]   
#
# Statistical significance: 94.2% confident "Build Better" beats control
```

### `headline-goat export <name>`

Export raw event data.

```bash
headline-goat export hero --format csv > hero-data.csv
headline-goat export hero --format json > hero-data.json
```

### `headline-goat serve`

Start the server.

```bash
headline-goat serve --port 8080 --db ./data/tests.db

# Output:
# 🐐 headline-goat running on :8080
# Dashboard: http://localhost:8080/dashboard
# Dashboard token: a1b2c3d4
# 
# Press Ctrl+C to stop
```

**Behavior:**
- Prints OTP/session token on startup (regenerates each restart)
- Serves beacon endpoint, dashboard, health check

### `headline-goat otp`

Show current dashboard token (for when you've scrolled past it).

```bash
headline-goat otp

# Output:
# Current dashboard token: a1b2c3d4
```

### `headline-goat snippet <name>`

Interactive snippet generator.

```bash
headline-goat snippet hero

# ? Select framework:
#   ❯ HTML (static)
#     Next.js (App Router)
#     React (non-SSR)
#     Vue
#     Svelte
#     Laravel (Blade)
#     Django
#
# ? Loading animation: (React/Vue/Svelte only)
#   ❯ scramble (default)
#     pixel
#     typewriter
#     none
#
# [outputs snippet code]
```

### `headline-goat winner <name> --variant <index>`

Lock in a winner (stops test, regenerates snippet as static).

```bash
headline-goat winner hero --variant 1

# Output:
# ✓ Test "hero" completed
# ✓ Winner locked: "Build Better" (variant 1)
# 
# Run `headline-goat snippet hero` for updated snippet
```

---

## HTTP Endpoints

### `GET /t/<test>.js`

Returns the client-side JavaScript for a test.

**Response:** JavaScript file

```javascript
(function(){
  var T='hero',V=["Ship Faster","Build Better","Scale Smart"],K='ht_'+T;
  var d=localStorage,i=d[K],vid=d['ht_vid'];
  if(!vid){vid=Math.random().toString(36).slice(2);d['ht_vid']=vid}
  if(i==null){i=Math.random()*V.length|0;d[K]=i}else{i=+i}
  
  var el=document.querySelector('[data-ht="'+T+'"]');
  if(el)el.textContent=V[i];
  
  document.querySelectorAll('[data-ht-convert="'+T+'"]').forEach(function(e){
    e.addEventListener('click',function(){C()});
  });
  
  var S='{{.ServerURL}}';
  function B(e){navigator.sendBeacon(S+'/b',JSON.stringify({t:T,v:i,e:e,vid:vid}))}
  function C(){B('convert')}
  
  B('view');
  
  window.HT=window.HT||{};
  window.HT[T]=C;
})();
```

**Headers:**
- `Content-Type: application/javascript`
- `Cache-Control: public, max-age=60`

### `POST /b`

Receives beacon events.

**Request body:**
```json
{
  "t": "hero",
  "v": 1,
  "e": "view",
  "vid": "abc123xyz"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `t` | string | Test name |
| `v` | int | Variant index |
| `e` | string | Event type: `view` or `convert` |
| `vid` | string | Visitor ID (from localStorage) |

**Response:** `204 No Content`

**Behavior:**
- Validate test exists
- Validate variant index in range
- Store event with timestamp
- Deduplicate by visitor_id + test + event type (count unique visitors)

**CORS:** Allow all origins (anonymous data)

### `GET /dashboard`

Dashboard UI (HTML).

**Auth:** Requires `?token=<otp>` query param or `ht_token` cookie.

**Response:** HTML page showing:
- List of all tests
- Click into test for detailed results
- Stats with confidence intervals
- Winner indicator when significant

### `GET /dashboard/api/tests`

JSON API for dashboard.

**Auth:** Same as dashboard.

**Response:**
```json
{
  "tests": [
    {
      "name": "hero",
      "state": "running",
      "variants": ["Ship Faster", "Build Better", "Scale Smart"],
      "created_at": "2024-01-15T10:00:00Z",
      "results": [
        {
          "variant": 0,
          "variant_name": "Ship Faster",
          "views": 412,
          "conversions": 32,
          "rate": 0.0777,
          "ci_lower": 0.052,
          "ci_upper": 0.103
        },
        // ...
      ],
      "significance": {
        "confident": true,
        "confidence_level": 0.942,
        "leading_variant": 1,
        "leading_variant_name": "Build Better"
      }
    }
  ]
}
```

### `GET /health`

Health check endpoint.

**Response:**
```json
{
  "status": "ok",
  "tests_count": 5,
  "db_size_bytes": 102400,
  "uptime_seconds": 3600
}
```

---

## Database Schema

SQLite with WAL mode enabled.

### `tests` table

```sql
CREATE TABLE tests (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT UNIQUE NOT NULL,
  variants TEXT NOT NULL,  -- JSON array: ["A", "B", "C"]
  weights TEXT,            -- JSON array: [0.5, 0.3, 0.2] (optional)
  state TEXT NOT NULL DEFAULT 'running',  -- running, paused, completed
  winner_variant INTEGER,  -- Set when state = completed
  created_at INTEGER NOT NULL DEFAULT (unixepoch()),
  updated_at INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE INDEX idx_tests_name ON tests(name);
CREATE INDEX idx_tests_state ON tests(state);
```

### `events` table

```sql
CREATE TABLE events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  test_name TEXT NOT NULL,
  variant INTEGER NOT NULL,
  event_type TEXT NOT NULL,  -- 'view' or 'convert'
  visitor_id TEXT NOT NULL,
  created_at INTEGER NOT NULL DEFAULT (unixepoch()),
  
  FOREIGN KEY (test_name) REFERENCES tests(name)
);

CREATE INDEX idx_events_test ON events(test_name);
CREATE INDEX idx_events_test_event ON events(test_name, event_type);
CREATE INDEX idx_events_visitor ON events(test_name, visitor_id, event_type);
```

### Queries

**Get unique views/conversions per variant:**
```sql
SELECT 
  variant,
  COUNT(DISTINCT CASE WHEN event_type = 'view' THEN visitor_id END) as views,
  COUNT(DISTINCT CASE WHEN event_type = 'convert' THEN visitor_id END) as conversions
FROM events
WHERE test_name = ?
GROUP BY variant;
```

---

## Statistical Significance

### Confidence Intervals

Use Wilson score interval for conversion rate confidence intervals (better for small samples than normal approximation).

```go
// Wilson score interval
func WilsonInterval(successes, trials int, confidence float64) (lower, upper float64) {
    if trials == 0 {
        return 0, 0
    }
    
    z := ZScore(confidence) // e.g., 1.96 for 95%
    p := float64(successes) / float64(trials)
    n := float64(trials)
    
    denominator := 1 + z*z/n
    center := (p + z*z/(2*n)) / denominator
    spread := (z / denominator) * math.Sqrt(p*(1-p)/n + z*z/(4*n*n))
    
    return center - spread, center + spread
}
```

### Winner Detection

Use a two-proportion z-test to determine if the leading variant significantly beats the control (variant 0).

```go
// Returns confidence level (0-1) that variant A beats variant B
func SignificanceTest(aConv, aViews, bConv, bViews int) float64 {
    // Two-proportion z-test
    // Returns p-value, convert to confidence level
}
```

**Significance threshold:** 95% confidence to declare winner

**Dashboard display:**
- `< 90%`: "Not enough data"
- `90-95%`: "Possibly better" 
- `≥ 95%`: "Winner" with indicator

---

## Dashboard UI

### Design Requirements

- Clean, minimal interface
- No external dependencies (all assets embedded)
- Works without JavaScript (progressive enhancement)
- Mobile-friendly

### Pages

**`/dashboard`** - Test list

```
┌─────────────────────────────────────────────────────────────┐
│  🐐 headline-goat                              [Logout]     │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  YOUR TESTS                                                 │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ hero                                      RUNNING   │   │
│  │ 3 variants · 1,234 views · 7.2% avg conversion      │   │
│  │ Created Jan 15, 2024                                │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ pricing                                   PAUSED    │   │
│  │ 2 variants · 567 views · 4.1% avg conversion        │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**`/dashboard/test/<name>`** - Test detail

```
┌─────────────────────────────────────────────────────────────┐
│  🐐 headline-goat                              [Logout]     │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ← Back to tests                                            │
│                                                             │
│  HERO                                          RUNNING      │
│  Created Jan 15, 2024                                       │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ RESULTS                              94.2% confident│   │
│  ├─────────────────────────────────────────────────────┤   │
│  │                                                     │   │
│  │  "Ship Faster"                                      │   │
│  │  ████████████░░░░░░░░  412 views · 32 conv · 7.8%  │   │
│  │                        95% CI: [5.2%, 10.3%]        │   │
│  │                                                     │   │
│  │  "Build Better"                          ← LEADING  │   │
│  │  ████████████████░░░░  398 views · 41 conv · 10.3% │   │
│  │                        95% CI: [7.4%, 13.2%]        │   │
│  │                                                     │   │
│  │  "Scale Smart"                                      │   │
│  │  ████░░░░░░░░░░░░░░░░  424 views · 16 conv · 3.8%  │   │
│  │                        95% CI: [2.0%, 5.6%]         │   │
│  │                                                     │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  [Get Snippet]  [Pause Test]  [Declare Winner]              │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Assets

All assets compiled into binary using `embed`:

```go
//go:embed assets/*
var assets embed.FS

//go:embed templates/*
var templates embed.FS
```

### Styling

- System font stack
- Minimal CSS (< 5KB)
- Dark mode via `prefers-color-scheme`
- No CSS framework

---

## Snippet Templates

### Framework: HTML (static)

```html
<!-- In <head> -->
<script>
  window.HT_{{.TestNameUpper}}=localStorage.ht_{{.TestName}}||(localStorage.ht_{{.TestName}}=Math.random()*{{.VariantCount}}|0);
  window.HT_{{.TestNameUpper}}_V={{.VariantsJSON}};
</script>

<!-- Where headline goes -->
<h1 data-ht="{{.TestName}}"><script>document.write(HT_{{.TestNameUpper}}_V[+HT_{{.TestNameUpper}}])</script></h1>

<!-- Before </body> -->
<script async src="{{.ServerURL}}/t/{{.TestName}}.js"></script>

<!-- For conversion tracking, add to your CTA: -->
<button data-ht-convert="{{.TestName}}">Sign Up</button>
<!-- Or call manually: HT.{{.TestName}}() -->
```

### Framework: Next.js (App Router)

**`middleware.ts`**
```typescript
import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

export function middleware(request: NextRequest) {
  const response = NextResponse.next();
  
  if (!request.cookies.has('ht_{{.TestName}}')) {
    const variant = Math.floor(Math.random() * {{.VariantCount}});
    response.cookies.set('ht_{{.TestName}}', String(variant), { 
      maxAge: 60 * 60 * 24 * 365,
      path: '/',
    });
  }
  
  return response;
}

export const config = {
  matcher: ['/'], // Adjust to pages where test runs
};
```

**`components/{{.TestNamePascal}}Headline.tsx`**
```tsx
import { cookies } from 'next/headers';
import { TrackImpression } from './TrackImpression';

const variants = {{.VariantsJSON}};

export async function {{.TestNamePascal}}Headline() {
  const cookieStore = await cookies();
  const variant = Number(cookieStore.get('ht_{{.TestName}}')?.value ?? 0);
  
  return (
    <>
      <TrackImpression test="{{.TestName}}" variant={variant} />
      <h1>{variants[variant]}</h1>
    </>
  );
}
```

**`components/TrackImpression.tsx`**
```tsx
'use client';

import { useEffect } from 'react';

export function TrackImpression({ test, variant }: { test: string; variant: number }) {
  useEffect(() => {
    const vid = localStorage.getItem('ht_vid') || (() => {
      const id = Math.random().toString(36).slice(2);
      localStorage.setItem('ht_vid', id);
      return id;
    })();
    
    navigator.sendBeacon(
      '{{.ServerURL}}/b',
      JSON.stringify({ t: test, v: variant, e: 'view', vid })
    );
  }, [test, variant]);
  
  return null;
}
```

**`components/ConvertButton.tsx`**
```tsx
'use client';

export function ConvertButton({ test, children }: { test: string; children: React.ReactNode }) {
  const handleClick = () => {
    const variant = document.cookie
      .split('; ')
      .find(row => row.startsWith(`ht_${test}=`))
      ?.split('=')[1];
    
    const vid = localStorage.getItem('ht_vid');
    
    if (variant !== undefined && vid) {
      navigator.sendBeacon(
        '{{.ServerURL}}/b',
        JSON.stringify({ t: test, v: Number(variant), e: 'convert', vid })
      );
    }
  };
  
  return <button onClick={handleClick}>{children}</button>;
}
```

### Framework: React (non-SSR)

With loading animation support.

**`components/{{.TestNamePascal}}Headline.tsx`** (scramble animation - default)
```tsx
import { useState, useEffect, useRef } from 'react';

const variants = {{.VariantsJSON}};
const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ';

export function {{.TestNamePascal}}Headline() {
  const [display, setDisplay] = useState('');
  const [resolved, setResolved] = useState(false);
  const finalText = useRef('');
  const sentBeacon = useRef(false);

  useEffect(() => {
    // Get or set visitor ID
    let vid = localStorage.getItem('ht_vid');
    if (!vid) {
      vid = Math.random().toString(36).slice(2);
      localStorage.setItem('ht_vid', vid);
    }

    // Get or set variant
    const k = 'ht_{{.TestName}}';
    let i = localStorage.getItem(k);
    if (i === null) {
      i = String(Math.floor(Math.random() * variants.length));
      localStorage.setItem(k, i);
    }
    finalText.current = variants[+i];

    // Scramble animation
    let frame = 0;
    const maxFrames = 20;
    
    const interval = setInterval(() => {
      frame++;
      setDisplay(
        finalText.current
          .split('')
          .map((char, idx) => {
            if (char === ' ') return ' ';
            if (frame > maxFrames - (finalText.current.length - idx) * 1.5) {
              return char;
            }
            return chars[Math.floor(Math.random() * chars.length)];
          })
          .join('')
      );
      
      if (frame >= maxFrames + finalText.current.length) {
        clearInterval(interval);
        setResolved(true);
      }
    }, 50);

    // Send beacon once resolved
    if (!sentBeacon.current) {
      sentBeacon.current = true;
      navigator.sendBeacon(
        '{{.ServerURL}}/b',
        JSON.stringify({ t: '{{.TestName}}', v: +i, e: 'view', vid })
      );
    }

    return () => clearInterval(interval);
  }, []);

  return (
    <h1 style={{ fontFamily: 'inherit' }}>
      {display.split('').map((char, i) => (
        <span 
          key={i} 
          style={{ color: resolved || char === finalText.current[i] ? 'inherit' : '#999' }}
        >
          {char === ' ' ? '\u00A0' : char}
        </span>
      ))}
    </h1>
  );
}

// Conversion helper
export function convert{{.TestNamePascal}}() {
  const vid = localStorage.getItem('ht_vid');
  const variant = localStorage.getItem('ht_{{.TestName}}');
  if (vid && variant !== null) {
    navigator.sendBeacon(
      '{{.ServerURL}}/b',
      JSON.stringify({ t: '{{.TestName}}', v: +variant, e: 'convert', vid })
    );
  }
}
```

### Framework: Vue

```vue
<script setup>
import { ref, onMounted } from 'vue';

const variants = {{.VariantsJSON}};
const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ';
const display = ref('');
const resolved = ref(false);

onMounted(() => {
  let vid = localStorage.getItem('ht_vid');
  if (!vid) {
    vid = Math.random().toString(36).slice(2);
    localStorage.setItem('ht_vid', vid);
  }

  const k = 'ht_{{.TestName}}';
  let i = localStorage.getItem(k);
  if (i === null) {
    i = String(Math.floor(Math.random() * variants.length));
    localStorage.setItem(k, i);
  }
  const finalText = variants[+i];

  let frame = 0;
  const maxFrames = 20;
  
  const interval = setInterval(() => {
    frame++;
    display.value = finalText
      .split('')
      .map((char, idx) => {
        if (char === ' ') return ' ';
        if (frame > maxFrames - (finalText.length - idx) * 1.5) return char;
        return chars[Math.floor(Math.random() * chars.length)];
      })
      .join('');
    
    if (frame >= maxFrames + finalText.length) {
      clearInterval(interval);
      resolved.value = true;
    }
  }, 50);

  navigator.sendBeacon(
    '{{.ServerURL}}/b',
    JSON.stringify({ t: '{{.TestName}}', v: +i, e: 'view', vid })
  );
});

function convert() {
  const vid = localStorage.getItem('ht_vid');
  const variant = localStorage.getItem('ht_{{.TestName}}');
  if (vid && variant !== null) {
    navigator.sendBeacon(
      '{{.ServerURL}}/b',
      JSON.stringify({ t: '{{.TestName}}', v: +variant, e: 'convert', vid })
    );
  }
}

defineExpose({ convert });
</script>

<template>
  <h1>{{ display }}</h1>
</template>
```

### Framework: Svelte

```svelte
<script>
  import { onMount } from 'svelte';
  
  const variants = {{.VariantsJSON}};
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ';
  let display = '';
  
  onMount(() => {
    let vid = localStorage.getItem('ht_vid');
    if (!vid) {
      vid = Math.random().toString(36).slice(2);
      localStorage.setItem('ht_vid', vid);
    }

    const k = 'ht_{{.TestName}}';
    let i = localStorage.getItem(k);
    if (i === null) {
      i = String(Math.floor(Math.random() * variants.length));
      localStorage.setItem(k, i);
    }
    const finalText = variants[+i];

    let frame = 0;
    const maxFrames = 20;
    
    const interval = setInterval(() => {
      frame++;
      display = finalText
        .split('')
        .map((char, idx) => {
          if (char === ' ') return ' ';
          if (frame > maxFrames - (finalText.length - idx) * 1.5) return char;
          return chars[Math.floor(Math.random() * chars.length)];
        })
        .join('');
      
      if (frame >= maxFrames + finalText.length) {
        clearInterval(interval);
      }
    }, 50);

    navigator.sendBeacon(
      '{{.ServerURL}}/b',
      JSON.stringify({ t: '{{.TestName}}', v: +i, e: 'view', vid })
    );
    
    return () => clearInterval(interval);
  });
  
  export function convert() {
    const vid = localStorage.getItem('ht_vid');
    const variant = localStorage.getItem('ht_{{.TestName}}');
    if (vid && variant !== null) {
      navigator.sendBeacon(
        '{{.ServerURL}}/b',
        JSON.stringify({ t: '{{.TestName}}', v: +variant, e: 'convert', vid })
      );
    }
  }
</script>

<h1>{display}</h1>
```

### Framework: Laravel (Blade)

**`app/Http/Middleware/HeadlineTest.php`**
```php
<?php

namespace App\Http\Middleware;

use Closure;
use Illuminate\Http\Request;

class HeadlineTest
{
    public function handle(Request $request, Closure $next)
    {
        $testName = '{{.TestName}}';
        $variantCount = {{.VariantCount}};
        $cookieName = 'ht_' . $testName;
        
        if (!$request->cookie($cookieName)) {
            $variant = random_int(0, $variantCount - 1);
            cookie()->queue($cookieName, $variant, 60 * 24 * 365);
            $request->attributes->set($cookieName, $variant);
        } else {
            $request->attributes->set($cookieName, (int) $request->cookie($cookieName));
        }
        
        return $next($request);
    }
}
```

**`resources/views/components/headline-test.blade.php`**
```blade
@php
  $variants = {!! json_encode({{.VariantsJSON}}) !!};
  $variant = request()->attributes->get('ht_{{.TestName}}') ?? request()->cookie('ht_{{.TestName}}') ?? 0;
@endphp

<h1>{{ $variants[$variant] }}</h1>

<script>
  (function() {
    var vid = localStorage.getItem('ht_vid');
    if (!vid) {
      vid = Math.random().toString(36).slice(2);
      localStorage.setItem('ht_vid', vid);
    }
    navigator.sendBeacon(
      '{{.ServerURL}}/b',
      JSON.stringify({ t: '{{.TestName}}', v: {{ $variant }}, e: 'view', vid: vid })
    );
    
    window.HT = window.HT || {};
    window.HT['{{.TestName}}'] = function() {
      navigator.sendBeacon(
        '{{.ServerURL}}/b',
        JSON.stringify({ t: '{{.TestName}}', v: {{ $variant }}, e: 'convert', vid: vid })
      );
    };
  })();
</script>
```

### Framework: Django

**`middleware.py`**
```python
import random

class HeadlineTestMiddleware:
    def __init__(self, get_response):
        self.get_response = get_response

    def __call__(self, request):
        test_name = '{{.TestName}}'
        variant_count = {{.VariantCount}}
        cookie_name = f'ht_{test_name}'
        
        if cookie_name not in request.COOKIES:
            request.ht_variant = random.randint(0, variant_count - 1)
            request.ht_set_cookie = True
        else:
            request.ht_variant = int(request.COOKIES[cookie_name])
            request.ht_set_cookie = False
        
        response = self.get_response(request)
        
        if getattr(request, 'ht_set_cookie', False):
            response.set_cookie(
                cookie_name,
                str(request.ht_variant),
                max_age=365 * 24 * 60 * 60
            )
        
        return response
```

**`templatetags/headline_test.py`**
```python
from django import template
from django.utils.safestring import mark_safe
import json

register = template.Library()

VARIANTS = {{.VariantsJSON}}

@register.simple_tag(takes_context=True)
def headline_{{.TestName}}(context):
    request = context['request']
    variant = getattr(request, 'ht_variant', 0)
    
    html = f'''
    <h1>{VARIANTS[variant]}</h1>
    <script>
      (function() {{
        var vid = localStorage.getItem('ht_vid');
        if (!vid) {{
          vid = Math.random().toString(36).slice(2);
          localStorage.setItem('ht_vid', vid);
        }}
        navigator.sendBeacon(
          '{{{{.ServerURL}}}}/b',
          JSON.stringify({{ t: '{{.TestName}}', v: {variant}, e: 'view', vid: vid }})
        );
        
        window.HT = window.HT || {{}};
        window.HT['{{.TestName}}'] = function() {{
          navigator.sendBeacon(
            '{{{{.ServerURL}}}}/b',
            JSON.stringify({{ t: '{{.TestName}}', v: {variant}, e: 'convert', vid: vid }})
          );
        }};
      }})();
    </script>
    '''
    return mark_safe(html)
```

---

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `HG_PORT` | `8080` | Server port |
| `HG_DB_PATH` | `./headline-goat.db` | SQLite database path |
| `HG_LOG_LEVEL` | `info` | Log level: debug, info, warn, error |

### Command Flags

All commands support:
- `--db <path>` - Database path (overrides `HG_DB_PATH`)

Server-specific:
- `--port <port>` - Port number (overrides `HG_PORT`)

---

## Error Handling

### CLI Errors

Exit codes:
- `0` - Success
- `1` - General error
- `2` - Invalid arguments
- `3` - Database error
- `4` - Test not found

### HTTP Errors

| Code | When |
|------|------|
| `204` | Beacon received successfully |
| `400` | Invalid beacon payload |
| `401` | Invalid/missing dashboard token |
| `404` | Test not found |
| `500` | Internal server error |

---

## Testing Strategy

### Unit Tests

**`internal/stats/significance_test.go`**
```go
func TestWilsonInterval(t *testing.T) {
    tests := []struct {
        name       string
        successes  int
        trials     int
        confidence float64
        wantLower  float64
        wantUpper  float64
        tolerance  float64
    }{
        {
            name:       "50% conversion rate",
            successes:  50,
            trials:     100,
            confidence: 0.95,
            wantLower:  0.40,
            wantUpper:  0.60,
            tolerance:  0.02,
        },
        // More test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            lower, upper := WilsonInterval(tt.successes, tt.trials, tt.confidence)
            // Assert within tolerance
        })
    }
}
```

### Integration Tests

**`tests/integration/beacon_test.go`**
```go
func TestBeaconEndpoint(t *testing.T) {
    // Setup: create test database
    // Setup: create test record
    // Setup: start server
    
    // Test: send valid beacon
    // Assert: 204 response
    // Assert: event recorded in database
    
    // Test: send beacon for nonexistent test
    // Assert: 400 response
    
    // Cleanup
}
```

### E2E Tests

**`tests/e2e/full_flow_test.go`**
```go
func TestFullFlow(t *testing.T) {
    // Test: init new test via CLI
    // Assert: test created in database
    
    // Test: start server
    // Test: request client JS
    // Assert: valid JavaScript returned
    
    // Test: send view beacons
    // Test: send convert beacons
    
    // Test: check results via CLI
    // Assert: correct counts and statistics
    
    // Test: export data
    // Assert: valid CSV/JSON output
}
```

---

## Dependencies

Minimal external dependencies:

```go
require (
    github.com/mattn/go-sqlite3 v1.14.x  // SQLite driver (CGO)
    // OR
    modernc.org/sqlite v1.x.x             // Pure Go SQLite (no CGO)
)
```

Consider for CLI:
- `github.com/spf13/cobra` - CLI framework
- `github.com/manifoldco/promptui` - Interactive prompts

Or keep it simple with standard library only.

---

## Build & Release

### Build

```bash
# Development
go build -o headline-goat ./cmd/headline-goat

# Production (embedded assets, optimized)
go build -ldflags="-s -w" -o headline-goat ./cmd/headline-goat

# Cross-compile
GOOS=linux GOARCH=amd64 go build -o headline-goat-linux-amd64 ./cmd/headline-goat
GOOS=darwin GOARCH=arm64 go build -o headline-goat-darwin-arm64 ./cmd/headline-goat
GOOS=windows GOARCH=amd64 go build -o headline-goat-windows-amd64.exe ./cmd/headline-goat
```

### Release Artifacts

- `headline-goat-linux-amd64`
- `headline-goat-linux-arm64`
- `headline-goat-darwin-amd64`
- `headline-goat-darwin-arm64`
- `headline-goat-windows-amd64.exe`

---

## License

MIT
