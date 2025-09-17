# Fix 3pio Test Failures

## Context
Running 3pio 0.5.0 on its own repository revealed several issues that need to be addressed.

## Issues Identified

### 1. Missing Python/pytest Installation
- **Problem**: 8 of 9 integration test failures are due to pytest not being installed on the system
- **Error**: `exec: "pytest": executable file not found in $PATH`
- **Affected Tests**:
  - TestPytestPatternMatching
  - TestPytestExitCodePass
  - TestPytestExitCodeError
  - TestPytestMultipleFiles
  - TestPytestDirectory
  - TestPytestSyntaxError
  - TestPytestImportError
  - TestPytestExceptionInTest
- **Solution**: Either install Python/pytest or skip these tests if Python isn't required

### 2. Jest TypeScript Configuration Issue
- **Problem**: TestJestConfigError failed
- **Expected**: Test expects ts-node error to be displayed when missing
- **Actual**: Jest ran successfully without showing the error
- **Location**: `tests/fixtures/jest-ts-config-error/`
- **Solution**: Either fix the test fixture to require ts-node or adjust test expectations

### 3. Incorrect Report Path in Console Output ✓
- **Problem**: Summary section shows wrong report path after failures
- **Example**: Shows `.3pio/runs/.../reports/github_com_zk_3pio_cmd_3pio/index.md`
- **Should be**: `.3pio/runs/.../reports/github_com_zk_3pio_tests_integration_go/index.md`
- **Root Cause**: Code shows first alphabetical directory instead of the failing group's directory
- **Location**: `internal/orchestrator/orchestrator.go:626-638`
- **Status**: FIXED - Now uses `o.groupFailedTests` to show correct failing group path

### 4. Missing Individual Failure Summaries
- **Problem**: Individual FAIL lines during execution don't show failure summaries
- **Expected**: Each FAIL should show up to 3 failed tests with report path (per code at lines 970-993)
- **Actual**: No failure details shown during execution, only in final summary
- **Investigation Needed**: Why `displayFileResult()` isn't showing failure details for Go test runner

## Test Coverage

- Created new test: `TestMultiPackageFailureReportPath` to catch the report path bug
- Test fixture: `tests/fixtures/multi-package-failure/` with pkg_alpha (passes) and pkg_zebra (fails)

## Next Steps

1. [ ] Install Python/pytest or document as test dependency
2. [ ] Fix TestJestConfigError test expectations
3. [x] Fix incorrect report path in summary
4. [ ] Investigate missing individual failure summaries
5. [ ] Run full test suite to verify all fixes

## Notes

- The system showed no Python installation (`python: command not found`)
- No package managers available (pip, uv)
- The report path fix ensures multi-package projects show the correct failing package in summary