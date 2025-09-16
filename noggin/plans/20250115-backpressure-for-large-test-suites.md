# Plan: Add Backpressure for Large Test Suites

**Date**: 2025-01-15
**Problem**: IPC event processing pipeline gets overwhelmed with large test suites (200k+ tests)
**Root Cause**: Go channel congestion - events are read faster than they can be processed

## Problem Analysis

### Current Architecture
```
IPC File → File Watcher → Parse Event → Channel (1s timeout) → Consumer → Report Generator
```

### Bottleneck Identified
- **Location**: `internal/ipc/manager.go:250-255`
- **Issue**: Channel send times out after 1 second when consumer can't keep up
- **Scale**: Pandas generates ~105 events/second, overwhelming the pipeline
- **Impact**: Events are dropped, reports incomplete, but tests continue running

### Evidence
- IPC file grows to 47MB+ with 189k+ events
- Debug log shows successful test execution
- Error: `[ERROR] Timeout sending event: testCase`
- Tests complete successfully despite timeouts

## Solution: Implement Backpressure

### Option 1: Buffered Channel with Dynamic Sizing (Quick Fix)
**File**: `internal/ipc/manager.go`

```go
// Current: Unbuffered or small buffer
m.Events = make(chan Event)

// Proposed: Large buffered channel
m.Events = make(chan Event, 10000)  // Handle burst of 10k events
```

**Pros**:
- Simple one-line change
- Handles bursts better

**Cons**:
- Still has a limit
- Doesn't solve root cause

### Option 2: Adaptive Reading Rate (Recommended)
**File**: `internal/ipc/manager.go`

Add backpressure by monitoring channel capacity:

```go
func (m *Manager) processLine(line []byte) {
    // Check channel capacity before reading more
    channelLoad := float64(len(m.Events)) / float64(cap(m.Events))

    if channelLoad > 0.8 {
        // Slow down reading when channel is 80% full
        time.Sleep(10 * time.Millisecond)
    }

    // Existing event parsing...
}
```

### Option 3: Blocking Send with No Timeout (Simplest)
**File**: `internal/ipc/manager.go`

Remove the timeout entirely - let the reader block naturally:

```go
// Current code with timeout
select {
case m.Events <- event:
    m.logger.Debug("Sent event: %s", eventType)
case <-time.After(time.Second):
    m.logger.Error("Timeout sending event: %s", eventType)
}

// Proposed: Blocking send (natural backpressure)
m.Events <- event
m.logger.Debug("Sent event: %s", eventType)
```

**This is the most elegant solution** - the file reader will naturally slow down when the consumer can't keep up.

### Option 4: Batched Event Processing
**File**: `internal/ipc/manager.go`

Batch events before sending to reduce channel operations:

```go
type EventBatch struct {
    Events []Event
}

// Collect events in batches of 100
batch := make([]Event, 0, 100)
for event := range incomingEvents {
    batch = append(batch, event)
    if len(batch) >= 100 {
        m.Events <- EventBatch{Events: batch}
        batch = make([]Event, 0, 100)
    }
}
```

## Recommended Implementation Plan

### Phase 1: Immediate Fix (Option 3)
1. Remove the timeout in `internal/ipc/manager.go`
2. Let the channel block naturally to create backpressure
3. Test with pandas to verify it handles 200k+ tests

### Phase 2: Optimize Channel Size (Option 1)
1. Increase channel buffer to 10,000 events
2. Monitor memory usage with large test suites
3. Adjust buffer size based on testing

### Phase 3: Performance Optimization (If Needed)
1. Implement batched processing (Option 4)
2. Add metrics to monitor event processing rate
3. Consider parallel consumers for event processing

## Root Cause Analysis - SOLVED

### The Real Problem: OS Buffer Not Flushed Before Python Exit
After extensive investigation with pandas test suite (200k+ tests):

1. **What appeared to happen**: IPC timeout errors, Go channel congestion
2. **What actually happened**: Python adapter successfully wrote all events, but 149 were lost
3. **Why they were lost**: Python's `flush()` only flushes to OS buffer, not to disk
4. **When Python exited**: Process terminated at 17:21:24, OS buffer wasn't written to disk
5. **Data loss**: 1,040 group results sent, only 891 made it to disk

### Evidence:
- Debug log: Shows 193,946 "Sending" messages logged successfully by adapter
- IPC file: Contains only 189,533 events (4,413 missing!)
- Lost events breakdown:
  - 287 group discoveries lost
  - 149 group results lost
  - ~4,000 test cases lost
- Timing: Python adapter stopped at 17:21:24.326649 when Python exited
- No errors: No exceptions logged, send_event() completed successfully each time
- OS buffering: 4,413 events were in OS buffer when Python exited, never written to disk

## Additional Issues Discovered

### Misleading Log Messages
The Go IPC manager logs "Sent event" when it's actually **received and processing** an event from the IPC file:
```go
// Current misleading message:
m.logger.Debug("Sent event: %s", eventType)

// Should be:
m.logger.Debug("Processing event: %s", eventType)
// or
m.logger.Debug("Received event from IPC: %s", eventType)
```

### Python Adapter Issues Found
1. **No OS-level sync**: `flush()` only flushes Python buffer, not OS buffer:
```python
with open(self.ipc_path, 'a') as f:
    f.write(json.dumps(event) + '\n')
    f.flush()  # Only flushes Python buffer to OS
    # Missing: os.fsync(f.fileno()) to force OS write to disk
```

2. **File reopening for each event** - inefficient (200,000+ opens):
```python
# Current: Opens file for EVERY event
with open(self.ipc_path, 'a') as f:
    f.write(json.dumps(event) + '\n')
```

3. **Silent exception handlers** hide critical failures:
```python
except Exception:
    pass  # Silent failure - no debugging possible!
```

**Root Cause Found**: When Python exited at 17:21:24, there were 149 events still in the OS buffer that never got written to disk. The adapter successfully called send_event() 1,040 times, but only 891 made it to the file because Python exited before the OS flushed its buffer.

## Implementation Steps

### URGENT Step 0: Fix Python Adapter File Handling (CRITICAL)
The Python adapter must be fixed to handle large test suites:

```python
class ThreepioReporter:
    def __init__(self, ipc_path: str):
        self.ipc_path = ipc_path
        self.ipc_file = None
        self._open_ipc_file()

    def _open_ipc_file(self):
        """Open IPC file once and keep it open"""
        try:
            self.ipc_file = open(self.ipc_path, 'a', buffering=1)  # Line buffering
        except Exception as e:
            self._log_error(f"Failed to open IPC file: {e}")

    def send_event(self, event_type: str, payload: Dict[str, Any]) -> None:
        """Send an event to the IPC file."""
        event = {
            "eventType": event_type,
            "payload": payload,
            "timestamp": time.time()
        }

        try:
            if not self.ipc_file or self.ipc_file.closed:
                self._open_ipc_file()

            json_line = json.dumps(event) + '\n'
            self.ipc_file.write(json_line)
            self.ipc_file.flush()
            # CRITICAL: Force OS to write to disk
            os.fsync(self.ipc_file.fileno())

        except Exception as e:
            self._log_error(f"Failed to send IPC event: {e}")
            # Try to reopen on next write
            self.ipc_file = None

    def cleanup(self):
        """Close the IPC file"""
        if self.ipc_file and not self.ipc_file.closed:
            self.ipc_file.close()
```

### Step 0.5: Fix Logging Terminology (5 minutes)
```diff
// internal/ipc/manager.go
- m.logger.Debug("Sent event: %s", eventType)
+ m.logger.Debug("Processing IPC event: %s", eventType)
```

### Step 1: Remove Timeout (5 minutes)
```diff
// internal/ipc/manager.go
- select {
- case m.Events <- event:
-     m.logger.Debug("Sent event: %s", eventType)
- case <-time.After(time.Second):
-     m.logger.Error("Timeout sending event: %s", eventType)
- }
+ m.Events <- event
+ m.logger.Debug("Sent event: %s", eventType)
```

### Step 2: Increase Buffer Size (5 minutes)
```diff
// internal/ipc/manager.go (in NewManager function)
- Events: make(chan Event),
+ Events: make(chan Event, 10000),  // Buffer for large test suites
```

### Step 3: Test with Pandas - CRITICAL VALIDATION (45 minutes)
1. Build 3pio with changes
2. Run full pandas test suite: `pytest pandas/tests/`
3. Verify:
   - No timeout errors
   - All 193,946+ events make it to IPC file
   - Compare: Events sent in debug log vs events in IPC file
   - Memory usage remains reasonable
4. Confirm test report completeness:
   - All 891+ test files have results
   - All 207,418+ test cases accounted for
5. Run twice to ensure consistency

### Step 4: Add Configuration (Optional, 15 minutes)
```go
// Add environment variable for buffer size
bufferSize := 1000  // default
if size := os.Getenv("THREEPIO_EVENT_BUFFER_SIZE"); size != "" {
    if s, err := strconv.Atoi(size); err == nil {
        bufferSize = s
    }
}
Events: make(chan Event, bufferSize),
```

## Success Criteria
- [ ] No "Timeout sending event" errors with 200k+ tests
- [ ] Complete test reports generated for pandas
- [ ] All 193,946+ events successfully written to IPC file (0 lost)
- [ ] Memory usage remains reasonable (<1GB for pandas)
- [ ] No performance regression for small test suites
- [ ] Python adapter successfully writes all events to IPC file with os.fsync()
- [ ] No silent failures in adapters
- [ ] Pandas test suite runs to completion twice with consistent results

## Additional Tasks

### Task 1: Audit Go Logging for Accuracy
Review all Go logging messages to ensure they accurately describe what's happening:
- `internal/ipc/manager.go` - "Sent event" → "Processing event"
- `internal/orchestrator/orchestrator.go` - Check all log messages
- `internal/report/manager.go` - Verify report generation logs
- `internal/runner/runner.go` - Check runner status logs

**Acceptance Criteria:**
- [ ] All log messages accurately describe the action being performed
- [ ] Distinguish between reading, processing, and writing operations
- [ ] Use consistent terminology across the codebase

### Task 2: Improve Python Adapter Error Handling
Fix silent failures in pytest adapter:
```python
# Current problematic code:
except Exception:
    pass  # Silent failure!

# Improved version:
except Exception as e:
    # Always attempt to log the error
    try:
        self._log_error(f"Unexpected error: {e}")
    except:
        # Last resort - print to stderr
        import sys
        print(f"CRITICAL: Adapter error: {e}", file=sys.stderr)
```

### Task 3: Add IPC Write Monitoring
Add counters to track IPC writes in Python adapter:
```python
def __init__(self, ipc_path: str):
    self.ipc_path = ipc_path
    self.events_written = 0
    self.write_failures = 0
    # Log stats every 1000 events
    self.log_interval = 1000

def send_event(self, event_type: str, payload: Dict[str, Any]) -> None:
    try:
        # ... write event ...
        self.events_written += 1
        if self.events_written % self.log_interval == 0:
            self._log_info(f"IPC stats: {self.events_written} written, {self.write_failures} failed")
    except Exception as e:
        self.write_failures += 1
        self._log_error(f"IPC write failed: {e}")
```

## Testing Plan
1. Small suite: 100 tests (should be unaffected)
2. Medium suite: 1,000 tests (should be unaffected)
3. Large suite: 10,000 tests (should work without timeouts)
4. **Massive suite: Pandas with 207,418 tests - CRITICAL TEST**
   - Must verify all 193,946+ events are written
   - Check: `grep -c "Sending" debug.log` == `wc -l ipc.jsonl`
   - No events lost due to missing os.fsync()

## Notes
- The blocking send (Option 3) is the most elegant solution
- Natural backpressure prevents overwhelming the system
- No events are dropped - they just process more slowly
- This maintains data integrity while handling scale