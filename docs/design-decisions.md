# Design Decisions

This document tracks important design decisions made during the development of 3pio.

## Native Runner Detection (2025-09-14)

**Problem**: When running cargo test on the Alacritty project, the test report showed `detected_runner: go test` instead of `cargo test`, and the report status remained "RUNNING" with 0 test counts despite tests completing successfully.

**Root Cause**: In `orchestrator.go`, when determining the detected runner name for native runners (those without external adapters), the code was hardcoded to return "go test" for all native runners instead of checking which specific native runner was being used.

**Solution**: Modified the runner detection logic in `orchestrator.go` to use a type switch on the runner definition:
- `*definitions.GoTestWrapper` → "go test"
- `*definitions.CargoTestWrapper` → "cargo test"
- `*definitions.NextestWrapper` → "cargo nextest"

This ensures the correct runner name is recorded in reports and proper processing occurs for each native runner type.

**Impact**: This fix ensures that:
1. The correct runner name appears in test reports
2. Native runners are properly identified for debugging
3. Future native runners can be added with proper detection

## Report Architecture

**Decision**: Use incremental file writing with complete regeneration on each update.

**Rationale**:
- Ensures reports are available even if the test run is interrupted
- Provides consistent formatting across all report files
- Simplifies the implementation by avoiding complex buffering

## IPC Communication

**Decision**: Use file-based JSON Lines format for IPC between adapters and the CLI.

**Rationale**:
- Simple and robust - no complex IPC mechanisms needed
- Easy to debug - IPC files can be inspected directly
- Language agnostic - works with any language that can write JSON to a file

## Native vs Adapter-based Runners

**Decision**: Support both native runners (Go test, cargo test) and adapter-based runners (Jest, Vitest, pytest).

**Rationale**:
- Native runners can process JSON output directly without embedding adapters
- Adapter-based runners need JavaScript/Python code to hook into test frameworks
- This hybrid approach provides flexibility and efficiency

## Output File Race Condition Fix (2025-09-14)

**Problem**: When running cargo test, an intermittent race condition (~40% occurrence) caused the error: `[ERROR] Failed to process native output: error reading cargo test output: read |0: file already closed`

**Root Cause**: In `orchestrator.go`, the output file was being closed via a defer statement immediately after the command started, but goroutines were still actively reading from it via a TeeReader. This created a race between:
1. The defer closing the file when the function returned
2. The goroutine reading from the TeeReader that writes to that file

**Solution**: Moved the `outputFile.Close()` call from a defer statement (line 266) to after `wg.Wait()` completes (lines 370-373). This ensures all goroutines finish their work before the file is closed.

**Test Case**: Added `TestOutputFileRaceCondition` in `orchestrator_test.go` that reproduces the race condition scenario and verifies the fix.

**Impact**: This fix eliminates the intermittent failures when running native test runners (cargo test, go test) that process output directly without adapters.

## Test Count Display (2025-09-18)

**Decision**: Use recursive test counting for summary table display instead of direct group statistics.

**Rationale**: The summary table in `test-run.md` needs to show accurate test counts for files with nested test structures (Jest describe blocks, Vitest suites, etc.). Using recursive counting ensures that files containing only nested tests display meaningful counts rather than "0 tests".

**Implementation**: Modified table generation logic in `internal/report/manager.go` to calculate recursive counts on-the-fly using `countTotalTestCases()`, `countPassedTestCases()`, etc. functions.

**Impact**: Summary tables now accurately reflect all test cases regardless of nesting structure.

## Future Decisions

(This section will be updated as new design decisions are made)