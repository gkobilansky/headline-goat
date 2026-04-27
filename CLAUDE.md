# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

headline-goat is a minimal, self-hosted A/B testing tool for headlines. Single Go binary, embedded SQLite, no external dependencies.

## Development Process: Strict TDD

This project follows Test-Driven Development with no exceptions:

1. **RED** → Write a failing test that defines behavior
2. **GREEN** → Write minimal code to make the test pass
3. **REFACTOR** → Improve code quality, keep tests green

**Mandatory Rules:**
- Write tests FIRST - no production code without a failing test
- Write minimal code only - just enough to pass the failing test
- Run full test suite before EVERY commit
- Tests must verify actual functionality - avoid over-mocking

## Commands

```bash
# Run all tests (REQUIRED before every commit)
go test ./... -v -race

# Build development binary
go build -o hlg ./cmd/hlg

# Build production binary (optimized with embedded assets)
go build -ldflags="-s -w" -o hlg ./cmd/hlg

# Cross-compile examples
GOOS=linux GOARCH=amd64 go build -o hlg-linux-amd64 ./cmd/hlg
GOOS=darwin GOARCH=arm64 go build -o hlg-darwin-arm64 ./cmd/hlg
```

## Architecture

```
headline-goat/
├── cmd/hlg/main.go              # CLI entry point
├── internal/
│   ├── cli/                     # CLI commands (init, list, serve, results, etc.)
│   ├── server/                  # HTTP server (handlers, middleware, routing)
│   ├── store/                   # Database layer (SQLite, models, interfaces)
│   ├── stats/                   # Statistical calculations (Wilson intervals, significance)
│   ├── snippets/                # Framework snippet generators
│   └── dashboard/               # Dashboard UI (embedded assets, templates)
└── tests/
    ├── unit/                    # Pure functions, isolated logic
    ├── integration/             # Database operations, HTTP handlers
    └── e2e/                     # Full user flows, CLI commands
```

## Data Flow

Client JS (browser) → `/b` beacon endpoint → SQLite storage → Stats Calculator → Dashboard UI

## Key Technical Details

- SQLite with WAL mode enabled
- All dashboard assets embedded in binary using `//go:embed`
- Wilson score interval for confidence intervals
- Two-proportion z-test for winner detection (95% threshold)
- Visitor deduplication by visitor_id + test + event type

## Agent Workflow for A/B Testing

Use `hlg` for headline/copy testing. Run `hlg --help` for commands.

Core workflow:
1. `hlg create <name> --variants "A,B"` - Create test
2. Deploy script tag to site (see `hlg token` for snippet)
3. `hlg results <name>` - Check status (note "check back in" time)
4. Sleep for suggested duration, repeat step 3
5. When ready: `hlg winner <name> --variant <index>`

The results command shows traffic rate and estimated time to significance.

### JSON Output for Automation

All commands support `--json` flag for programmatic use:

```bash
# Get results as JSON
hlg results hero --json

# List tests as JSON
hlg list --json

# Create test with JSON response
hlg create hero --variants "A,B" --json
```

### Provisioning a VPS

`hlg deploy` provisions a fresh VPS and installs hlg as a systemd service.
It uses your authenticated provider CLI — no separate API tokens needed:

```bash
hlg deploy                       # auto-picks if only doctl or hcloud is authenticated
hlg deploy --provider hetzner    # explicit when both are authenticated
hlg deploy --size cpx21 --region hel1
hlg deploy --yes                 # skip confirmation prompts (CI/scripts)
```

Prereqs (install one):
- DigitalOcean: [doctl](https://docs.digitalocean.com/reference/doctl/how-to/install/) + `doctl auth init`
- Hetzner Cloud: [hcloud](https://github.com/hetznercloud/cli) + `hcloud context create <name>`

The deploy flow:
1. Picks your local SSH pubkey (`~/.ssh/id_ed25519.pub` preferred)
2. Checks if it's registered with the provider; if not, asks once to upload it
3. Creates the VM, installs `hlg` binary + systemd unit via cloud-init
4. Writes `~/.hlg/config.json` so `hlg list`, `hlg results`, etc. work immediately

### Remote Mode (SSH dispatch)

Data commands (`list`, `results`, `create`, `winner`, `export`, `token`) can
run against a remote `hlg` server over SSH. This is the recommended way to
drive a deployed VPS from your local machine without ever opening an
interactive SSH session.

Activation (first match wins):
1. `HLG_REMOTE=user@host[:port]` env var
2. `hlg.json` in the current repo (walked up from cwd, like `.git`)
3. `~/.hlg/config.json` global default

Flags:
- `--remote` force remote (errors if no config is discoverable)
- `--local` force local even if `hlg.json` is present

Example `hlg.json` (safe to commit — no secrets):
```json
{"host": "hlg.yourdomain.com", "user": "hlg"}
```

Under the hood, `hlg <cmd> <args>` becomes `ssh user@host 'hlg <cmd> <args>'`
with stdio streamed through. Authentication is handled by your existing SSH
agent / `~/.ssh/config`.

JSON output includes:
- Test metadata (name, state, variants)
- Per-variant stats (views, conversions, rate, confidence intervals)
- Significance data (confidence level, leading variant)
- Status estimates (views needed, traffic rate, check-back time)
