# Worker-Based Filtering Implementation Plan

## Executive Summary
Implement a context-aware pytest adapter that detects whether it's running in a worker process and disables reporting in workers, eliminating duplicates while preserving all test information through the controller/standalone process.

## Architecture Overview

### Execution Contexts
1. **Worker Mode**: Child process executing assigned tests (xdist worker)
2. **Non-Worker Mode**: Either standalone execution OR xdist controller process

### Core Principle
Workers stay silent. Only non-worker processes (standalone or controller) report test events, preventing double-reporting since the controller already aggregates all worker events through pytest-xdist's hook system.

## Detection Strategy

### Context Detection Logic
The adapter will detect if it's running in a worker process:

**Primary Detection**:
- Check for `PYTEST_XDIST_WORKER` environment variable (present only in workers)

**Secondary Detection** (for verification/fallback):
- Check for `hasattr(config, 'workerinput')` - present in workers
- Check for `hasattr(config, 'workeroutput')` - present in workers
- Check for worker-specific config options

### Reporting Strategy Decision Tree

```
IF PYTEST_XDIST_WORKER environment variable exists:
    → Worker mode detected
    → DO NOT initialize reporter
    → DO NOT send IPC events
    → Return early from all hooks
ELSE:
    → Non-worker mode (standalone or controller)
    → Initialize reporter normally
    → Report all test events (local or from workers)
    → Handle all phases normally
```

## Implementation Details

### Phase 1: Context Detection Module

**Objectives**:
- Create simple, robust detection for worker vs non-worker mode
- Handle edge cases (missing environment variables, older versions)
- Provide clear logging of detected context

**Components**:
1. Simple worker detection function checking `PYTEST_XDIST_WORKER`
2. Fallback mechanisms if environment variable is missing
3. Debug logging to trace detection decision

### Phase 2: Conditional Reporter Initialization

**Objectives**:
- Initialize reporter only in non-worker contexts
- Ensure clean shutdown regardless of context
- Maintain backward compatibility

**Key Decisions**:
- Non-worker processes (standalone/controller) handle all reporting
- Workers stay completely silent (no IPC writes)
- Both standalone and controller modes work identically

### Phase 3: Hook Registration Management

**Objectives**:
- Register hooks appropriately based on context
- Prevent any processing in worker contexts
- Ensure all events are captured in non-worker processes

**Implementation Notes**:
- In worker: Skip all hook registrations or return early from hooks
- In non-worker: Register all hooks and process normally
- Add context check at the beginning of each hook function

### Phase 4: Output Capture Coordination

**Objectives**:
- Ensure stdout/stderr capture works correctly
- Prevent duplicate output capture
- Maintain test output association

**Strategy**:
- Non-worker processes handle all output capture
- Workers don't interfere with output streams
- Output flows naturally through pytest-xdist's aggregation

## Edge Cases & Error Handling

### Edge Case 1: Dynamic Worker Spawning
**Scenario**: Workers are created/destroyed during test run
**Solution**: Detection happens at configure time, remains stable

### Edge Case 2: Mixed Mode Execution
**Scenario**: Some tests run in controller, some in workers
**Solution**: Non-worker process catches all events regardless of source

### Edge Case 3: Plugin Load Order
**Scenario**: Our adapter loads before/after xdist
**Solution**: Defer detection until all plugins are loaded

### Edge Case 4: Custom xdist Configurations
**Scenario**: Non-standard worker configurations
**Solution**: Multiple detection methods provide redundancy

### Edge Case 5: Network-Distributed Testing
**Scenario**: Workers on different machines
**Solution**: Environment variable detection still works

## Progress Checklist

### Research & Analysis ✓
- [ ] Analyze current pytest adapter code for modification points
- [ ] Study pytest-xdist source to understand hook flow
- [ ] Document all environment variables and config attributes
- [ ] Identify all hooks that might be affected
- [ ] Create test harness to reproduce duplicate issue

### Design & Planning ✓
- [ ] Finalize detection logic flowchart
- [ ] Document state machine for reporter lifecycle
- [ ] Create decision matrix for edge cases
- [ ] Design logging strategy for debugging
- [ ] Plan rollback strategy if issues arise

### Implementation Phase 1: Detection
- [ ] Implement context detection module
- [ ] Add comprehensive debug logging
- [ ] Create unit tests for detection logic
- [ ] Test with various xdist configurations
- [ ] Verify detection with different pytest versions

### Implementation Phase 2: Conditional Initialization
- [ ] Modify pytest_configure to use detection
- [ ] Update reporter initialization logic
- [ ] Handle initialization failures gracefully
- [ ] Add initialization status logging
- [ ] Test initialization in all contexts

### Implementation Phase 3: Hook Management
- [ ] Audit all existing hooks for context checks
- [ ] Add early returns for worker context
- [ ] Ensure controller processes all events
- [ ] Verify no events are lost
- [ ] Test hook behavior in each context

### Implementation Phase 4: Testing & Validation
- [ ] Run with langchain test suite
- [ ] Verify test count matches baseline
- [ ] Check for any missing test results
- [ ] Validate output capture completeness
- [ ] Performance comparison with baseline

### Implementation Phase 5: Polish & Documentation
- [ ] Remove debug logging or make conditional
- [ ] Update documentation with architecture
- [ ] Add troubleshooting guide
- [ ] Create migration notes
- [ ] Update CLAUDE.md with new behavior

## Test Plan

### Unit Tests

#### Test Suite 1: Context Detection
**Purpose**: Verify accurate detection of worker vs non-worker mode

**Test Cases**:
1. **test_non_worker_detection**: No PYTEST_XDIST_WORKER env var, should detect non-worker
2. **test_worker_detection**: PYTEST_XDIST_WORKER='gw0' set, should detect worker
3. **test_missing_environment**: Handle missing/malformed environment variables
4. **test_version_compatibility**: Test with different pytest/xdist versions

#### Test Suite 2: Reporter Initialization
**Purpose**: Verify reporter initializes only when appropriate

**Test Cases**:
1. **test_worker_no_init**: Reporter should NOT initialize in worker
2. **test_non_worker_init**: Reporter SHOULD initialize in non-worker mode
3. **test_init_failure_handling**: Graceful handling of IPC path issues
4. **test_multiple_init_attempts**: Prevent double initialization

#### Test Suite 3: Hook Behavior
**Purpose**: Verify hooks behave correctly in each context

**Test Cases**:
1. **test_worker_hooks_disabled**: All hooks return early in worker
2. **test_non_worker_hooks_active**: All hooks process in non-worker mode
3. **test_hook_error_handling**: Exceptions don't break test run
4. **test_collection_phase_hooks**: Collection events handled properly
5. **test_session_finish_behavior**: Clean shutdown in all contexts

### Integration Tests

#### Test Suite 1: Full Run Scenarios
**Purpose**: End-to-end validation with real test suites

**Test Cases**:
1. **test_no_xdist_run**: Run without -n flag, verify normal operation
2. **test_xdist_auto_run**: Run with -n auto, verify no duplicates
3. **test_xdist_specific_workers**: Run with -n 4, verify correct count
4. **test_mixed_test_results**: Pass/fail/skip/xfail all recorded once
5. **test_test_reruns**: --reruns flag doesn't cause issues

#### Test Suite 2: Output Verification
**Purpose**: Ensure all test output is captured correctly

**Test Cases**:
1. **test_stdout_capture**: Print statements appear in reports
2. **test_stderr_capture**: Error output appears in reports
3. **test_no_duplicate_output**: Output not duplicated
4. **test_output_association**: Output linked to correct tests
5. **test_interleaved_output**: Parallel execution output handled

#### Test Suite 3: Compatibility Matrix
**Purpose**: Verify compatibility across versions

**Test Cases**:
1. **test_pytest_versions**: Test with pytest 6.x, 7.x, 8.x
2. **test_xdist_versions**: Test with xdist 2.x, 3.x
3. **test_python_versions**: Test with Python 3.8-3.12
4. **test_platform_differences**: Windows/Mac/Linux behave identically
5. **test_large_test_suites**: Scales to 10K+ tests

### Validation Criteria

#### Correctness Criteria
1. Test count MUST equal baseline (no duplicates, no missing)
2. Test results MUST match baseline (pass/fail status)
3. Test duration SHOULD be within 10% of baseline
4. Exit codes MUST match baseline
5. All test names MUST be present exactly once

#### Performance Criteria
1. No more than 5% overhead vs baseline
2. IPC file size reduced by ~50% (no duplicates)
3. Memory usage stable (no leaks from tracking)
4. Startup time < 100ms additional
5. No impact on test execution performance

#### Robustness Criteria
1. Handles worker crashes gracefully
2. Survives malformed environment variables
3. Works with custom pytest plugins
4. Handles filesystem errors (disk full, etc.)
5. Clean shutdown even on SIGTERM/SIGINT

## Risk Mitigation

### Risk 1: Breaking Changes in pytest-xdist
**Mitigation**: Version detection with compatibility modes

### Risk 2: Incomplete Event Capture
**Mitigation**: Non-worker process as authoritative source sees all events

### Risk 3: Performance Degradation
**Mitigation**: Early returns minimize processing in workers

### Risk 4: Complex Debugging
**Mitigation**: Comprehensive logging with context information

### Risk 5: Backward Compatibility
**Mitigation**: Feature flag to enable/disable new behavior

## Success Metrics

1. **Primary**: Test count matches baseline exactly (1,371 for langchain)
2. **Primary**: No duplicate test events in IPC file
3. **Secondary**: IPC file size reduced by 40-50%
4. **Secondary**: Clean debug logs with clear context detection
5. **Tertiary**: Improved performance from reduced IPC writes

## Summary

### Core Solution
Only **non-worker processes** (standalone or xdist controller) report test events to 3pio. **Workers stay completely silent**. This eliminates duplicates since each test result flows through the controller exactly once.

### Detection Logic
- **Worker**: Has `PYTEST_XDIST_WORKER` env var → Don't initialize reporter
- **Non-Worker**: No `PYTEST_XDIST_WORKER` env var → Initialize reporter normally

### Key Benefits
- Eliminates duplicate test reporting
- Reduces IPC file size by ~50%
- Maintains complete test visibility
- Works with all pytest-xdist configurations
- No performance impact on test execution

### Timeline Estimate
- Research & Analysis: 2-3 hours
- Implementation: 4-6 hours
- Testing & Validation: 3-4 hours
- Documentation: 1-2 hours
- **Total: 10-15 hours**

### Next Steps
1. Begin with Research & Analysis phase
2. Create minimal reproduction test case
3. Implement detection module with logging
4. Test with langchain suite to validate approach
5. Roll out incrementally with feature flag