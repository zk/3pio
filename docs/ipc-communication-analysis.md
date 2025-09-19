# IPC Communication Analysis: How Each Runner Handles Test Data

## Overview

3pio supports multiple test runners with two distinct approaches to IPC communication:

1. **Native Runners** (Go, Cargo) - Use file tailing to read output.log, then write to IPC file
2. **Adapter-Based Runners** (Jest, Vitest, Mocha, Cypress, pytest) - Use embedded adapters that write IPC events directly

**IMPORTANT**: ALL runners have their IPC events read through the SAME mechanism - the IPC Manager watches and tails the `ipc.jsonl` file using fsnotify for ALL runner types.

## Architecture Breakdown

### Native Runners (File Tailing Approach)

Native runners use the `TailReader` pattern to poll the `output.log` file:

#### Go Test Runner
- **Path**: `internal/runner/definitions/gotest.go`
- **Method**: `ProcessOutput(reader io.Reader, ipcPath string)`
- **Process**:
  1. Reads JSON output from `go test -json`
  2. Uses `TailReader` to poll `output.log` file
  3. Parses JSON events and writes to IPC file
  4. TailReader relies on `cargoProcessExited` channel to stop polling

#### Cargo Test Runner
- **Path**: `internal/runner/definitions/cargo.go`
- **Method**: `ProcessOutput(reader io.Reader, ipcPath string)`
- **Process**:
  1. Reads libtest JSON output from `cargo test --format=json`
  2. Uses `TailReader` to poll `output.log` file
  3. Parses JSON events and writes to IPC file
  4. TailReader relies on `cargoProcessExited` channel to stop polling

### Adapter-Based Runners (Direct IPC Writing)

Adapter-based runners DO NOT use file tailing. They use embedded JavaScript/Python adapters:

#### Jest Runner
- **Adapter**: `internal/adapters/jest.js` (embedded in binary)
- **Process**:
  1. Jest loads the custom reporter
  2. Reporter hooks into Jest's event system
  3. Reporter writes IPC events directly to the IPC file
  4. No file tailing involved - direct event streaming

#### Vitest Runner
- **Adapter**: `internal/adapters/vitest.js` (embedded in binary)
- **Process**:
  1. Vitest loads the custom reporter
  2. Reporter hooks into Vitest's event system
  3. Reporter writes IPC events directly to the IPC file
  4. No file tailing involved - direct event streaming

#### Mocha Runner
- **Adapter**: `internal/adapters/mocha.js` (embedded in binary)
- **Process**:
  1. Mocha loads the custom reporter
  2. Reporter hooks into Mocha runner events
  3. Reporter writes IPC events directly to the IPC file
  4. No file tailing involved - direct event streaming

#### Cypress Runner
- **Adapter**: `internal/adapters/cypress.js` (embedded in binary)
- **Process**:
  1. Cypress runs Mocha and loads our custom reporter
  2. Reporter hooks into Mocha runner events exposed by Cypress
  3. Reporter writes IPC events directly to the IPC file
  4. No file tailing involved - direct event streaming

#### pytest Runner
- **Adapter**: `internal/adapters/pytest_adapter.py` (embedded in binary)
- **Process**:
  1. pytest loads the custom plugin via `-p` flag
  2. Plugin hooks into pytest's hook system
  3. Plugin writes IPC events directly to the IPC file
  4. No file tailing involved - direct event streaming

## Key Code Paths

### Orchestrator Decision Logic

In `internal/orchestrator/orchestrator.go:436-443`:

```go
// Process output through native definition (no TeeReader needed)
if nd, ok := nativeDef.(interface {
    ProcessOutput(io.Reader, string) error
}); ok {
    o.logger.Debug("Processing output for native runner")
    if err := nd.ProcessOutput(fileReader, o.ipcPath); err != nil {
        o.logger.Error("Failed to process native output: %v", err)
    }
}
```

This code checks if the runner implements `ProcessOutput` (native runners only).

### TailReader Implementation

In `internal/orchestrator/orchestrator.go:64-87`:

The `TailReader` is ONLY created for native runners that need to poll output.log:

```go
type TailReader struct {
    file          io.ReadCloser
    processExited <-chan struct{}
    logger        *logger.FileLogger
}
```

## The SIGINT Bug Context

The SIGINT bug ONLY affected native runners because:

1. **Native runners** create a `TailReader` goroutine that polls `output.log`
2. This goroutine relies on `cargoProcessExited` channel being closed to terminate
3. Without closing the channel on SIGINT, the goroutine spins forever
4. **Adapter-based runners** never create this goroutine - they don't use TailReader

## Channel Usage Summary

The `cargoProcessExited` channel (misleadingly named):
- **Created**: For ALL runners in `orchestrator.go`
- **Used**: ONLY by native runners (Go, Cargo) via TailReader
- **Not Used**: By adapter-based runners (Jest, Vitest, Mocha, Cypress, pytest)
- **Bug Impact**: Only affects runners that actually use the channel

## Verification

To verify which approach a runner uses:

1. Check if the runner definition has a `ProcessOutput` method → Native runner
2. Check if the runner uses an adapter file → Adapter-based runner
3. Native runners appear in `internal/runner/definitions/` with `ProcessOutput`
4. Adapter-based runners have corresponding files in `internal/adapters/`

## Complete File Tailing Picture

There are TWO types of file tailing happening in 3pio:

### 1. TailReader for output.log (Native Runners Only)
- **Who uses it**: Only Go and Cargo runners
- **What it does**: 3pio reads its own `output.log` file to parse JSON test output
- **Where**: In orchestrator.go via `TailReader` struct
- **Why polling**: 3pio writes to output.log AND reads from it - can't use fsnotify on our own writes
- **Bug impact**: SIGINT bug affected this because goroutine relies on `cargoProcessExited` channel

### 2. IPC File Watching (ALL Runners)
- **Who uses it**: ALL runners (Go, Cargo, Jest, Vitest, Mocha, Cypress, pytest)
- **What it does**: Watches and tails `ipc.jsonl` using fsnotify
- **Where**: In `internal/ipc/manager.go` via `WatchEvents()` and `watchLoop()`
- **How it works**:
  - Uses fsnotify to watch for write events on ipc.jsonl
  - Reads new lines as they're appended
  - Parses JSON events and sends to Events channel
  - This goroutine properly terminates via `Cleanup()` method

## Conclusion

The complete picture:

1. **Native runners (Go, Cargo)**:
   - Use TailReader to poll `output.log` → parse JSON → write to `ipc.jsonl`
   - Then IPC Manager tails `ipc.jsonl` → sends events to report manager

2. **Adapter-based runners (Jest, Vitest, pytest)**:
   - Adapters write directly to `ipc.jsonl`
   - Then IPC Manager tails `ipc.jsonl` → sends events to report manager

So yes, ALL runners involve tailing the `ipc.jsonl` file, but only native runners also tail `output.log`. The SIGINT bug only affected the `output.log` tailing (TailReader), not the `ipc.jsonl` watching.
