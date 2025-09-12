# Component Design: IPC Manager

## 1. Core Purpose

The IPC Manager provides file-based inter-process communication between 3pio's orchestrator and test runner adapters. Implemented in Go, it uses a JSON Lines format for structured event passing, with file watching capabilities for real-time event processing. This design ensures reliable, debuggable communication across different test runner processes.

## 2. Key Features

### File-Based Communication
- Simple, reliable mechanism that works everywhere
- Easy to debug and inspect (just text files)
- No network dependencies or permissions issues
- Atomic writes prevent partial message corruption

### JSON Lines Format
- One JSON object per line
- Human-readable and machine-parseable
- Streaming-friendly for real-time processing
- Standard format with wide tooling support

### Real-Time Event Processing
- File watching using fsnotify library
- Immediate event detection and processing
- Non-blocking channel-based communication
- Graceful handling of rapid event sequences

## 3. Implementation Details

### Structure

The IPC Manager maintains:

- **IPC Path**: Path to the JSON Lines communication file
- **File Watcher**: fsnotify watcher for file changes
- **File Handle**: Open file for reading events
- **Events Channel**: Channel for sending parsed events
- **Error Channel**: Channel for error reporting
- **Done Channel**: Signal for shutdown coordination
- **Logger**: Debug logging interface
- **Last Position**: Track read position in file
- **Mutex**: Synchronization for thread safety

### Event Types

**Base Event Structure:**
- EventType: String identifying the event type
- Payload: JSON payload with event-specific data

**Specific Event Types:**

- **TestFileStartEvent**: Signals test file execution beginning
  - FilePath: Path to the test file

- **TestCaseEvent**: Individual test case result
  - FilePath: Test file path
  - TestName: Name of the test
  - SuiteName: Optional suite grouping
  - Status: PASS, FAIL, or SKIP
  - Duration: Optional execution time
  - Error: Optional error message

- **TestFileResultEvent**: Test file completion
  - FilePath: Test file path
  - Status: Overall file status

- **StdoutChunkEvent**: Console output chunk
  - FilePath: Associated test file
  - Chunk: Output content

- **StderrChunkEvent**: Error output chunk
  - FilePath: Associated test file
  - Chunk: Error content

## 4. Public API

### NewManager

Creates a new IPC Manager:
- Creates IPC directory structure
- Initializes IPC file for writing
- Sets up fsnotify watcher
- Creates event and error channels

### WatchEvents

Starts event monitoring:
- Launches file watching goroutine
- Monitors for file changes
- Reads new lines as they're written
- Parses and sends events to channel

### Cleanup

Shuts down the IPC system:
- Stops file watching
- Closes channels
- Releases file handles
- Ensures graceful shutdown

### Static Helper Functions

For adapter usage (writing events):

**SendEvent**: Writes an event to the IPC file
- Reads THREEPIO_IPC_PATH from environment
- Serializes event to JSON
- Appends to IPC file with newline
- Thread-safe with file locking

## 5. Communication Protocol

### IPC File Format

JSON Lines format with one event per line:
- testFileStart: Test file beginning execution
- testCase: Individual test result with status and duration
- stdoutChunk: Console output from test
- testFileResult: Test file completion status

Each line is a complete JSON object that can be parsed independently.

### Event Flow

1. **Adapter writes event** → IPC file
2. **File system notifies** → fsnotify watcher
3. **Manager reads new lines** → From last position
4. **Parse JSON** → Validate structure
5. **Determine event type** → Based on eventType field
6. **Unmarshal payload** → Into specific event struct
7. **Send to channel** → For orchestrator processing

## 6. File Watching Implementation

### Reading Strategy

The manager reads new events efficiently:
1. Seek to last read position in file
2. Scan for new lines using buffered reader
3. Parse each line as JSON event
4. Send parsed event to channel
5. Update position for next read

### File Watching Loop

The watch loop handles:
- **Write events**: Trigger reading of new content
- **Error events**: Forward to error channel
- **Done signal**: Exit gracefully

This approach ensures only new content is processed, avoiding re-reading the entire file.

## 7. Error Handling

### File System Errors
- Directory creation failure → Return error to caller
- File permission issues → Log error and exit
- Watcher creation failure → Fallback to polling

### Parsing Errors
- Invalid JSON → Log warning, skip line
- Unknown event type → Create UnknownEvent
- Malformed payload → Use partial data

### Communication Errors
- Channel blocked → Drop events with warning
- File locked → Retry with backoff
- Watcher stopped → Attempt restart

## 8. Performance Considerations

### Efficient File Reading
- Track last read position
- Only read new content
- Use buffered scanner
- Minimize file seeks

### Channel Management
- Buffered event channel (size 1000)
- Non-blocking sends where possible
- Graceful handling of slow consumers

### Memory Usage
- Stream processing (no full file load)
- Bounded channel buffers
- Efficient JSON parsing

## 9. Thread Safety

### Concurrent Access
- Mutex protection for file position
- Thread-safe channel operations
- Atomic file writes from adapters

### Multiple Writers
- File system handles concurrent appends
- Each adapter process writes independently
- No coordination required between writers

## 10. Testing Strategy

### Unit Tests (`manager_test.go`)
- Event parsing and validation
- File watching simulation
- Channel communication
- Error handling scenarios

### Integration Tests
- Multi-process writing
- Large event volumes
- Rapid event sequences
- File system edge cases

### Test Utilities

Provides helper functions for testing:
- Create test manager with temporary directory
- Generate test IPC path
- Initialize with test logger
- Return manager and path for test assertions

## 11. Debugging Support

### IPC File Inspection
- Human-readable JSON format
- Can tail -f during execution
- Use jq for pretty printing
- Timestamps in event data

### Debug Logging
- Event reception logging
- Parse error details
- File position tracking
- Performance metrics

### Common Issues
- **Events not received**: Check THREEPIO_IPC_PATH
- **Parsing errors**: Validate JSON format
- **Missing events**: Check file permissions
- **Delayed events**: Monitor file system latency

## 12. Platform Considerations

### File System Differences
- Windows: Different path separators
- Linux: inotify for file watching
- macOS: FSEvents/kqueue support
- All: Atomic append operations

### Path Handling

Uses platform-agnostic path construction:
- Join path segments with proper separators
- Create IPC directory under .3pio
- Generate unique IPC file names with run ID
- Handle different path conventions across OS

## 13. Future Enhancements

- WebSocket transport option
- Binary protocol for performance
- Event compression for large outputs
- Bi-directional communication
- Event replay capabilities
- Real-time streaming to web UI
- Message acknowledgments
- Event filtering and routing