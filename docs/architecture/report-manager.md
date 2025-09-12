# Component Design: Report Manager

## 1. Core Purpose

The Report Manager handles all file I/O operations for test reports in 3pio. Implemented in Go, it provides incremental writing capabilities, thread-safe state management, and generates both summary reports and individual test logs. The design prioritizes reliability, ensuring partial results are available even if the process is interrupted.

## 2. Key Features

### Incremental Writing
- Individual log files created immediately upon test file registration
- Output written as events arrive, not batched until completion
- Partial results available if process interrupted (Ctrl+C)
- All file handles properly managed and flushed

### Thread-Safe State Management
- sync.RWMutex for concurrent access protection
- In-memory state tracking for performance
- Debounced writes to reduce I/O overhead

### Comprehensive Reporting
- Main test-run.md report with test case details
- Individual log files per test file
- Complete output.log with all stdout/stderr
- Test case level tracking with suite organization

## 3. Implementation Details

### Structure

The Report Manager maintains:

- **Run Directory**: Path to the run-specific output directory
- **Test Run State**: Complete in-memory state of the test run
- **Output Parser**: Runner-specific output parsing logic
- **Logger**: Interface for debug logging
- **File Handles**: Map of open file handles for incremental writing
- **File Buffers**: Map of output buffers per test file
- **Debouncers**: Timers for debounced write operations
- **Output File**: Handle to the main output.log file
- **Synchronization**: Read/write mutex for thread safety
- **Timing Configuration**: Debounce and max wait durations

### TestRunState Structure

The test run state tracks:

- **Metadata**: Timestamp, status (RUNNING/COMPLETE/ERROR), last update time
- **Command**: Test command arguments
- **File Counters**: Total, completed, passed, failed, skipped
- **Test Files**: Array of individual test file states

Each test file contains:

- **Path**: File path
- **Status**: PENDING, RUNNING, PASS, FAIL, or SKIP
- **Timing**: Start time and duration
- **Test Cases**: Array of individual test results

Each test case includes:

- **Name**: Test name
- **Suite**: Optional suite grouping
- **Status**: Test result
- **Duration**: Execution time
- **Error**: Optional error message

## 4. Public API

### NewManager

Creates a new Report Manager instance:
- Creates run directory structure
- Opens output.log for streaming writes
- Initializes file handle maps
- Sets debounce timings

### Initialize

Sets up the initial test run:
- Creates initial test run state
- Registers known test files (static discovery)
- Writes initial test-run.md report
- Supports empty file list for dynamic discovery

### HandleEvent

Central event processing for IPC events:
- **TestFileStartEvent**: Mark file as RUNNING
- **TestCaseEvent**: Update test case status, duration, errors
- **TestFileResultEvent**: Update file completion and counters
- **StdoutChunkEvent**: Buffer output for file logs
- **StderrChunkEvent**: Buffer error output for file logs

### Finalize

Completes report generation:
- Flushes all pending writes
- Writes individual test log files
- Updates final status (COMPLETE/ERROR)
- Closes all file handles
- Ensures data persistence

## 5. File Management

### Directory Structure

Each run creates:
- **test-run.md**: Main report file
- **output.log**: Complete stdout/stderr capture
- **logs/**: Directory containing individual test file logs with sanitized names

### File Handle Lifecycle

1. **Creation**
   - output.log opened in NewManager
   - Individual logs created on first write

2. **Writing**
   - Incremental writes as events arrive
   - Buffering for efficiency
   - Debounced report updates

3. **Cleanup**
   - All handles closed in Finalize
   - Buffers flushed before closing
   - Temporary files cleaned up

## 6. Event Processing

### Dynamic Test File Registration

Automatically registers newly discovered test files:
- Adds files discovered during execution
- Updates total file count
- Creates TestFile entry in state
- Thread-safe with mutex protection

### Test Case Management

Handles individual test case events:
- Ensures test file is registered
- Adds/updates test case in file's cases
- Tracks suite organization
- Updates file status based on results

### Output Buffering

Manages per-file output collection:
- Maintains output buffers for each file
- Appends chunks as they arrive
- Flushes to disk during finalization
- Memory-efficient for large outputs

## 7. Report Generation

### test-run.md Format

The main report includes:

**Header Section:**
- Run ID with timestamp and memorable name
- Overall status (COMPLETE/ERROR)
- Test command executed
- Total duration

**Summary Section:**
- Total file count
- Passed/failed/skipped counts

**Test Files Section:**
- File path with status indicator
- Execution duration
- Test cases organized by suite
- Pass/fail indicators (✓/✕/○)
- Individual test durations
- Error messages and stack traces for failures

### Individual Log Files

Each test file gets its own log with:
- Header showing file path and timestamp
- Console output from that file's tests
- Test case boundaries when identifiable
- Error stack traces

## 8. Performance Optimizations

### Debounced Writes

Configured with:
- **Debounce time**: 100ms default
- **Max wait time**: 500ms default

Benefits:
- Batches rapid state updates
- Reduces file system operations
- Configurable timing parameters

### Memory Management
- Bounded buffers with periodic flushes
- Stream processing for large outputs
- Efficient string concatenation

### Concurrent Safety
- Read/write mutex for state access
- Goroutine-safe event handling
- Lock-free reads where possible

## 9. Error Handling

### File System Errors
- Directory creation failures → Return error
- Permission issues → Log and attempt recovery
- Disk full → Degrade gracefully

### Event Processing Errors
- Invalid event data → Log warning, continue
- Missing file paths → Auto-register file
- Malformed test cases → Skip with warning

### Recovery Mechanisms
- Partial writes preserved on crash
- State persistence via incremental updates
- Cleanup attempted even on error

## 10. Testing Strategy

### Unit Tests (`manager_test.go`)
- Event handling sequences
- Dynamic file registration
- State management consistency
- Debounce mechanism timing

### Integration Tests
- Real file system operations
- Concurrent event processing
- Large output handling
- Error recovery scenarios

### Edge Cases
- Empty test runs
- Interrupted processes
- Malformed events
- File system limitations

## 11. Thread Safety

### Mutex Usage

The system uses read/write mutexes to:
- Protect state modifications
- Ensure consistent reads
- Prevent race conditions
- Allow concurrent read operations

### Concurrent Operations
- Multiple goroutines can send events
- Output capture runs in parallel
- Report writes are synchronized

## 12. Future Enhancements

- Streaming updates for real-time viewing
- Compressed storage for large outputs
- Differential reports for re-runs
- Custom report formats (JSON, XML)
- Test result caching
- Failure pattern analysis
- Performance metrics tracking