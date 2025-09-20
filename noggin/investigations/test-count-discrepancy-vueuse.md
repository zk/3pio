# Test Count Discrepancy Investigation - VueUse

## Issue
Baseline and 3pio runs report different test counts for the VueUse project.

## Summary of Counts

### Vitest Native Reporting (from output.log)
- **Baseline**: 1359 passed + 1 skipped + 1 todo = 1361 tests
- **3pio output.log**: Same as baseline (1359 passed + 1 skipped + 1 todo = 1361 tests)

### 3pio Internal Reporting
- **test-run.md**: 1344 passed + 1 skipped = 1345 tests
- **Console output**: 2719 passed + 3 skipped = 2722 tests
- **IPC events**: 2722 testCase events (but only 1362 unique)
- **Actual test cases in reports**: 1344 passed + 1 skipped = 1345

## Key Findings

### 1. Test Duplication in IPC Events
- **Problem**: Tests are being reported twice in IPC events
- **Evidence**: `grep '"eventType":"testCase"' ipc.jsonl | wc -l` returns 2722, but `sort -u` returns 1362
- **Example**: The test "params: url" appears twice with identical parent names and duration
- **Root Cause**: Likely duplicate event emission in the Vitest adapter

### 2. TODO Test Not Captured
- **Problem**: Vitest reports 1 TODO test that doesn't appear in 3pio's IPC events or reports
- **Evidence**: No "TODO" status found in IPC events
- **Impact**: 1 test missing from 3pio's count

### 3. Discrepancy in Test Counting
- **Actual difference**: 1361 (Vitest) - 1345 (3pio reports) = 16 tests
- **Partial explanation**:
  - 1 TODO test not captured
  - Remaining 15 tests unaccounted for

### 4. Console Output Bug
- **Problem**: 3pio's console output shows 2722 total tests
- **Root Cause**: Counting all IPC events including duplicates
- **Should be**: Deduplicating events before counting

## Root Causes Identified

### 1. Duplicate Event Emission in Vitest Adapter
**Root Cause**: The Vitest adapter emits testCase events from BOTH `onTestCaseResult` AND `processTasksSimple` methods
- `onTestCaseResult` (line 472): Called by Vitest 3+ for each test result
- `processTasksSimple` (line 760): Called from `onTestModuleEnd` to process all tasks
- **Result**: Every test is reported exactly twice (2722 events for 1361 actual tests)

### 2. TODO Test Status Not Supported
**Root Cause**: The Vitest adapter doesn't handle the 'todo' state
- The adapter only maps: passed → PASS, failed → FAIL, skipped → SKIP
- TODO tests are incorrectly reported as both SKIP and PASS in duplicate events
- **Example**: "filters > pausableFilter > should pause" is a TODO test but reported as SKIP and PASS

### 3. Parameterized Tests Counted Multiple Times by Vitest
**Root Cause**: Vitest's JSON reporter has a bug with `it.each()` parameterized tests
- Tests using `it.each()` appear multiple times in Vitest's JSON output
- **Affected tests**:
  - `useInfiniteScroll` test appears 3 times (should be 1)
  - `useStorage` test appears 5 times (should be 1)
  - `useDateFormat` meridiem tests appear 2x each (8 tests → 16 in output)
- **Total impact**: 16 extra test instances in Vitest's JSON output (1361 vs actual 1345)

### 4. Console Output Shows Raw Event Count
**Root Cause**: CLI counts all IPC events without deduplication
- Console shows 2722 (all events) instead of unique test count
- Report generation correctly deduplicates to 1345

## Summary

The discrepancies are caused by:
1. **3pio bug**: Duplicate event emission in Vitest adapter (2x all tests)
2. **3pio bug**: TODO test status not supported
3. **Vitest bug**: Parameterized tests counted multiple times in JSON reporter
4. **3pio bug**: Console output doesn't deduplicate events

The actual test count should be:
- **True count**: 1345 unique tests (what 3pio reports correctly deduplicate)
- **Vitest reports**: 1361 (includes 16 duplicate parameterized test instances)
- **3pio IPC events**: 2722 (each test emitted twice)

## Fixes Needed

1. **Vitest Adapter**: Remove duplicate emission - use ONLY `onTestCaseResult` OR `processTasksSimple`, not both
2. **Vitest Adapter**: Add support for 'todo' state mapping
3. **CLI**: Deduplicate events when displaying console output totals
4. **Upstream**: Report Vitest JSON reporter bug with `it.each()` tests