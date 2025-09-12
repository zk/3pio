# Component Design: Test Runner Adapters

## 1. Core Purpose

Test Runner Adapters are specialized reporters for Jest, Vitest, and pytest that run inside test processes. They capture test execution events, individual test case results, and console output, transmitting this data to 3pio via IPC without interfering with normal test output. These adapters are embedded in the Go binary and extracted at runtime.

## 2. Architecture Overview

### Embedded Distribution
- JavaScript adapters (Jest, Vitest) and Python adapter (pytest) embedded in Go binary
- Extracted to temporary directory at runtime using Go's embed package
- Automatic cleanup after test completion
- Version consistency guaranteed between CLI and adapters

### Language Support
- **JavaScript**: Jest and Vitest adapters written in JavaScript
- **Python**: pytest adapter implemented as pytest plugin
- Both use same IPC protocol for communication

## 3. JavaScript Adapters

### Jest Adapter

**Class Structure:**

The Jest adapter implements the reporter interface with these lifecycle methods:
- constructor: Initialize with global config and options
- onRunStart: Setup before test run begins
- onTestStart: Handle test file start
- onTestCaseResult: Capture individual test results
- onTestResult: Process file completion
- onRunComplete: Final cleanup

**Key Features:**
- Implements Jest's reporter interface
- Captures individual test case results via onTestCaseResult hook
- Handles suite hierarchy from ancestorTitles array
- Works with Jest's worker process architecture
- Patches stdout/stderr per test file

**Console Capture Strategy:**

The adapter intercepts console output by:
1. Storing original write functions
2. Replacing with wrapper functions
3. Capturing output chunks
4. Sending IPC events with file context
5. Forwarding to original functions

### Vitest Adapter

**Class Structure:**

The Vitest adapter implements these reporter methods:
- onInit: Initialize and start global capture
- onPathsCollected: Track discovered test paths
- onTestFileStart: Switch context to new file
- onTestFileResult: Process file results and test cases
- onFinished: Fallback processing and cleanup

**Key Features:**
- Implements Vitest's reporter interface
- Recursive test case extraction from file.tasks tree
- Global console capture with context switching
- Handles both synchronous and asynchronous tests
- Fallback processing in onFinished for edge cases

**Test Case Extraction:**

Recursively processes the task tree:
1. Check task type (suite or test)
2. For suites: Build path and recurse into children
3. For tests: Send test case event with suite path
4. Handle nested describe blocks properly

## 4. Python Adapter

### pytest Adapter

**Plugin Structure:**

The pytest adapter implements these plugin hooks:
- __init__: Initialize plugin state
- pytest_sessionstart: Setup before test session
- pytest_runtest_protocol: Handle test execution
- pytest_runtest_logreport: Process test results
- pytest_sessionfinish: Cleanup after session

**Key Features:**
- Implements pytest plugin hook interface
- Captures test output via capsys fixture
- Handles test lifecycle events
- Supports parametrized tests
- Works with pytest's assertion rewriting

**IPC Communication:**

Sends events by:
1. Reading THREEPIO_IPC_PATH from environment
2. Opening file in append mode
3. Serializing event to JSON
4. Writing with newline delimiter

## 5. Embedding and Extraction

### Go Embed System

The adapters are embedded in the Go binary using embed directives:

1. **Compile-time embedding**: Adapters included in binary
2. **Runtime extraction**: Written to temporary directory
3. **Path generation**: Unique paths prevent conflicts
4. **Cleanup**: Temporary files removed after use

The ExtractAdapter function:
- Determines adapter content based on runner name
- Creates temporary directory with hash-based name
- Writes adapter file with appropriate extension
- Returns absolute path for injection

### Adapter Injection

Each test runner has a specific injection pattern:

**Jest:**
- Uses --reporters flag
- Replaces default reporter
- Accepts absolute path to adapter

**Vitest:**
- Uses --reporter flag (can specify multiple)
- Includes default reporter for user visibility
- Adds 3pio adapter as additional reporter

**pytest:**
- Uses PYTHONPATH environment variable
- Loads via -p (plugin) flag
- References module name without extension

## 6. IPC Protocol

### Event Structure

All adapters use the same JSON Lines format:

- **testFileStart**: Indicates test file beginning
  - Payload: filePath
- **testCase**: Individual test result
  - Payload: filePath, testName, status, duration (optional), error (optional)
- **stdoutChunk**: Console output chunk
  - Payload: filePath, chunk
- **testFileResult**: Test file completion
  - Payload: filePath, status

Each event is a single line of JSON for streaming compatibility.

### Environment Variables
- `THREEPIO_IPC_PATH`: Path to IPC file for event transmission
- `THREEPIO_DEBUG`: Enable debug logging when set to "1"

## 7. Console Output Capture

### Strategy Differences

**Jest**: No default reporter included
- Clean, single output stream
- All output comes through 3pio adapter

**Vitest**: Default reporter included
- Better user experience with progress indicators
- Dual output (default + 3pio capture)

**pytest**: Plugin architecture
- Captures via capsys fixture
- Integrates with pytest's output handling

### Implementation Pattern
1. Store original write functions
2. Replace with instrumented wrappers
3. Capture and forward chunks
4. Send IPC events with file context
5. Restore original functions

## 8. Error Handling

### Resilience Principles
- Never crash the test runner
- Graceful degradation without IPC
- Log errors to debug.log
- Continue test execution

### Common Error Scenarios
- Missing THREEPIO_IPC_PATH → Skip IPC, log warning
- File permission errors → Log error, continue
- JSON serialization failure → Drop event, log error
- Adapter extraction failure → Fall back to no adapter

## 9. Performance Characteristics

### Minimal Overhead
- Direct IPC writes (no buffering)
- Pass-through console output
- No memory accumulation
- Immediate event transmission

### Resource Usage
- Temporary disk space for extracted adapters
- Single file handle for IPC
- Minimal CPU overhead
- No network operations

## 10. Testing Strategy

### Unit Tests
- Event serialization
- Console capture logic
- Error handling paths
- IPC initialization

### Integration Tests (`tests/integration_go/`)
- Full test runs with real runners
- Event sequence validation
- Console output verification
- Error recovery scenarios

### Fixture Projects (`tests/fixtures/`)
- basic-jest: Simple Jest project
- basic-vitest: Simple Vitest project
- basic-pytest: Simple pytest project
- Each with passing, failing, and skipped tests

## 11. Development and Debugging

### Adapter Development
1. Edit adapter file in `internal/adapters/`
2. Run `make adapters` to prepare for embedding
3. Build with `make build`
4. Test with fixture projects

### Debug Output
When `THREEPIO_DEBUG=1`:
- Adapter initialization logged
- Each event transmission logged
- Errors include stack traces
- Performance metrics recorded

### Common Issues
- **No events received**: Check THREEPIO_IPC_PATH is set
- **Console output missing**: Verify capture patches applied
- **Test cases not reported**: Check reporter hooks called
- **Adapter not found**: Ensure extraction succeeded

## 12. Future Enhancements

- Support for additional test runners (Mocha, Jasmine, RSpec)
- Test coverage integration
- Performance profiling data
- Parallel execution tracking
- Watch mode support
- Custom reporter configurations
- Real-time streaming to web UI