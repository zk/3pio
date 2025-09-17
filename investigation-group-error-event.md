# Investigation: Adding Group Error Event to 3pio

## Problem Summary

3pio's Go adapter has a critical issue where package-level failures (setup/compilation errors) report incorrect totals in `testGroupResult` events, leading to confusion between console output and final reports.

## Current Issue Analysis

### The Bug in Detail

**Example from etcd tests:**
```json
// IPC Event shows:
{"eventType":"testGroupResult","payload":{"groupName":"go.etcd.io/etcd/tests/v3/common","status":"FAIL","totals":{"failed":0,"passed":0,"skipped":0,"total":0}}}

// But Go test actually failed:
{"Action":"fail","Package":"go.etcd.io/etcd/tests/v3/common","Elapsed":0.856}
```

**Root Cause:** In `gotest.go:608-629`, the adapter calculates totals by counting individual test case events, but when packages fail at setup level, no test case events are generated, resulting in `{0,0,0,0}` totals despite `"status":"FAIL"`.

### Current Adapter Behavior Comparison

| Adapter | Setup Error Handling | Event Used |
|---------|---------------------|------------|
| **Go** | ❌ Broken totals in `testGroupResult` | None |
| **pytest** | ✅ Has `collectionError` event | `collectionError` |
| **Jest** | ✅ Calculates totals correctly | `testGroupResult` |
| **Vitest** | ✅ Handles errors properly | `testGroupResult` |

## Proposed Solution: `testGroupError` Event

### New Event Schema

```json
{
  "eventType": "testGroupError",
  "payload": {
    "groupName": "go.etcd.io/etcd/tests/v3/common",
    "parentNames": [],
    "errorType": "SETUP_FAILURE",
    "duration": 856,
    "error": {
      "message": "No test mode selected, please selected either e2e mode with \"--tags e2e\" or integration mode with \"--tags integration\"",
      "phase": "setup"
    }
  }
}
```

### Error Types

- `SETUP_FAILURE`: Package failed before tests could run (Go, build errors)
- `COLLECTION_FAILURE`: Test discovery/collection failed (pytest style)
- `COMPILATION_FAILURE`: Code compilation/transpilation failed
- `IMPORT_FAILURE`: Module import/dependency errors
- `CONFIGURATION_FAILURE`: Invalid test configuration

### Event Flow Changes

**Current (Broken):**
```
testGroupDiscovered → testGroupStart → [no test events] → testGroupResult{totals:{0,0,0,0}}
```

**Proposed:**
```
testGroupDiscovered → testGroupStart → testGroupError → testGroupResult{setupFailed:true}
```

### Modified `testGroupResult` Schema

Add optional `setupFailed` flag:
```json
{
  "eventType": "testGroupResult",
  "payload": {
    "groupName": "go.etcd.io/etcd/tests/v3/common",
    "parentNames": [],
    "status": "FAIL",
    "duration": 856,
    "setupFailed": true,  // NEW FIELD
    "totals": {
      "failed": 0,
      "passed": 0,
      "skipped": 0,
      "total": 0
    }
  }
}
```

## Implementation Plan

### Phase 1: Schema Updates

1. **Update IPC event documentation** in `docs/architecture/test-runner-adapters.md`
2. **Add `testGroupError` event type** to schema
3. **Update `testGroupResult` schema** with optional `setupFailed` field

### Phase 2: Go Adapter Fix

**File:** `internal/runner/definitions/gotest.go`

**Changes in `handlePackageResult()` method:**

```go
// NEW: Detect setup failures
if event.Action == "fail" && len(g.packageGroups[event.Package].Tests) == 0 {
    // This is a setup failure - send testGroupError
    g.sendGroupError(event.Package, []string{}, "SETUP_FAILURE", event.Elapsed,
        "Package failed before tests could run")

    // Mark setupFailed in testGroupResult
    totals["setupFailed"] = true
}
```

**New method to add:**

```go
func (g *GoTestDefinition) sendGroupError(groupName string, parentNames []string,
    errorType string, duration float64, message string) {
    event := map[string]interface{}{
        "eventType": "testGroupError",
        "payload": map[string]interface{}{
            "groupName":   groupName,
            "parentNames": parentNames,
            "errorType":   errorType,
            "duration":    duration * 1000,
            "error": map[string]interface{}{
                "message": message,
                "phase":   "setup",
            },
        },
    }
    if err := g.ipcWriter.WriteEvent(event); err != nil {
        g.logger.Error("Failed to send testGroupError: %v", err)
    }
}
```

### Phase 3: Report Generator Updates

**File:** `internal/report/` (report generation logic)

1. **Handle `testGroupError` events** in report generation
2. **Display setup failures clearly** in reports
3. **Fix console vs report metric confusion**

### Phase 4: Other Adapters (Optional)

1. **Migrate pytest's `collectionError`** to `testGroupError` for consistency
2. **Add `testGroupError` support** to Jest/Vitest for compilation failures

## Benefits

### 1. **Accurate Reporting**
- Console and report metrics will align
- Clear distinction between test failures and setup failures
- Proper totals calculation

### 2. **Better User Experience**
```bash
# Current confusing output:
"Results: 21 passed, 3 failed" vs "131 test cases failed"

# Proposed clear output:
"Results: 21 passed, 2 failed, 1 setup failed" vs "131 test cases failed, 1 setup failure"
```

### 3. **Enhanced Debugging**
- Setup failures clearly identified
- Error messages preserved and categorized
- Phase information helps debugging

### 4. **Consistency Across Adapters**
- Unified error handling approach
- Common event schema for all test runners

## Risk Assessment

### Low Risk
- **Backward compatible:** Existing events unchanged
- **Additive change:** New event is optional
- **Isolated to Go adapter:** Other adapters unaffected initially

### Testing Strategy
1. **Unit tests:** Test setup failure detection
2. **Integration tests:** Verify event generation with failing packages
3. **Real-world validation:** Test against etcd, kubernetes projects

## Alternative Approaches Considered

### 1. **Fix totals calculation only**
❌ **Rejected:** Doesn't provide error details, loses information

### 2. **Add error field to testGroupResult**
❌ **Rejected:** Conflates success reporting with error reporting

### 3. **Use existing groupStderr for errors**
❌ **Rejected:** Stdout/stderr is for output capture, not structured errors

## Conclusion

Adding `testGroupError` event is the cleanest solution that:
- ✅ Fixes the Go adapter totals bug
- ✅ Provides rich error information
- ✅ Maintains backward compatibility
- ✅ Establishes pattern for other adapters
- ✅ Improves user experience significantly

This change addresses the core issue found in etcd testing while establishing a robust foundation for error handling across all test runners.