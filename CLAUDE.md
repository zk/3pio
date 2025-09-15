# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

3pio is a context friendly test runner. It translates traditional test runner output (Jest, Vitest, pytest) into a format optimized for coding agents - providing persistent, structured, file-based records that are context-efficient and searchable.

## Key Architecture Components

### Core Components (Go Implementation)
1. **CLI Orchestrator** (`cmd/3pio/main.go`) - Main entry point, manages test lifecycle
2. **Report Manager** (`internal/report/`) - Handles all report file I/O with incremental writing
3. **IPC Manager** (`internal/ipc/`) - File-based communication between adapters and CLI
4. **Test Runner Adapters** (`internal/adapters/`) - JavaScript/Python reporters embedded in the binary
5. **Process Manager** (`internal/process/`) - Spawns and monitors test runner processes
6. **Output Parser** (`internal/output/`) - Parses stdout/stderr streams

### Data Flow
- CLI attempts dry run (optional) → creates run directory → spawns test runner with adapter → adapter sends test events via IPC → CLI captures all stdout/stderr at process level → Report Manager generates unified test reports → final report at `.3pio/runs/[timestamp]-[memorable-name]/test-run.md`

### Unified Report Generation
- Individual test reports use a single `generateIndividualFileReport()` function for consistent formatting
- Reports are regenerated completely when test state changes (no incremental buffering)
- All reports use the same clean format with YAML frontmatter and structured test case listings
- Consistent format across all phases: file start, test execution, and completion
- Partial results are available even if the test run is interrupted (e.g., Ctrl+C)

### Console Output Capture Strategy
- **Jest**: 3pio does NOT use Jest's default reporter to avoid duplicate output
- **Vitest**: 3pio DOES include Vitest's default reporter for better user experience
- **pytest**: Uses custom plugin that integrates with pytest's hook system
- All console output from tests is captured at the CLI process level by monitoring stdout/stderr streams
- The captured output is stored in `.3pio/runs/*/output.log` as a complete record
- Individual test log files contain headers with file path and timestamp information

### Error Display Strategy
- **Configuration errors**: Displayed immediately to console for user visibility (e.g., missing presets, syntax errors)
- **Test failures**: Shown in summary format, full details in reports
- **Command errors**: Both displayed to console and captured in reports
- Errors are detected by checking exit codes and parsing initial output for error indicators
- This ensures users see critical errors immediately while still maintaining complete logs

## Development Commands

### Build
```bash
# Build the Go binary
make build

# Build with all adapters embedded
make adapters && make build

# Build all platform binaries
goreleaser build --snapshot --clean
```

### Test
```bash
# Run all tests (unit + integration)
make test

# Run Go tests directly
go test ./...

# Run integration tests only
go test ./tests/integration_go

# Test with fixtures
cd tests/fixtures/basic-jest && ../../../build/3pio npx jest
cd tests/fixtures/basic-vitest && ../../../build/3pio npx vitest run
```

### Development
```bash
# Build and install locally
make build
./build/3pio --version

# Run with debug logs written to .3pio/debug.log
./build/3pio npm test
```

### Local Testing
```bash
# Test with sample projects
./build/3pio npx jest
./build/3pio npx vitest run
./build/3pio pytest
./build/3pio npm test
```

## Implementation Guidelines

### IPC Event Schema
Events written to `.3pio/ipc/[timestamp].jsonl`:
- `testGroupDiscovered`: `{ eventType: "testGroupDiscovered", payload: { groupName, parentNames } }`
- `testGroupStart`: `{ eventType: "testGroupStart", payload: { groupName, parentNames } }`
- `testCase`: `{ eventType: "testCase", payload: { testName, parentNames, status: "PASS"|"FAIL"|"SKIP", duration?, error? } }`
- `testGroupResult`: `{ eventType: "testGroupResult", payload: { groupName, parentNames, status, duration?, totals? } }`
- `groupStdout`: `{ eventType: "groupStdout", payload: { groupName, parentNames, chunk } }`
- `groupStderr`: `{ eventType: "groupStderr", payload: { groupName, parentNames, chunk } }`

### Adapter Development
- Adapters must be **silent** - no stdout/stderr output
- Read `THREEPIO_IPC_PATH` environment variable for IPC file location
- JavaScript adapters patch `process.stdout.write` and `process.stderr.write` during test execution
- Python adapter uses pytest hooks to capture output
- Adapters are embedded in the Go binary using `embed` package

### Error Handling
- Mirror exit codes from underlying test runners
- No report generation if startup fails (before test runner starts)
- All startup failures should exit with non-zero code and clear error to stderr

### File Structure Conventions
- Reports: `.3pio/runs/[ISO8601_timestamp]-[memorable-name]/`
- IPC files: `.3pio/ipc/[timestamp].jsonl`
- Output log: `.3pio/runs/[timestamp]-[name]/output.log` (contains all stdout/stderr from test run)
- Test logs: `.3pio/runs/[timestamp]-[name]/logs/[sanitized-test-file].log` (per-file output with test case boundaries)

## Testing Requirements

### Unit Tests Required For
- Argument parsing logic (CLI)
- Test runner detection
- Command modification for adapter injection
- IPC event serialization/deserialization
- Report state management

### Integration Tests Required For
- Full CLI flow with real test runners
- Adapter lifecycle hooks
- IPC file watching and event handling
- Report generation accuracy
- Process management and signal handling

### End-to-End Tests Required For
- Complete runs against fixture projects (Jest/Vitest/pytest)
- Correct preamble generation
- Accurate report file generation
- Exit code mirroring
- Interrupt handling (SIGINT/SIGTERM)

## Technical Stack
- **Language**: Go
- **Build**: Go compiler with embedded resources
- **Adapters**: JavaScript (Jest/Vitest) and Python (pytest)
- **IPC**: File-based JSON Lines format
- **File Watching**: fsnotify (Go package)
- **Testing**: Go testing package + integration fixtures

## Known Issues and Gotchas

### Test Runner Detection
- Commands invoked via `npx`, `yarn`, or `pnpm` require special handling to detect the actual test runner
- The system checks both the package manager and the subsequent test runner argument

### Vitest-Specific Behaviors
- **Important**: `vitest list` doesn't just list files - it runs tests in watch mode
- When specific test files are provided as arguments, they are extracted directly rather than using dry run
- Vitest runs WITH the default reporter included for better user visibility

### Environment Variables
- `THREEPIO_IPC_PATH` must be explicitly passed to child processes
- Adapter paths must use absolute paths to avoid resolution issues

For detailed information about these issues and their solutions, see `docs/known-issues.md`.
- Never use emojis in output
- All debug and info logging is written to `.3pio/debug.log`
- Each test run session is clearly marked with timestamps in the debug log
- Only critical errors that require user attention are displayed to stderr
- Parsing errors and internal issues are logged to debug.log only, not console output



## Misc

- Test fixtures for Jest, Vitest, and pytest are at `tests/fixtures/`
- Generated test files and scripts should not be put in the root directory. Any temporary files should go in the `./scratch` directory.
- When we make design decisions update `docs/design-decisions.md` noting the decision and rationale.
- Adapters are prepared for embedding using `make adapters` which runs `scripts/prepare-adapters.sh`
- When asked to update documentation find a place that fits in `docs/`. If no good file exists for the information create it.
- Always use the logger class when logging. See `internal/logger/file_logger.go`.
- After making changes lint and `gofmt`.
- Before opening a pr, lint and `gofmt`.
