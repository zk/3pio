# Issue: 3pio Adapter Injection Fails for Custom npm Script Wrappers

## Problem Description

When running tests through npm scripts that use custom wrapper scripts (rather than directly invoking the test runner), 3pio fails to properly inject its adapter, resulting in:
- Tests run successfully but 3pio captures 0 test results
- Empty IPC file (no test events captured)
- No structured test reports despite successful test execution

## Example Case

**Project:** manapua
**Command:** `npm run system-test -- test/system/mcp-tools/press-key.test.js`

**package.json script:**
```json
{
  "scripts": {
    "system-test": "node scripts/run-tests-with-logging.js"
  }
}
```

The custom script `scripts/run-tests-with-logging.js` eventually calls Jest internally, but doesn't pass through the `--reporters` argument that 3pio tries to inject.

## Current Behavior

1. 3pio detects Jest (from package.json devDependencies)
2. 3pio modifies the command to include `--reporters /path/to/adapter/jest.js`
3. The custom wrapper script doesn't forward this argument to the actual Jest command
4. Tests run with default Jest reporter, not the 3pio adapter
5. Result: Empty IPC file, 0 test results in report

## Root Cause

The detection logic checks for test runners in package.json dependencies/devDependencies, which succeeds even when the npm script doesn't directly invoke the test runner. The adapter injection assumes the script will pass arguments through, which custom wrapper scripts often don't do.

## Attempted Fix

Added validation in `internal/runner/manager.go` to check if npm scripts directly reference supported test runners:

```go
// Read package.json and check what the script actually runs
if scriptCommand := resolvePackageScript(scriptName); scriptCommand != "" {
    // Check if the resolved script uses a known test runner
    supportedRunners := []string{"jest", "vitest", "mocha", "cypress", "pytest"}
    hasKnownRunner := false
    for _, runner := range supportedRunners {
        if strings.Contains(scriptCommand, runner) {
            hasKnownRunner = true
            // ... attempt to match runner
        }
    }

    // If the script doesn't directly use a known runner, fail
    if !hasKnownRunner {
        return nil, fmt.Errorf("npm script '%s' runs '%s' which doesn't use a supported test runner",
            scriptName, scriptCommand)
    }
}
```

However, this fix doesn't fully solve the problem because the Jest definition's `Matches()` function detects Jest from package.json before this validation runs.

## Proper Solution Options

### Option 1: Strict Script Validation (Recommended)
- Check npm script commands BEFORE runner detection
- Only allow scripts that directly invoke supported test runners
- Fail fast with clear error message for custom wrapper scripts

### Option 2: Adapter Injection Verification
- After test execution, verify the IPC file has content
- If empty but tests ran (detected from output), fail with clear error
- Helps identify the issue but doesn't prevent wasted execution time

### Option 3: Documentation and Warnings
- Document that custom wrapper scripts are not supported
- Add warning when detecting npm scripts that don't directly invoke test runners
- Provide guidance on how to modify scripts for 3pio compatibility

## Impact

This issue affects any project using custom test wrapper scripts, which is common in larger projects for:
- Custom logging/reporting
- Environment setup
- Test file filtering
- CI/CD integration

## Workaround

Users can work around this by:
1. Running the test command directly: `npx jest test/system/mcp-tools/press-key.test.js`
2. Modifying the wrapper script to pass through all arguments
3. Creating a separate npm script that directly invokes the test runner

## Test Case

```bash
# In manapua directory with the custom script
cd /Users/edie/code/manapua
./build/3pio npm run system-test -- test/system/mcp-tools/press-key.test.js

# Expected: Error indicating unsupported script
# Actual: Tests run with 0 results captured
```

## Related Files

- `/Users/edie/code/3pio/internal/runner/manager.go` - Runner detection logic
- `/Users/edie/code/3pio/internal/runner/definition.go` - Individual runner definitions
- `/Users/edie/code/manapua/package.json` - Example problematic configuration
- `/Users/edie/code/manapua/.3pio/runs/20250921T233527-quirky-travis/` - Example failed run