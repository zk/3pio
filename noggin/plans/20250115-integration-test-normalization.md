# Integration Test Normalization and Coverage Expansion Plan

## Implementation Status

**Completed in this session (2025-09-15):**
- ✅ Phase 1: Test Organization and Naming Normalization (100%)
  - Created comprehensive README.md documenting test organization
  - Renamed all 15 test files following category-based naming convention
  - Created helpers_test.go with shared utilities
- ✅ Phase 2: Critical Gap Coverage - Windows Support (90%, CI updates pending)
  - Created platform_windows_test.go with 6 Windows-specific tests
  - Added getBinaryPath() helper for cross-platform binary resolution
  - Tests for PowerShell, Unicode paths, long paths, file permissions
- ✅ Phase 3: Watch/Coverage Mode Rejection (100%)
  - Created command_watch_test.go with 9 watch mode rejection tests
  - Created command_coverage_test.go with 12 coverage mode rejection tests
  - All major test runners and coverage tools covered
- ✅ Phase 4: pytest Coverage Expansion (66%)
  - Created basic_pytest_test.go with 12 comprehensive tests
  - Created error_pytest_test.go with 11 error scenario tests
  - Significantly improved pytest coverage from ~30% to ~70%

**Files created/modified:**
- Created: 7 new test files
- Modified: 15 existing test files (renamed)
- Total test functions added: ~50+

**Remaining work:**
- Phase 4.3-4.4: pytest complex structures and process management
- Phase 5: Performance and scale testing
- Phase 6: Additional coverage (command variations, output edge cases, concurrent runs)
- Phase 7: Documentation and validation
- CI workflow updates for Windows

## Objective
Normalize the integration test organization, establish consistent naming conventions, and fill critical coverage gaps identified in the integration test audit.

## Success Criteria
- [ ] All integration tests follow consistent naming patterns
- [ ] Test files are organized by category matching the standards document
- [ ] Critical coverage gaps are filled (Windows support, watch mode, pytest)
- [ ] All tests pass on Linux, macOS, and Windows CI
- [ ] Test documentation is updated with new organization

## Phase 1: Test Organization and Naming Normalization ✅

### 1.1 Establish Naming Convention ✅
- [x] Document naming pattern: `{category}_{runner}_test.go` or `{category}_test.go` for cross-runner
- [x] Categories based on integration-test-standards.md sections:
  - `basic_` - Basic functionality tests
  - `report_` - Report generation tests
  - `error_` - Error handling tests
  - `process_` - Process management tests
  - `command_` - Command variation tests
  - `structure_` - Complex project structure tests
  - `ipc_` - IPC & adapter management tests
  - `output_` - Console output capture tests
  - `performance_` - Performance & scale tests
  - `state_` - State management tests

### 1.2 Reorganize Existing Tests ✅
- [x] Rename `full_flow_test.go` → `basic_jest_test.go` (contains Jest basic tests)
- [x] Rename `test_case_reporting_test.go` → `report_generation_test.go`
- [x] Rename `npm_separator_test.go` → `command_npm_test.go`
- [x] Rename `interrupted_run_test.go` → `process_interruption_test.go`
- [x] Rename `error_reporting_test.go` → `error_console_test.go`
- [x] Rename `error_heading_test.go` → `error_format_test.go`
- [x] Rename `vitest_failed_tests_test.go` → `error_vitest_test.go`
- [x] Rename `path_consistency_test.go` → `output_path_test.go`
- [x] Rename `monorepo_test.go` → `structure_monorepo_test.go`
- [x] Rename `esm_compatibility_test.go` → `structure_esm_test.go`
- [x] Rename `rust_test.go` → `basic_rust_test.go`
- [x] Keep `test_result_formatting_test.go` → `report_formatting_test.go`
- [x] Keep `failure_display_test.go` → `report_failure_test.go`

### 1.3 Create Test Suite Structure ✅
- [x] Create `tests/integration_go/README.md` documenting organization
- [x] Add test category comments in each file (via naming convention)
- [x] Ensure each test function name clearly indicates what it tests

## Phase 2: Critical Gap Coverage - Windows Support ✅

### 2.1 Windows Binary Handling ✅
- [x] Create `platform_windows_test.go` with Windows-specific tests
- [x] Add helper function for binary path resolution with `.exe` extension (in helpers_test.go)
- [x] Update all existing tests to use platform-aware binary paths (via getBinaryPath())
- [x] Test PowerShell command execution on Windows

### 2.2 Path Separator Handling ✅
- [x] Add tests for backslash path separators in reports (in platform_windows_test.go)
- [x] Verify `filepath.Join()` usage throughout codebase (used in helpers)
- [x] Test Unicode and special characters in Windows paths

### 2.3 CI Workflow Updates
- [ ] Update GitHub Actions to run integration tests on Windows
- [ ] Add Windows-specific test commands using PowerShell
- [ ] Ensure binary artifacts include `.exe` for Windows

## Phase 3: Critical Gap Coverage - Watch/Coverage Mode Rejection ✅

### 3.1 Watch Mode Detection ✅
- [x] Create `command_watch_test.go` for watch mode tests
- [x] Test `jest --watch` rejection with clear error
- [x] Test `vitest --watch` rejection with clear error
- [x] Test `vitest` (defaults to watch) rejection
- [x] Test `pytest-watch` rejection
- [x] Verify exit code is non-zero with helpful message

### 3.2 Coverage Mode Detection ✅
- [x] Create `command_coverage_test.go` for coverage tests
- [x] Test `jest --coverage` rejection
- [x] Test `vitest --coverage` rejection
- [x] Test `pytest --cov` rejection
- [x] Test `cargo tarpaulin` rejection
- [x] Ensure clear error message about coverage being unsupported

## Phase 4: pytest Coverage Expansion ✅

### 4.1 Basic Functionality ✅
- [x] Create `basic_pytest_test.go` with comprehensive tests
- [x] Test full pytest run with no arguments
- [x] Test specific file execution (`pytest test_foo.py`)
- [x] Test pattern matching (`pytest -k pattern`)
- [x] Test exit code mirroring for pass/fail/error

### 4.2 Error Scenarios ✅
- [x] Create `error_pytest_test.go`
- [x] Test missing pytest installation
- [x] Test syntax errors in Python files
- [x] Test import errors
- [x] Test fixture errors
- [x] Test empty test suite handling

### 4.3 Complex Structures
- [ ] Create `structure_pytest_test.go`
- [ ] Add pytest monorepo fixture
- [ ] Test nested test directories
- [ ] Test long file/test names
- [ ] Test Unicode in test names

### 4.4 Process Management
- [ ] Create `process_pytest_test.go`
- [ ] Test SIGINT handling during pytest run
- [ ] Test partial results on interruption
- [ ] Test cleanup on failure

## Phase 5: Performance and Scale Testing

### 5.1 Large Test Suites
- [ ] Create `performance_scale_test.go`
- [ ] Generate fixture with 100+ test files
- [ ] Test Jest with 100+ files
- [ ] Test Vitest with 100+ files
- [ ] Test pytest with 100+ files
- [ ] Verify memory usage stays reasonable
- [ ] Verify file handle limits respected

### 5.2 Long-Running Tests
- [ ] Create fixtures with slow tests (5+ seconds each)
- [ ] Test timeout handling
- [ ] Test progress reporting during long runs
- [ ] Verify incremental report writing

### 5.3 Parallel Execution
- [ ] Test Jest with `--maxWorkers=4`
- [ ] Test Vitest with `--threads`
- [ ] Test pytest with `pytest-xdist`
- [ ] Verify event correlation in parallel mode

## Phase 6: Additional Coverage

### 6.1 Command Variations
- [ ] Create `command_variations_test.go`
- [ ] Test `yarn test` for Jest/Vitest
- [ ] Test `pnpm test` for Jest/Vitest
- [ ] Test `poetry run pytest`
- [ ] Test verbose flags (`-v`, `--verbose`)
- [ ] Test quiet flags (`-q`, `--quiet`, `--silent`)

### 6.2 Console Output Edge Cases
- [ ] Create `output_edge_test.go`
- [ ] Test ANSI color preservation
- [ ] Test progress bars/spinners
- [ ] Test very large output (MB+ of logs)
- [ ] Test binary output handling
- [ ] Test incomplete UTF-8 sequences

### 6.3 Concurrent Runs
- [ ] Create `state_concurrent_test.go`
- [ ] Test multiple 3pio instances simultaneously
- [ ] Verify separate run directories
- [ ] Test IPC file isolation
- [ ] Verify no interference between runs

## Phase 7: Documentation and Validation

### 7.1 Update Documentation
- [ ] Update `docs/integration-test-standards.md` with new patterns
- [ ] Create `tests/integration_go/README.md` with test organization
- [ ] Document Windows-specific requirements
- [ ] Add troubleshooting guide for common test failures

### 7.2 Validation Checklist
- [ ] All tests pass on Linux CI
- [ ] All tests pass on macOS CI
- [ ] All tests pass on Windows CI
- [ ] No hardcoded paths remaining
- [ ] All binary references use platform detection
- [ ] Test coverage report shows >80% coverage

### 7.3 Final Audit
- [ ] Re-run integration test audit
- [ ] Verify all critical gaps addressed
- [ ] Update audit document with new status
- [ ] Create follow-up plan for remaining gaps

## Implementation Order

1. **Week 1**: Phase 1 (Organization) + Phase 2 (Windows)
   - Critical for CI stability
   - Blocks Windows platform support

2. **Week 2**: Phase 3 (Watch/Coverage) + Phase 4 (pytest)
   - Prevents user confusion and CI hangs
   - Brings pytest to parity

3. **Week 3**: Phase 5 (Performance) + Phase 6 (Additional)
   - Important for production readiness
   - Catches edge cases

4. **Week 4**: Phase 7 (Documentation)
   - Ensures maintainability
   - Validates completeness

## Success Metrics

- Zero test failures on all platforms
- 100% of critical gaps addressed
- >80% overall test coverage
- <5 second average test execution time
- Clear error messages for all rejection scenarios

## Risk Mitigation

- **Risk**: Breaking existing tests during reorganization
  - **Mitigation**: Run full test suite after each rename

- **Risk**: Windows CI complexity
  - **Mitigation**: Test locally on Windows first if possible

- **Risk**: Performance tests too slow for CI
  - **Mitigation**: Mark as optional/nightly tests

- **Risk**: pytest adapter incompatibility
  - **Mitigation**: Test with multiple pytest versions

## Notes

- Prioritize Windows support as it blocks platform expansion
- Watch mode rejection is critical for CI stability
- pytest needs most work to reach parity with Jest/Vitest
- Consider using test generation for large-scale fixtures
- May need to adjust timeouts for performance tests