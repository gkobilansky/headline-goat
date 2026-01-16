# 🐐 Headline Goat

A/B test any text on any website. Minimal setup, maximum flexibility.

```bash
# Create a test via CLI - targets elements by CSS selector
hlg create hero --variants "Ship Faster,Build Better" --url "/" --target "h1"

# Or define tests inline with data attributes
<h1 data-hlg-name="hero" data-hlg-variants='["Ship Faster","Build Better"]'>Ship Faster</h1>
```

Single Go binary with embedded SQLite. No external services, no dependencies.

- Run `hlg` to serve global script
- Add `<script src='…/hlg.js'>` to your site and mark any text with `data-hlg-name` and `data-hlg-variants`. 
- The script assigns a variant, records views/conversions to SQLite, and you inspect results via CLI or dashboard.
- You can also predefine tests in the DB with URL/selector targeting.
- No external services, just Go + SQLite.

---

## What makes it useful

**Test any text element** — Headlines, subheadings, CTAs, value props. If it's text, you can test it.

**Two ways to define tests** — Use CLI commands when you want centralized control, or data attributes when you want tests defined alongside the markup. Mix both approaches for the same test.

**Minimal by design** — ~2000 lines of Go. Easy to read, easy to understand, easy to extend. The entire codebase fits in your head.

**Self-hosted** — Your data stays on your server. Run it locally, on a VPS, or behind a tunnel.

---

## Quick Start

### 1. Install

```bash
go install github.com/gkobilansky/headline-goat/cmd/hlg@latest
```

This puts `hlg` in your `$GOPATH/bin` (usually `~/go/bin`). Make sure it's in your PATH.

### 2. Start the server

```bash
hlg
```

You'll see setup prompts and get a dashboard URL with your access token.

### 3. Add to your site

Drop the script tag in your `<head>`:

```html
<script src="http://localhost:8080/hlg.js" defer></script>
```

### 4. Create a test

**Option A: Via CLI** (centralized, no HTML changes needed)

```bash
hlg create hero --variants "Ship Faster,Build Better" --url "/" --target "h1"
```

**Option B: Via data attributes** (inline, self-documenting)

```html
<h1 data-hlg-name="hero" data-hlg-variants='["Ship Faster","Build Better"]'>
  Ship Faster
</h1>
<button data-hlg-convert="hero">Sign Up</button>
```

Use CLI when you want central control or can't easily edit HTML. Use data attributes when you want tests defined alongside the elements they modify.

### 5. Watch the results

```bash
hlg results hero
```

```
TEST: hero
STATE: running
CREATED: 2026-01-16

VARIANT           VIEWS    CONVERSIONS  RATE     95% CI
────────────────────────────────────────────────────────────
Ship Faster       412      32           7.77%    [5.2%, 10.3%]
Build Better      398      41           10.30%   [7.4%, 13.2%]  ← LEADING

Statistical significance: 94.2% confident "Build Better" beats control (not yet significant)

STATUS
──────
Progress: 810 / 16300 views (5%)
Traffic: 58 views/hour
Check back in: ~267 hours
```

When the test reaches 95% confidence:

```
STATUS
──────
✓ Ready to declare winner
Run: hlg winner hero --variant 1
```

---

## How It Works

1. **Visitor loads your page** → Script picks a random variant, stores it in localStorage
2. **Headline text swaps** → Visitor sees their assigned variant
3. **View beacon fires** → Server records the impression
4. **Visitor clicks CTA** → Convert beacon fires, conversion recorded
5. **You check results** → CLI or dashboard shows stats with confidence intervals

Tests auto-create on first beacon. No pre-registration needed.

---

## Deployment

Headline Goat needs a persistent process and SQLite storage, eg a VPS, container host, or local machine.

### Same server vs. separate

| | Same Server | Separate Server |
|---|-------------|-------------------------|
| **Complexity** | One thing to manage | Two servers |
| **Cost** | Free | $3-5/mo |
| **Latency** | Fastest | Extra hop |
| **CORS** | None | Works, but extra headers |

**Recommendation:** Run on the same server as your website if you can. Headline Goat is tiny (~8MB, minimal CPU/RAM) and won't compete for resources. Use a separate server only if you're on a serverless platform and have no choice.

### Same server (recommended)

Run hlg alongside your website and route via nginx or Caddy:

```nginx
# nginx - route hlg endpoints to the binary
location ~ ^/(hlg\.js|b|dashboard) {
    proxy_pass http://localhost:8080;
}
```

```html
<script src="/hlg.js" defer></script>
```

No CORS, same domain, clean URLs.

### Cloudflare Tunnel (quickest for development)

```bash
# Terminal 1
hlg

# Terminal 2
cloudflared tunnel --url http://localhost:8080
# Gives you https://random-words.trycloudflare.com
```

### Docker (for VPS or container hosts)

A `Dockerfile` is included for deploying to any Docker host.

```bash
# Build and run locally
docker build -t hlg .
docker run -p 8080:8080 -v hlg-data:/data hlg
```

---

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  /hlg.js    │────▶│  /b beacon  │────▶│   SQLite    │
│  (browser)  │     │  endpoint   │     │   database  │
└─────────────┘     └─────────────┘     └─────────────┘
                                               │
                                               ▼
                    ┌─────────────┐     ┌─────────────┐
                    │  Dashboard  │◀────│   Stats     │
                    │  /dashboard │     │  (Wilson)   │
                    └─────────────┘     └─────────────┘
```

**Key components:**

| Path | Purpose |
|------|---------|
| `cmd/hlg/` | CLI entry point |
| `internal/cli/` | Command implementations |
| `internal/server/` | HTTP handlers, `/hlg.js` generation |
| `internal/store/` | SQLite database layer |
| `internal/stats/` | Wilson intervals, z-test significance |
| `internal/dashboard/` | Embedded HTML/CSS templates |

Everything compiles into a single binary (~8MB). No runtime dependencies.

---

## Dashboard

The dashboard shows all tests, conversion rates, and statistical significance.

**Authentication:** Token-based. On first startup, hlg generates an 8-character token stored in `.hlg-token` alongside your database.

```bash
# Get your dashboard URL anytime
hlg token
# → Dashboard: http://localhost:8080/dashboard?token=a1b2c3d4
```

First visit with `?token=` sets an auth cookie (24h).

---

## Creating Tests

Two approaches, same results. Pick what fits your workflow.

### How they differ

| | CLI tests | Data attribute tests |
|---|-----------|---------------------|
| **Created** | `hlg create` command | Auto-created on first page view |
| **Targeting** | CSS selectors (`--target "h1"`) | Element has the attributes |
| **Variants stored** | In database | In HTML (sent with beacon) |
| **Source** | `server` | `client` |

Both methods work. You can even mix them — use CLI to create the test and set `--conversion-url`, use data attributes to define variants. 
The `hlg list` command shows the source for each test.

**Note:** If you create a test via CLI and also have data attributes for the same test name, the dashboard will flag a "source conflict." This isn't an error — just a heads-up that the test has mixed origins.

### Option A: CLI (centralized control)

Create tests from the command line with CSS selector targeting:

```bash
# Basic test
hlg create hero --variants "Ship Faster,Build Better"

# With URL and element targeting
hlg create hero \
  --variants "Ship Faster,Build Better" \
  --url "/" \
  --target "h1" \
  --cta-target "button.signup"
```

| Flag | Description |
|------|-------------|
| `--variants` | Comma-separated variant text (required) |
| `--url` | Page path to match (e.g., "/", "/pricing") |
| `--target` | CSS selector for the headline element |
| `--cta-target` | CSS selector for the conversion button |
| `--conversion-url` | Track conversion on page load (e.g., "/thanks") |

**Best for:** Central test management, can't easily edit HTML, multiple tests across pages.

### Option B: Data Attributes (inline definition)

Define tests directly in your HTML:

```html
<h1
  data-hlg-name="hero"
  data-hlg-variants='["Option A","Option B","Option C"]'
>
  Option A
</h1>
```

| Attribute | Required | Description |
|-----------|----------|-------------|
| `data-hlg-name` | Yes | Unique test identifier |
| `data-hlg-variants` | Yes | JSON array of text variants |

**Best for:** Self-documenting tests, quick iteration, tests defined where they're used.

### Tracking Conversions

**Via CLI:**

```bash
# Convert when user reaches a thank-you page
hlg create hero --variants "A,B" --conversion-url "/thanks"

# Convert on button click (CSS selector)
hlg create hero --variants "A,B" --cta-target "button.signup"
```

**Via data attributes:**

```html
<!-- Click conversion (buttons, links) -->
<button data-hlg-convert="hero">Sign Up</button>

<!-- Page-load conversion (thank-you pages) -->
<div data-hlg-convert="hero" data-hlg-convert-type="url" hidden></div>

<!-- CTA with variant text -->
<button
  data-hlg-convert="hero"
  data-hlg-convert-variants='["Get Started","Sign Up Free"]'
>
  Get Started
</button>
```

| Attribute | Required | Description |
|-----------|----------|-------------|
| `data-hlg-convert` | Yes | Test name to track |
| `data-hlg-convert-type` | No | Set to `"url"` for page-load conversion |
| `data-hlg-convert-variants` | No | JSON array of button text variants |

### SSR Support

For server-rendered apps where you want to avoid a text flash:

```html
<h1
  data-hlg-name="hero"
  data-hlg-variants='["A","B"]'
  data-hlg-selected="1"
>
  B
</h1>
```

When `data-hlg-selected` is present, the script skips text swap and just sends the beacon.

---

## CLI Commands

| Command | Description |
|---------|-------------|
| `hlg` | Start server (interactive setup on first run) |
| `hlg list` | List all tests with summary stats |
| `hlg results <name>` | Detailed results for a test |
| `hlg winner <name> --variant N` | Declare a winner |
| `hlg export <name>` | Export raw data (CSV/JSON) |
| `hlg create <name> --variants "A,B"` | Create test via CLI |
| `hlg token` | Show dashboard URL |

### Global flags

```bash
--db <path>    # Database path (default: ./hlg.db, env: HG_DB_PATH)
--port <port>  # Server port (default: 8080, env: HG_PORT)
```

---

## Framework Examples

### React / Next.js

```jsx
function Hero() {
  return (
    <>
      <h1
        data-hlg-name="hero"
        data-hlg-variants='["Ship Faster","Build Better"]'
      >
        Ship Faster
      </h1>
      <button data-hlg-convert="hero">Sign Up</button>
    </>
  );
}
```

### Vue

```vue
<template>
  <h1
    data-hlg-name="hero"
    :data-hlg-variants='JSON.stringify(["Ship Faster", "Build Better"])'
  >
    Ship Faster
  </h1>
  <button data-hlg-convert="hero">Sign Up</button>
</template>
```

### Svelte

```svelte
<h1
  data-hlg-name="hero"
  data-hlg-variants={JSON.stringify(["Ship Faster", "Build Better"])}
>
  Ship Faster
</h1>
<button data-hlg-convert="hero">Sign Up</button>
```

---

## Statistics

headline-goat uses proper statistics:

- **Wilson score intervals** for confidence intervals (accurate even with small samples)
- **Two-proportion z-test** for significance testing
- **95% confidence threshold** to declare a winner

No more "this variant is winning" with 12 visits.

---

## Usage with AI Agents

The CLI is designed for AI agent workflows with `--json` output and time-to-significance estimates.

### Agent Workflow

```bash
# 1. Create test
hlg create hero --variants "A,B" --json

# 2. Check results (includes when to check back)
hlg results hero --json
# → "check_back_at": "2026-01-17T02:00:00Z"
# → "message": "Check back in ~9 hours"

# 3. Sleep until check_back_at, repeat step 2

# 4. When ready: declare winner
hlg winner hero --variant 1
```

The `results --json` output includes:
- `status.ready` — boolean, true when statistically significant
- `status.check_back_at` — ISO timestamp for next check
- `status.estimated_hours` — hours until significance
- `status.recommended_variant` — variant index to pick (-1 if not ready)

### JSON Output

All commands support `--json` for programmatic use:

```bash
hlg results hero --json    # Full results with status estimates
hlg list --json            # All tests with summary stats
hlg create hero -v "A,B" --json  # Confirmation with test details
```

### Claude Code Integration

Add this to your project's `CLAUDE.md`:

```markdown
## A/B Testing with Headline Goat

Use `hlg` for headline/copy A/B testing. Run `hlg --help` for commands.

Core workflow:
1. `hlg create <name> --variants "A,B"` - Create test
2. `hlg results <name> --json` - Check status (note check_back_at time)
3. Sleep until check_back_at, repeat step 2
4. When status.ready=true: `hlg winner <name> --variant <index>`

The results command shows traffic rate and estimated time to significance.
Use --json flag for all commands when automating.
```

### Claude Code Skill

For automatic hlg awareness, install the skill:

```bash
mkdir -p .claude/skills/hlg
curl -o .claude/skills/hlg/SKILL.md \
  https://raw.githubusercontent.com/gkobilansky/headline-goat/main/skills/hlg/SKILL.md
```

Claude will then know when and how to manage A/B tests in your project.

---

## Configuration

| Env Variable | Default | Description |
|--------------|---------|-------------|
| `HG_PORT` | `8080` | Server port |
| `HG_DB_PATH` | `./hlg.db` | SQLite database path |

---

## FAQ

**How do I avoid the text flash on page load?**

Use `data-hlg-selected` for SSR, or add a CSS rule:
```css
[data-hlg-name] { visibility: hidden; }
```
The script adds a class after swapping that you can use to show it.

**Can I test more than headlines?**

Yes. Any text element works. Subheadings, CTAs, value props.

**What about images/styles?**

Not yet. Text variants only for now. Open an issue if you need this.

**How long should I run a test?**

Until you hit 95% confidence. The CLI and dashboard tell you when you're there.

**Can I run multiple tests on one page?**

Yes. Each `data-hlg-name` is independent.

---

## Contributing

headline-goat follows strict TDD. Every change needs a failing test first.

```bash
# Run tests (required before every commit)
go test ./... -v -race

# Build
go build -o hlg ./cmd/hlg
```

See `CLAUDE.md` for development guidelines and architecture details.

---

## License

MIT
