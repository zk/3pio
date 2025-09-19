# Fix Pytest Setup Phase Skip Capture

**Date:** 2025-09-19
**Author:** Claude
**Issue:** 3pio pytest adapter misses tests skipped via markers
**Impact:** Incorrect test counts (e.g., 44 vs 45 in LangChain)
**Status:** ✅ IMPLEMENTED & VALIDATED

## Executive Summary

The 3pio pytest adapter fails to capture tests skipped during the 'setup' phase, missing all tests with `@pytest.mark.skip` or `@pytest.mark.skipif` markers. This is due to an overly restrictive filter at line 442 that only processes 'call' phase events.

## Problem Analysis

### Root Cause
```python
# internal/adapters/pytest_adapter.py line 442
def pytest_runtest_logreport(report: TestReport) -> None:
    # ...
    # Only process the 'call' phase (actual test execution)
    if report.when != 'call':
        return  # BUG: Filters out setup phase skips
```

### Impact
- Tests with `@pytest.mark.skipif(condition)` where condition=True → Missed
- Tests with `@pytest.mark.skip` → Missed
- Tests with `pytest.skip()` inside function → Captured (call phase)

### Evidence from LangChain Testing
- **Baseline pytest:** 1346 passed, 12 skipped, 10 xfailed, 2 xpassed
- **3pio (current):** Reports fewer skipped tests
- **Missing test:** `test_async_custom_event_implicit_config` with `@pytest.mark.skipif(sys.version_info < (3, 11))`

## Technical Investigation

### Pytest Execution Phases

Each test goes through three phases:

1. **Setup Phase**
   - Evaluates skip conditions from markers
   - Reports skips for `@pytest.mark.skip` and `@pytest.mark.skipif`
   - `pytest_runtest_setup()` NOT called for marker skips (optimization)

2. **Call Phase**
   - Actual test execution
   - Reports skips from `pytest.skip()` calls inside test
   - Only reached if test wasn't skipped in setup

3. **Teardown Phase**
   - Always executed
   - Cleanup operations

### Key Finding

The `pytest_runtest_logreport` hook DOES receive setup phase events with full skip information:
```python
# Setup phase skip event received by the hook:
report.when = 'setup'
report.skipped = True
report.outcome = 'skipped'
report.longrepr = ('skipif', condition, 'Skipped: Requires Python 3.11+')
```

**The adapter receives this data but explicitly ignores it.**

## Implementation Plan

### Core Changes

#### 1. Add Skip Tracking (Prevent Duplicates)
```python
class ThreepioReporter:
    def __init__(self, ipc_path: str):
        # ... existing init ...
        self.processed_skips = set()  # Track (file_path, test_name) tuples
```

#### 2. Add Skip Reason Extraction
```python
def _extract_skip_reason(report: TestReport) -> str:
    """Extract skip reason from pytest report object."""
    if hasattr(report, 'longrepr'):
        if isinstance(report.longrepr, tuple) and len(report.longrepr) >= 3:
            # Format: (category, condition, reason)
            reason = str(report.longrepr[2])
            # Remove 'Skipped: ' prefix if present
            if reason.startswith('Skipped: '):
                return reason[9:]
            return reason
        elif isinstance(report.longrepr, str):
            return report.longrepr

    return "Test skipped"
```

#### 3. Fix pytest_runtest_logreport
```python
def pytest_runtest_logreport(report: TestReport) -> None:
    # ... initialization ...

    # Parse test info (needed for all phases)
    file_path, suite_chain, test_name = _reporter.parse_test_hierarchy(report.nodeid)

    # Initialize results for file if needed
    if file_path not in _reporter.test_results:
        _reporter.test_results[file_path] = {
            "passed": 0, "failed": 0, "skipped": 0,
            "xfailed": 0, "xpassed": 0, "failed_tests": []
        }

    # HANDLE SKIPPED TESTS IN ANY PHASE
    if report.skipped and report.when in ('setup', 'call'):
        # Check for duplicate processing
        skip_key = (file_path, test_name)
        if skip_key in _reporter.processed_skips:
            return
        _reporter.processed_skips.add(skip_key)

        # Extract skip reason
        skip_reason = _extract_skip_reason(report)

        # Update skip count
        _reporter.test_results[file_path]["skipped"] += 1

        # Send IPC event for skipped test
        _reporter.send_event("testCase", {
            "testName": test_name,
            "parentNames": suite_chain + [file_path],
            "status": "SKIP",
            "skipReason": skip_reason,
            "skipPhase": report.when  # 'setup' or 'call'
        })

        return

    # Only process non-skip events from call phase
    if report.when != 'call':
        return

    # ... rest of existing logic ...
```

### IPC Event Structure

```json
{
  "eventType": "testCase",
  "payload": {
    "testName": "test_async_custom_event_implicit_config",
    "parentNames": ["tests/unit_tests/callbacks/test_dispatch_custom_event.py"],
    "status": "SKIP",
    "skipReason": "Requires Python 3.11+",
    "skipPhase": "setup"
  }
}
```

## Testing Strategy

### Unit Tests
```python
def test_setup_phase_skip_captured():
    """Test that skips in setup phase are captured."""
    # Mock report with when='setup', skipped=True
    # Verify IPC event sent

def test_skip_deduplication():
    """Test that same skip isn't reported twice."""
    # Send same skip in multiple phases
    # Verify only one IPC event
```

### Integration Test Fixtures
```python
# tests/fixtures/skip_tests.py
import sys
import pytest

@pytest.mark.skip(reason="Always skip")
def test_always_skip():
    pass

@pytest.mark.skipif(sys.version_info < (3, 11), reason="Needs 3.11+")
def test_conditional_skip():
    pass

def test_dynamic_skip():
    pytest.skip("Runtime skip")
```

### Verification with Real Projects
1. Run baseline pytest to get expected counts
2. Run with 3pio and verify:
   - Total test count matches
   - Skip count matches
   - Skip reasons preserved
   - No duplicate events

## Edge Cases

| Skip Type | Phase | Current | Fixed |
|-----------|-------|---------|-------|
| `@pytest.mark.skip` | setup | ❌ Missed | ✅ Captured |
| `@pytest.mark.skipif(True)` | setup | ❌ Missed | ✅ Captured |
| `@pytest.mark.skipif(False)` | N/A | ✅ Runs | ✅ Runs |
| `pytest.skip()` in test | call | ✅ Captured | ✅ Captured |

## Risk Assessment

- **Risk Level:** Low
- **Change Scope:** ~40 lines in one file
- **Backward Compatibility:** Maintained
- **Performance Impact:** Negligible
- **Rollback:** Can add feature flag if needed

## Success Criteria

- [x] All marker-based skips captured (setup phase)
- [x] All programmatic skips captured (call phase)
- [x] No duplicate skip events
- [x] Skip reasons preserved accurately
- [x] Test counts match baseline pytest exactly
- [x] Works with xdist parallel execution (LangChain uses -n auto)
- [x] Integration tests pass
- [x] Tested with LangChain project (12 skips correctly captured)

## Code Diff Summary

**File:** `internal/adapters/pytest_adapter.py`

**Changes:**
1. Add `self.processed_skips = set()` to track processed skips
2. Add `_extract_skip_reason(report)` helper function
3. Modify `pytest_runtest_logreport()`:
   - Remove early return for non-call phases
   - Add skip handling for setup and call phases
   - Add deduplication logic
   - Preserve existing logic for pass/fail/xfail

**Lines Changed:** ~40 additions, 5 deletions

## Implementation Checklist

### Phase 1: Core Implementation ✅
- [x] Add `processed_skips` set to `ThreepioReporter.__init__()`
- [x] Implement `_extract_skip_reason()` helper function
- [x] Modify `pytest_runtest_logreport()` to handle setup phase skips
- [x] Add skip deduplication logic
- [x] Update skip event IPC structure to include phase and reason

### Phase 2: Testing ✅
- [x] Create test fixtures with various skip types
- [x] Test setup phase skip capture (marker and skipif)
- [x] Test call phase skip capture (pytest.skip())
- [x] Test skip deduplication
- [x] Run tests to verify skip counts match baseline

### Phase 3: Validation ✅
- [x] Test with LangChain project (12 skips captured correctly)
- [x] Test with xdist parallel execution (via LangChain -n auto)
- [x] Verify skip reasons are preserved correctly
- [x] Confirm no duplicate events in IPC
- [x] Verify xfail/xpass are not confused with skips

### Phase 4: Documentation & Review
- [ ] Add inline comments explaining phase handling
- [ ] Update pytest adapter documentation
- [ ] Create PR with clear description of bug and fix
- [ ] Include before/after test results
- [ ] Get code review

### Phase 5: Post-Merge
- [ ] Monitor for any issues in CI/CD
- [ ] Test with other open-source projects
- [ ] Close related issue(s)
- [ ] Update release notes

## Conclusion

This is a straightforward bug fix. The pytest adapter already receives all necessary information about skipped tests but discards it due to an overly restrictive filter. The fix involves processing skip events from both 'setup' and 'call' phases while preventing duplicates.

The change is minimal, well-contained, and addresses the root cause directly without affecting other functionality.