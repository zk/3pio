# 3pio Issues to Fix

This document tracks issues discovered during testing with real-world open source projects.

## Priority 1 - Critical Issues

### 1. Glob Pattern Support
**Discovered**: Testing Supabase repository
**Test Case**: `3pio npx vitest run "**/Alert/**"`

**Problem**: 
Glob patterns are not properly handled, resulting in "No test files found" errors even when matching test files exist.

**Context**:
- When running `3pio npx vitest run "**/Alert/**"` in packages/ui
- The glob should match `src/components/Alert/Alert.test.tsx`
- Instead, Vitest reports: "No test files found, exiting with code 1"
- The issue appears to be with shell expansion - the quotes are being escaped incorrectly

**Expected Behavior**:
```bash
# Should work like:
vitest run "**/Alert/**"  # Finds and runs Alert tests
```

**Actual Behavior**:
```bash
# 3pio seems to escape it as:
vitest run \"**/Alert/**\"  # Vitest can't interpret the escaped quotes
```

**Impact**: 
Developers can't use glob patterns to run subset of tests, which is common when working on specific components.

**Suggested Fix**:
- Review command argument handling in Go
- Ensure quotes are preserved but not double-escaped
- Test with various shell expansion patterns

## Priority 2 - Important Issues

### 2. Coverage Mode Compatibility
**Discovered**: Testing Supabase repository
**Test Case**: `3pio pnpm test:ci` (which runs `vitest --run --coverage`)

**Problem**:
When coverage is enabled, 3pio's adapter fails to capture individual test results. Tests run successfully but only the final summary appears in output.log, with test-run.md showing 0 files.

**Context**:
- Coverage reporters seem to take precedence over custom reporters
- The IPC events are not being sent when coverage is active
- This affects all test runners (Vitest, Jest, pytest with coverage)

**Impact**:
CI/CD workflows that require coverage reports lose granular test tracking.

**Suggested Fix**:
- Investigate if multiple reporters can be chained
- Consider detecting coverage mode and warning users
- Explore alternative reporter registration when coverage is detected

### 3. Collection Phase Errors Don't Stop Execution (pytest)
**Discovered**: Testing agno repository

**Problem**:
While we now capture collection errors, pytest continues trying to run (and fails) even when collection fails.

**Context**:
- Collection errors are captured successfully
- But the test run continues with confusing error messages
- Users see both collection error AND execution failure

**Suggested Fix**:
- Detect collection failures and halt execution cleanly
- Provide clear final status when collection fails

## Priority 3 - Enhancement Opportunities

### 4. Turborepo Environment Variable Propagation
**Discovered**: Testing Supabase repository

**Problem**:
THREEPIO_IPC_PATH environment variable doesn't propagate through Turborepo to child processes, preventing adapter communication.

**Context**:
- Turborepo filters environment variables for security/caching
- Custom reporters can be passed through but can't communicate without IPC path
- Affects all monorepo orchestration tools (Turborepo, Nx, Lerna)

**Potential Solutions**:
- Embed IPC path in adapter at injection time
- Use alternative communication methods (temp files with known paths)
- Provide Turborepo-specific adapter that doesn't need env vars

### 5. Source Map Warnings
**Discovered**: Throughout Supabase testing

**Problem**:
Vite complains about missing source maps for 3pio adapters:
```
Failed to load source map for /path/to/.3pio/adapters/xxx/vitest.js
```

**Context**:
- Not breaking functionality but creates noise in output
- Happens because adapters don't have .map files
- Vite tries to load source maps for all JS files

**Suggested Fix**:
- Generate source maps for adapters
- Or add inline source map comment to suppress warnings
- Or configure Vite to ignore adapter directory

## Priority 4 - Future Enhancements

### 6. Watch Mode Support
**Discovered**: General testing

**Problem**:
Watch mode isn't fully supported - 3pio captures initial run but not subsequent test runs.

**Context**:
- Developers use watch mode extensively during development
- Would need to handle continuous output stream
- Need to detect test re-runs and update reports

### 7. Better Error Messages for Unsupported Commands
**Discovered**: Turborepo testing

**Problem**:
When 3pio can't detect test runner, error message could be more helpful:
```
Error: Could not detect test runner from command: pnpm test:ui
```

**Suggested Enhancement**:
```
Error: Could not detect test runner from command: pnpm test:ui

This appears to be a Turborepo command. Try running 3pio from within the package:
  cd packages/ui && 3pio pnpm test

Or use the test runner directly:
  3pio npx vitest run
```

## Testing Notes

### Projects Used for Discovery
1. **Supabase** - Large monorepo with Vitest, Turborepo, pnpm workspaces
2. **agno** - Python project with pytest, collection phase errors
3. **ms** - ES Module project with Jest
4. **unplugin-auto-import** - Vitest with watch mode issues

### Test Commands to Verify Fixes
```bash
# Glob patterns
cd packages/ui && 3pio npx vitest run "**/Button/**"

# Coverage mode
cd packages/ui && 3pio pnpm test:ci

# Collection errors
cd libs/agno && 3pio pytest tests/unit/

# Turborepo (future)
3pio pnpm test:ui
```

## Implementation Priority

1. **Fix glob patterns** - Common developer use case
2. **Handle coverage mode** - Important for CI/CD
3. **Clean up collection errors** - Better UX
4. **Address source map warnings** - Reduce noise
5. **Everything else** - Nice to have

## Success Criteria

Each fix should:
1. Not break existing functionality
2. Include tests to prevent regression
3. Update documentation if behavior changes
4. Work across all supported test runners where applicable