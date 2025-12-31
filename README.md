# headline-goat

A/B test your headlines without the enterprise BS.

Single binary. Zero dependencies. Self-hosted. **Ship in 30 seconds.**

```html
<script src="https://your-server.com/hlg.js" defer></script>

<h1 data-hlg-name="hero" data-hlg-variants='["Ship Faster","Build Better"]'>
  Ship Faster
</h1>
<button data-hlg-convert="hero">Sign Up</button>
```

That's it. Tests auto-create when traffic arrives. View results in the dashboard or CLI.

---

## Why headline-goat?

You want to know which headline converts better. You don't want:

- Monthly SaaS fees for something this simple
- Complex SDKs with 47 configuration options
- Your visitor data sitting on someone else's server
- A "free tier" that expires right when you need it

headline-goat is a single Go binary with embedded SQLite. Download it, run it, own your data forever.

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

Add data attributes to any headline:

```html
<h1 data-hlg-name="hero" data-hlg-variants='["Ship Faster","Build Better"]'>
  Ship Faster
</h1>
<button data-hlg-convert="hero">Sign Up</button>
```

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

## Data Attributes

### Define a test

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

### Track conversions

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

### SSR support

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
| `hlg otp` | Show dashboard access token |

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

## Server-Side Tests

For more control, create tests via CLI with URL targeting:

```bash
# Create a test that targets a specific page and element
hlg create "hero" \
  --variants "Ship Faster,Build Better" \
  --url "/" \
  --target "h1" \
  --cta-target "button.signup"
```

The script fetches test config from the server and applies variants automatically. No data attributes needed in HTML.

---

## Deployment

headline-goat needs to be accessible from your website. Pick your poison:

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

## License

MIT. Do whatever you want.

---

Built for indie hackers who ship.
