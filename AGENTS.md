# Repository Guidelines

## Project Structure & Module Organization
- `cmd/3pio/` — CLI entrypoint (main package).
- `internal/` — core packages: `orchestrator/`, `runner/`, `adapters/`, `report/`, `ipc/`, `logger/`.
- `tests/` — Go integration tests under `tests/integration_go/` and language fixtures in `tests/fixtures/` (jest, vitest, pytest, go).
- `docs/` — architecture and usage docs; `open-source/`, `noggin/`, and `claude-plans/` hold planning notes.
- `packaging/` — npm, pip, and brew packaging; `scripts/` for release/build helpers.
- `.3pio/` — runtime test reports (ignored; never commit).

## Build, Test, and Development Commands
- `make build` — build CLI to `build/3pio`.
- `make dev` — debug build with symbols.
- `make test` / `make test-integration` / `make test-all` — unit, integration, both.
- `make coverage` — generate `coverage.html`.
- `make fmt` / `make lint` — format and lint (uses `go fmt`, `go vet` or `golangci-lint`).
- Without make: `go build -o build/3pio cmd/3pio/main.go` and `go test ./...`.

## Coding Style & Naming Conventions
- Go-first repository: run `make fmt` before pushing. Keep packages small, lower-case names; exported identifiers use CamelCase. Prefer table-driven tests, early returns, and `%w` error wrapping.
- JS/Python adapters in `internal/adapters/` should remain minimal and cross-platform; avoid external deps and prefer Node 18+ compatible JS.

## Testing Guidelines
- Unit tests live beside code as `*_test.go`; integration tests in `tests/integration_go/`.
- Name tests `TestXxx` and organize with subtests where useful. Keep fixtures deterministic (no network).
- Run full suite: `make test-all`. For coverage locally: `make coverage`.

## Commit & Pull Request Guidelines
- Use Conventional Commits: `feat:`, `fix:`, `chore:`, etc. Example: `fix(runner): handle vitest pnpm watch mode`.
- PRs should include: clear description, linked issues, reproduction steps, and screenshots or snippets of CLI output when user-facing behavior changes. Update `README.md`/`docs/` when behavior or flags change.

## Security & Configuration Tips
- Never commit `.3pio/` outputs or secrets. Be cautious with paths logged to reports; sanitize when adding new writers.
- For releases/packaging, see `.goreleaser.yml`, `packaging/`, and `Makefile` targets.

## Agent-Specific Notes
- When running tests in this repo or downstream projects, prefer `3pio <your test command>`.
- Supported runners include:
  - JavaScript: Jest, Vitest, Mocha, Cypress
  - Python: pytest
  - Go: go test (native)
  - Rust: cargo test and cargo nextest (native)
- Consult `README.md` and `docs/` for runner-specific notes.
