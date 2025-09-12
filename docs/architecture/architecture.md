# 3pio Architecture

## Overview

3pio is a context-friendly test runner adapter that translates traditional test runner output (Jest, Vitest, pytest) into structured, persistent reports optimized for AI agents and developers. This document provides a comprehensive view of the Go-based architecture.

## System Components

The system consists of six primary components:

### 1. CLI Entry Point (`cmd/3pio/main.go`)
- Parses command-line arguments
- Initializes file-based logger for debug output
- Creates and configures the Orchestrator
- Handles version and help commands
- Passes control to Orchestrator for test execution

### 2. Orchestrator (`internal/orchestrator/`)
The central controller managing the entire test execution lifecycle:
- Generates unique run IDs (format: `{timestamp}-{adjective}-{character}`, e.g., `20250911T194308-sneaky-yoda`)
- Detects test runner using Runner Manager
- Creates run directory structure (`.3pio/runs/[runID]/`)
- Initializes IPC and Report managers
- Extracts and prepares embedded adapters
- Spawns test process with adapter injection
- Captures stdout/stderr through pipes
- Processes IPC events concurrently
- Handles signals (SIGINT/SIGTERM) gracefully
- Mirrors test runner exit codes

### 3. Runner Manager (`internal/runner/`)
Manages test runner detection and configuration:
- Registry of supported test runners (Jest, Vitest, pytest)
- Detects runner from command arguments
- Parses package.json for npm/yarn/pnpm commands
- Builds modified commands with adapter injection
- Extracts test files from arguments
- Handles various invocation patterns

### 4. Report Manager (`internal/report/`)
Handles all report generation with incremental writing:
- Creates and manages run directory structure
- Processes IPC events to update test state
- Manages file handles for incremental log writing
- Writes test-run.md report with test case details
- Generates individual test log files per test file
- Implements debounced writes for performance
- Thread-safe state management with sync.RWMutex

### 5. IPC Manager (`internal/ipc/`)
Provides file-based communication between CLI and test adapters:
- Creates IPC directory and file structure
- Watches IPC file for new events using fsnotify
- Parses JSON Lines format events
- Validates event schema and types
- Provides Events channel for orchestrator consumption

### 6. Embedded Adapters (`internal/adapters/`)
JavaScript and Python reporters embedded in the Go binary:
- `jest.js`: Jest reporter implementation
- `vitest.js`: Vitest reporter implementation
- `pytest_adapter.py`: pytest plugin implementation
- Embedded at compile time using `//go:embed`
- Extracted to temporary directory at runtime
- Cleaned up after test completion

## Data Flow

### Execution Sequence

1. User executes `3pio npm test`
2. CLI parses arguments and creates Orchestrator
3. Orchestrator generates run ID (e.g., `20250911T120000Z-brave-luke`)
4. Runner Manager detects test runner
5. Orchestrator creates directory structure
6. Embedded adapters extracted to temp directory
7. Report Manager initialized with run metadata
8. IPC Manager starts watching for events
9. Orchestrator spawns test process with modified command
10. Test adapter sends events via IPC
11. IPC Manager reads and parses events
12. Report Manager processes events incrementally
13. Orchestrator displays progress to console
14. Process completes or is interrupted
15. Report Manager finalizes all files
16. Orchestrator exits with test runner's exit code

### Concurrent Processing

The system uses Go's concurrency features:
- **Main Process**: Coordinates all operations
- **IPC Event Processing** (goroutine): Reads events and updates report state
- **Stdout Capture** (goroutine): Pipes stdout to output.log and console
- **Stderr Capture** (goroutine): Pipes stderr to output.log and error buffer
- **Signal Handler** (goroutine): Handles SIGINT/SIGTERM for graceful shutdown

## IPC Protocol

### Event Types

All adapters communicate using JSON Lines format with these events:

#### testFileStart
```json
{
  "eventType": "testFileStart",
  "payload": {"filePath": "src/math.test.js"}
}
```

#### testCase
```json
{
  "eventType": "testCase",
  "payload": {
    "filePath": "src/math.test.js",
    "testName": "should add numbers",
    "suiteName": "Math operations",
    "status": "PASS",
    "duration": 0.023,
    "error": null
  }
}
```

#### testFileResult
```json
{
  "eventType": "testFileResult",
  "payload": {"filePath": "src/math.test.js", "status": "PASS"}
}
```

#### stdoutChunk / stderrChunk
```json
{
  "eventType": "stdoutChunk",
  "payload": {
    "filePath": "src/math.test.js",
    "chunk": "Console log output\n"
  }
}
```

#### collectionStart / collectionFinish
```json
{
  "eventType": "collectionStart",
  "payload": {}
}

{
  "eventType": "collectionFinish",
  "payload": {"collected": 67}
}
```
Note: Collection events provide immediate feedback during test discovery phase

## File Structure

### Runtime Directory Structure
```
.3pio/
├── runs/
│   └── [runID]/
│       ├── test-run.md           # Main report
│       ├── output.log             # Complete stdout/stderr
│       └── reports/              # Individual test reports
│           ├── math.test.js.log
│           └── string.test.js.log
├── ipc/
│   └── [runID].jsonl            # IPC communication
└── debug.log                      # Debug logging
```

### Source Code Structure
```
cmd/3pio/                # CLI entry point
internal/
├── orchestrator/        # Central controller
├── runner/             # Test runner management
├── report/             # Report generation
├── ipc/                # Inter-process communication
├── logger/             # Debug logging
└── adapters/           # Embedded test adapters
tests/                  # Integration tests and fixtures
```

## Key Design Decisions

### Embedded Adapters
- Compiled into binary for zero dependencies
- Extracted at runtime with IPC path injection
- Each run gets unique adapter instance

### File-Based IPC
- Simple, reliable cross-platform mechanism
- Human-readable JSON Lines format
- Easy to debug and inspect

### Incremental Writing
- Results available even if process killed
- Better reliability for long-running tests
- Memory-efficient buffering

### Go Implementation
- Single binary distribution
- Excellent cross-platform support
- Superior performance and concurrency
- Built-in goroutine primitives

## Performance Characteristics

### Concurrency
- Parallel processing of events and output
- Non-blocking IPC reading
- Concurrent file writes with mutex protection

### Memory Management
- Bounded buffers for output capture
- Incremental file writing (no full memory load)
- Stream processing for large outputs

### I/O Optimization
- Debounced report writes (100ms default)
- Buffered file operations
- Minimal file system calls

## Error Handling

### Startup Failures
- Test runner not found → Clear error message
- Permission issues → Fallback paths
- Missing dependencies → Helpful suggestions

### Runtime Failures
- Process crashes → Partial reports saved
- Signal interruption → Graceful shutdown
- IPC failures → Continue with degraded functionality

### Recovery Mechanisms
- Incremental writes ensure partial data preserved
- File handles properly closed on exit
- Exit codes accurately mirrored