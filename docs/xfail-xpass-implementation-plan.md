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
1. Test pytest adapter correctly identifies xfail/xpass from TestReport
2. Test IPC events contain correct status for xfail/xpass
3. Test group manager correctly processes xfail/xpass statuses
4. Test report generation includes xfail/xpass in summaries

### Integration Tests
1. Create pytest test fixtures with xfail/xpass tests
2. Run 3pio against these fixtures
3. Verify reports correctly show xfail/xpass statuses
4. Verify exit codes are correct (xfail shouldn't fail the suite)

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

1. **Exit Codes**: xfail tests should not affect exit codes (they're expected failures)
2. **Strict Mode**: pytest's strict xfail mode makes xpass fail the suite - consider how to handle this
3. **Other Test Runners**: This implementation is pytest-specific. Consider if/how to extend to other runners
4. **Backward Compatibility**: Ensure existing functionality isn't broken
5. **Report Clarity**: Make it clear in reports what xfail/xpass mean for users unfamiliar with the concepts

## Implementation Order
1. Start with Python adapter changes (easiest to test in isolation)
2. Update Go types and IPC handling
3. Update report generation
4. Add tests
5. Update documentation

## Success Criteria
- pytest tests marked with @pytest.mark.xfail are correctly identified as XFAIL or XPASS
- Reports clearly show xfail/xpass counts separate from regular failures/passes
- Exit codes remain correct (xfail doesn't fail the suite)
- Integration tests pass with real pytest fixtures