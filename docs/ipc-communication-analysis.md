# IPC Communication Analysis

## Overview

3pio uses two approaches for IPC communication:

1. **Native Runners** (Go, Cargo) - Use file tailing to read output.log, then write to IPC
2. **Adapter-Based Runners** (Jest, Vitest, Mocha, Cypress, pytest) - Embedded adapters write IPC events directly

ALL runners have their IPC events read through the IPC Manager, which watches and tails the `ipc.jsonl` file using fsnotify.

## Native Runners (File Tailing)

### Go Test
- Uses `TailReader` to poll `output.log`
- Parses `go test -json` output
- Writes parsed events to IPC file

### Cargo Test
- Uses `TailReader` to poll `output.log`
- Parses libtest JSON format
- Writes parsed events to IPC file

## Adapter-Based Runners (Direct Writing)

### JavaScript Runners (Jest, Vitest, Mocha, Cypress)
- Load embedded JS adapter as custom reporter
- Hook into test runner's event system
- Write IPC events directly to file

### Python Runner (pytest)
- Loads embedded Python plugin via `-p` flag
- Hooks into pytest's plugin system
- Writes IPC events directly to file

## Key Components

### TailReader
- **Used by**: Native runners only
- **Purpose**: Poll output.log for test results
- **Location**: `internal/orchestrator/orchestrator.go`
- **Termination**: Relies on `processExited` channel

### IPC Manager
- **Used by**: ALL runners
- **Purpose**: Watch and tail ipc.jsonl
- **Location**: `internal/ipc/manager.go`
- **Method**: fsnotify for file watching

## File Tailing Summary

Two types of file tailing:

1. **output.log tailing** (Native runners only)
   - 3pio reads its own output file
   - Uses polling (can't use fsnotify on own writes)
   - Affected by SIGINT bug if channel not closed

2. **ipc.jsonl watching** (ALL runners)
   - Uses fsnotify for efficient watching
   - Reads appended lines and parses events
   - Properly terminates via Cleanup()

## Data Flow

**Native runners**:
Test output → output.log → TailReader → ipc.jsonl → IPC Manager → Reports

**Adapter-based runners**:
Test events → Adapter → ipc.jsonl → IPC Manager → Reports

## Verification

To identify runner type:
- Has `ProcessOutput` method → Native runner
- Has adapter file in `internal/adapters/` → Adapter-based runner