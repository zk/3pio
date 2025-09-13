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
Handles all report generation with hierarchical group-based organization:
- Creates and manages run directory structure
- Delegates test state management to Group Manager
- Manages hierarchical group reports (no longer file-centric)
- Writes test-run.md report with group hierarchy
- Generates group-based reports with nested structure
- Implements debounced writes for performance
- Thread-safe state management with sync.RWMutex

#### Group Manager (`internal/report/group_manager.go`)
Manages test hierarchy using universal group abstractions:
- Processes group discovery, start, and result events
- Maintains hierarchical group tree structure
- Generates deterministic IDs from group paths using SHA256
- Creates filesystem-safe paths for group reports
- Propagates status from children to parent groups
- Handles arbitrary nesting depth (files, describes, suites, classes)
- Supports all test runners with unified abstraction

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

1. User executes `3pio npm test` or `3pio go test`
2. CLI parses arguments and creates Orchestrator
3. Orchestrator generates run ID (e.g., `20250911T120000Z-brave-luke`)
4. Runner Manager detects test runner
5. Orchestrator creates directory structure
6. For adapter-based runners (Jest/Vitest/pytest):
   - Embedded adapters extracted to temp directory
   - Command modified to include adapter
7. For native runners (Go test):
   - Command modified to add `-json` flag
   - No adapter extraction needed
8. Report Manager initialized with Group Manager
9. IPC Manager starts watching for events
10. Orchestrator spawns test process with modified command
11. For adapter-based runners:
    - Test adapter discovers group hierarchy
    - Sends testGroupDiscovered events for all groups
    - Sends testGroupStart when groups begin execution
    - Sends testCase events with parent hierarchy
    - Sends testGroupResult when groups complete
12. For native runners:
    - Orchestrator processes JSON output directly
    - Generates group events from package/test structure
13. IPC Manager reads and parses events
14. Group Manager processes events:
    - Creates groups on-demand from discovery events
    - Builds hierarchical tree structure
    - Generates deterministic IDs from paths
15. Report Manager generates group reports incrementally
16. Orchestrator displays hierarchical progress to console
17. Process completes or is interrupted
18. Group Manager finalizes all group reports
19. Orchestrator exits with test runner's exit code

### Concurrent Processing

The system uses Go's concurrency features:
- **Main Process**: Coordinates all operations
- **IPC Event Processing** (goroutine): Reads events and updates report state
- **Stdout Capture** (goroutine): Pipes stdout to output.log and console
- **Stderr Capture** (goroutine): Pipes stderr to output.log and error buffer
- **Signal Handler** (goroutine): Handles SIGINT/SIGTERM for graceful shutdown

## IPC Protocol

### Event Types

All adapters communicate using JSON Lines format with group-based events:

#### testGroupDiscovered
```json
{
  "eventType": "testGroupDiscovered",
  "payload": {
    "groupName": "Button Component",
    "parentNames": ["src/components/Button.test.js", "UI Components"]
  }
}
```

#### testGroupStart
```json
{
  "eventType": "testGroupStart",
  "payload": {
    "groupName": "Math operations",
    "parentNames": ["src/math.test.js"]
  }
}
```

#### testCase
```json
{
  "eventType": "testCase",
  "payload": {
    "testName": "should add numbers",
    "parentNames": ["src/math.test.js", "Math operations"],
    "status": "PASS",
    "duration": 0.023,
    "error": null,
    "stdout": "optional captured stdout",
    "stderr": "optional captured stderr"
  }
}
```

#### testGroupResult
```json
{
  "eventType": "testGroupResult",
  "payload": {
    "groupName": "Math operations",
    "parentNames": ["src/math.test.js"],
    "status": "PASS",
    "duration": 1.23,
    "totals": {
      "passed": 10,
      "failed": 2,
      "skipped": 1
    }
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
│       ├── test-run.md                         # Main report with group hierarchy
│       ├── output.log                          # Complete stdout/stderr
│       └── reports/                            # Hierarchical group reports
│           ├── src_components_button_test_js/  # File group directory
│           │   ├── index.md                    # File-level tests
│           │   ├── button_rendering.md         # Describe block tests
│           │   └── button_rendering/           # Nested describe
│           │       ├── with_props.md          # Nested tests
│           │       └── without_props.md       # Nested tests
│           └── test_math_py/                   # Python file group
│               ├── index.md                    # Module-level tests
│               └── testmathoperations.md       # Class-based tests
├── ipc/
│   └── [runID].jsonl                          # IPC communication
└── debug.log                                   # Debug logging
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

### Universal Group Abstractions
- Hierarchical model supports all test runners
- Groups replace file-centric organization
- Arbitrary nesting depth (files → describes → suites)
- Deterministic ID generation using SHA256
- Filesystem-safe path generation with sanitization

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