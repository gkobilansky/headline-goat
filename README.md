# 🐐 Headline Goat

A/B test any text on any website. Minimal setup, maximum flexibility. Built for humans and AI agents.

```bash
# Create a test via CLI
hlg create hero --variants "Ship Faster,Build Better" --url "/" --target "h1"

# Or define tests inline with data attributes
<h1 data-hlg-name="hero" data-hlg-variants='["Ship Faster","Build Better"]'>Ship Faster</h1>
```

Single Go binary with embedded SQLite. No external services, no dependencies.

---

## Why Headline Goat

- **Test any text** — Headlines, CTAs, value props. If it's text, you can test it.
- **CLI or data attributes** — Centralized control or inline definitions. Mix both.
- **AI-agent friendly** — JSON output with time-to-significance estimates for automated workflows.
- **Minimal** — ~2000 lines of Go. The entire codebase fits in your head.
- **Self-hosted** — Your data stays on your server.

---

## Quick Start

### 1. Install

```bash
go install github.com/gkobilansky/headline-goat/cmd/hlg@latest
```

### 2. Start the server

```bash
hlg
```

### 3. Add to your site

```html
<!-- Replace with your server URL -->
<script src="https://your-server.com/hlg.js" defer></script>
```

> **Note:** The script auto-detects the correct protocol (http/https) from request headers, so it works seamlessly behind reverse proxies.

### 4. Create a test

**Via CLI:**
```bash
hlg create hero --variants "Ship Faster,Build Better" --url "/" --target "h1"
```

**Via data attributes:**
```html
<h1 data-hlg-name="hero" data-hlg-variants='["Ship Faster","Build Better"]'>Ship Faster</h1>
<button data-hlg-convert="hero">Sign Up</button>
```

### 5. Check results

```bash
hlg results hero
```

```
TEST: hero
STATE: running

VARIANT           VIEWS    CONVERSIONS  RATE     95% CI
────────────────────────────────────────────────────────────
Ship Faster       412      32           7.77%    [5.2%, 10.3%]
Build Better      398      41           10.30%   [7.4%, 13.2%]  ← LEADING

Statistical significance: 94.2% confident "Build Better" beats control

STATUS
──────
Progress: 810 / 16300 views (5%)
Traffic: 58 views/hour
Check back in: ~267 hours
```

When ready: `hlg winner hero --variant 1`

---

## AI Agent Integration

The CLI supports `--json` output with time-to-significance estimates for automated workflows.

### Workflow

```bash
# 1. Create test
hlg create hero --variants "A,B" --json

# 2. Check results
hlg results hero --json
# → "status.ready": false
# → "status.check_back_at": "2026-01-17T02:00:00Z"

# 3. Sleep until check_back_at, repeat step 2

# 4. When status.ready=true
hlg winner hero --variant 1
```

### JSON Output

```bash
hlg results hero --json    # Full results with status estimates
hlg list --json            # All tests
hlg create hero -v "A,B" --json
```

Key fields: `status.ready`, `status.check_back_at`, `status.recommended_variant`

### Claude Code

Add to your `CLAUDE.md`:

```markdown
## A/B Testing

Use `hlg` for headline A/B testing. Workflow:
1. `hlg create <name> --variants "A,B"` - Create test
2. `hlg results <name> --json` - Check status
3. When status.ready=true: `hlg winner <name> --variant <index>`
```

Or install the skill:

```bash
mkdir -p .claude/skills/hlg
curl -o .claude/skills/hlg/SKILL.md \
  https://raw.githubusercontent.com/gkobilansky/headline-goat/main/skills/hlg/SKILL.md
```

---

## CLI Reference

| Command | Description |
|---------|-------------|
| `hlg` | Start server |
| `hlg create <name> --variants "A,B"` | Create test |
| `hlg results <name>` | Show results |
| `hlg list` | List all tests |
| `hlg winner <name> --variant N` | Declare winner |
| `hlg export <name>` | Export data |
| `hlg token` | Show dashboard URL |

Add `--json` to any command for JSON output.

### Create Options

| Flag | Description |
|------|-------------|
| `--variants`, `-v` | Comma-separated variants (required) |
| `--url` | Page path to match |
| `--target` | CSS selector for headline |
| `--cta-target` | CSS selector for conversion button |
| `--conversion-url` | Track conversion on page load |

---

## Data Attributes

| Attribute | Description |
|-----------|-------------|
| `data-hlg-name` | Test identifier (required) |
| `data-hlg-variants` | JSON array of variants (required) |
| `data-hlg-convert` | Test name to track conversion |
| `data-hlg-convert-type="url"` | Convert on page load |
| `data-hlg-selected="N"` | Pre-selected variant (SSR) |

---

## Deployment

Headline Goat needs a persistent process. The server automatically detects HTTPS when behind a reverse proxy via `X-Forwarded-Proto` header.

### Reverse Proxy (nginx, Caddy, etc.)

```nginx
location ~ ^/(hlg\.js|b|dashboard|api) {
    proxy_pass http://localhost:8080;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Forwarded-Host $host;
}
```

The `hlg.js` script will automatically use the correct `https://` URLs.

### Cloudflare Tunnel

```bash
cloudflared tunnel --url http://localhost:8080
```

### Cloud Platforms

Most platforms handle TLS termination automatically. Just expose the port:

```bash
HG_PORT=8000 ./hlg
```

**Note:** You need shell access to run CLI commands (`hlg results`, `hlg winner`). Platforms with SSH access (exe.dev, VPS) are recommended. The dashboard at `/dashboard?token=xxx` provides view-only access to results.

### Docker

```bash
docker build -t hlg .
docker run -p 8080:8080 -v hlg-data:/data hlg
```

### Systemd (keep running)

```ini
# /etc/systemd/system/hlg.service
[Unit]
Description=Headline Goat A/B Testing
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/hlg
ExecStart=/opt/hlg/hlg
Restart=always
Environment=HG_PORT=8080
Environment=HG_DB_PATH=/opt/hlg/hlg.db

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl enable --now hlg
```

---

## Framework Examples

**React/Next.js:**
```jsx
<h1 data-hlg-name="hero" data-hlg-variants='["A","B"]'>A</h1>
<button data-hlg-convert="hero">Sign Up</button>
```

**Vue:**
```vue
<h1 data-hlg-name="hero" :data-hlg-variants='JSON.stringify(["A","B"])'>A</h1>
```

**Svelte:**
```svelte
<h1 data-hlg-name="hero" data-hlg-variants={JSON.stringify(["A","B"])}>A</h1>
```

---

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `HG_PORT` | `8080` | Server port |
| `HG_DB_PATH` | `./hlg.db` | Database path |
| `HG_SERVER_URL` | (auto-detect) | Override server URL in generated script |

The `hlg.js` script automatically detects the correct URL from request headers, including `X-Forwarded-Proto` and `X-Forwarded-Host` for reverse proxy setups.

---

## FAQ

**How do I avoid text flash?**
Use `data-hlg-selected` for SSR, or: `[data-hlg-name] { visibility: hidden; }`

**How long should I run a test?**
Until 95% confidence. The STATUS section tells you when.

**Multiple tests on one page?**
Yes. Each `data-hlg-name` is independent.

---

## Contributing

Strict TDD. Every change needs a failing test first.

```bash
go test ./... -v -race
go build -o hlg ./cmd/hlg
```

See `CLAUDE.md` for architecture details.

---

## License

MIT
