# Plan: Incremental Log File Writing

## Objective
Change 3pio to write individual test log files incrementally as output arrives, ensuring partial results are available even if the test run is interrupted (e.g., Ctrl+C).

## Success Criteria
- [ ] Individual log files are created as soon as a test file starts
- [ ] Output is written to individual log files within 500ms of being received
- [ ] Log files contain partial output if the test run is interrupted
- [ ] No regression in existing functionality
- [ ] Performance impact is minimal (< 5% overhead)

## Implementation Tasks

### 1. Core Infrastructure Changes
- [ ] Add file handle management properties to ReportManager class
  - `testFileHandles: Map<string, fs.FileHandle>`
  - `testFileBuffers: Map<string, string[]>`
  - `debouncedFileWrites: Map<string, (() => void) & { cancel: () => void }>`
- [ ] Create logs directory during initialization (in `initialize()` method)
- [ ] Import required debounce for per-file debouncing

### 2. File Handle Lifecycle Management
- [ ] Modify `ensureTestFileRegistered()` to open file handles immediately
  - Open file handle when test file is first registered
  - Write file header immediately
  - Create per-file debounced write function (100ms delay, 500ms max)
- [ ] Add `flushFileBuffer()` method to write buffered content to file
- [ ] Update `finalize()` to properly close all file handles
  - Cancel all pending debounced writes
  - Flush all buffers
  - Close all file handles
  - Clear all maps

### 3. Incremental Writing Logic
- [ ] Modify `appendToLogFile()` to buffer output for incremental writing
  - Add output to per-file buffer
  - Trigger debounced write for the file
  - Remove in-memory collections (testFileOutputs, testCaseOutputs)
- [ ] Handle test case boundaries in incremental output
  - Write test case markers when test starts/completes
  - Ensure output is properly attributed to test cases
- [ ] Remove `parseOutputIntoTestLogs()` method entirely
  - Delete the method from ReportManager
  - Remove call to it from `finalize()`
  - Remove related helper methods if any

### 4. Error Handling & Resilience
- [ ] Add try-catch blocks around file operations
- [ ] Log warnings if file writes fail but don't crash
- [ ] Ensure data remains in memory if file write fails
- [ ] Handle ENOENT errors if logs directory is deleted mid-run

### 5. Testing
- [ ] Update unit tests in `src/ReportManager.test.ts`
  - Test file handle creation on registration
  - Test incremental writing with mocked timers
  - Test buffer flushing
  - Test file handle cleanup
- [ ] Create new integration test `tests/integration/interrupted-run.test.ts`
  - Start test run with multiple test files
  - Kill process (SIGTERM) after first test file completes
  - Verify `.3pio/runs/*/logs/` directory exists
  - Verify individual log files exist for started test files
  - Verify log files contain partial output
  - Follow default verifications from README where applicable
- [ ] Update integration tests in `tests/integration/full-flow.test.ts`
  - Add test for interrupted run (simulate SIGINT)
  - Verify partial log files are created
- [ ] Create new system test `tests/system/incremental-logs.test.ts`
  - Test with real Jest/Vitest runs
  - Verify logs appear before test completion
  - Test Ctrl+C scenario
- [ ] Update existing tests that may be affected:
  - `tests/integration/test-case-reporting.test.ts` - May need timing adjustments
  - `tests/system/console-output/console-output.test.ts` - Verify output format unchanged

### 6. Documentation
- [ ] Update `CLAUDE.md` with new incremental writing behavior
- [ ] Update `docs/architecture/report-manager.md` with new design
- [ ] Add entry to `docs/design-decisions.md` explaining the change and rationale

## Failure Modes & Mitigations

### 1. File Handle Exhaustion
**Risk**: Opening too many file handles for large test suites
**Mitigation**:
- Monitor handle count in tests
- Consider handle pooling if > 100 test files
- Document OS limits in README

### 2. Data Loss During Crash
**Risk**: Buffered data not written if process crashes
**Mitigation**:
- Keep buffer size small (flush every 100ms)
- Max wait of 500ms ensures data is written quickly

### 3. Concurrent Write Conflicts
**Risk**: Multiple writes to same file causing corruption
**Mitigation**:
- Single file handle per file
- Debouncing prevents write storms
- Sequential writes through buffer

### 4. Disk Space Issues
**Risk**: Disk full during test run
**Mitigation**:
- Log warning but don't fail test run
- Test with limited disk space

### 5. Performance Degradation
**Risk**: Too many I/O operations slowing down tests
**Mitigation**:
- Debouncing (100ms/500ms max)
- Buffering to batch writes

## Testing Strategy

### Unit Tests (src/ReportManager.test.ts)
```typescript
describe('ReportManager - Incremental Writing', () => {
  it('should create log file immediately when test file is registered')
  it('should write buffered output within debounce window')
  it('should flush all buffers on finalize')
  it('should handle file write errors gracefully')
  it('should clean up file handles on finalize')
})
```

### Integration Tests
1. **Interruption Test** (`tests/integration/interrupted-run.test.ts`)
   ```typescript
   describe('Interrupted test runs', () => {
     it('should create individual log files even when killed mid-run', async () => {
       // Use fixture with multiple test files (e.g., basic-jest)
       // Spawn 3pio process
       // Wait for first test file to complete (monitor IPC or output)
       // Send SIGTERM to process
       // Wait for process to exit
       // Verify:
       //   - logs/ directory exists
       //   - At least one .log file exists in logs/
       //   - Log file has proper header format
       //   - Log file contains test output
       //   - test-run.md shows RUNNING status
     });
   });
   ```

2. **Full Flow Update** (`tests/integration/full-flow.test.ts`)
   - Add similar interruption test with mocked components
   - Verify partial state is preserved

3. **Performance Test** (`tests/integration/performance.test.ts`)
   - Run with 50+ test files
   - Measure time overhead
   - Verify < 5% performance impact

### System Tests
1. **Real Jest Run** (`tests/system/incremental-logs.test.ts`)
   - Run actual Jest tests
   - Read log files while tests are running
   - Verify content appears incrementally

2. **Real Vitest Run** (`tests/system/incremental-logs.test.ts`)
   - Same as Jest but with Vitest
   - Verify both adapters work correctly

### Manual Testing Checklist
- [ ] Run `3pio run jest` and Ctrl+C after first test - verify logs exist
- [ ] Run `3pio run vitest` with 20+ test files - verify no handle errors
- [ ] Run with `--debug` flag - verify no excessive I/O warnings
- [ ] Delete logs directory while running - verify graceful handling
- [ ] Run on system with low ulimit - verify appropriate error message
