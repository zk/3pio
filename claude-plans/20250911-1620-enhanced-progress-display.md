# Enhanced Progress Display System - Implementation Plan

## Overview
Create a clean, agent-friendly progress system with distinct phases and file-by-file reporting to improve user experience during long-running test suites.

## Problem Statement
When running `3pio pnpm test` on large test suites (like Mastra), users see no output for ~60 seconds during the test collection phase, creating a poor user experience. This plan addresses the issue by providing immediate feedback and collection phase visibility.

## Display States
```
# Existing startup (preserve current format)
Greetings! I will now execute the test command:
`pnpm test`

Full report: .3pio/runs/20250911T152436-perky-robo/test-run.md

Beginning test execution now...

# Collection phase (if supported by runner)
Collecting tests...
Found 150 test files

# Execution phase 
RUNNING    ./src/math.test.js
PASS       ./src/math.test.js
RUNNING    ./src/string.test.js  
FAIL       ./src/string.test.js
```

## Architecture

### 1. New IPC Events (Optional)
```json
{"eventType": "testCollectionStart", "payload": {}}
{"eventType": "testCollectionComplete", "payload": {"totalFiles": 150}}
```

### 2. Simple Progress State
```go
type ProgressState struct {
    Phase      string  // "starting", "collecting", "executing", "complete"  
    TotalFiles int     // Total files (for final summary only)
}
```

### 3. Enhanced Console Output Logic
```go
func (o *Orchestrator) handleConsoleOutput(event ipc.Event) {
    switch e := event.(type) {
    case ipc.TestCollectionStartEvent:
        fmt.Println("Collecting tests...")
        
    case ipc.TestCollectionCompleteEvent:
        fmt.Printf("Found %d test files\n\n", e.Payload.TotalFiles)
        o.progressState.TotalFiles = e.Payload.TotalFiles
        
    case ipc.TestFileStartEvent:
        fmt.Printf("RUNNING    %s\n", relativePath)
        
    case ipc.TestFileResultEvent:
        fmt.Printf("%s %s\n", status, relativePath)
    }
}
```

### 4. Preserve Existing Startup Output
```go
// IMPORTANT: Do not modify existing startup output in orchestrator
// Current greeting format must be preserved:
// "Greetings! I will now execute the test command:"
// "`{command}`"
// ""
// "Full report: {reportPath}"
// ""
// "Beginning test execution now..."

// Collection events will appear AFTER this existing output
```

## Implementation Steps

### Step 1: Add Minimal Progress Events
- Add `testCollectionStart` and `testCollectionComplete` to `internal/ipc/events.go`
- Optional events - no breaking changes to existing adapters

### Step 2: Preserve Existing Startup Format
- **DO NOT MODIFY** existing startup greeting in orchestrator
- Collection phase output will appear after "Beginning test execution now..."
- Maintain current spacing and formatting

### Step 3: Update Orchestrator Console Output
- Extend `handleConsoleOutput()` to handle new collection events
- Add proper spacing with newlines after collection complete

### Step 4: Update Vitest Adapter
```javascript
onInit(ctx) {
    // Send collection start event
    IPCSender.sendEvent({
        eventType: "testCollectionStart",
        payload: {}
    });
    
    // Existing initialization code...
}

onCollected(files) {
    // Send collection complete with file count
    IPCSender.sendEvent({
        eventType: "testCollectionComplete",
        payload: { totalFiles: files?.length || 0 }
    });
    
    // Existing collected logic...
}
```

### Step 5: Clean Phase Transitions
- "Greetings! ..." → "Beginning test execution now..." → "Collecting tests..." → "Found X test files" → "RUNNING ./file.js"
- Each phase is distinct and clear  
- No progress bars or percentages
- Preserve existing startup format exactly

## Test Runner Collection Support Analysis

Based on investigation of existing adapters, here's the collection event support by runner:

| Test Runner | Collection Start | Collection Complete | File Count Available | Status |
|-------------|------------------|-------------------|---------------------|---------|
| **Vitest**  | ✅ `onInit()`    | ✅ `onCollected()` | ✅ Yes | Needs implementation |
| **pytest** | ✅ Already implemented | ✅ Already implemented | ✅ Yes | **Ready to use!** |
| **Jest**    | ✅ `onRunStart()` | ❌ No hook available | ❌ No | Partial support only |

### Detailed Analysis

#### Jest Adapter Collection Support
**Available Jest Reporter Hooks (Verified from Source):**
```typescript
// Complete Jest Reporter interface from jest/packages/jest-reporters/src/types.ts
onRunStart?: (results: AggregatedResult, options: ReporterOnStartOptions) => Promise<void> | void;
onTestStart?: (test: Test) => Promise<void> | void;
onTestFileStart?: (test: Test) => Promise<void> | void;
onTestCaseStart?: (test: Test, testCaseStartInfo: Circus.TestCaseStartInfo) => Promise<void> | void;
onTestCaseResult?: (test: Test, testCaseResult: TestCaseResult) => Promise<void> | void;
onTestFileResult?: (test: Test, testResult: TestResult, aggregatedResult: AggregatedResult) => Promise<void> | void;
onTestResult?: (test: Test, testResult: TestResult, aggregatedResult: AggregatedResult) => Promise<void> | void;
onRunComplete?: (testContexts: Set<TestContext>, results: AggregatedResult) => Promise<void> | void;
getLastError?: () => Error | void;
```

**Collection Phase Support: ❌ Limited (Confirmed)**
- **No explicit collection phase hooks** - Jest reporter interface has no collection-specific methods
- Jest discovery happens internally during `onRunStart()` without separate reporter events
- `onRunStart()` is called after Jest has already collected tests, not before
- Jest immediately begins test execution after collection, without a distinct phase separation

**Recommendation for Jest:**
- Send `testCollectionStart` in `onRunStart()` (representing the combined collection+start phase)
- **Cannot** send `testCollectionComplete` with file count (no appropriate hook available)
- Falls back to showing individual `RUNNING ./file.js` as tests start via `onTestFileStart()`

#### pytest Adapter Collection Support  
**Available pytest Hooks:**
- `pytest_configure()` - Called when pytest starts (already implemented)
- `pytest_collectreport()` - Called during collection for each item
- `pytest_collection_finish(session)` - Called when collection completes ✅
- `pytest_runtest_protocol()` - Called when test execution starts

**Collection Phase Support: ✅ Excellent**
- pytest already sends `collectionStart` in `pytest_configure()`
- pytest already sends `collectionFinish` with test count in `pytest_collection_finish()`
- **pytest already supports the collection events we need!**

**Current pytest Events:**
```python
# In pytest_configure():
_reporter.send_event("collectionStart", {"phase": "collection"})

# In pytest_collection_finish():  
_reporter.send_event("collectionFinish", {
    "collected": session.testscollected if hasattr(session, 'testscollected') else 0
})
```

#### Vitest Adapter Collection Support
**Available Vitest Reporter Hooks:**
- `onInit(ctx)` - Called when Vitest initializes
- `onPathsCollected(paths)` - Called with discovered test files
- `onCollected(files)` - Called when collection is complete
- `onTestFileStart(file)` - Called when test file execution starts

**Collection Phase Support: ✅ Excellent**
- Can send `testCollectionStart` in `onInit()`
- Can send `testCollectionComplete` in `onCollected()` with file count
- Full support for enhanced progress display

## Compatibility Strategy

### Implementation Priority by Runner
1. **pytest**: ✅ **Already supported** - just needs orchestrator event handlers
2. **Vitest**: 🔄 **Needs implementation** - add collection events to adapter
3. **Jest**: ⚠️ **Partial support** - can only show collection start, not completion

### Progressive Enhancement
1. **Phase 1**: Add orchestrator support for existing pytest events
2. **Phase 2**: Add Vitest collection events  
3. **Phase 3**: Add Jest partial collection support
4. **Future**: Other runners can adopt collection events as needed

## Benefits for Agent Friendliness

1. **Clear Phases**: Distinct states are easy to parse
2. **File-by-File**: Each test file gets individual RUNNING/PASS/FAIL lines
3. **No Complex Progress**: No percentages or changing numbers to parse
4. **Predictable Output**: Consistent format across all runners
5. **Backward Compatible**: Existing behavior preserved
6. **Immediate Feedback**: Users know something is happening right away

## Testing Strategy

### Integration Tests
- Test with Vitest (enhanced progress with collection phase)
- Test with Jest (fallback behavior, no collection phase) 
- Test with pytest (fallback behavior, no collection phase)
- Test interrupted runs (Ctrl+C during collection)

### Edge Cases
- No collection events → graceful fallback to current behavior
- Collection without execution → handle timeout scenarios
- Very large test suites → ensure performance is maintained

## Final Output Example
```
Greetings! I will now execute the test command:
`pnpm test`

Full report: .3pio/runs/20250911T152436-perky-robo/test-run.md

Beginning test execution now...

Collecting tests...
Found 150 test files

RUNNING    ./src/math.test.js
PASS       ./src/math.test.js
RUNNING    ./src/string.test.js
FAIL       ./src/string.test.js
RUNNING    ./src/utils.test.js
PASS       ./src/utils.test.js

Test failures! This is madness!
FAIL 1 file, PASS 2 files
```

## Critical Discovery: Vitest Parallel Mode Progress

### The Problem
When Vitest runs in parallel mode (default for performance), the traditional `onTestFileStart`/`onTestFileResult` hooks are NOT called in real-time. They're batched until the end, causing no progress feedback during execution.

### The Solution  
Use Vitest V3 module hooks which ARE called in real-time:
- `onTestModuleCollected` - Called when a test file is queued (use for `testFileStart`)
- `onTestModuleEnd` - Called when a test file completes (use for `testFileResult`)
- Use `testModule.moduleId` (not `filepath`) for the file path

These hooks work because the main process receives module-level events from workers as they complete, providing real-time progress even in parallel mode.

### Key Code Changes
```javascript
onTestModuleCollected(testModule) {
  const filePath = testModule?.moduleId; // Note: moduleId, not filepath
  if (filePath) {
    IPCSender.sendEvent({
      eventType: "testFileStart",
      payload: { filePath }
    });
  }
}

onTestModuleEnd(testModule) {
  const filePath = testModule?.moduleId;
  if (filePath) {
    // Determine status from module.children test results
    IPCSender.sendEvent({
      eventType: "testFileResult", 
      payload: { filePath, status, failedTests }
    });
  }
}
```

## Implementation Checklist

### Phase 1: Core Infrastructure ✅ Foundation

#### 1.1 Add New IPC Event Types
- [x] Add `TestCollectionStartEvent` to `internal/ipc/events.go` (Already existed as CollectionStartEvent)
- [x] Add `TestCollectionCompleteEvent` to `internal/ipc/events.go` (Already existed as CollectionFinishEvent)
- [x] Add event type constants for new events (Already existed)
- [x] Update event parser to handle new event types (Already working)
- [x] Ensure backward compatibility (existing events unchanged)

#### 1.2 Update Orchestrator Progress State  
- [x] Add `ProgressState` struct to orchestrator (Added lastCollected field for deduplication)
  ```go
  type ProgressState struct {
      Phase      string  // "starting", "collecting", "executing", "complete"  
      TotalFiles int     // Total files (for final summary only)
  }
  ```
- [x] Initialize progress state in orchestrator constructor
- [x] Add progress state field to orchestrator struct

#### 1.3 Enhance Console Output Handler
- [x] Add `TestCollectionStartEvent` handler to `handleConsoleOutput()`
  ```go
  case ipc.TestCollectionStartEvent:
      fmt.Println("Collecting tests...")
  ```
- [x] Add `TestCollectionCompleteEvent` handler to `handleConsoleOutput()`
  ```go
  case ipc.TestCollectionCompleteEvent:
      fmt.Printf("Found %d test files\n\n", e.Payload.TotalFiles)
      o.progressState.TotalFiles = e.Payload.TotalFiles
  ```
- [x] Ensure proper spacing and formatting
- [x] **DO NOT MODIFY** existing startup greeting format (Preserved)

#### 1.4 Handle Existing pytest Events
- [x] Add `collectionStart` event handler (map to `TestCollectionStartEvent`)
- [x] Add `collectionFinish` event handler (map to `TestCollectionCompleteEvent`)
- [x] Test with existing pytest adapter to verify collection display

### Phase 2: pytest Integration ✅ Ready to Test

#### 2.1 Verify pytest Support
- [x] Test with basic pytest project to confirm collection events work
- [x] Verify "Collecting tests..." appears after "Beginning test execution now..."
- [x] Verify "Found X test files" appears with correct count (Fixed to use session.items)
- [ ] Test collection error handling (malformed test files)

#### 2.2 pytest Edge Cases
- [ ] Test with zero test files discovered
- [ ] Test with collection errors (import failures)
- [ ] Test with interrupted collection (Ctrl+C during discovery)
- [ ] Test with very large test suites (100+ files)

### Phase 3: Vitest Integration 🔄 Needs Implementation

#### 3.1 Update Vitest Adapter
- [x] Add collection start event in `onInit()`
  ```javascript
  onInit(ctx) {
      IPCSender.sendEvent({
          eventType: "testCollectionStart",
          payload: {}
      });
      // ... existing code
  }
  ```
- [x] Add collection complete event in `onCollected()`
  ```javascript
  onCollected(files) {
      IPCSender.sendEvent({
          eventType: "testCollectionComplete",
          payload: { totalFiles: files?.length || 0 }
      });
      // ... existing code
  }
  ```
- [x] Test that existing Vitest functionality remains unchanged
- [x] Rebuild adapters: `make adapters && make build`

#### 3.2 Vitest Integration Testing
- [x] Test with basic Vitest project (tests/fixtures/basic-vitest)
- [x] Test with Mastra test suite (long collection phase)
- [x] Verify collection progress shows during slow discovery
- [ ] Test with parallel execution (multiple workers)
- [ ] Test with file filtering (`--testPathPattern`)

#### 3.3 Vitest Edge Cases  
- [ ] Test with `vitest list` command (should not run collection)
- [ ] Test with watch mode (if applicable)
- [ ] Test with coverage mode
- [ ] Test with specific file arguments

### Phase 4: Jest Integration ⚠️ Partial Support

#### 4.1 Update Jest Adapter
- [x] Add collection start event in `onRunStart()`
  ```javascript
  onRunStart() {
      IPCSender.sendEvent({
          eventType: "testCollectionStart", 
          payload: {}
      });
      // ... existing code
  }
  ```
- [x] **NOTE**: Jest cannot provide collection complete with file count (Confirmed)
- [x] Test that Jest falls back to individual file progress
- [x] Rebuild adapters: `make adapters && make build`

#### 4.2 Jest Integration Testing
- [x] Test with basic Jest project (tests/fixtures/basic-jest)
- [x] Verify "Collecting tests..." appears
- [x] Verify individual `RUNNING ./file.js` still works
- [ ] Test with parallel execution (`--maxWorkers`)
- [ ] Test with test filtering (`--testPathPattern`)

### Phase 5: Cross-Runner Testing 🧪 Integration

#### 5.1 Compatibility Testing
- [x] Test all three runners preserve existing behavior
- [x] Test runners without collection support gracefully degrade
- [x] Test mixed usage (switching between runners)
- [x] Performance testing (no significant slowdown)

#### 5.2 User Experience Testing
- [x] Test with small test suites (1-10 files) - should not feel cluttered
- [ ] Test with medium test suites (10-50 files) - useful progress
- [x] Test with large test suites (50+ files) - essential progress (Mastra)
- [ ] Test interruption scenarios (Ctrl+C during collection)

#### 5.3 Agent Friendliness Validation
- [x] Verify output is easily parseable by agents
- [x] Confirm no unexpected format changes
- [x] Test phase transitions are clear and distinct
- [x] Validate no complex progress indicators (percentages, etc.)

### Phase 6: Documentation & Polish 📚 Finalization

#### 6.1 Update Documentation
- [ ] Update architecture docs with new event types
- [ ] Document collection phase behavior per runner
- [ ] Update debugging guide for new events
- [ ] Add troubleshooting section for collection issues

#### 6.2 Code Quality
- [ ] Add unit tests for new IPC event types
- [ ] Add integration tests for collection progress
- [ ] Update error handling for collection failures  
- [ ] Code review and cleanup

#### 6.3 Future Extensibility
- [ ] Document pattern for other test runners to adopt
- [ ] Consider additional progress events (if needed)
- [ ] Plan for potential collection progress indicators (future)

## Testing Commands

### Quick Development Testing
```bash
# Build with new changes
make adapters && make build

# Test pytest (should already work)
cd tests/fixtures/basic-pytest
../../../build/3pio pytest

# Test Vitest (after implementation) 
cd tests/fixtures/basic-vitest
../../../build/3pio npx vitest run

# Test Jest (after implementation)
cd tests/fixtures/basic-jest  
../../../build/3pio npx jest

# Test with Mastra (the original problem case)
cd open-source/mastra/packages/core
../../../../build/3pio pnpm test
```

### Validation Checklist
- [x] All existing tests pass
- [x] No breaking changes to current behavior
- [x] Collection phase visible for supported runners
- [x] Graceful degradation for unsupported runners  
- [x] Performance impact minimal
- [x] User experience improved for long-running suites

## Success Criteria

✅ **Phase 1 Complete**: Infrastructure supports collection events  
✅ **Phase 2 Complete**: pytest shows collection progress  
✅ **Phase 3 Complete**: Vitest shows collection progress  
✅ **Phase 4 Complete**: Jest shows partial collection progress  
✅ **Phase 5 Complete**: All runners work without regressions  
🔄 **Phase 6 In Progress**: Documentation updates pending

**Final Goal**: Users running `3pio pnpm test` on Mastra see immediate collection feedback instead of 60 seconds of silence.

This approach provides immediate feedback during collection while maintaining the clean, parseable output format that agents can easily understand.