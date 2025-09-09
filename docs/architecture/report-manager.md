# Component Design: Report Manager

## 1. Core Purpose

The Report Manager encapsulates all file system logic for creating and updating persistent test reports. It manages test state at both file and individual test case levels, supporting dynamic test discovery and providing context-efficient output for AI agents.

## 2. Key Features

### Test Case Tracking
- Individual test case results with suite organization
- Status tracking: PASS, FAIL, SKIP, PENDING, RUNNING
- Duration and error message capture
- Hierarchical organization by suite

### Dynamic Test Discovery
- Optional test file list during initialization
- Runtime registration of discovered test files
- Seamless handling of both static and dynamic modes

### Performance Optimization
- Debounced writes to reduce I/O overhead
- In-memory state management
- Parallel output collection strategies

### Output Management
- Unified output.log for complete test run
- Per-file output Maps for individual test logs
- Test case boundary tracking

## 3. Internal State

### TestRunState
Maintains the complete state of the test run in memory:
- Timestamp and status tracking (RUNNING, COMPLETE, ERROR)
- Test command arguments
- File-level counters (total, completed, passed, failed, skipped)
- Individual test file states with their test cases
- Links to generated log files

### TestCase Tracking
Each test case maintains:
- Test name and optional suite grouping
- Status (PASS, FAIL, SKIP, PENDING, RUNNING)
- Execution duration when available
- Error messages and stack traces for failures

### Output Collection Strategy
Three-tier Map structure for organizing output:
- File-level output collection for general console logs
- Test-case-level output for specific test boundaries
- Active test tracking to associate output with running tests

## 4. Public API

### Constructor
Initializes the Report Manager with:
- Unique run ID for directory naming
- Test command for documentation
- OutputParser instance for runner-specific parsing
- Debounced write configuration for performance

### Initialization
Creates the report infrastructure:
- Establishes run directory structure
- Opens output.log for streaming writes
- Supports both static mode (with predefined test files) and dynamic mode (files discovered during execution)
- Generates initial test-run.md report

### Event Handling
Central method for processing IPC events from adapters:
- **testCase events**: Track individual test results with suite, status, duration, and errors
- **testFileStart events**: Mark files as currently running
- **testFileResult events**: Update file completion status and aggregate counters
- **stdout/stderr chunks**: Collect and organize output by file and test case

### Dynamic Registration
Handles test files discovered during execution:
- Automatically adds new files to tracking state
- Maintains accurate file counts
- Preserves all test case information

### Finalization
Completes the report generation:
- Closes all file handles
- Writes individual test log files from collected output
- Performs final state write with completion status
- Ensures all data is persisted to disk

### Helper Methods
Utility functions for:
- Direct output log appending
- Report path retrieval
- Summary statistics generation

## 5. Report Generation

### Test Run Report (test-run.md)
Primary report in Markdown format containing:
- Header section with timestamp (including memorable Star Wars name), status, and executed command
- Summary statistics for the entire test run
- Per-file sections showing individual test cases organized by suite
- Status indicators (✓ for pass, ✕ for fail, ○ for skip)
- Test execution durations when available
- Error messages and stack traces for failures
- Links to individual test log files

### Individual Test Logs
Separate log files for each test file:
- Generated from output collected during test execution
- Contains only output specific to that test file
- Organized by test case boundaries when possible
- Stored in the logs/ subdirectory with sanitized filenames

## 6. Performance Considerations

### Debounced Writes
- Batches multiple state updates into single writes
- Reduces I/O overhead during rapid test execution
- Configurable delay and max wait times

### Memory Management
- Bounded Maps for output collection
- Periodic cleanup of completed test data
- Efficient string concatenation for large outputs

### File Handle Management
- Single output.log handle kept open during execution
- Atomic writes for test-run.md updates
- Proper cleanup in finalization

## 7. Error Handling

### File System Errors
- Permission issues: Log error and attempt alternate location
- Disk full: Gracefully degrade to essential reports only
- Path conflicts: Use unique identifiers to avoid collisions

### Event Processing Errors
- Invalid event data: Log warning and continue
- Missing file paths: Use dynamic registration
- Malformed test cases: Skip with warning

### Crash Recovery
- Periodic state persistence via debounced writes
- Graceful degradation if finalization fails
- Partial reports better than no reports

## 8. Integration with OutputParser

The ReportManager uses OutputParser for runner-specific logic:
- Parse test boundaries from console output
- Associate output chunks with test files
- Extract error messages and stack traces
- Handle runner-specific output formats

## 9. Testing Strategy

### Unit Tests
- State management with various event sequences
- Dynamic test file registration
- Debounced write mechanism with mock timers
- Test case hierarchical organization
- Error handling scenarios

### Integration Tests
- Real file system operations in temp directories
- Stream of IPC events with expected outputs
- Concurrent event processing
- Large output handling
- Crash recovery scenarios

### Performance Tests
- Large test suite handling (1000+ files)
- Memory usage under load
- I/O throughput optimization
- Debounce effectiveness

## 10. Future Enhancements

- Streaming report updates for real-time viewing
- Compressed output storage for large test runs
- Incremental report updates for watch mode
- Custom report formats (JSON, XML, HTML)
- Test result caching and comparison
- Failure pattern analysis