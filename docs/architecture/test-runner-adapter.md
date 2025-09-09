# Component Design: Test Runner Adapters

## 1. Core Purpose

Test Runner Adapters are silent reporters that run inside Jest and Vitest processes. They capture test execution events, individual test case results, and console output, transmitting this data to the CLI via IPC events without interfering with normal test output.

## 2. General Principles

### Silent Operation
- No stdout/stderr output during normal operation
- All communication via IPC channel only
- Debug logging to `.3pio/debug.log` when THREEPIO_DEBUG=1
- Coexist with default test runner reporters

### Configuration
- Read THREEPIO_IPC_PATH environment variable for IPC file location
- Validate IPC connection during initialization
- Log startup preamble for debugging

### Error Resilience
- All IPC operations wrapped in error handlers
- Failures don't crash test runners
- Graceful degradation when IPC unavailable

## 3. Event Reporting

### Test Case Events
Adapters report individual test case results including:
- Test name and suite organization
- Status (PASS, FAIL, SKIP)
- Execution duration
- Error messages and stack traces for failures

### File Level Events
- **testFileStart**: Signals beginning of test file execution
- **testFileResult**: Reports final status for entire file
- Aggregate statistics for the file

### Console Output Capture
- Patch process.stdout.write and process.stderr.write
- Capture output chunks during test execution
- Associate output with current test file
- Send as stdoutChunk/stderrChunk events

## 4. Implementation Strategies

### Jest Adapter (ThreePioJestReporter)

#### Lifecycle Hooks
- **onRunStart**: Initialize IPC connection
- **onTestStart**: Begin capturing for specific test file
- **onTestCaseResult**: Report individual test case results
- **onTestResult**: Send file completion status
- **onRunComplete**: Cleanup and final processing

#### Console Capture Strategy
- Start/stop capture per test file
- Track current test file context
- Handle Jest's worker process architecture
- Work around Jest's empty testResult.console issue

#### Test Case Extraction
- Process test results from onTestCaseResult hook
- Extract suite hierarchy from ancestorTitles
- Include duration and error details
- Handle nested describe blocks

### Vitest Adapter (ThreePioVitestReporter)

#### Lifecycle Hooks
- **onInit**: Initialize and start global capture
- **onPathsCollected**: Track discovered test paths
- **onTestFileStart**: Switch context to new file
- **onTestFileResult**: Process file results and test cases
- **onFinished**: Fallback processing and cleanup

#### Console Capture Strategy
- Global capture started in onInit
- Context switching based on current test file
- Handle output without clear file context
- Use 'global' as fallback for unattributed output

#### Test Case Processing
- Extract test cases from file.tasks recursively
- Handle suite nesting and organization
- Process both synchronous and asynchronous tests
- Fallback processing in onFinished for edge cases

## 5. Stream Tapping Architecture

### Capture Lifecycle
1. Store original write functions
2. Replace with instrumented wrappers
3. Capture and forward output chunks
4. Send IPC events with file association
5. Restore original functions when done

### Pass-Through Behavior
- Maintain normal console output
- Preserve output formatting
- No visible changes to test runner behavior
- Support for color codes and special characters

## 6. Failure Modes

### Environment Issues
- Missing THREEPIO_IPC_PATH variable
- IPC file permissions problems
- File system errors

### API Compatibility
- Test runner version changes
- Breaking changes in reporter interfaces
- Hook behavior modifications

### Runtime Errors
- Stream patching conflicts with other tools
- Memory issues with large output
- Race conditions in cleanup

### Context Loss
- Output without clear test file association
- Worker process isolation in Jest
- Async test execution ordering

## 7. Performance Considerations

### Minimal Overhead
- Lightweight event transmission
- No buffering or batching
- Direct pass-through for console output
- Efficient IPC writes

### Memory Management
- No accumulation of output in memory
- Immediate event transmission
- Cleanup after each test file

## 8. Debugging Support

### Startup Preamble
Each adapter logs initialization info:
- Adapter version
- IPC path configuration
- Process ID
- Timestamp

### Debug Logging
When THREEPIO_DEBUG=1:
- Detailed lifecycle events
- IPC transmission logs
- Error details with context
- Performance metrics

## 9. Testing Strategy

### Unit Tests
- IPC initialization and validation
- Stream tapping logic
- Event formatting
- Error handling

### Integration Tests
- Real Jest/Vitest project execution
- IPC file monitoring
- Event sequence validation
- Console output capture verification

### Compatibility Tests
- Multiple test runner versions
- Various project configurations
- Edge cases and error scenarios
- Performance under load

## 10. Future Enhancements

- Support for additional test runners (Mocha, Jasmine)
- Test coverage integration
- Performance metrics collection
- Parallel test execution tracking
- Watch mode support
- Custom event types for extensions