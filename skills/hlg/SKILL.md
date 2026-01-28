# Headline Goat (hlg) - A/B Testing Tool

Use `hlg` for headline and copy A/B testing. Self-hosted, single binary, SQLite storage.

## When to Use

- User asks to A/B test headlines, copy, CTAs, or any text
- User wants to optimize conversion rates
- User needs statistical significance for text variants

## Core Workflow

```bash
# 1. Create a test
hlg create hero --variants "Ship Faster,Build Better"

# 2. Check results (use --json for automation)
hlg results hero --json

# 3. When status.ready=true, declare winner
hlg winner hero --variant 1
```

## Commands

| Command | Description |
|---------|-------------|
| `hlg create <name> --variants "A,B"` | Create test with variants |
| `hlg results <name>` | Show results with STATUS section |
| `hlg results <name> --json` | Results as JSON (for automation) |
| `hlg list` | List all tests |
| `hlg list --json` | List as JSON |
| `hlg winner <name> --variant N` | Declare winning variant |
| `hlg export <name>` | Export raw data |
| `hlg token` | Show dashboard URL |

## Create Options

```bash
hlg create hero \
  --variants "A,B,C" \        # Comma-separated variants (required)
  --url "/" \                 # Page path to match
  --target "h1" \             # CSS selector for headline
  --cta-target "button.cta" \ # CSS selector for conversion button
  --conversion-url "/thanks"  # Track conversion on page load
  --json                      # Output as JSON
```

## JSON Output Structure

`hlg results <name> --json` returns:

```json
{
  "name": "hero",
  "state": "running",
  "variants": [
    {"index": 0, "name": "A", "views": 142, "conversions": 12, "rate": 0.085, "leading": false},
    {"index": 1, "name": "B", "views": 138, "conversions": 18, "rate": 0.130, "leading": true}
  ],
  "significance": {
    "confident": false,
    "confidence_level": 0.87,
    "leading_variant": 1
  },
  "status": {
    "ready": false,
    "views_needed": 780,
    "views_current": 280,
    "progress_percent": 36,
    "traffic_rate_per_hour": 58,
    "estimated_hours": 8.6,
    "check_back_at": "2026-01-16T22:00:00Z",
    "message": "Check back in ~9 hours",
    "recommended_variant": -1
  }
}
```

Key fields for automation:
- `status.ready` — true when test has reached statistical significance
- `status.check_back_at` — when to check again (ISO 8601)
- `status.recommended_variant` — variant index to use (-1 if not ready)
- `significance.confident` — true if >= 95% confidence

## Agent Automation Pattern

```bash
# Create test
hlg create hero --variants "Original,New Copy" --json

# Poll until ready (check status.ready)
while true; do
  result=$(hlg results hero --json)
  ready=$(echo "$result" | jq -r '.status.ready')
  if [ "$ready" = "true" ]; then
    variant=$(echo "$result" | jq -r '.status.recommended_variant')
    hlg winner hero --variant $variant
    break
  fi
  # Sleep until check_back_at
  sleep 3600
done
```

## HTML Integration

Add script to site (URL auto-detects from request):
```html
<script src="https://your-hlg-server.com/hlg.js" defer></script>
```

Data attributes (alternative to CLI):
```html
<h1 data-hlg-name="hero" data-hlg-variants='["A","B"]'>A</h1>
<button data-hlg-convert="hero">Sign Up</button>
```

## Deployment

Start the server:
```bash
HG_PORT=8000 ./hlg
```

Behind a reverse proxy (nginx, Cloudflare, cloud platforms), the script automatically uses HTTPS via `X-Forwarded-Proto` header.

**Important:** CLI commands (`hlg results`, `hlg winner`) require shell access. Deploy on a VPS or platform with SSH access (exe.dev, DigitalOcean, etc.) for full functionality. The web dashboard provides view-only access.

## Tips

- Tests auto-create on first visitor if using data attributes
- Use `--json` for all commands when automating
- 95% confidence threshold for statistical significance
- Wilson score intervals for accurate small-sample stats
- Script URL is dynamic — auto-detects protocol and host from request headers
- Works behind any TLS-terminating proxy (nginx, Cloudflare, AWS ALB, etc.)
