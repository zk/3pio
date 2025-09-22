# Fix Test Tracking: Single Source of Truth Plan

## Problem Statement

The Svelte open-source test revealed a critical bug where 3pio reports incorrect test counts in console output:
- **Console shows**: 6707 passed (incorrect)
- **Reports show**: 6758 passed (correct)
- **Difference**: 51 tests missing from console count

## Root Cause Analysis

### Current Architecture (Broken)
```
IPC Events
    ├─→ Report Manager → GroupManager (CORRECT tracking)
    └─→ Orchestrator Console Counters (BROKEN tracking)
```

### The Bug
1. Vitest v2 sends lifecycle events for some tests:
   - First event: `status: "PENDING"`
   - Second event: `status: "PASS"`

2. Orchestrator's broken logic (`orchestrator.go`):
   ```go
   if o.seenTestCases[key] {
       return  // BUG: Rejects state transitions as duplicates
   }
   ```

3. GroupManager's correct logic (`group_manager.go`):
   ```go
   if existingTest.ID == testCase.ID {
       parentGroup.TestCases[i] = testCase  // Correctly updates state
   }
   ```

## Solution: GroupManager as Single Source of Truth

Make GroupManager the authoritative source for all test state, eliminating duplicate tracking in orchestrator.

## Implementation Plan

### Phase 1: Expose GroupManager Statistics
**Goal**: Allow orchestrator to query test counts from GroupManager

**Checklist**:
- [ ] Add `GetStats()` method to GroupManager
- [ ] Add `GetGroupStats()` method for group-level stats
- [ ] Add `TestStats` struct with all counter fields
- [ ] Add unit tests for GetStats methods
- [ ] Verify stats calculation matches current report generation

**Files to modify**:
- `internal/report/group_manager.go`
- `internal/report/group_types.go`

### Phase 2: Add Accessor Methods
**Goal**: Provide clean API for orchestrator to access GroupManager

**Checklist**:
- [ ] Add `GetGroupManager()` method to Report Manager
- [ ] Add `GetTestCount()` convenience method
- [ ] Add `GetTestsByStatus()` method for filtering
- [ ] Add integration tests for accessor methods
- [ ] Document new API methods

**Files to modify**:
- `internal/report/manager.go`

### Phase 3: Remove Duplicate Tracking
**Goal**: Eliminate orchestrator's broken test tracking

**Checklist**:
- [ ] Remove test counter fields from Orchestrator struct
- [ ] Remove `seenTestCases` map
- [ ] Remove `testStatuses` map if present
- [ ] Remove duplicate counting logic from `handleConsoleOutput()`
- [ ] Update all references to use GroupManager stats
- [ ] Run tests to ensure no broken references

**Fields to remove from orchestrator**:
```go
passedTests      int
failedTests      int
skippedTests     int
xfailedTests     int
xpassedTests     int
erroredTests     int
totalTests       int
seenTestCases    map[string]bool
```

**Files to modify**:
- `internal/orchestrator/orchestrator.go`

### Phase 4: Update Console Output
**Goal**: Query GroupManager for accurate console display

**Checklist**:
- [ ] Update `displayTestFailures()` to query GroupManager
- [ ] Update `displayTestSummary()` to use GroupManager stats
- [ ] Update progress display during test execution
- [ ] Ensure console format remains unchanged
- [ ] Add debug logging for stat queries
- [ ] Test with various runners (Jest, Vitest v2, pytest)

**Files to modify**:
- `internal/orchestrator/orchestrator.go`
- `internal/orchestrator/console_output.go` (if exists)

### Phase 5: Fix Edge Cases
**Goal**: Handle all test lifecycle scenarios correctly

**Checklist**:
- [ ] Verify PENDING → PASS transitions work
- [ ] Verify PENDING → FAIL transitions work
- [ ] Verify PENDING → SKIP transitions work
- [ ] Handle duplicate PASS events gracefully
- [ ] Test with xfail/xpass scenarios (pytest)
- [ ] Test with setup failures
- [ ] Add logging for state transitions

**Files to modify**:
- `internal/report/group_manager.go`

## Test Migration Strategy

### Unit Tests
1. **New Tests Required**:
   - `TestGroupManagerGetStats` - Verify stat calculation
   - `TestGroupManagerStateTransitions` - Test PENDING→PASS/FAIL/SKIP
   - `TestGroupManagerDuplicateHandling` - Ensure duplicates handled correctly
   - `TestReportManagerAccessors` - Test new accessor methods

2. **Tests to Update**:
   - Remove orchestrator test counter assertions
   - Update console output tests to mock GroupManager stats
   - Update integration tests to verify single source of truth

3. **Test Files**:
   - Create: `internal/report/group_manager_stats_test.go`
   - Update: `internal/orchestrator/orchestrator_test.go`
   - Update: `internal/report/manager_test.go`

### Integration Tests
1. **Scenarios to Test**:
   ```go
   // tests/integration_go/vitest_v2_lifecycle_test.go
   - Test with Vitest v2 project sending PENDING events
   - Verify console and report counts match
   - Test interruption during PENDING state
   ```

2. **Fixture Updates**:
   - Add `tests/fixtures/vitest-v2-pending/` with tests that emit PENDING
   - Add `tests/fixtures/mixed-lifecycle/` with various state transitions

3. **Regression Tests**:
   - Run all existing integration tests
   - Verify Svelte counts are correct (6758 passed)
   - Test other verified libraries remain accurate

### Performance Tests
1. **Benchmarks**:
   - Benchmark GroupManager.GetStats() with 10k+ tests
   - Compare performance vs old duplicate tracking
   - Ensure no performance regression

## Verification Steps

### Manual Testing
1. **Svelte Repository**:
   ```bash
   cd /tmp/3pio-open-source/svelte
   rm -rf .3pio
   3pio pnpm exec vitest run
   # Verify: Console shows 6758 passed (not 6707)
   ```

2. **Other Verified Libraries**:
   - Test Jest (facebook/jest)
   - Test Vitest v2 (vuejs/vueuse)
   - Test pytest (pallets/flask)
   - Test Go (google/uuid)
   - Test Rust (serde-rs/serde)

### Automated Verification
1. **CI Pipeline**:
   - Add test case for PENDING→PASS transitions
   - Add assertion for console/report count matching
   - Run against all supported test runners

2. **Regression Suite**:
   ```bash
   make test
   go test ./tests/integration_go -run TestVitestV2
   ```

## Rollback Plan

If issues discovered:
1. Keep old tracking code commented out initially
2. Add feature flag `THREEPIO_LEGACY_TRACKING=1` to revert
3. Can hot-fix by restoring orchestrator counters

## Success Criteria

- [ ] Console output shows correct test counts for Svelte (6758 passed)
- [ ] All integration tests pass
- [ ] No performance regression
- [ ] Console and report counts always match
- [ ] All verified libraries maintain correct counts
- [ ] No new bugs introduced

## Timeline

- Phase 1: 30 minutes (expose stats)
- Phase 2: 20 minutes (add accessors)
- Phase 3: 40 minutes (remove duplicate tracking)
- Phase 4: 30 minutes (update console)
- Phase 5: 30 minutes (edge cases)
- Testing: 60 minutes
- **Total**: ~3.5 hours

## Notes

- GroupManager already handles state transitions correctly
- This simplifies architecture by removing duplicate logic
- Future benefit: Easier to add new test states (e.g., TODO, FLAKY)
- Maintains backward compatibility with existing reports