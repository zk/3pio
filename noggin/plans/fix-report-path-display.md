# Fix Inline Failure Display After FAIL Messages

## Context
When running 3pio, test failures should be displayed inline immediately after each FAIL message in the console output, but currently they are not shown until the end in a summary section.

## Issue: Missing Inline Failure Display After FAIL Messages

### Problem
When individual test groups fail during execution, they should display a failure summary immediately below the FAIL line, but instead failures are only shown in the final summary section after "Test failures!". The final summary section should be removed entirely - failures should only appear inline.

### Expected Behavior
```
FAIL     github.com/zk/3pio/tests/integration_go (50.44s)
  x TestPytestPatternMatching
  x TestPytestExitCodePass
  x TestPytestExitCodeError
  +6 more
  See .3pio/runs/.../reports/github_com_zk_3pio_tests_integration_go/index.md
```

### Actual Behavior
```
FAIL     github.com/zk/3pio/tests/integration_go (50.44s)
[no failure summary shown - failures only appear in final summary after "Test failures!"]
```

### Code Location
The `displayFileResult()` function at `internal/orchestrator/orchestrator.go:968-994` has the logic to show failures inline, but it's not being triggered for native Go test runner.

### Final Summary Section to Remove
The summary section after "Test failures! This is madness!" at `internal/orchestrator/orchestrator.go:626-638` should be removed entirely. Failures should only appear inline after each FAIL message.

### Investigation Needed
1. Check when `displayFileResult()` is called for native runners
2. Verify `group.Status` is correctly set to `TestStatusFail`
3. Ensure `collectFailedTests()` returns the failed test names
4. Check if native runners bypass this display logic
5. Remove the final "Test failures!" summary section

### Possible Causes
- Native runners (Go test) might handle display differently than Jest/Vitest
- The group status might not be set correctly when tests fail
- The failed test collection might not work for native test output format

## Status
- ✅ COMPLETED - Display failures inline after FAIL messages
- ✅ COMPLETED - Removed the "Test failures!" summary section

## Next Steps
1. [x] Enable inline failure display after FAIL messages for all test runners
2. [x] Remove the "Test failures! This is madness!" summary section
3. [x] Verify that inline failure display shows correct report path pointing to the actual failing test group
4. [x] Verify the fix works for Go test runner (other runners need separate verification)
5. [x] Add integration tests for inline failure display

### Verification Requirements
After implementing inline failure display:
- The report path shown inline must point to the correct failing test group's report
- For multi-package projects, ensure the path points to the specific package that failed, not alphabetically first
- Example: If `pkg_zebra` fails while `pkg_alpha` passes, the inline display should show path to `pkg_zebra/index.md`

## Existing Test Coverage

### Found: `TestMultiPackageFailureReportPath`
Location: `tests/integration_go/multi_package_failure_test.go`

This test currently verifies:
1. ✅ The final summary section (after "Test failures!") shows the correct failing package path
2. ❌ Does NOT test inline failure display after FAIL messages (test looks for it but doesn't find it)

The test's `individual_file_shows_correct_path` subtest (lines 101-121) attempts to verify inline display but currently doesn't find failures displayed inline after FAIL messages.

### Test Updates Needed
The existing test should be updated after the fix to:
1. Verify failures ARE shown inline after each FAIL message
2. Verify the inline display includes the correct report path
3. Verify the "Test failures!" summary section is removed

### New Test Needed
Create a dedicated test for inline failure display across all test runners:
- `TestInlineFailureDisplay` - Verify Jest, Vitest, pytest, and Go test all show failures inline
- Should test both single and multi-package scenarios
- Should verify report paths are correct in inline display

## Implementation Checklist (TDD Approach) - COMPLETED

### Phase 1: Write Failing Tests ✅
- [x] Create `TestInlineFailureDisplay` in `tests/integration_go/inline_failure_display_test.go`
- [x] Update `TestMultiPackageFailureReportPath` to expect inline failures (will fail initially)
- [x] Run tests to confirm they fail as expected

### Phase 2: Fix Inline Display ✅
- [x] Fixed race condition in `processEvents()` - report manager must update BEFORE console display
- [x] Fixed Go test definition to override status to FAIL when tests fail (even if Go reports PASS)
- [x] Ensured `group.Status` is set correctly when tests fail
- [x] Inline display now works for Go test runner

### Phase 3: Remove Summary Section ✅
- [x] Located summary section code at `internal/orchestrator/orchestrator.go:597-641`
- [x] Removed the "Test failures!" output with random exclamations
- [x] Removed the summary list of failed tests
- [x] Removed the summary report path display

### Phase 4: Verify and Clean Up ✅
- [x] Run all tests to ensure they pass
- [x] Run manual tests with fixtures for Go test runner
- [x] Update plan with completion status
- [x] Run linter (`make lint`) and formatter (`gofmt -w`)