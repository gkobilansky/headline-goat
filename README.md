# 🐐 Headline Goat

A/B test any text on any website. Minimal setup, maximum flexibility.

```bash
# Create a test via CLI - targets elements by CSS selector
hlg create hero --variants "Ship Faster,Build Better" --url "/" --target "h1"

# Or define tests inline with data attributes
<h1 data-hlg-name="hero" data-hlg-variants='["Ship Faster","Build Better"]'>Ship Faster</h1>
```

Single Go binary with embedded SQLite. No external services, no dependencies.

- Run `./hlg`, drop <script src='…/hlg.js'> on your site, and mark any text with data-hlg-name/data-hlg-variants. 
- The script assigns a variant, records views/conversions to SQLite, and you inspect results via CLI or dashboard.
- You can also predefine tests in the DB with URL/selector targeting.
- No external services—just Go + SQLite.

---

## What makes it useful

**Test any text element** — Headlines, subheadings, CTAs, value props. If it's text, you can test it.

**Two ways to define tests** — Use CLI commands when you want centralized control, or data attributes when you want tests defined alongside the markup. Mix both approaches on the same site.

**Minimal by design** — ~2000 lines of Go. Easy to read, easy to understand, easy to extend. The entire codebase fits in your head.

**Self-hosted** — Your data stays on your server. Run it locally, on a VPS, or behind a tunnel.

---

## Quick Start

### 1. Get the binary

```bash
# Download latest release (macOS ARM)
curl -L -o hlg https://github.com/gkobilansky/headline-goat/releases/latest/download/hlg-darwin-arm64
chmod +x hlg

# Or build from source
go install github.com/gkobilansky/headline-goat/cmd/hlg@latest
```

### 2. Start the server

```bash
./hlg
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
./hlg results hero
```

```
TEST: hero
STATE: running

VARIANT           VIEWS    CONVERSIONS  RATE     95% CI
────────────────────────────────────────────────────────────
Ship Faster       412      32           7.77%    [5.2%, 10.3%]
Build Better      398      41           10.30%   [7.4%, 13.2%]  ← LEADING

Statistical significance: 94.2% confident "Build Better" beats control
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

First visit with `?token=` sets a cookie (24h). After that, the cookie handles auth automatically.

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

Both methods work. You can even mix them — use CLI for some tests, data attributes for others. The `hlg list` command shows the source for each test.

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

## Deployment

headline-goat needs to be accessible from your website. A few options:

### Cloudflare Tunnel (quickest)

```bash
# Terminal 1
./hlg

# Terminal 2
cloudflared tunnel --url http://localhost:8080
# Gives you https://random-words.trycloudflare.com
```

### Fly.io (production-ready)

```bash
fly launch --name my-headline-goat
fly deploy
# Gives you https://my-headline-goat.fly.dev
```

### Any VPS + Caddy

```bash
# On your server
./hlg --port 8080 --db /var/lib/hlg/data.db
```

```
# Caddyfile
hlg.yourdomain.com {
  reverse_proxy localhost:8080
}
```

---

## Statistics

headline-goat uses proper statistics:

- **Wilson score intervals** for confidence intervals (accurate even with small samples)
- **Two-proportion z-test** for significance testing
- **95% confidence threshold** to declare a winner

No more "this variant is winning" with 12 visits.

---

## Works with AI Coding Assistants

The CLI is just shell commands, so AI tools like Claude Code, Cursor, or Copilot can help you manage tests:

```
You: "Create an A/B test for the homepage hero"

AI: hlg create hero --variants "Ship Faster,Build Better" --url "/" --target "h1"
```

Create tests, check results, export data, declare winners — all through simple commands.

**Claude Code users:** You can create a [Skill](https://docs.anthropic.com/en/docs/claude-code/skills) to teach Claude about hlg. Add a `SKILL.md` file to `.claude/skills/hlg/` with the command reference, and Claude will automatically know when and how to manage your tests.

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
