# Performance Optimizations

## Report Writing Debouncing (2025-01-15)

### Problem
When testing with massive test suites like pandas (50,000+ tests, 189,505 IPC events), 3pio's event processing became slower than test execution itself:
- Test execution: 21 minutes
- Event processing: 23 minutes (processing backlog after tests completed)
- Root cause: Writing to disk on every single event (189,505+ disk writes)

### Solution
Implemented debounced file writing to dramatically reduce disk I/O:

#### Changes Made
1. **Manager Report Debouncing** (`internal/report/manager.go`)
   - Added debounce timer with 200ms delay for main `test-run.md` writes
   - Batches multiple state updates into single disk write
   - Force flush on finalization ensures final state is written

2. **Group Report Debouncing** (`internal/report/group_manager.go`)
   - Already had 100ms debouncing, kept as-is
   - Works well for individual test file reports

### Implementation Details

The debouncing mechanism works as follows:
- Each state update schedules a write after 200ms
- If another update comes within that window, the timer resets
- This batches rapid updates into single disk writes
- On finalization, pending writes are cancelled and final state written immediately

Key code additions:
- `scheduleWrite()`: Sets up debounced write with timer
- `flushWrite()`: Executes pending write to disk
- Timer management in Manager struct
- Force flush in `Finalize()` method

### Expected Impact

For pandas test suite (189,505 events):
- **Before**: ~189,505 disk writes to `test-run.md`
- **After**: ~100-200 disk writes (99.9% reduction)
- **Time saved**: ~20+ minutes of processing overhead

### Configuration

Default timing:
- Main report (`test-run.md`): 200ms debounce delay
- Group reports: 100ms debounce delay

These values balance between:
- Responsiveness (reports update in near-realtime)
- Performance (minimal disk I/O)
- Data safety (no loss of state information)

### Testing

The changes are backwards compatible and have been tested with:
- Small test suites (immediate writes still happen on completion)
- Large test suites (dramatic reduction in disk I/O)
- Interrupted test runs (partial results still saved)

### Future Improvements

Potential areas for further optimization:
1. Event batching at IPC level (process multiple events per read)
2. Async console output (non-blocking progress updates)
3. Configurable debounce timing via environment variables
4. Memory-mapped files for extremely large reports