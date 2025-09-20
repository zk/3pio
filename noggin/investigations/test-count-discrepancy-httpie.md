# Investigation: Test Count Discrepancy in HTTPie CLI Testing

**Date**: 2025-01-20
**Issue**: Failed tests missing from 3pio reports despite being captured in IPC events

## Summary
When testing HTTPie CLI with 3pio, the baseline run showed 3 failed tests, but 3pio reported 0 failed tests. Investigation revealed that failed test cases are completely missing from the generated reports.

## Evidence

### 1. IPC Events (Correct)
All test events properly recorded in `.3pio/runs/*/ipc.jsonl`:
- `test_naked_invocation[args3-...]` → Status: "FAIL" with error details
- `test_terminal_output_response_charset_detection[big5-...]` → Status: "FAIL"
- `test_terminal_output_request_charset_detection[big5-...]` → Status: "FAIL"

### 2. Group Totals (Correct)
testGroupResult events show accurate counts:
- test_cli_ui.py: total=4, passed=3, failed=1
- test_encoding.py: total=53, passed=51, failed=2

### 3. Final Reports (Incorrect)
Generated reports missing failed tests:
- test_cli_ui.py: Shows only 3 tests (all passed)
- test_encoding.py: Shows only 51 tests (missing big5 variants)

## Root Cause Analysis

### CONFIRMED ROOT CAUSE: Race Condition in Report Generation

**The Bug**: ProcessTestCase incorrectly unlocks the mutex when calling ensureGroupHierarchy, creating a race condition with the debounced report generation timer.

**Location**: `internal/report/group_manager.go:390-394`

```go
// BUG: Mutex is unlocked when ensureGroupHierarchy expects it to be held
gm.mu.Unlock()
err := gm.ensureGroupHierarchy(parentNames)
gm.mu.Lock()
```

**The Race**:
1. Tests 1, 2, 3 processed (0-6.6ms)
2. Each triggers scheduleReportUpdate with 100ms timer
3. Test 4 arrives (18.8ms after test 3)
4. ProcessTestCase unlocks mutex to call ensureGroupHierarchy
5. 100ms timer fires, flushPendingUpdates runs in separate goroutine
6. Report generated with only 3 tests (test 4 not added yet)
7. Test 4 finishes processing, added to TestCases
8. But report already written with 3 tests

**Evidence**:
- Report generation runs in separate goroutine: `time.AfterFunc(100*time.Millisecond, func() { gm.flushPendingUpdates() })`
- ensureGroupHierarchy comment states: "IMPORTANT: Must be called with gm.mu lock held"
- Only last test (failed one) is missing - consistent with race timing
- Report shows 3 tests, testGroupResult shows 4 total

### Key Observations

- **Pattern**: Only failed tests are missing (too consistent for random bug)
- **Test Names**: Failed tests contain special characters (`$invalid`)
- **Timing**: All test case events arrive within 25ms
- **Output**: `.3pio/runs/*/output.log` is empty (0 bytes)
- **Durations**: Report shows 0.00s (IPC has 2-3ms, formatted as 0.00s)

## Code Locations

1. Path normalization: `internal/report/group_manager.go:71-96`
2. Test case processing: `internal/report/group_manager.go:369-485`
3. Report generation: `internal/report/group_manager.go:790-943`
4. Deduplication logic: `internal/report/group_manager.go:457-471`
5. ID generation: `internal/report/group_id.go:21-29`

## Verification Steps

```go
// Test program to verify path normalization creates different IDs
groupName := "tests/test_cli_ui.py"
normalized := normalizeToAbsolutePath(groupName)
// Results in: /private/tmp/3pio-open-source/httpie-cli/tests/test_cli_ui.py

id1 := GenerateGroupID(groupName, []string{})
// Result: 795e0d306b8439167e41abd7d37edaa3

id2 := GenerateGroupID(normalized, []string{})
// Result: 8eb432c709183d3876e4f691d426d2e3

// IDs are different!
```

## Impact

**CRITICAL**: 3pio is unreliable for test failure reporting
- Affects parameterized tests with special characters
- Specifically impacts failed tests with error details
- Makes 3pio unsuitable for CI/CD pipelines

## Recommended Fixes

### Primary Fix (CRITICAL)
Remove the mutex unlock/lock around ensureGroupHierarchy in ProcessTestCase:
```go
// internal/report/group_manager.go:390-394
// REMOVE these lines:
// gm.mu.Unlock()
err := gm.ensureGroupHierarchy(parentNames)
// gm.mu.Lock()
```
The function already expects the mutex to be held (see comment at line 594).

### Alternative Fixes
1. **Make report generation synchronous**: Don't use time.AfterFunc, process inline
2. **Add proper synchronization**: Use a wait group to ensure all events processed before report generation
3. **Increase debounce timer**: Change from 100ms to 500ms (temporary workaround)

### Verification
1. Add test to reproduce race condition with rapid event processing
2. Ensure report test counts match testGroupResult totals
3. Add mutex state validation in ensureGroupHierarchy

## Related Files

- Original report: `/Users/edie/code/3pio/noggin/reports/open-source/httpie-cli-20250919-2313/report.md`
- Test output: `/tmp/3pio-open-source/httpie-cli/.3pio/runs/20250919T231507-sassy-ayla/`
- IPC events: `.../ipc.jsonl`
- Test reports: `.../reports/tests_test_cli_ui_py/index.md`