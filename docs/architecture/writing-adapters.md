# Writing Test Runner Adapters

## Overview

Test runner adapters are specialized reporters that 3pio injects into test runners (Jest, Vitest, pytest) to capture test events and output. These adapters are embedded in the Go binary and extracted at runtime for use.

## Architecture

### Embedding Process

1. **Development**: Adapters are written in JavaScript (Jest/Vitest) or Python (pytest)
2. **Build Time**: Go's embed directive includes adapters in the binary
3. **Runtime**: Adapters are extracted to a temporary directory
4. **Injection**: Test runner commands are modified to include the adapter
5. **Cleanup**: Temporary files are removed after test completion

### Communication Protocol

Adapters communicate with 3pio via file-based IPC using JSON Lines format:
- Each event is a single line of JSON
- Events are appended to a file specified by `THREEPIO_IPC_PATH`
- The orchestrator watches this file for new events

## Event Types

All adapters must emit these standard events:

### testFileStart
Signals that a test file is beginning execution:
```json
{
  "eventType": "testFileStart",
  "payload": {
    "filePath": "src/math.test.js"
  }
}
```

### testCase
Reports an individual test result:
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

### testFileResult
Indicates test file completion:
```json
{
  "eventType": "testFileResult",
  "payload": {
    "filePath": "src/math.test.js",
    "status": "PASS"
  }
}
```

### stdoutChunk / stderrChunk
Captures console output:
```json
{
  "eventType": "stdoutChunk",
  "payload": {
    "filePath": "src/math.test.js",
    "chunk": "Console log output\n"
  }
}
```

## JavaScript Adapter Guidelines

### Jest Adapter Structure

Jest adapters implement the Reporter interface:

```javascript
class ThreePioJestReporter {
  constructor(globalConfig, reporterOptions, reporterContext) {
    // Initialize IPC path from environment
    this.ipcPath = process.env.THREEPIO_IPC_PATH;
  }
  
  onRunStart(results) {
    // Setup before test run
  }
  
  onTestStart(test) {
    // File beginning - send testFileStart
    // Start console capture
  }
  
  onTestCaseResult(test, testCaseResult) {
    // Individual test - send testCase
  }
  
  onTestResult(test, testResult) {
    // File complete - send testFileResult
    // Stop console capture
  }
  
  onRunComplete(testContexts, results) {
    // Final cleanup
  }
}
```

### Vitest Adapter Structure

Vitest adapters use a similar but distinct interface:

```javascript
class ThreePioVitestReporter {
  onInit(ctx) {
    // Initialize and start global capture
  }
  
  onPathsCollected(paths) {
    // Track discovered test files
  }
  
  onTestFileStart(file) {
    // Send testFileStart event
  }
  
  onTestFileResult(file) {
    // Process test cases recursively
    // Send testCase events
    // Send testFileResult
  }
  
  onFinished(files, errors) {
    // Handle any remaining files
  }
}
```

### Console Capture Pattern

Both JavaScript adapters patch console output:

1. Store original write functions
2. Replace with wrapper that captures and forwards
3. Send chunks via IPC with file association
4. Restore original functions when done

## Python Adapter Guidelines

### pytest Plugin Structure

The pytest adapter uses plugin hooks:

```python
class ThreePioPytestPlugin:
    def __init__(self):
        self.ipc_path = os.environ.get('THREEPIO_IPC_PATH')
        
    def pytest_sessionstart(self, session):
        # Initialize before session
        
    def pytest_runtest_protocol(self, item, nextitem):
        # Handle test execution
        
    def pytest_runtest_logreport(self, report):
        # Process test results
        # Send testCase events
        
    def pytest_sessionfinish(self, session):
        # Final cleanup
```

### Output Capture

pytest adapters use the capsys fixture to capture output and send it via IPC events.

## Best Practices

### 1. Silent Operation
- **Never** write to stdout/stderr directly
- All communication must go through IPC
- All debug output written to `.3pio/debug.log`

### 2. Error Resilience
- Wrap all IPC operations in try/catch
- Never let adapter errors crash the test runner
- Gracefully degrade if IPC unavailable

### 3. Minimal Overhead
- Don't buffer large amounts of data
- Send events immediately as they occur
- Clean up resources promptly

### 4. File Association
- Always include the test file path in events
- Handle worker processes correctly (Jest)
- Track context switching (Vitest)

## Adding a New Test Runner

### Step 1: Create the Adapter

Write adapter in the test runner's native language:
- Study existing adapters for patterns
- Implement all required event types
- Test with the runner's example projects

### Step 2: Add to Go Binary

1. Place adapter in `internal/adapters/`
2. Add embed directive in `embedded.go`:
   ```go
   //go:embed newrunner.js
   var newRunnerAdapter string
   ```
3. Update `ExtractAdapter()` function

### Step 3: Create Runner Definition

In `internal/runner/definition.go`:
1. Create new definition struct
2. Implement the Definition interface
3. Add detection patterns
4. Define command building logic

### Step 4: Register Runner

In `internal/runner/manager.go`:
1. Add to Manager initialization
2. Set detection priority order

### Step 5: Write Tests

Create comprehensive tests:
- Unit tests for detection logic
- Integration tests with real runner
- Test fixture in `tests/fixtures/`

### Step 6: Update Documentation

- Add to supported runners list
- Document any special considerations
- Update architecture diagrams if needed

## Testing Adapters

### Manual Testing

1. Build 3pio with your adapter:
   ```bash
   make adapters
   make build
   ```

2. Test with a sample project:
   ```bash
   ./build/3pio npx [test-runner]
   ```

3. Check IPC file for events:
   ```bash
   cat .3pio/ipc/*.jsonl | jq
   ```

4. Verify report generation:
   ```bash
   cat .3pio/runs/*/test-run.md
   ```

### Debugging

Check debug logs for detailed information:
```bash
./build/3pio npm test
tail -f .3pio/debug.log
```

Common issues to check:
- THREEPIO_IPC_PATH is set correctly
- Adapter file is extracted successfully
- Events are properly formatted JSON
- File paths are absolute or properly resolved

## Examples

See the existing adapters for reference:
- `internal/adapters/jest.js` - Jest reporter implementation
- `internal/adapters/vitest.js` - Vitest reporter implementation
- `internal/adapters/pytest_adapter.py` - pytest plugin implementation

Each demonstrates the complete pattern of:
- Event lifecycle management
- Console output capture
- Error handling
- IPC communication