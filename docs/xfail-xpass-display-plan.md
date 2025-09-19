# Implementation Plan: Display xfail/xpass in Console and Reports

## Overview
This plan outlines the necessary changes to make xfail (expected failures) and xpass (unexpected passes) visible in both console output and generated reports. The implementation ensures these statuses are clearly distinguished from regular pass/fail/skip statuses.

## Current State
- Backend correctly processes XFAIL and XPASS events from pytest
- Console output shows xfail/xpass tests as "0 passed"
- Reports display xfail/xpass tests with regular checkmarks (✓)
- No distinction in visual presentation or counting

## Implementation Tasks

### 1. Orchestrator Console Output (`internal/orchestrator/orchestrator.go`)

#### Add Counter Fields
```go
type Orchestrator struct {
    // ... existing fields ...
    xfailedTests  int
    xpassedTests  int
    xfailedGroups int
    xpassedGroups int
}
```

#### Update Console Summary Display (lines 626-648)
- Modify the `Results:` line format to conditionally include xfail/xpass counts
- Only show xfail/xpass counts when they are > 0
- Format examples:
  - Normal: `Results: 10 passed, 2 failed, 12 total`
  - With xfail: `Results: 10 passed, 2 failed, 3 xfailed, 15 total`
  - With both: `Results: 10 passed, 2 failed, 3 xfailed, 1 xpassed, 16 total`
  - With skip: `Results: 10 passed, 2 failed, 1 skipped, 3 xfailed, 16 total`

#### Update Event Handlers
- Handle `TestStatusXFail` and `TestStatusXPass` in IPC event processing
- Increment appropriate counters when receiving these statuses

### 2. Report Status Symbols (`internal/report/group_manager.go`)

#### Update Test Status Icons (lines 880, 1075)
```go
switch tc.Status {
case TestStatusPass:
    icon = "✓"
case TestStatusFail:
    icon = "✗"
case TestStatusSkip:
    icon = "○"
case TestStatusXFail:
    icon = "⊗"  // Expected failure
case TestStatusXPass:
    icon = "⊕"  // Unexpected pass
default:
    icon = "•"
}
```

Alternative ASCII-safe symbols:
- XFAIL: `xf` or `~`
- XPASS: `xp` or `!`

### 3. Report Summary Statistics

#### Update Summary Sections
- Modify summary generation to include xfail/xpass counts
- Add to both file-level and overall summaries
- Format: "X passed, Y failed, Z skipped, A xfailed, B xpassed"

#### Update Group Statistics Display
- Include xfail/xpass in `TestGroupStats` calculations
- Display in group summary tables when counts > 0

### 4. Test Failure Classification

#### Exit Code Handling
- XFAIL tests should NOT cause non-zero exit codes
- XPASS tests may warrant warnings but shouldn't fail by default
- Only actual FAIL status should trigger failure exit codes

#### Group Status Determination
- Groups with only xfail tests should show as PASS (expected behavior)
- Groups with xpass tests might show a warning indicator

### 5. Visual Enhancements

#### Console Color Coding (if colors are added later)
- XFAIL: Gray/muted (expected, not concerning)
- XPASS: Yellow/amber (unexpected, needs attention)
- FAIL: Red (actual failure)
- PASS: Green (success)

#### Report Formatting
- Add xfail reason display when available
- Group xfail/xpass tests separately in detailed listings
- Show xfail reason as a note under the test name

### 6. Report Generation Updates

#### Modify Report Functions
- Update `generateGroupBasedReport` to handle xfail/xpass
- Include xfail/xpass in statistics calculations
- Add xfail reason to test detail output

#### Update Markdown Templates
- Add legend explaining status symbols
- Include xfail/xpass counts in summary tables
- Format xfail reasons as blockquotes or italics

## Implementation Order

1. **Phase 1: Console Output** (High Priority)
   - Add xfail/xpass counters to Orchestrator
   - Update Results line format
   - Handle new statuses in event processing

2. **Phase 2: Report Symbols** (High Priority)
   - Update status icon mapping
   - Ensure consistent symbol usage across all reports

3. **Phase 3: Summary Statistics** (Medium Priority)
   - Update all summary calculations
   - Add conditional display of xfail/xpass counts

4. **Phase 4: Visual Polish** (Low Priority)
   - Add xfail reason display
   - Improve visual hierarchy in reports
   - Add legend/documentation

## Testing Requirements

- Verify xfail tests don't affect exit codes
- Ensure xpass tests are visually distinct
- Test with mixed status scenarios
- Validate count accuracy in summaries
- Check report symbol consistency

## Success Criteria

- [x] Console shows xfail/xpass counts when present
- [x] Reports use distinct symbols for xfail/xpass
- [x] Exit codes remain correct (xfail doesn't fail)
- [x] Summaries accurately reflect all test statuses
- [x] xfail reasons appear in reports when available

## TODO List for Implementation

- [x] Add xfailed/xpassed counters to Orchestrator struct
- [x] Update IPC event handlers to process XFAIL/XPASS statuses
- [x] Modify console Results line format with conditional xfail/xpass display
- [x] Update test status icon mapping in group_manager.go
- [x] Add xfail/xpass to report summary statistics
- [x] Update group status determination logic
- [x] Test exit code behavior with xfail tests
- [x] Add xfail reason display in reports
- [x] Create test fixtures for mixed status scenarios
- [ ] Update documentation with new status explanations
- [x] Run integration tests to verify implementation
- [x] Format code with gofmt

## Implementation Complete

The xfail/xpass display feature has been successfully implemented. The implementation includes:

1. **Console Output**: Now displays xfail and xpass counts conditionally (only when > 0)
2. **Report Icons**: Uses distinct symbols (⊗ for xfail, ⊕ for xpass)
3. **Exit Codes**: Xfail tests correctly return exit code 0
4. **Summary Statistics**: Reports show xfail/xpass counts in summaries
5. **All tests pass**: Integration tests confirm the implementation works correctly