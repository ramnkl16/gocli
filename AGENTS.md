# Agent instructions (Cursor, Antigravity, others)

Use this file as the single source of truth for how AI agents should work in this repository. Tool-specific rules should stay thin and defer here when possible.

## Project

- **Language:** Go (see `go.mod` for version).
- **Module:** `github.com/yourorg/gocli`.
- **Layout:** `cmd/` (Cobra commands), `internal/` (packages), `main.go` (entry + `.env` loading).

## Expectations

- Match existing patterns: naming, error handling, imports, and comment style in nearby code.
- Keep changes minimal and scoped to the task; avoid drive-by refactors or unrelated edits.
- Do not widen scope (extra features, broad renames, formatting churn) unless explicitly requested.
- Config and secrets: user config under `~/.gocli/config.yaml` (see `internal/config`); secrets via keyring or env, not committed files.

## Checks

- `go test ./...`
- `go vet ./...` (when changing non-trivial logic)

## Docs

- Prefer updating `README.md` only when the user asks or when behavior user-facing in the CLI changes in the same change.
