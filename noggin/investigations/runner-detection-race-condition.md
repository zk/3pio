# Runner Detection Race Condition Investigation

## Summary
3pio incorrectly detects test runners when multiple runners are present in package.json, causing Jest adapter to be used for Vitest commands and vice versa. This is a critical bug that affects all JavaScript test runners.

## Root Cause

### The Flawed Logic
Both Jest and Vitest definitions use the same flawed `Matches()` logic:

```go
// Jest
func (j *JestDefinition) Matches(command []string) bool {
    return containsTestRunner(command, "jest") || j.isJestInPackageJSON()
}

// Vitest
func (v *VitestDefinition) Matches(command []string) bool {
    return containsTestRunner(command, "vitest") || v.isVitestInPackageJSON()
}
```

The problem: The `is*InPackageJSON()` fallback makes the runner match ANY command if it's in package.json, even commands explicitly for other runners.

### The Race Condition
1. When `Detect()` is called with command `["npx", "vitest", "run"]`
2. It iterates through runners using `for _, def := range m.runners`
3. Since `m.runners` is a Go map, iteration order is non-deterministic
4. Both Jest and Vitest return `true` from `Matches()`:
   - Jest: `containsTestRunner` returns false, but `isJestInPackageJSON` returns true
   - Vitest: `containsTestRunner` returns true
5. Whichever runner is checked first "wins" and is returned
6. Due to random map iteration, Jest wins ~90% of the time in testing

### Evidence
Test results show the race condition clearly:
- Command `npx vitest run` detected as `jest.js` in 9/10 iterations
- Command `npx jest` always detected as `jest.js` (because Jest is usually checked first)
- All explicit runner commands fail when both runners are in package.json

### Why It Matters
This bug causes:
- Wrong adapter to be injected (e.g., `--reporters` flag for Vitest)
- Test runs to fail with "Unknown option" errors
- Inconsistent behavior between runs
- Complete failure when projects use multiple test frameworks

## Solution

The `is*InPackageJSON()` check should only be used as a fallback for generic commands that don't specify a runner explicitly. The logic should be:

1. If command explicitly contains a test runner name, that runner should ALWAYS be selected
2. Only use package.json detection for generic commands like `npm test`, `yarn test`
3. Never let package.json presence override explicit runner specification

### Proposed Fix

```go
func (j *JestDefinition) Matches(command []string) bool {
    // If Jest is explicitly in the command, match it
    if containsTestRunner(command, "jest") {
        return true
    }

    // If another test runner is explicitly specified, DON'T match
    if containsTestRunner(command, "vitest") ||
       containsTestRunner(command, "mocha") ||
       containsTestRunner(command, "cypress") {
        return false
    }

    // Only use package.json as fallback for generic commands
    return j.isJestInPackageJSON()
}
```

This ensures:
- Explicit runner commands always work correctly
- Generic test commands use package.json detection
- No race conditions or inconsistent behavior

## Test Case
Created `runner_detection_race_test.go` that demonstrates the issue:
- Tests explicit Vitest commands with both runners in package.json
- Shows consistent misdetection (Jest selected for Vitest commands)
- Validates that explicit runner specification should take precedence

## Impact
This bug affects any project that:
- Has multiple test frameworks installed
- Uses explicit runner commands (npx vitest, yarn jest, etc.)
- Relies on 3pio for test execution

## Next Steps
1. Fix the Matches() logic for all JavaScript test runners
2. Add comprehensive tests for runner detection precedence
3. Consider using a priority system instead of first-match-wins