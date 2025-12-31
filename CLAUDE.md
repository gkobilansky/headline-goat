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
