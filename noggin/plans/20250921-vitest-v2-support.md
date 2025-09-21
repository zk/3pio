# Plan: Add Vitest v2 Support

## Summary
Extend the existing Vitest adapter (currently v3+) to also support Vitest v2, keeping v3 behavior unchanged. Detect the installed Vitest major version at runtime and switch adapter logic accordingly. Add fixtures and integration tests to validate both versions, and update docs/CI.

## Goals
- Support Vitest v2 and v3 with a single adapter file.
- Preserve current v3 behavior and output format (IPC + console summaries).
- Keep CLI injection stable: `--reporter <adapter> --reporter default`, auto-add `run` (no watch).
- Add clear tests (fixtures + integration) and CI coverage for v2.

## Non‑Goals
- Watch mode support (intentionally disabled by adding `run`).
- Feature parity beyond what 3pio consumes (we only emit IPC events needed for reports/console).

## Background
- Current adapter (`internal/adapters/vitest.js`) gates Vitest to major >= 3 and uses v3‑specific reporter hooks (e.g., `onTestRunStart`, `onTestCaseResult`, `onTestModuleEnd`).
- We need a compatibility path for v2 where the reporter API differs. We will branch behavior based on `require('vitest/package.json').version`.

## Approach
1) Version detection
   - Keep requiring `vitest/package.json` and parse `major`.
   - Change gate from `>=3` to `>=2` and warn for `<2`.
   - Set `this.vitestMajor` and branch all event handling.

2) Adapter compatibility layer
   - Maintain v3 path exactly as today.
   - Add v2 path implementing equivalent IPC emission using stable v2 hooks, e.g.:
     - Initialization/collection: `onInit`, `onPathsCollected`, `onCollected`.
     - File lifecycle: `onTestFileStart`, `onTestFileResult`.
     - Test results: prefer `onTaskUpdate`/task processing to emit `testCase` events with status mapping (PASS/FAIL/SKIP). Avoid duplicate emissions.
     - Run end: `onFinished`.
   - Keep group discovery/parent hierarchy consistent with v3 output (derive suite chain from task parent links when available). If missing, fall back to file‑only grouping.

3) Event mapping and deduplication
   - For v2: emit `testCase` once per test case. Do not also re‑walk the same tasks elsewhere.
   - Map states consistently: passed→PASS, failed→FAIL, skipped/todo→SKIP. Include error object (message/stack) when failed.

4) CLI injection remains unchanged
   - Continue to inject `--reporter <adapter> --reporter default`.
   - Force `run` subcommand when omitted to avoid watch mode, for both v2 and v3.

5) Output parsing
   - The Go Vitest parser is heuristic (checkmarks/markers). Validate it still buckets per‑file with v2. If v2 differs, minimally tune heuristics without expanding scope.

## Work Items
1. Relax version gate and add `this.vitestMajor` branching in adapter.
2. Implement v2 reporter handlers and test case emission (no duplicates).
3. Keep v3 path as‑is (zero behavior change).
4. Add fixture `tests/fixtures/basic-vitest-v2` with `vitest@^2` and small pass/fail tests.
5. Add integration tests mirroring existing v3 cases (failure summary line, report path checks).
6. Validate injected command for common package managers (npm, pnpm, yarn, bun) still correct for v2.
7. Adjust Vitest output parser heuristics only if needed.
8. Update README support matrix: “Vitest (v2–v3)”. Add short note in docs about watch mode.
9. Expand CI workflow to run v2 fixture on Linux/macOS (Windows if feasible alongside current runs).

## Risks & Mitigations
- Duplicate event emission (seen in prior iterations): Ensure only one code path emits `testCase` per test (v2 uses task updates OR final aggregation, not both).
- API drift between v2 minor versions: Prefer stable hooks; guard undefined fields defensively.
- Performance on large suites: Avoid expensive per‑event filesystem calls; batch only via IPC appends as today.

## Acceptance Criteria
- Running `3pio npx vitest run` with vitest@^2 produces:
  - Proper `.3pio` run directory with reports populated.
  - Console summary lines per failing file, including report path.
  - Exit codes mirror Vitest outcome.
- Existing v3 fixtures/tests remain green without changes.
- README updated, CI runs both v2 and v3 jobs.

## Follow‑ups (Optional)
- Add a richer v2 fixture (skipped/todo/parameterized) to validate edge states.
- Document any known discrepancies between Vitest native counts and 3pio counts if discovered.

