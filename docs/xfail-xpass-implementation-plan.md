# Implementation Plan: Adding xfail and xpass Support to 3pio

## Overview
This document outlines the plan to add support for pytest's xfail (expected failures) and xpass (unexpected passes) test statuses to 3pio.

## Background
Currently, 3pio only recognizes these test statuses:
- PASS - Test passed
- FAIL - Test failed
- SKIP - Test was skipped
- PENDING - Test is pending
- RUNNING - Test is running
- NO_TESTS - No tests found
- ERROR - Test encountered an error

pytest has two additional important statuses:
- **xfail** - Test failed as expected (marked with @pytest.mark.xfail)
- **xpass** - Test passed unexpectedly (marked with @pytest.mark.xfail but passed)

## Technical Details

### How pytest Reports xfail/xpass
- pytest reports xfail/xpass tests as "skipped" in the TestReport
- The presence of the `wasxfail` attribute indicates an xfail marker
- Detection logic:
  - **xfail**: `hasattr(report, 'wasxfail') and report.skipped`
  - **xpass**: `hasattr(report, 'wasxfail') and report.passed`
  - The `wasxfail` attribute contains the reason string

## Implementation Steps

### 1. Update IPC Event Types (Go)
**File**: `internal/ipc/events.go`
- Add new TestStatus constants:
  ```go
  TestStatusXFail TestStatus = "XFAIL"  // Test failed as expected
  TestStatusXPass TestStatus = "XPASS"  // Test passed unexpectedly
  ```

### 2. Update Group Types (Go)
**File**: `internal/report/group_types.go`
- Add corresponding TestStatus constants:
  ```go
  TestStatusXFail TestStatus = "XFAIL"
  TestStatusXPass TestStatus = "XPASS"
  ```

### 3. Update pytest Adapter (Python)
**File**: `internal/adapters/pytest_adapter.py`
- Modify `pytest_runtest_logreport` function (around line 385-414):
  ```python
  # Determine test status
  has_xfail = hasattr(report, 'wasxfail')

  if has_xfail:
      # Handle xfail/xpass cases
      if report.passed:
          status = "XPASS"  # Test passed unexpectedly
          _reporter.test_results[file_path]["xpassed"] = _reporter.test_results[file_path].get("xpassed", 0) + 1
      else:  # report.skipped is True for xfail
          status = "XFAIL"  # Test failed as expected
          _reporter.test_results[file_path]["xfailed"] = _reporter.test_results[file_path].get("xfailed", 0) + 1
  elif report.passed:
      status = "PASS"
      _reporter.test_results[file_path]["passed"] += 1
  elif report.failed:
      status = "FAIL"
      _reporter.test_results[file_path]["failed"] += 1
  elif report.skipped:
      status = "SKIP"
      _reporter.test_results[file_path]["skipped"] += 1
  else:
      status = "UNKNOWN"
  ```

- Add xfail reason to payload when present:
  ```python
  # Add xfail reason if available
  if has_xfail:
      payload["xfailReason"] = str(report.wasxfail)
  ```

- Update totals calculation in `pytest_sessionfinish` (around line 489):
  ```python
  totals = {
      'total': (results.get("passed", 0) + results.get("failed", 0) +
                results.get("skipped", 0) + results.get("xfailed", 0) +
                results.get("xpassed", 0)),
      'passed': results.get("passed", 0),
      'failed': results.get("failed", 0),
      'skipped': results.get("skipped", 0),
      'xfailed': results.get("xfailed", 0),
      'xpassed': results.get("xpassed", 0)
  }
  ```

### 4. Update Group Manager (Go)
**File**: `internal/report/group_manager.go`
- Update `HandleTestCase` method to handle new statuses (around line 416-422):
  ```go
  case "XFAIL":
      testCase.Status = TestStatusXFail
  case "XPASS":
      testCase.Status = TestStatusXPass
  ```

- Update `HandleGroupResult` method similarly (around line 259-265)

### 5. Update IPC GroupTotals Structure (Go)
**File**: `internal/ipc/group_events.go`
- Add xfail/xpass counts to GroupTotals:
  ```go
  type GroupTotals struct {
      Total   int `json:"total"`
      Passed  int `json:"passed"`
      Failed  int `json:"failed"`
      Skipped int `json:"skipped"`
      XFailed int `json:"xfailed,omitempty"`
      XPassed int `json:"xpassed,omitempty"`
  }
  ```

### 6. Update Report Formatting (Go)
**Files**: Various report generation files
- Update test count displays to include xfail/xpass
- Update status symbols/formatting to handle new statuses
- Consider display format:
  - XFAIL could display as "⊗" or "xf"
  - XPASS could display as "⊕" or "xp"

### 7. Update Report Generation Logic
- Ensure xfail tests don't count as failures in overall status
- Ensure xpass tests are highlighted (they might indicate fixed bugs)
- Update summary statistics to show xfail/xpass counts

## Testing Strategy

### Unit Tests

#### Pytest Adapter Tests (Python)
- **Test xfail detection**: Mock a TestReport with wasxfail attribute and skipped=True, verify adapter sends XFAIL status
- **Test xpass detection**: Mock a TestReport with wasxfail attribute and passed=True, verify adapter sends XPASS status
- **Test xfail reason extraction**: Verify wasxfail reason string is included in the IPC event payload
- **Test normal skip differentiation**: Verify skipped test without wasxfail attribute still reports as SKIP, not XFAIL
- **Test totals accumulation**: Verify xfailed and xpassed counts are tracked separately in test_results dictionary
- **Test group totals**: Verify pytest_sessionfinish sends correct xfailed/xpassed counts in totals

#### IPC Event Tests (Go)
- **Test XFAIL event parsing**: Verify IPC manager correctly parses testCase events with status="XFAIL"
- **Test XPASS event parsing**: Verify IPC manager correctly parses testCase events with status="XPASS"
- **Test xfail reason field**: Verify xfailReason field is properly extracted from event payload
- **Test group totals with xfail/xpass**: Verify GroupTotals correctly deserializes with xfailed and xpassed fields

#### Group Manager Tests (Go)
- **Test XFAIL status mapping**: Verify HandleTestCase maps "XFAIL" string to TestStatusXFail enum
- **Test XPASS status mapping**: Verify HandleTestCase maps "XPASS" string to TestStatusXPass enum
- **Test group status aggregation with xfail**: Verify group containing xfailed tests doesn't become FAIL status
- **Test group status with mixed results**: Verify correct status when group has mix of pass, fail, skip, xfail, xpass
- **Test recursive count aggregation**: Verify xfailed/xpassed counts bubble up through parent groups correctly

#### Report Generation Tests (Go)
- **Test status symbol formatting**: Verify XFAIL and XPASS get appropriate display symbols
- **Test summary line format**: Verify test count summaries show "X xfailed, Y xpassed" when applicable
- **Test report table rows**: Verify groups with xfail/xpass tests show correct counts in table columns
- **Test no xfail display**: Verify reports don't show xfail/xpass counts when all are zero
- **Test overall run status**: Verify xfailed tests don't affect overall PASS/FAIL determination

### Integration Tests

#### Basic xfail/xpass Fixtures
- **Simple xfail test**: Single test marked with xfail that fails - should report as XFAIL
- **Simple xpass test**: Single test marked with xfail that passes - should report as XPASS
- **Mixed results file**: File with normal pass, fail, skip, xfail, and xpass tests all together
- **xfail with reason**: Test with xfail(reason="...") - verify reason appears in report
- **Conditional xfail**: Test with xfail(condition=True/False) - verify conditional behavior works

#### Complex Scenarios
- **Nested test classes with xfail**: Class containing xfailed tests - verify group hierarchy handles xfail
- **Parametrized xfail tests**: Parametrized test with some xfail markers - verify each parameter reports correctly
- **xfail on setup/teardown**: Tests where fixture fails with xfail marker - verify proper handling
- **Strict xfail mode**: Tests with xfail(strict=True) - verify xpass causes failure in strict mode
- **xfail with pytest.xfail()**: Test that calls pytest.xfail() function - verify runtime xfail works

#### Report Content Verification
- **Summary statistics**: Verify final report shows correct counts for xfailed and xpassed
- **Individual test status**: Verify each test shows correct XFAIL or XPASS status in report
- **No false positives**: Verify normal skipped tests don't show as xfailed
- **Reason display**: Verify xfail reasons appear in appropriate report sections
- **Group aggregation**: Verify parent groups show accumulated xfail/xpass counts from children
- **Mixed status files**: Verify files with combination of all statuses display correctly
- **Nested group summaries**: Verify xfail/xpass counts appear in nested group tables

### Test Fixtures Needed
```python
# test_xfail_xpass.py
import pytest

@pytest.mark.xfail
def test_expected_failure():
    assert False  # This will be XFAIL

@pytest.mark.xfail
def test_unexpected_pass():
    assert True  # This will be XPASS

@pytest.mark.xfail(reason="Feature not implemented")
def test_xfail_with_reason():
    raise NotImplementedError()

@pytest.mark.xfail(strict=True)
def test_strict_xpass():
    assert True  # This will fail the suite in strict mode
```

## Considerations

1. **Other Test Runners**: This implementation is pytest-specific. Consider if/how to extend to other runners
2. **Backward Compatibility**: Ensure existing functionality isn't broken
3. **Report Clarity**: Make it clear in reports what xfail/xpass mean for users unfamiliar with the concepts
4. **Performance**: Minimal overhead for checking wasxfail attribute
5. **Edge Cases**: Handle tests that are both xfailed and skipped conditionally

## Implementation Order
1. Start with Python adapter changes (easiest to test in isolation)
2. Update Go types and IPC handling
3. Update report generation
4. Add tests
5. Update documentation

## Success Criteria
- pytest tests marked with @pytest.mark.xfail are correctly identified as XFAIL or XPASS
- Reports clearly show xfail/xpass counts separate from regular failures/passes
- Integration tests pass with real pytest fixtures
- No regression in existing test status handling
- Clear visual distinction in reports between xfail (expected) and fail (unexpected)