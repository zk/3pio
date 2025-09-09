# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

3pio is an AI-first test runner adapter that acts as a "protocol droid" for test frameworks. It translates traditional test runner output (Jest, Vitest) into a format optimized for AI agents - providing persistent, structured, file-based records that are context-efficient and searchable.

## Key Architecture Components

### Core Components
1. **CLI Orchestrator** (`src/cli.ts`) - Main entry point, manages test lifecycle
2. **Report Manager** (`src/ReportManager.ts`) - Handles all report file I/O with debounced writes
3. **IPC Manager** (`src/ipc.ts`) - File-based communication between adapters and CLI
4. **Test Runner Adapters** (`src/adapters/`) - Silent reporters running inside test processes

### Data Flow
- CLI performs dry run → creates run directory → spawns test runner with adapter → adapter sends test events via IPC → CLI captures all stdout/stderr at process level → Report Manager writes structured logs → final report at `.3pio/runs/[timestamp]/test-run.md`

### Console Output Capture Strategy
- **Important**: 3pio does NOT use Jest's default reporter to avoid duplicate output
- All console output from tests is captured at the CLI process level by monitoring stdout/stderr streams
- Jest runs tests in worker processes, so the reporter cannot directly capture console output
- The captured output is stored in `.3pio/runs/*/output.log` as a complete record
- Individual test log files may be empty if the output parser cannot attribute console logs to specific test files

## Development Commands

### Build
```bash
npm run build  # Use esbuild to compile TypeScript
```

### Test
```bash
# Run all tests
npm test

# Test specific adapter
npm test -- src/adapters/jest.test.ts
npm test -- src/adapters/vitest.test.ts

# Test CLI orchestrator
npm test -- src/cli.test.ts
```

### Development
```bash
npm run dev    # Watch mode for development
npm run lint   # Run linter
npm run typecheck  # Type checking with tsc
```

### Local Testing
```bash
# Link package locally for testing
npm link

# Test with sample projects
3pio run jest
3pio run vitest
3pio run npm test
```

## Implementation Guidelines

### IPC Event Schema
Events written to `.3pio/ipc/[timestamp].jsonl`:
- `stdoutChunk`: `{ eventType: "stdoutChunk", payload: { filePath, chunk } }`
- `stderrChunk`: `{ eventType: "stderrChunk", payload: { filePath, chunk } }`
- `testFileResult`: `{ eventType: "testFileResult", payload: { filePath, status: "PASS"|"FAIL" } }`

### Adapter Development
- Adapters must be **silent** - no stdout/stderr output
- Read `THREEPIO_IPC_PATH` environment variable for IPC file location
- Patch `process.stdout.write` and `process.stderr.write` during test execution
- Restore original functions after test completion

### Error Handling
- Mirror exit codes from underlying test runners
- No report generation if startup fails (before test runner starts)
- All startup failures should exit with non-zero code and clear error to stderr

### File Structure Conventions
- Reports: `.3pio/runs/[ISO8601_timestamp]/`
- IPC files: `.3pio/ipc/[timestamp].jsonl`
- Output log: `.3pio/runs/[timestamp]/output.log` (contains all stdout/stderr from test run)

## Testing Requirements

### Unit Tests Required For
- Argument parsing logic (CLI)
- Test runner detection from package.json
- Command modification for adapter injection
- IPC event serialization/deserialization
- Report state management and debounced writes

### Integration Tests Required For
- Full CLI flow with mock components
- Adapter lifecycle hooks with real test runners
- IPC file watching and event handling
- Report generation accuracy

### End-to-End Tests Required For
- Complete runs against sample Jest/Vitest projects
- Correct preamble generation
- Accurate report file generation
- Exit code mirroring

## Technical Stack
- **Language**: TypeScript
- **Runtime**: Node.js
- **Build**: esbuild (for speed)
- **CLI Framework**: commander.js
- **Shell Execution**: zx (for robust command handling)
- **File Watching**: chokidar (for IPC monitoring)
- **Debouncing**: lodash.debounce (for report writes)

## Known Issues and Gotchas

### Test Runner Detection
- Commands invoked via `npx`, `yarn`, or `pnpm` require special handling to detect the actual test runner
- The system checks both the package manager and the subsequent test runner argument

### Vitest-Specific Behaviors
- **Important**: `vitest list` doesn't just list files - it runs tests in watch mode
- When specific test files are provided as arguments, they are extracted directly rather than using dry run
- Duplicate output may appear (from both default reporter and 3pio adapter) - this is expected behavior

### Environment Variables
- `THREEPIO_IPC_PATH` must be explicitly passed to child processes
- Adapter paths must use absolute paths to avoid resolution issues

For detailed information about these issues and their solutions, see `docs/known-issues.md`.
- Never use emojis in output
- System debug logging is avaialable at `.3pio/debug.log`



## Misc

- There are sample projects for jest and vitest at `sample-projects/`
- Generated test files and scripts should not be put in the root directory. Any temporary files should go in the `./scratch` directory.
