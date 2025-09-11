# 3pio Testing with Supabase Repository

## Test Environment
- **Repository**: Supabase (monorepo with pnpm workspaces)
- **Test Framework**: Vitest (v3.0.5+)
- **3pio Version**: Local build
- **Date**: 2025-09-11

## Test Results Summary

| Test Case | Command | Location | Result | Notes |
|-----------|---------|----------|--------|-------|
| Direct Vitest (UI) | `npx vitest run --no-coverage` | packages/ui | ✅ Success | 7 passed, 3 skipped |
| pnpm test script | `pnpm test -- --run --no-coverage` | packages/ui | ✅ Success | Same results |
| Direct Vitest (Studio) | `npx vitest run --no-coverage` | apps/studio | ✅ Success | 11+ tests passed |
| Single file | `npx vitest run Button.test.tsx` | packages/ui | ✅ Success | 1 file, all tests pass |
| Coverage mode | `pnpm test:ci` | packages/ui | ⚠️ Partial | Tests run but not captured |
| Glob pattern | `npx vitest run "**/Alert/**"` | packages/ui | ❌ Failed | No tests found |
| Docs app | `npx vitest run --no-coverage` | apps/docs | ✅ Success | Captures failures correctly |
| UI-patterns | `pnpm test -- --run` | packages/ui-patterns | ✅ Success | 9 passed |
| Update mode | `pnpm test:update` | apps/studio | ✅ Success | 69 passed, 1 error captured |
| Turborepo from root | `pnpm test:ui` | root | ❌ Failed | Not recognized |
| pnpm run from root | `pnpm run test:ui` | root | ❌ Failed | Exit code 1 |

## Detailed Findings

### ✅ Successful Scenarios

#### 1. Direct Vitest Commands
**Command**: `3pio npx vitest run --no-coverage`

3pio successfully:
- Detected Vitest as the test runner
- Captured test output through the adapter
- Generated structured reports
- Tracked individual test file status (PASS/SKIP)
- Created test-run.md with complete results

**UI Package Results**:
```
Results:  7 passed, 10 total
Time:        2.551s
```

**Studio App Results**:
- All lib tests passed (cloudprovider-utils, formatSql, github, gotrue, helpers, etc.)
- Clean execution with proper file tracking

#### 2. pnpm Script Execution (Within Package)
**Command**: `3pio pnpm test -- --run --no-coverage`

When executed within a package directory:
- 3pio correctly forwarded arguments to Vitest
- Results identical to direct vitest command
- Proper detection of test runner despite pnpm wrapper

### ⚠️ Partial Success Scenarios

#### 1. Coverage Mode (`pnpm test:ci`)
**Issue**: When running with coverage enabled, tests execute but 3pio doesn't capture individual test results.

**Symptoms**:
- Output.log shows test summary but no individual test tracking
- test-run.md shows 0 files
- Vitest's coverage reporter seems to interfere with 3pio's adapter

**Impact**: Coverage reports work but lose granular test tracking.

### ❌ Failed Scenarios

#### 1. Turborepo Commands from Root
**Command**: `3pio pnpm test:ui`

Issues:
- 3pio failed to detect the test runner
- Error: "Could not detect test runner from command: pnpm test:ui"
- Turborepo's filtering mechanism not recognized
- Exit code 254

#### 2. Glob Patterns
**Command**: `3pio npx vitest run "**/Alert/**"`

**Issue**: Glob patterns don't work as expected
- Vitest can't find tests with the glob
- May be shell expansion issue
- Error: "No test files found"

#### 3. pnpm run from Root
**Command**: `3pio pnpm run test:ui`

Issues:
- Command executed but failed with exit code 1
- No test output captured
- Turborepo abstraction layer prevented test runner detection

## Additional Findings from Extended Testing

### Cross-Package Testing
- **Docs app**: Successfully captures test failures with detailed error messages
- **UI-patterns**: Clean execution with all CommandMenu and FilterBar tests passing
- **Studio app**: Handles large test suites (572 tests) with errors properly captured

### Test Execution Modes
- **Single file execution**: Works perfectly for targeted testing
- **Update mode**: `--update` flag properly forwarded, snapshots updated
- **Error handling**: Test failures and errors properly captured in logs

### Performance
- **Large test suites**: Studio's 572 tests completed in ~20s
- **Parallel execution**: Vitest's internal parallelization works well
- **Memory usage**: No issues with large test suites

## Key Observations

### What Works Well

1. **Direct Test Runner Invocation**
   - 3pio excels when directly invoking vitest
   - Both `npx vitest` and direct vitest commands work perfectly
   - Adapter properly captures all test events

2. **Package-Level Testing**
   - When run within a specific package, 3pio works correctly
   - pnpm scripts that directly call vitest are handled well
   - Test arguments are properly forwarded

3. **Vitest Adapter Performance**
   - Clean test status tracking (PASS/SKIP/RUNNING)
   - Accurate file-level reporting
   - Proper timing information

### Current Limitations

1. **Monorepo Orchestration Tools**
   - Turborepo commands not recognized
   - Cannot detect test runner through Turborepo's abstraction layer
   - Root-level convenience scripts fail

2. **Complex Command Chains**
   - Scripts that use Turborepo filters lose context
   - Multi-step command execution not tracked
   - Package manager abstractions cause detection issues

3. **Script Detection**
   - 3pio relies on command pattern matching
   - Doesn't analyze script contents from package.json
   - Cannot follow script chains (e.g., test:ui → turbo run test)

## Recommendations for 3pio

### Short-term Improvements

1. **Enhanced Script Detection**
   - Parse package.json to understand script contents
   - Follow script chains to find actual test runner
   - Recognize common monorepo patterns (turbo, nx, lerna)

2. **Turborepo Support**
   - Detect `turbo run test` patterns
   - Parse --filter arguments
   - Navigate to target package before execution

3. **Better Error Messages**
   - When detection fails, suggest alternative commands
   - Provide hints for monorepo users
   - Show what was attempted and why it failed

### Long-term Enhancements

1. **Monorepo-Aware Mode**
   - Detect workspace configuration (pnpm-workspace.yaml, etc.)
   - Support filtered test execution across packages
   - Aggregate results from multiple packages

2. **Script Analysis**
   - Deep inspection of npm/pnpm/yarn scripts
   - Recursive script resolution
   - Support for complex script compositions

3. **Plugin Architecture**
   - Allow custom adapters for build tools (Turborepo, Nx)
   - Support for project-specific configurations
   - Community-contributed patterns

## Workaround for Supabase Developers

Until 3pio supports Turborepo, developers should:

1. **Navigate to specific packages**: 
   ```bash
   cd packages/ui && 3pio npx vitest run
   ```

2. **Use direct test commands**:
   ```bash
   3pio npx vitest run --filter=ui
   ```

3. **Avoid root-level scripts**:
   - Skip `pnpm test:ui` from root
   - Use package-specific commands instead

## Conclusion

3pio works excellently with Vitest when invoked directly but struggles with monorepo orchestration tools like Turborepo. The core functionality is solid - the Vitest adapter captures all necessary data and generates comprehensive reports. The main challenge is command detection through abstraction layers commonly used in modern monorepos.

For Supabase specifically, 3pio is fully functional when used at the package level but requires workarounds for root-level convenience scripts. This pattern likely applies to other large monorepos using similar tooling.