# 3pio: System Architecture

## 1. Introduction

3pio is a context-friendly test runner adapter that translates traditional test runner output (Jest, Vitest, pytest) into structured, persistent reports optimized for AI agents and developers. This document describes the Go-based architecture that prioritizes reliability, performance, and minimal context overhead.

## 2. High-Level Architecture

The system consists of five primary components implemented in Go:

1. **CLI Entry Point** (`cmd/3pio/main.go`) - Command-line interface and initialization
2. **Orchestrator** (`internal/orchestrator/`) - Central controller managing test execution lifecycle
3. **Runner Manager** (`internal/runner/`) - Test runner detection and command building
4. **Report Manager** (`internal/report/`) - Incremental report generation and file I/O
5. **IPC Manager** (`internal/ipc/`) - File-based communication with test adapters
6. **Embedded Adapters** (`internal/adapters/`) - JavaScript and Python reporters embedded in binary

## 3. Component Architecture

### 3.1. CLI Entry Point (`cmd/3pio/main.go`)

The main entry point that handles command-line parsing and orchestrator initialization.

**Responsibilities:**
- Parse command-line arguments using Go's flag package
- Initialize file-based logger for debug output
- Create and configure the Orchestrator
- Handle version and help commands
- Pass control to Orchestrator for test execution

### 3.2. Orchestrator (`internal/orchestrator/orchestrator.go`)

The central controller that manages the entire test execution lifecycle.

**Responsibilities:**
- Generate unique run IDs (timestamp + memorable Star Wars names)
- Detect test runner using Runner Manager
- Create run directory structure (`.3pio/runs/[runID]/`)
- Initialize IPC and Report managers
- Extract and prepare embedded adapters
- Spawn test process with adapter injection
- Capture stdout/stderr through pipes
- Process IPC events concurrently
- Handle signals (SIGINT/SIGTERM) gracefully
- Mirror test runner exit codes

**Key Features:**
- Concurrent goroutines for event processing and output capture
- Real-time console output display with progress tracking
- Signal handling for graceful shutdown
- Comprehensive error capture and reporting

### 3.3. Runner Manager (`internal/runner/manager.go`)

Manages test runner detection and configuration through a registry pattern.

**Components:**
- **Manager**: Registry of supported test runners
- **Definition Interface**: Contract for test runner implementations
- **Implementations**: JestDefinition, VitestDefinition, PytestDefinition

**Responsibilities:**
- Detect test runner from command arguments
- Parse package.json for npm/yarn/pnpm commands
- Build modified commands with adapter injection
- Extract test files from arguments
- Handle various invocation patterns (direct, npm scripts, npx)

### 3.4. Report Manager (`internal/report/manager.go`)

Handles all report generation with incremental writing for reliability.

**Responsibilities:**
- Create and manage run directory structure
- Initialize test run state with metadata
- Process IPC events to update test state
- Manage file handles for incremental log writing
- Write test-run.md report with test case details
- Generate individual test log files
- Implement debounced writes for performance
- Ensure proper cleanup and finalization

**Key Features:**
- Incremental writing (results available even if interrupted)
- Per-file log separation with headers
- Test case level tracking with suite organization
- Memory-efficient buffering
- Thread-safe state management with sync.RWMutex

### 3.5. IPC Manager (`internal/ipc/manager.go`)

Provides file-based communication between the CLI and test adapters.

**Responsibilities:**
- Create IPC directory and file structure
- Watch IPC file for new events using fsnotify
- Parse JSON Lines format events
- Validate event schema and types
- Provide Events channel for orchestrator consumption
- Handle cleanup and resource management

**Event Types:**
- `testFileStart`: Test file execution beginning
- `testCase`: Individual test case result
- `testFileResult`: Test file completion
- `stdoutChunk`: Console output from test
- `stderrChunk`: Error output from test

### 3.6. Embedded Adapters (`internal/adapters/`)

JavaScript and Python reporters embedded directly in the Go binary.

**Structure:**
- `jest.js`: Jest reporter implementation
- `vitest.js`: Vitest reporter implementation
- `pytest_adapter.py`: pytest plugin implementation
- `embedded.go`: Go embed directives and extraction logic

**Embedding Process:**
- Adapters embedded at compile time using `//go:embed`
- Extracted to temporary directory at runtime
- Cleaned up after test completion
- Paths passed to test runners via command modification

**Adapter Responsibilities:**
- Silent operation (no console output)
- Capture test events and results
- Send structured events via IPC
- Patch stdout/stderr for output capture
- Handle test lifecycle hooks

## 4. Data Flow

### Execution Sequence

1. **User** executes `3pio npm test`
2. **CLI** parses arguments and creates Orchestrator
3. **Orchestrator** generates run ID (e.g., `20250911T120000Z-brave-luke`)
4. **Runner Manager** detects test runner (Jest/Vitest/pytest)
5. **Orchestrator** creates directories:
   - `.3pio/runs/[runID]/`
   - `.3pio/runs/[runID]/logs/`
   - `.3pio/ipc/`
6. **Embedded Adapters** extracted to temp directory
7. **Report Manager** initialized with run metadata
8. **IPC Manager** starts watching for events
9. **Orchestrator** spawns test process with:
   - Modified command including adapter
   - THREEPIO_IPC_PATH environment variable
   - Pipes for stdout/stderr capture
10. **Test Adapter** initializes and sends events:
    - `testFileStart` when file begins
    - `testCase` for each test result
    - `stdoutChunk`/`stderrChunk` for output
    - `testFileResult` when file completes
11. **IPC Manager** reads events from file
12. **Report Manager** processes events:
    - Updates in-memory state
    - Writes to individual log files incrementally
    - Debounces report updates
13. **Orchestrator** displays progress to console
14. **Process** completes or is interrupted
15. **Report Manager** finalizes:
    - Flushes all buffers
    - Writes final test-run.md
    - Closes file handles
16. **Orchestrator** exits with test runner's exit code

### Concurrent Processing

The system uses Go's concurrency features for efficient processing:

- **Main Process**: Coordinates all operations
- **IPC Event Processing** (goroutine): Reads events and updates report state
- **Stdout Capture** (goroutine): Pipes stdout to output.log and console
- **Stderr Capture** (goroutine): Pipes stderr to output.log and error buffer
- **Signal Handler** (goroutine): Handles SIGINT/SIGTERM for graceful shutdown

All goroutines communicate through channels and are properly synchronized during shutdown.

## 5. File Structure

### Runtime Directory Structure

The system creates the following directory structure during execution:

- **.3pio/runs/[timestamp]-[name]/**: Run-specific directory
  - **test-run.md**: Main report file
  - **output.log**: Complete stdout/stderr capture
  - **logs/**: Directory containing individual test file logs
- **.3pio/ipc/**: IPC communication directory
  - **[timestamp].jsonl**: JSON Lines format event file
- **.3pio/debug.log**: Debug logging output

### Source Code Structure

The codebase follows standard Go project layout:

- **cmd/3pio/**: CLI entry point
- **internal/orchestrator/**: Central controller component
- **internal/runner/**: Test runner detection and management
- **internal/report/**: Report generation and file I/O
- **internal/ipc/**: Inter-process communication
- **internal/logger/**: File-based debug logging
- **internal/adapters/**: Embedded test runner adapters
- **tests/**: Integration tests and fixtures
- **Makefile**: Build automation scripts

## 6. Build and Deployment

### Build Process
1. `make adapters` prepares JavaScript/Python adapters
2. `go:embed` directives include adapters in binary
3. `go build` creates single executable
4. `goreleaser` handles cross-platform builds

### Binary Distribution
- Single static binary with embedded adapters
- No runtime dependencies required
- Cross-platform: macOS (arm64/amd64), Linux, Windows
- Distributed via npm, pip, and Homebrew

## 7. Key Design Decisions

### Embedded Adapters
- Adapters compiled into binary for zero dependencies
- Extracted to temp directory at runtime
- Ensures version compatibility

### File-Based IPC
- Simple, reliable communication mechanism
- Works across all platforms
- Easy to debug and inspect

### Incremental Writing
- Results available even if process killed
- Better reliability for long-running tests
- Immediate feedback for users

### Go Implementation
- Single binary distribution
- Excellent cross-platform support
- Superior performance and memory efficiency
- Built-in concurrency primitives

## 8. Error Handling

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

## 9. Performance Characteristics

### Concurrency
- Parallel processing of events and output
- Non-blocking IPC reading
- Concurrent file writes with mutex protection

### Memory Management
- Bounded buffers for output capture
- Incremental file writing (no full memory load)
- Efficient string handling

### I/O Optimization
- Debounced report writes
- Buffered file operations
- Minimal file system calls

## 10. Future Considerations

- WebSocket-based IPC for real-time monitoring
- Distributed test execution support
- Custom output formatters
- Test result caching and comparison
- Integration with CI/CD platforms