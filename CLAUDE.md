# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run

```bash
go build -o instap main.go    # Build binary
./instap                       # Launch TUI (requires auth setup first)
./instap save <url>            # Save bookmark from CLI
./instap list                  # List bookmarks as JSON
echo "https://example.com tag1,tag2" | ./instap  # Pipe URLs via stdin
```

No Makefile or test suite exists. No `*_test.go` files in the project.

## Architecture

Instap is a CLI/TUI client for the Instapaper API, built with Go.

**Flow:** `main.go` → `cmd.Execute()` (Cobra) → either CLI subcommand or interactive TUI

### Packages

- **`cmd/`** — Cobra commands: `root.go` (entry point, stdin detection, TUI launch), `auth.go` (OAuth credential setup + xAuth login), `list.go`, `save.go`, `tag.go`
- **`internal/api/client.go`** — Instapaper REST API client. OAuth 1.0 via `dghubble/oauth1`. All API methods (CRUD bookmarks, folders, tags, text extraction) live here.
- **`internal/tui/tui.go`** — Bubble Tea TUI. Single-file Elm-architecture state machine with 4 states: `stateBrowsing`, `stateMoving`, `stateReading`, `stateTagging`. Uses Catppuccin Macchiato color palette.

### Key Patterns

- The TUI layout in `View()` must carefully account for border/padding heights when calculating `windowHeight` — lipgloss borders add rendered lines that aren't part of the content height.
- Config lives at `~/.instap.yaml` (permissions 0600) managed by Viper. Fields: `consumer_key`, `consumer_secret`, `access_token`, `access_token_secret`.
- Reader view converts HTML → Markdown (`html-to-markdown`) → styled terminal output (`glamour`).
