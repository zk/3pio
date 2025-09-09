# Component Design: IPC Manager

## 1. Core Purpose

The IPC (Inter-Process Communication) Manager provides a reliable file-based event communication channel between Test Runner Adapters (running inside test processes) and the CLI Orchestrator (main process). It uses JSONL (JSON Lines) format for structured event transmission.

## 2. Architecture

### Class Design
- **IPCManager class**: Instance methods for CLI-side operations
- **Static methods**: For adapter-side event sending
- **File-based communication**: Uses `.3pio/ipc/[runId].jsonl` files

### Communication Flow
1. Adapters write events to IPC file using static methods
2. CLI watches file for changes using instance methods
3. Events processed sequentially to maintain order
4. File acts as persistent audit log of all events

## 3. Public API

### For Adapters (Static Methods)
- **sendEvent**: Writes events to IPC file specified in THREEPIO_IPC_PATH environment variable
- **Synchronous writes**: Uses fs.appendFileSync for reliability in test runner context
- **Error resilience**: Catches errors to prevent test runner crashes

### For CLI (Instance Methods)
- **watchEvents**: Monitors IPC file for new events using chokidar
- **Event callback**: Invokes callback for each parsed event
- **Position tracking**: Maintains read position to process only new events
- **cleanup**: Stops file watching and releases resources

### Directory Management
- **ensureIPCDirectory**: Creates `.3pio/ipc/` directory structure
- **Returns absolute path**: For IPC file creation

## 4. Event Schema

All events follow a consistent structure with eventType and payload:

### Test Execution Events
- **testCase**: Individual test case results with suite, status, duration, and errors
- **testFileStart**: Signals beginning of test file execution
- **testFileResult**: Final status for a test file (PASS/FAIL/SKIP)

### Output Capture Events
- **stdoutChunk**: Console output from stdout with associated file path
- **stderrChunk**: Console output from stderr with associated file path

### Event Format
JSONL format (one JSON object per line):
- Each event on separate line
- Newline character as delimiter
- Enables streaming and partial reads

## 5. Implementation Details

### File Watching Strategy
- Uses chokidar for cross-platform file monitoring
- Configurable polling and stability thresholds
- Handles rapid file changes efficiently

### Read Position Management
- Tracks last read line number
- Processes only new lines on file change
- Prevents duplicate event processing

### Error Handling
- Malformed JSON lines logged but don't stop processing
- File permission errors handled gracefully
- Partial writes detected and logged

## 6. Performance Considerations

### Write Performance
- Synchronous writes in adapters for reliability
- Minimal overhead to avoid affecting test execution
- No buffering to ensure real-time event delivery

### Read Performance
- Efficient file watching without polling when possible
- Batch processing of multiple events
- Minimal memory footprint with streaming reads

## 7. Logging Integration

### Debug Logging
- Event transmission logged when THREEPIO_DEBUG=1
- File operations tracked for troubleshooting
- Error conditions logged with context

### Event Flow Tracking
- Incoming events logged with type
- Processing errors captured with details
- Lifecycle events (start, stop) recorded

## 8. Failure Modes

### File System Issues
- **Permission denied**: Cannot create or write to IPC file
- **Disk full**: Write operations fail
- **File deleted**: IPC file removed during execution

### Data Integrity Issues
- **Malformed JSON**: Invalid event data
- **Partial writes**: Incomplete event at file end
- **Encoding issues**: Non-UTF8 characters

### Process Issues
- **Adapter crash**: Events stop arriving
- **CLI crash**: Events not processed
- **Race conditions**: Multiple writers (prevented by design)

## 9. Testing Strategy

### Unit Tests
- Event serialization and deserialization
- File watching with mock file changes
- Read position tracking accuracy
- Error handling for malformed data

### Integration Tests
- Multi-process communication scenarios
- High-volume event streams
- Concurrent reads and writes
- File system error simulation

### Performance Tests
- Event throughput measurement
- File size limits
- Memory usage under load
- Latency measurements

## 10. Security Considerations

- File permissions restricted to user
- No sensitive data in IPC events
- Temporary files cleaned up after use
- Input validation on all events

## 11. Future Enhancements

- Event compression for large payloads
- Binary protocol for efficiency
- Socket-based IPC option
- Event replay capabilities
- Real-time event streaming to external consumers