# headline-goat 🐐

## Overview

A minimal, self-hosted A/B testing tool for headlines. Single Go binary, embedded SQLite, no external dependencies.

**Key principles:**
- Configuration lives in HTML via data attributes
- One global script for entire site
- Tests auto-create on first beacon
- Server is just a beacon collector + stats calculator

---

## Quick Start

```bash
# 1. Deploy headline-goat (see Deployment section)
#    You'll get a URL like: https://ht.example.com

# 2. Add to your site
<script src="https://ht.example.com/ht.js" defer></script>

# 3. Create a test with data attributes
<h1 data-ht-name="hero" data-ht-variants='["Ship Faster","Build Better"]'>
  Ship Faster
</h1>
<button data-ht-convert="hero">Sign Up</button>
```

That's it. Tests auto-create when traffic arrives.

> **Note:** headline-goat is designed to be deployed to a VPS, Fly.io, or accessed via Cloudflare Tunnel. See the [Deployment](#deployment) section for setup instructions.

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
│   ├── cli/                  # CLI commands
│   ├── server/               # HTTP server
│   │   ├── handlers.go       # Route handlers
│   │   ├── auth.go           # Dashboard auth
│   │   ├── globaljs.go       # Global script generator
│   │   └── server.go         # Server setup
│   ├── store/                # Database layer
│   │   ├── sqlite.go         # SQLite implementation
│   │   └── models.go         # Data structures
│   ├── stats/                # Statistical calculations
│   │   └── significance.go   # Confidence intervals, winner detection
│   └── dashboard/            # Dashboard UI
│       ├── assets/           # CSS (embedded)
│       └── templates/        # HTML templates (embedded)
├── tests/
│   ├── unit/
│   ├── integration/
│   └── e2e/
└── go.mod
```

### Data Flow

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  Global JS   │────▶│  /b beacon   │────▶│   SQLite     │
│  (ht.js)     │     │  endpoint    │     │   storage    │
└──────────────┘     └──────────────┘     └──────────────┘
                                                 │
       Config via                                ▼
       data-ht-*        ┌──────────────┐   ┌──────────────┐
       attributes       │  Dashboard   │◀──│  Stats       │
                        │  UI          │   │  Calculator  │
                        └──────────────┘   └──────────────┘
```

---

## Data Attributes

All test configuration lives in HTML:

### `data-ht-name` - Define a test

```html
<h1
  data-ht-name="hero"
  data-ht-variants='["Ship Faster","Build Better","Scale Smart"]'
>
  Ship Faster
</h1>
```

| Attribute | Required | Description |
|-----------|----------|-------------|
| `data-ht-name` | Yes | Unique test identifier |
| `data-ht-variants` | Yes | JSON array of text variants |

### `data-ht-convert` - Track conversions

```html
<!-- Button click conversion -->
<button data-ht-convert="hero">Sign Up</button>

<!-- With variant text for the button itself -->
<button
  data-ht-convert="hero"
  data-ht-convert-variants='["Get Started","Sign Up Free"]'
>
  Get Started
</button>

<!-- URL-based conversion (for thank-you pages) -->
<span data-ht-convert="hero" data-ht-convert-type="url" hidden></span>
```

| Attribute | Required | Description |
|-----------|----------|-------------|
| `data-ht-convert` | Yes | Test name to track conversion for |
| `data-ht-convert-variants` | No | JSON array of button text variants |
| `data-ht-convert-type` | No | Set to `"url"` for page-load conversion |

### Variant Index Sync

The same variant index is used for both headline and convert button:
- If visitor gets variant 1 for headline, they get variant 1 for convert button
- Ensures consistent experience

### Complete Example

```html
<!DOCTYPE html>
<html>
<head>
  <!-- Replace with your deployed headline-goat URL -->
  <script src="https://ht.example.com/ht.js" defer></script>
</head>
<body>
  <h1
    data-ht-name="hero"
    data-ht-variants='["Ship Faster","Build Better"]'
  >
    Ship Faster
  </h1>

  <button
    data-ht-convert="hero"
    data-ht-convert-variants='["Start Now","Get Started"]'
  >
    Start Now
  </button>
</body>
</html>
```

---

## CLI Commands

### `headline-goat` (or `headline-goat init`)

Start the server and show integration instructions.

```bash
./headline-goat

# Or with options
./headline-goat --port 8080 --db ./data/tests.db
```

**Output:**
```
? Framework: HTML

headline-goat running on :8080
Dashboard: http://localhost:8080/dashboard?token=a1b2c3d4

=== Deployment ===

Deploy headline-goat to make it accessible from your site.
See: https://github.com/headline-goat/headline-goat#deployment

=== Add to your site ===

<script src="https://YOUR-DEPLOYED-URL/ht.js" defer></script>

=== Create a test ===

<h1 data-ht-name="hero" data-ht-variants='["A","B"]'>A</h1>
<button data-ht-convert="hero">Sign Up</button>

=== Commands ===

  headline-goat results <name>   View test results
  headline-goat winner <name>    Declare a winner
  headline-goat list             List all tests
  headline-goat otp              Open dashboard

Press Ctrl+C to stop
```

> **Note:** The dashboard URL shown uses localhost for local access. Your deployed URL will be different (e.g., `https://ht.example.com/dashboard?token=...`).

### `headline-goat list`

List all tests (auto-created from beacons).

```bash
headline-goat list

# Output:
# NAME        STATE     VIEWS    CONVERSIONS  RATE     CREATED
# hero        running   1,234    89           7.2%     2024-01-15
# pricing     running   567      23           4.1%     2024-01-10
# cta-button  completed 10,000   892          8.9%     2024-01-01
```

### `headline-goat results <name>`

Show detailed results for a test.

```bash
headline-goat results hero

# Output:
# TEST: hero
# STATE: running
#
# VARIANT           VIEWS    CONVERSIONS  RATE     95% CI
# ─────────────────────────────────────────────────────────────────
# Ship Faster       412      32           7.77%    [5.2%, 10.3%]
# Build Better      398      41           10.30%   [7.4%, 13.2%]  ← LEADING
#
# Statistical significance: 94.2% confident "Build Better" beats others
```

### `headline-goat winner <name> --variant <index>`

Declare a winner and complete the test.

```bash
headline-goat winner hero --variant 1

# Output:
# Declared winner for test 'hero': "Build Better" (variant 1)
# Test has been marked as completed.
#
# To finalize, replace your headline-goat code with:
#   <h1>Build Better</h1>
#
# You can now remove the headline-goat script tag.
```

### `headline-goat otp`

Show dashboard URL with current token.

```bash
headline-goat otp

# Output (local):
# Dashboard: http://localhost:8080/dashboard?token=a1b2c3d4

# For deployed instances, access via your deployed URL:
# https://ht.example.com/dashboard?token=a1b2c3d4
```

### `headline-goat export <name>`

Export raw event data.

```bash
headline-goat export hero --format csv > hero-data.csv
headline-goat export hero --format json > hero-data.json
```

---

## HTTP Endpoints

### `GET /ht.js`

Global JavaScript that powers all tests.

**Behavior:**
1. Finds all `[data-ht-name]` elements
2. For each test:
   - Gets or creates visitor ID (localStorage)
   - Gets or assigns variant index (localStorage)
   - Swaps text content from `data-ht-variants`
   - Sends view beacon
3. Finds all `[data-ht-convert]` elements
4. For each convert element:
   - Swaps text if `data-ht-convert-variants` present
   - Adds click handler to send convert beacon
   - If `data-ht-convert-type="url"`, sends beacon on page load

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
| `vid` | string | Visitor ID |

**Response:** `204 No Content`

**Behavior:**
- Auto-create test if it doesn't exist
- Validate variant index >= 0
- Store event with timestamp
- Deduplicate by (visitor_id, test, event_type)

**CORS:** Allow all origins

### `GET /dashboard`

Dashboard UI.

**Auth:** Requires `?token=<otp>` query param or `ht_token` cookie.

### `GET /dashboard/api/tests`

JSON API for dashboard.

**Response:**
```json
{
  "tests": [
    {
      "name": "hero",
      "state": "running",
      "created_at": "2024-01-15T10:00:00Z",
      "results": [
        {
          "variant": 0,
          "views": 412,
          "conversions": 32,
          "rate": 0.0777,
          "ci_lower": 0.052,
          "ci_upper": 0.103
        }
      ],
      "significance": {
        "confident": true,
        "confidence_level": 0.942,
        "leading_variant": 1
      }
    }
  ]
}
```

### `GET /health`

Health check endpoint.

```json
{
  "status": "ok",
  "tests_count": 5,
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
  state TEXT NOT NULL DEFAULT 'running',  -- running, completed
  winner_variant INTEGER,                  -- Set when completed
  created_at INTEGER NOT NULL DEFAULT (unixepoch()),
  updated_at INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE INDEX idx_tests_name ON tests(name);
```

Note: Variant names are NOT stored server-side. They come from HTML data attributes.

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
CREATE UNIQUE INDEX idx_events_dedup ON events(test_name, visitor_id, event_type);
```

---

## Global Script (ht.js)

```javascript
(function() {
  var S = '{{.ServerURL}}';

  // Get or create visitor ID
  var vid = localStorage.getItem('ht_vid');
  if (!vid) {
    vid = crypto.randomUUID();
    localStorage.setItem('ht_vid', vid);
  }

  // Process all test elements
  document.querySelectorAll('[data-ht-name]').forEach(function(el) {
    var name = el.dataset.htName;
    var variants = JSON.parse(el.dataset.htVariants || '[]');
    if (!variants.length) return;

    // Get or assign variant
    var key = 'ht_' + name;
    var v = localStorage.getItem(key);
    if (v === null) {
      v = Math.floor(Math.random() * variants.length);
      localStorage.setItem(key, v);
    } else {
      v = parseInt(v);
    }

    // Swap text
    el.textContent = variants[v];

    // Send view beacon
    beacon(name, v, 'view');
  });

  // Process convert elements
  document.querySelectorAll('[data-ht-convert]').forEach(function(el) {
    var name = el.dataset.htConvert;
    var v = parseInt(localStorage.getItem('ht_' + name) || '0');

    // Swap text if variants provided
    var variants = el.dataset.htConvertVariants;
    if (variants) {
      variants = JSON.parse(variants);
      if (variants[v]) el.textContent = variants[v];
    }

    // URL type: beacon on load
    if (el.dataset.htConvertType === 'url') {
      beacon(name, v, 'convert');
      return;
    }

    // Click handler
    el.addEventListener('click', function() {
      beacon(name, v, 'convert');
    });
  });

  function beacon(t, v, e) {
    navigator.sendBeacon(S + '/b', JSON.stringify({t:t, v:v, e:e, vid:vid}));
  }
})();
```

---

## Framework Integration

All examples assume you've deployed headline-goat and have a URL like `https://ht.example.com`.

### HTML

Just add the script and data attributes:

```html
<script src="https://ht.example.com/ht.js" defer></script>

<h1 data-ht-name="hero" data-ht-variants='["A","B"]'>A</h1>
<button data-ht-convert="hero">Sign Up</button>
```

### React / Next.js / Vue / Svelte

Add global script to layout, then use data attributes:

```jsx
// React example
function Hero() {
  return (
    <>
      <h1
        data-ht-name="hero"
        data-ht-variants='["Ship Faster","Build Better"]'
      >
        Ship Faster
      </h1>
      <button data-ht-convert="hero">Sign Up</button>
    </>
  );
}
```

For SSR (to avoid flash), use middleware + cookies:

```jsx
// Next.js server component
import { cookies } from 'next/headers';

const variants = ["Ship Faster", "Build Better"];

export async function Hero() {
  const cookieStore = await cookies();
  const v = parseInt(cookieStore.get('ht_hero')?.value ?? '0');

  return (
    <h1
      data-ht-name="hero"
      data-ht-variants={JSON.stringify(variants)}
      data-ht-selected={v}
    >
      {variants[v]}
    </h1>
  );
}
```

The global script skips text swap if `data-ht-selected` is present.

### Laravel / Django

Same pattern - add script to layout, use data attributes in templates:

```blade
{{-- Laravel Blade --}}
<h1
  data-ht-name="hero"
  data-ht-variants='@json(["Ship Faster", "Build Better"])'
>
  Ship Faster
</h1>
```

---

## Statistical Significance

### Confidence Intervals

Wilson score interval for conversion rates:

```go
func WilsonInterval(successes, trials int, confidence float64) (lower, upper float64)
```

### Winner Detection

Two-proportion z-test to determine if leading variant significantly beats others.

**Threshold:** 95% confidence to declare winner

**Dashboard display:**
- `< 90%`: "Not enough data"
- `90-95%`: "Possibly better"
- `≥ 95%`: "Winner"

---

## Design Decisions

### Configuration in HTML

Test configuration lives in data attributes, not on the server:
- **Simplicity**: No CLI step to define tests
- **Flexibility**: Change variants without restarting server
- **Transparency**: See exactly what's being tested in the HTML

### Tests Auto-Create

Tests are created automatically on first beacon:
- No explicit registration required
- Just add data attributes and deploy
- Typos create new tests (visible in dashboard, easy to spot)

### Global Script

One script for entire site instead of per-test scripts:
- Single `<script>` tag in layout
- Add new tests without changing layout
- Smaller total payload

### No Site URL Required

Server doesn't store the URL of the site being tested:
- CORS is open (anonymous data)
- Same test name could run on multiple sites
- User knows their site

### Variant Index Sync

Headline and convert button share the same variant index:
- Consistent experience for visitors
- If headline shows variant 1, button shows variant 1

### Static Winner Output

After declaring a winner, CLI shows plain HTML to replace test code:
- No dependency on headline-goat after test completes
- Clean transition to production

---

## Roadmap

### v1.0 (Current)
- [x] Global script with data attributes
- [x] Text variants
- [x] Conversion tracking (click and URL)
- [x] Dashboard with statistics
- [x] Winner declaration

### v1.1 (Future)
- [ ] Style variants (`data-ht-style-variants`)
- [ ] Weight distribution (`data-ht-weights`)
- [ ] Pause/resume tests
- [ ] Test archiving

### v2.0 (Future)
- [ ] Multi-page funnels
- [ ] Revenue tracking
- [ ] Segment analysis

---

## Deployment

headline-goat needs to be accessible from your website to receive beacons. Choose one of these deployment options:

### Option 1: Cloudflare Tunnel (Recommended for getting started)

Run headline-goat locally and expose via Cloudflare Tunnel (free):

```bash
# Terminal 1: Start headline-goat
./headline-goat

# Terminal 2: Create tunnel (gives you a URL like https://xyz.trycloudflare.com)
cloudflared tunnel --url http://localhost:8080
```

Use the tunnel URL in your script tag:
```html
<script src="https://xyz.trycloudflare.com/ht.js" defer></script>
```

For a permanent subdomain, set up a named tunnel:
```bash
cloudflared tunnel create headline-goat
cloudflared tunnel route dns headline-goat ht.yourdomain.com
cloudflared tunnel run headline-goat
```

### Option 2: Fly.io (Recommended for production)

Deploy to Fly.io for a permanent URL with automatic SSL:

```bash
# First time setup
fly launch --name my-headline-goat

# Deploy
fly deploy

# Add custom domain (optional)
fly certs add ht.yourdomain.com
```

Your URL will be: `https://my-headline-goat.fly.dev`

**fly.toml example:**
```toml
app = "my-headline-goat"
primary_region = "sjc"

[build]
  builder = "paketobuildpacks/builder:base"

[http_service]
  internal_port = 8080
  force_https = true

[mounts]
  source = "headline_goat_data"
  destination = "/data"
```

### Option 3: VPS (DigitalOcean, Linode, Hetzner)

Deploy to any VPS with Caddy for automatic SSL:

```bash
# On your server
./headline-goat --port 8080 --db /var/lib/headline-goat/data.db
```

**Caddyfile:**
```
ht.yourdomain.com {
  reverse_proxy localhost:8080
}
```

Your URL will be: `https://ht.yourdomain.com`

### After Deployment

Once deployed, update your site's script tag with your URL:

```html
<!-- Replace with your actual deployed URL -->
<script src="https://ht.yourdomain.com/ht.js" defer></script>
```

The dashboard is available at `https://ht.yourdomain.com/dashboard?token=<your-token>`

---

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `HG_PORT` | `8080` | Server port |
| `HG_DB_PATH` | `./headline-goat.db` | SQLite database path |

### Command Flags

- `--port <port>` - Port number
- `--db <path>` - Database path

---

## License

MIT
