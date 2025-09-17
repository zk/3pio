# Error Message Construction for testGroupError Event

## Overview

The `testGroupError` event's error message will be constructed by capturing and buffering package-level output events, then assembling them into a structured error message when a setup failure is detected.

## Current vs Proposed Flow

### **Current (Broken) Flow**
```
Go JSON Output → handleOutput() → Debug log only → LOST ❌
```

### **Proposed (Fixed) Flow**
```
Go JSON Output → handleOutput() → Buffer package errors → handlePackageResult() → testGroupError with constructed message ✅
```

## Implementation Details

### 1. **Data Structure for Error Buffering**

Add to `GoTestDefinition` struct:
```go
type GoTestDefinition struct {
    // ... existing fields
    packageErrors map[string][]string  // NEW: Buffer package-level error output
    mu            sync.Mutex
}
```

### 2. **Enhanced handleOutput() Method**

**File:** `internal/runner/definitions/gotest.go:650-683`

**Current code:**
```go
func (g *GoTestDefinition) handleOutput(event *GoTestEvent) {
    // ... existing code for individual test output

    } else {
        // Package-level output - only logged, never captured
        g.logger.Debug("Package output for %s: %s", event.Package, strings.TrimSpace(event.Output))
    }
}
```

**Proposed enhancement:**
```go
func (g *GoTestDefinition) handleOutput(event *GoTestEvent) {
    g.mu.Lock()
    defer g.mu.Unlock()

    if event.Test != "" {
        // Individual test output (existing logic)
        key := fmt.Sprintf("%s/%s", event.Package, event.Test)
        if state, ok := g.testStates[key]; ok {
            state.Output = append(state.Output, event.Output)
        }
    } else {
        // Package-level output - NOW CAPTURED!

        // Initialize package error buffer if needed
        if g.packageErrors == nil {
            g.packageErrors = make(map[string][]string)
        }

        // Filter and capture relevant error lines
        output := strings.TrimSpace(event.Output)
        if g.isErrorOutput(output) {
            g.packageErrors[event.Package] = append(g.packageErrors[event.Package], output)
        }

        // Also check for "no test files" indicator (existing logic)
        if strings.Contains(event.Output, "[no test files]") {
            // ... existing no test files logic
        }

        // Keep debug logging for development
        g.logger.Debug("Package output for %s: %s", event.Package, output)
    }
}
```

### 3. **Error Output Detection Logic**

```go
func (g *GoTestDefinition) isErrorOutput(output string) bool {
    // Skip empty lines and standard go test output
    if output == "" {
        return false
    }

    // Skip standard success/info lines
    skipPatterns := []string{
        "?   \t",           // No test files indicator
        "ok  \t",           // Package passed
        "coverage:",        // Coverage information
        "=== RUN",          // Test start (should be captured by test-specific logic)
        "--- PASS",         // Test pass (should be captured by test-specific logic)
        "--- FAIL",         // Test fail (should be captured by test-specific logic)
        "--- SKIP",         // Test skip (should be captured by test-specific logic)
    }

    for _, pattern := range skipPatterns {
        if strings.Contains(output, pattern) {
            return false
        }
    }

    // Capture everything else as potential error output
    return true
}
```

### 4. **Error Message Construction in handlePackageResult()**

**File:** `internal/runner/definitions/gotest.go:577-648`

**Enhanced logic:**
```go
func (g *GoTestDefinition) handlePackageResult(event *GoTestEvent) {
    g.mu.Lock()
    defer g.mu.Unlock()

    // ... existing setup logic ...

    // NEW: Detect setup failures and construct error message
    if event.Action == "fail" && len(g.packageGroups[event.Package].Tests) == 0 {
        // This is a setup failure - construct error message
        errorMessage := g.constructErrorMessage(event.Package)

        // Send testGroupError event
        g.sendGroupError(event.Package, []string{}, "SETUP_FAILURE", event.Elapsed, errorMessage)

        // Mark setupFailed in testGroupResult totals
        totals["setupFailed"] = true
    }

    // ... rest of existing logic ...
}
```

### 5. **Error Message Construction Logic**

```go
func (g *GoTestDefinition) constructErrorMessage(packageName string) string {
    // Get buffered error output for this package
    errorLines, hasErrors := g.packageErrors[packageName]

    if !hasErrors || len(errorLines) == 0 {
        // Fallback message if no specific error captured
        return "Package failed during setup or compilation"
    }

    // Join error lines with newlines for readability
    message := strings.Join(errorLines, "\n")

    // Clean up common noise
    message = g.cleanErrorMessage(message)

    return message
}

func (g *GoTestDefinition) cleanErrorMessage(message string) string {
    // Remove leading/trailing whitespace
    message = strings.TrimSpace(message)

    // Remove ANSI color codes if present
    ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
    message = ansiRegex.ReplaceAllString(message, "")

    // Remove redundant "FAIL" lines since status is already FAIL
    lines := strings.Split(message, "\n")
    var cleanLines []string

    for _, line := range lines {
        line = strings.TrimSpace(line)
        if line == "" {
            continue
        }

        // Skip redundant FAIL lines like "FAIL\tpackage.name\t1.23s"
        if strings.HasPrefix(line, "FAIL\t") {
            continue
        }

        cleanLines = append(cleanLines, line)
    }

    return strings.Join(cleanLines, "\n")
}
```

## Example: Error Message Construction for etcd

### **Input: Go Test JSON Output**
```json
{"Action":"output","Package":"go.etcd.io/etcd/tests/v3/common","Output":"No test mode selected, please selected either e2e mode with \"--tags e2e\" or integration mode with \"--tags integration\"\n"}
{"Action":"output","Package":"go.etcd.io/etcd/tests/v3/common","Output":"FAIL\tgo.etcd.io/etcd/tests/v3/common\t0.854s\n"}
{"Action":"fail","Package":"go.etcd.io/etcd/tests/v3/common","Elapsed":0.856}
```

### **Processing Steps**

1. **First output event processed by `handleOutput()`:**
   ```
   isErrorOutput("No test mode selected, please selected either e2e mode...") → true
   packageErrors["go.etcd.io/etcd/tests/v3/common"] = ["No test mode selected, please selected either e2e mode with \"--tags e2e\" or integration mode with \"--tags integration\""]
   ```

2. **Second output event processed by `handleOutput()`:**
   ```
   isErrorOutput("FAIL\tgo.etcd.io/etcd/tests/v3/common\t0.854s") → false (filtered out)
   ```

3. **Package failure event processed by `handlePackageResult()`:**
   ```
   len(g.packageGroups[event.Package].Tests) == 0 → Setup failure detected
   constructErrorMessage("go.etcd.io/etcd/tests/v3/common") → "No test mode selected, please selected either e2e mode with \"--tags e2e\" or integration mode with \"--tags integration\""
   ```

### **Output: testGroupError Event**
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

## Memory Management

### **Cleanup Logic**
```go
func (g *GoTestDefinition) cleanupPackageErrors(packageName string) {
    // Clean up buffered errors after processing to prevent memory leaks
    delete(g.packageErrors, packageName)
}
```

Called at the end of `handlePackageResult()` after sending events.

## Error Message Quality Examples

### **Good Error Messages (Preserved)**
- `"No test mode selected, please selected either e2e mode with \"--tags e2e\" or integration mode with \"--tags integration\""`
- `"undefined: someFunction"`
- `"cannot find package \"missing/dependency\" in any of"`
- `"syntax error: unexpected token"`

### **Noise (Filtered Out)**
- `"FAIL\tpackage.name\t1.23s"` (redundant with status)
- `"?   \tpackage.name\t[no test files]"` (handled separately)
- `"coverage: 0.0% of statements"` (not an error)

## Benefits of This Approach

1. **Preserves Original Error Messages**: Users see the actual Go compiler/test runner errors
2. **Structured Information**: Error details are available in both IPC events and reports
3. **Backward Compatible**: Existing functionality is preserved
4. **Memory Efficient**: Errors are cleaned up after processing
5. **Debuggable**: Both structured events and debug logs are available

This construction method ensures that setup failure error messages are captured, cleaned, and presented in a structured format that provides maximum value to users debugging their test issues.

## Implementation Checklist

### Phase 1: Schema and Documentation
- [x] Update `docs/architecture/test-runner-adapters.md` with `testGroupError` event schema
- [x] Update `CLAUDE.md` with new IPC event type

### Phase 2: Go Adapter Implementation
- [x] Add `packageErrors map[string][]string` field to `GoTestDefinition` struct
- [x] Implement `isErrorOutput()` method for filtering relevant error lines
- [x] Enhance `handleOutput()` method to capture package-level errors
- [x] Implement `constructErrorMessage()` method for message assembly
- [x] Implement `cleanErrorMessage()` method for noise filtering
- [x] Add `sendGroupError()` method for sending `testGroupError` events
- [x] Enhance `handlePackageResult()` to detect setup failures and send error events
- [x] Add cleanup logic to prevent memory leaks
- [x] Add `setupFailed` flag to `testGroupResult` events

### Phase 3: Report Generation Updates
- [ ] Update report generators to handle `testGroupError` events
- [ ] Add setup failure display to individual reports
- [ ] Update console output to distinguish setup vs test failures

### Phase 4: Testing
- [ ] Add unit tests for error message construction logic
- [ ] Add unit tests for setup failure detection
- [ ] Add integration tests with failing Go packages
- [ ] Test against etcd to verify fix works
- [ ] Test memory cleanup and thread safety

### Phase 5: Code Quality
- [x] Run `go fmt` on all modified files
- [x] Run linter on all modified files
- [x] Run complete test suite (`make test`)
- [x] Verify no regressions in existing functionality

### Phase 6: Validation
- [ ] Test against etcd to confirm totals are now correct
- [ ] Verify error messages are properly captured and displayed
- [ ] Confirm console vs report metrics now align