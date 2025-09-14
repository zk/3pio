# Test Runner Adapters

## Overview

Test runner adapters are specialized reporters that 3pio injects into test runners (Jest, Vitest, pytest) to capture test events and output. These adapters are embedded in the Go binary and extracted at runtime.

## Adapter Architecture

### Embedding and Extraction

1. **Development**: Adapters written in JavaScript (Jest/Vitest) or Python (pytest)
2. **Build Time**: Go's embed directive includes adapters in the binary
3. **Runtime**: Adapters extracted to temporary directory with IPC path injection
4. **Injection**: Test runner commands modified to include the adapter
5. **Cleanup**: Temporary files removed after test completion

### IPC Path Injection

Since v0.0.1, IPC paths are injected directly into adapter code at runtime:
- Template markers in source: `/*__IPC_PATH__*/"WILL_BE_REPLACED"/*__IPC_PATH__*/` (JavaScript)
- Template markers in source: `#__IPC_PATH__#"WILL_BE_REPLACED"#__IPC_PATH__#` (Python)
- Each test run gets its own adapter instance in `.3pio/adapters/[runID]/`
- Ensures 100% reliability in monorepos and complex process hierarchies

## Test Runner Support

### Go Test (Native)

**Implementation**: Native JSON processing without external adapter
- No adapter file required - processes `go test -json` output directly
- `GoTestDefinition` in `internal/runner/definitions/gotest.go`
- Automatically adds `-json` flag if not present
- Maps packages to test files via `go list -json`
- Tracks test state for parallel test attribution
- Handles cached test results with CACH status

**Special Considerations**:
- Uses Go's built-in JSON output format (available since Go 1.10)
- Processes stdout directly in the orchestrator
- Supports subtests with "/" separator in names
- Handles parallel test output with pause/cont state tracking
- Detects cached packages and reports them separately

### Jest Adapter

**Implementation**: Reporter interface with lifecycle methods
- `onRunStart`: Setup before test run
- `onTestStart`: File beginning, start console capture
- `onTestCaseResult`: Individual test results
- `onTestResult`: File completion, stop console capture
- `onRunComplete`: Final cleanup

**Special Considerations**:
- testResult.console is always undefined (verified with Jest 29.x)
- Must patch both console methods AND stdout/stderr writers
- No default reporter included (clean output)
- Reporter flag must come LAST in command line

### Vitest Adapter

**Implementation**: Reporter with V3 module hooks for parallel support
- `onInit`: Initialize and send collection start event
- `onPathsCollected`: Send collection complete with file count
- `onTestModuleCollected`: Send testFileStart when module queued (V3 hook, real-time)
- `onTestModuleEnd`: Send testFileResult and test cases (V3 hook, real-time)
- `sendTestCasesFromModule`: Recursively extract and send individual test case events
- `onTestFileStart`/`onTestFileResult`: Fallback for non-parallel mode
- `onFinished`: Final fallback processing and cleanup

**Special Considerations**:
- Uses V3 module hooks for real-time progress in parallel mode
- Module hooks (`onTestModule*`) work across worker processes
- Default reporter included for better UX
- `vitest list` unreliable (runs in watch mode)
- Dynamic test discovery when files unknown upfront
- Sends individual test case events with status, duration, and errors

### pytest Adapter

**Implementation**: Plugin using pytest hooks
- `pytest_sessionstart`: Initialize before session
- `pytest_runtest_protocol`: Handle test execution
- `pytest_runtest_logreport`: Process test results
- `pytest_sessionfinish`: Final cleanup

**Special Considerations**:
- Uses plugin architecture, not reporter
- Captures output via capsys fixture
- Handles collection phase errors
- Supports parametrized tests

## Console Output Capture

### Capture Strategy

All adapters implement output capture to handle:
- Direct stdout/stderr writes that bypass test frameworks
- Console methods intercepted by frameworks
- Output from worker processes and threads
- Buffered output from parallel execution

### Implementation Pattern

1. Store original write functions
2. Replace with instrumented wrappers
3. Capture and forward chunks
4. Send IPC events with file context
5. Restore original functions

### Framework Differences

**Jest**:
- No default reporter → All output through 3pio
- Must capture at multiple levels
- Worker process architecture considerations

**Vitest**:
- Default reporter included → Dual output
- Global capture with context switching
- Thread-based parallelization

**pytest**:
- Plugin architecture → Different capture mechanism
- Uses capsys fixture
- Process-based parallelization with xdist

## Adding New Test Runners

### Step 1: Create the Adapter

Write adapter in the test runner's native language:
```javascript
// For JavaScript runners
class ThreePioReporter {
  constructor() {
    this.ipcPath = /*__IPC_PATH__*/"WILL_BE_REPLACED"/*__IPC_PATH__*/;
  }
  // Implement runner-specific interface
}
```

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
- Create definition struct implementing interface
- Add detection patterns
- Define command building logic
- Handle adapter injection

### Step 4: Register and Test

1. Register in Manager initialization
2. Create test fixtures in `tests/fixtures/`
3. Write comprehensive tests
4. Update documentation

## Best Practices

### Silent Operation
- Never write to stdout/stderr directly
- All communication through IPC
- All debug output written to `.3pio/debug.log`

### Error Resilience
- Wrap all IPC operations in try/catch
- Never crash the test runner
- Gracefully degrade if IPC unavailable

### Performance
- Don't buffer large amounts of data
- Send events immediately
- Clean up resources promptly

### File Association
- Always include test file path in events
- Handle worker processes correctly
- Track context switching

## Debugging Adapters

### Manual Testing
```bash
# Build with adapter
make adapters && make build

# Test with sample project
./build/3pio npx [test-runner]

# Check IPC events
cat .3pio/ipc/*.jsonl | jq

# View debug logs
tail -f .3pio/debug.log
```

### Common Issues
- **No events**: Check IPC path injection
- **Missing output**: Verify capture patches
- **Wrong file association**: Check context tracking
- **Adapter not found**: Ensure extraction succeeded