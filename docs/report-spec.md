# 3pio Report Spec and Samples

This spec defines the canonical report artifacts produced by 3pio, the states they represent, and how to curate sample outputs for verification and documentation.

## Goals
- Provide a stable contract for report files and their frontmatter.
- Show how different states (pending, running, pass, fail, skip, error, xfail/xpass, no-tests) appear in reports.
- Offer a reproducible way to collect samples from fixtures across runners.
- Enable lightweight visual/manual regression checks without committing `.3pio/` runtime artifacts.

## Artifacts
- Root summary: `.3pio/runs/<run-id>/test-run.md`
- Per-group report: `.3pio/runs/<run-id>/<sanitized-path...>/index.md`
- Combined output log for the whole run: `.3pio/runs/<run-id>/output.log`
- Per-group output log: `.3pio/runs/<run-id>/<sanitized-path...>/output.log`

Note: Directory names use `report.SanitizeGroupName()`. All report files inside directories are named `index.md`. The only non-index report file is the root `test-run.md`.

## States

Run-level states (frontmatter `status` in `test-run.md`):
- PENDING, RUNNING, COMPLETED, ERRORED

Group/Test-level states (frontmatter `status` in `index.md` and inline rows):
- PENDING, RUNNING, PASS, FAIL, SKIP, NO_TESTS, ERROR, XFAIL, XPASS

Derivation rules are centralized in `docs/report-status-rules.md`. Highlights:
- PASS only when all children pass.
- SKIP only when everything under a group was skipped.
- Otherwise FAIL (mixtures or presence of fail/error).
- XFAIL/XPASS at test level; aggregate into group status per rules.

## File Formats

### Root summary: `test-run.md`
- YAML frontmatter keys:
  - `run_id: <string>` — basename of run directory
  - `run_path: <string>` — absolute path to run directory
  - `detected_runner: <string>` — e.g., vitest, jest, pytest, go test
  - `modified_command: <string>` — effective command 3pio executed
  - `created: <ISO-8601>` — run start timestamp (UTC)
  - `updated: <ISO-8601>` — last update timestamp (UTC)
  - `status: PENDING|RUNNING|COMPLETED|ERRORED`
- Body sections:
  - H1: “3pio Test Run”
  - Test command and pointer to `./output.log`
  - If `status: ERRORED`, an “Error” code block with details
  - Hierarchical group summary table using root groups

### Group report: `index.md`
- YAML frontmatter keys:
  - `group_name: <string>`
  - `parent_path: <a/b/c or empty>` — slash-joined `ParentNames`
  - `status: PENDING|RUNNING|PASS|FAIL|SKIP|ERROR|XFAIL|XPASS|NO_TESTS`
  - `duration: <seconds>s` — optional if known
  - `created: <ISO-8601>`
  - `updated: <ISO-8601>`
- Body sections:
  - H1: “Test Report: <path or name>” (shows full path for nested groups)
  - Summary bullets: direct test stats and/or subgroup counts
  - “Test case results” list (when direct tests exist)
  - “Subgroups” table with status, name, test breakdown, duration, and link to each subgroup’s report
  - “stdout/stderr” combined code block if any output captured

## Sample Matrix

Curate samples that demonstrate representative states for each runner. Suggested baseline set:
- PASS: All tests pass (basic fixtures)
- FAIL: At least one failure (many-failures or failing-mocha)
- SKIP: Entire group skipped (empty fixtures)
- ERROR: Setup/teardown failure or configuration error (jest-config-error, jest-ts-config-error)
- XFAIL/XPASS: Pytest `xfail` cases (pytest-xfail)
- RUNNING/PENDING: Mid-run snapshot (use `--maxWorkers=1`/`-j1` and capture during execution)

Map to fixtures in `tests/fixtures/`:
- Jest: `basic-jest`, `empty-jest`, `jest-config-error`
- Vitest: `basic-vitest`, `empty-vitest`, `long-names-vitest`
- Pytest: `basic-pytest`, `empty-pytest`, `pytest-xfail`
- Mocha/Cypress: `basic-mocha`, `failing-mocha`, `basic-cypress`
- Go: `basic-go`

## Collecting Samples

We keep curated samples under `docs/report-samples/` (never commit `.3pio/`). Each scenario contains:
- `test-run.md` — copied from the run directory
- Selected group `index.md` files that illustrate the targeted state(s)
- `manifest.yaml` — metadata for reproducibility (runner, fixture, command, date, notes)

Use `scripts/generate-report-samples.sh` to:
- Build `3pio` if needed
- Run selected fixtures via `3pio <test command>`
- Locate the latest `.3pio/runs/<run-id>`
- Copy `test-run.md` and relevant `index.md` files into `docs/report-samples/<scenario>/`
- Create or update `manifest.yaml`

## Curation Guidelines
- Keep samples small and focused; include only a few representative group reports per scenario.
- Redact absolute paths in explanatory text if necessary; frontmatter `run_path` remains verbatim for fidelity.
- Prefer deterministic fixtures (no network, no time-based flakiness).
- When demonstrating RUNNING/PENDING, clearly note timing and environment in the manifest.

## Versioning
- This spec describes the report as of this repository’s commit date.
- Any user‑visible change to frontmatter keys or report body sections should update this document and affected fixtures.
- Backward‑compatible additions (new optional fields) are allowed; removals/renames must be documented.

## Open Questions
- Do we want a JSON export alongside Markdown for machine consumption? If so, define a parallel JSON schema mirroring frontmatter fields and per‑section summaries.
- For very large suites, should root `test-run.md` paginate/group beyond current table? If yes, specify rules here.

## How To Propose Changes
- Open a PR that edits this file and updates samples under `docs/report-samples/`.
- Include before/after snippets of the relevant `index.md`/`test-run.md` blocks.
- Link to `docs/report-status-rules.md` if status logic changes.

