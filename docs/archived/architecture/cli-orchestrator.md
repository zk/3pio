# Component Design: Orchestrator

## 1. Core Purpose

The Orchestrator is the central controller of 3pio, managing the entire test execution lifecycle from command parsing to report finalization. Implemented in Go, it coordinates between test runners, adapters, and report generation while providing real-time feedback to users.

## 2. Key Responsibilities

### Command Processing
- Receive parsed arguments from CLI entry point
- Detect test runner using Runner Manager
- Support Jest, Vitest, and pytest across various invocation patterns

### Infrastructure Setup
- Generate unique run IDs with timestamps and memorable names
- Create directory structure for reports and IPC
- Extract embedded adapters to temporary locations
- Initialize Report and IPC managers

### Process Management
- Spawn test processes with modified commands
- Inject adapters through command-line arguments
- Set THREEPIO_IPC_PATH environment variable
- Capture stdout/stderr through pipes
- Handle interactive input via stdin passthrough

### Concurrent Operations
- Process IPC events in dedicated goroutine
- Capture output streams in parallel goroutines
- Handle signals (SIGINT/SIGTERM) gracefully
- Coordinate shutdown across all components

### Output and Reporting
- Display real-time progress to console
- Track file completion statistics
- Generate memorable failure messages
- Mirror test runner exit codes accurately

## 3. Implementation Details

### Structure

The Orchestrator maintains the following state:

- **Manager References**: Pointers to Runner, Report, and IPC managers
- **Logger**: Interface for debug logging
- **Run Metadata**: Run ID, directory path, IPC path, command arguments
- **Exit Code**: Tracks process exit status
- **Console Output State**: Start time, file counters (passed/failed/total), displayed files tracking
- **Error Capture**: String builder for stderr content

### Run Sequence

1. **Generate Run ID**
   - ISO8601 timestamp (e.g., `20250911T120000Z`)
   - Memorable suffix (e.g., `brave-luke`)
   - Combined format: `[timestamp]-[adjective]-[character]`

2. **Print Greeting**
   - Display welcome message
   - Show the test command being executed
   - Print full report path
   - Indicate test execution is beginning

3. **Detect Test Runner**
   - Use Runner Manager to identify Jest/Vitest/pytest
   - Handle npm/yarn/pnpm script resolution
   - Support direct invocations and npx patterns

4. **Extract Test Files**
   - Call runner definition's GetTestFiles()
   - Support both static and dynamic discovery
   - Handle empty file lists for dynamic mode

5. **Setup Infrastructure**
   - Create run directory at `.3pio/runs/[runID]/`
   - Create IPC file at `.3pio/ipc/[runID].jsonl`
   - Extract embedded adapters using adapters.ExtractAdapter()

6. **Initialize Managers**
   - Create IPC Manager with file path
   - Create Report Manager with run directory
   - Initialize report with test files and command

7. **Build Modified Command**
   - Use runner definition's BuildCommand()
   - Inject adapter path into arguments
   - Handle different injection patterns per runner

8. **Spawn Process**
   - Create command with first argument as executable
   - Pass remaining arguments to command
   - Append THREEPIO_IPC_PATH to environment
   - Connect stdin for interactive input

9. **Setup Concurrent Operations**
   - Start IPC event processing goroutine
   - Start stdout capture goroutine
   - Start stderr capture goroutine
   - Setup signal handler for interruption

10. **Process Events**
    - Read from IPC Manager's Events channel
    - Update Report Manager state
    - Display progress for completed files

11. **Handle Completion**
    - Wait for process exit or signal
    - Stop IPC watching
    - Wait for all goroutines to complete
    - Finalize report with exit code

12. **Display Summary**
    - Show memorable failure message if tests failed
    - Display test counts (failed, passed, total)
    - Show execution time

## 4. Concurrent Architecture

### Goroutine Structure

- **Main Goroutine**: Coordinates all operations
- **processEvents() goroutine**: Reads IPC events, updates report, shows progress
- **captureOutput(stdout) goroutine**: Reads stdout pipe, writes to file, echoes to console
- **captureOutput(stderr) goroutine**: Reads stderr pipe, writes to file, captures errors
- **Signal handler**: Waits for SIGINT/SIGTERM, kills process, sets exit code

### Synchronization
- WaitGroup for output capture goroutines
- Channel for process completion signal
- Channel for IPC events completion
- Mutex protection in Report Manager

## 5. Output Capture Strategy

### Pipe-Based Capture

The system creates pipes for both stdout and stderr before starting the process, allowing the orchestrator to intercept all output while the test runner executes.

### captureOutput Method
- Uses bufio.Scanner for line-by-line reading
- Writes all output to `output.log`
- Optionally captures stderr to error buffer
- Non-blocking reads with scanner

### Console Display
- Real-time test file progress
- Shows only newly completed files
- Tracks passed/failed counts
- Displays timing information

## 6. Error Handling

### Test Runner Detection
- Unknown runner → Show error and supported runners
- Missing package.json → Continue if possible

### Process Execution
- Command not found → Clear error message
- Permission denied → Suggest fixes
- Adapter extraction failure → Log and exit

### Graceful Shutdown
- SIGINT (Ctrl+C) → Kill process, exit code 130
- SIGTERM → Clean shutdown, preserve reports
- Always finalize reports if possible

### Error Reporting
- Capture stderr for command failures
- Include error details in final summary
- Preserve partial results on crash

## 7. Run ID Generation

### Format
`[ISO8601_timestamp]-[adjective]-[starwars_character]`

### Components
- **Timestamp**: `20250911T120000Z` format
- **Adjective**: Random from curated list (brave, clever, swift, etc.)
- **Character**: Star Wars names (luke, leia, yoda, etc.)

### Example IDs
- `20250911T120000Z-brave-luke`
- `20250911T143022Z-clever-leia`
- `20250911T091511Z-swift-yoda`

## 8. Integration Points

### Runner Manager
- Calls `Detect()` to identify test runner
- Gets `Definition` for command building
- Uses `GetTestFiles()` for discovery

### Report Manager
- Calls `Initialize()` with test files
- Sends events via `HandleEvent()`
- Calls `Finalize()` with exit code

### IPC Manager
- Calls `WatchEvents()` to start monitoring
- Reads from `Events` channel
- Calls `Cleanup()` to stop watching

### Embedded Adapters
- Calls `ExtractAdapter()` for each runner
- Gets temporary path for adapter
- Cleans up after completion

## 9. Configuration

### Environment Variables
- `THREEPIO_IPC_PATH`: Set for child process
- `THREEPIO_DEBUG`: Enables debug logging

### Exit Codes
- 0: All tests passed
- 1: Test failures or command error
- 130: Interrupted by SIGINT
- Mirror test runner's exit code otherwise

## 10. Testing Strategy

### Unit Tests (`orchestrator_test.go`)
- Run ID generation format
- Command detection logic
- Exit code handling
- Signal handling simulation

### Integration Tests
- Full execution with mock managers
- Concurrent operations coordination
- Error scenarios and recovery
- Output capture verification

### End-to-End Tests
- Complete runs with real test runners
- Report generation accuracy
- Console output format
- Exit code mirroring

## 11. Performance Optimizations

### Concurrent Processing
- Parallel goroutines for I/O operations
- Non-blocking event processing
- Efficient channel communication

### Memory Management
- Bounded error buffer (strings.Builder)
- Stream processing for output
- Map for tracking displayed files

### File Operations
- Single output.log handle kept open
- Incremental writes through Report Manager
- Cleanup of temporary files
