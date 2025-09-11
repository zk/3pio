# Successfully Tested Open Source Projects

This document tracks open source projects that have been successfully tested with 3pio.

## Summary

| Project | Test Framework | Module Type | Status | Notes |
|---------|----------------|-------------|--------|-------|
| ms | Jest | ES Module | ✅ Success | Fixed with ES module compatibility |
| unplugin-auto-import | Vitest | ES Module | ✅ Success | Fixed vitest watch mode hanging issue |

## Detailed Results

### ms (Microsoft Millisecond Conversion Utility)
- **Repository**: https://github.com/vercel/ms  
- **Test Framework**: Jest
- **Module Type**: ES Module (`"type": "module"`)
- **Package Manager**: pnpm
- **Test Command**: `npm test` (runs `pnpm run test:nodejs && pnpm run test:edge`)

**Results**:
- ✅ **Tests Passed**: 4 passed, 167 total tests
- ✅ **3pio Compatibility**: Works with ES module compatibility fix
- ✅ **Report Generation**: Successfully generated test-run.md with COMPLETE status
- ✅ **Console Output**: Proper test results displayed

**Initial Issue**: 
- Failed with `module is not defined` error before ES module compatibility fix
- Required `pnpm` installation

**Resolution**:
- Added ES module detection in 3pio
- Jest adapter now uses `.cjs` extension for ES module projects
- Installed pnpm dependency

**Test Output**:
```
PASS src/format.test.ts
PASS src/index.test.ts  
PASS src/parse-strict.test.ts
PASS src/parse.test.ts

Test Suites: 4 passed, 4 total
Tests:       167 passed, 167 total
Time:        0.261 s
```

### unplugin-auto-import
- **Repository**: https://github.com/unjs/unplugin-auto-import
- **Test Framework**: Vitest
- **Module Type**: ES Module (`"type": "module"`)
- **Package Manager**: pnpm
- **Test Command**: `npm test` (runs `vitest`)

**Results**:
- ✅ **Tests Passed**: All tests pass
- ✅ **3pio Compatibility**: Works with vitest watch mode fix
- ✅ **Report Generation**: Successfully generated test-run.md
- ✅ **Console Output**: Proper test results displayed

**Initial Issue**: 
- Tests were hanging because the default `npm test` command was running vitest in watch mode
- Vitest watch mode waits for user input, causing 3pio to hang indefinitely

**Resolution**:
- Modified 3pio to detect when vitest is running in watch mode
- Implemented best-effort conversion from watch mode to run mode
- Automatically replaces watch mode command with `vitest run` to execute tests once and exit

**Key Fix**:
- 3pio now detects test runner commands that would run in watch mode
- Automatically converts them to run mode for compatibility
- This ensures tests complete and 3pio can generate reports properly

### agno (Multi-Agent System Library)
- **Repository**: https://github.com/agno-agi/agno
- **Test Framework**: pytest
- **Module Type**: Python package
- **Package Manager**: pip
- **Test Command**: `pytest`

**Initial Results (Before Fix)**:
- ❌ **Tests Failed**: Exit code 2 - tests fail during collection phase
- ⚠️ **3pio Compatibility**: Correctly reports failure but captures no output
- ❌ **Report Generation**: Report generated but contains no test output
- ❌ **Console Output**: No output captured from pytest

**Issues Identified**:
1. **Project has dependency errors** - Missing `docstring_parser` module causes import failure during collection
2. **3pio pytest adapter wasn't capturing collection-phase errors** - Adapter hooks engaged too late

**Fix Applied**:
- Modified pytest adapter to hook into `pytest_configure()` for immediate capture
- Added `pytest_collectreport()` hook to capture collection errors
- Added `pytest_collection_finish()` hook to track collection completion
- Adapter now starts capturing with special `__collection__` file path

**Results After 3pio Fix**:
- ✅ **Collection errors now captured** - Full traceback and error messages captured
- ✅ **IPC events generated** - New events: `collectionStart`, `collectionError`, `collectionFinish`
- ✅ **Output.log populated** - Contains complete error information
- ✅ **3pio provides useful feedback** - Users can now see why tests fail to start

**Root Cause Identified**:
- Missing `docstring-parser` Python package (despite being in pyproject.toml)
- Once installed, tests run successfully

**Final Results (After Installing Dependencies)**:
- ✅ **Tests Passed**: 11 passed, 9 skipped (async tests)
- ✅ **3pio Compatibility**: Full test execution and reporting
- ✅ **Report Generation**: Complete test-run.md with all test details
- ✅ **Console Output**: All test output captured correctly

**Test Details**:
- File: `tests/unit/reader/test_json_reader.py`
- Tests cover JSON reading functionality
- Sync tests: 11 (all passed)
- Async tests: 9 (skipped)
- Total execution time: 0.807s

**Key Achievement**:
- 3pio now handles the entire pytest lifecycle, including collection phase failures
- This is critical for real-world projects where dependency issues are common
- The agno case validates both the pytest adapter improvements and demonstrates a typical open source project challenge

---

## Project Selection Criteria

Projects were selected to test different combinations of:
- Test frameworks (Jest, Vitest, pytest, Go test)
- Module types (CommonJS, ES Module)
- Package managers (npm, pnpm, yarn)
- Project structures and configurations

## Testing Process

1. Clone project to `open-source/` directory
2. Install dependencies using project's package manager
3. Run `3pio [test-command]` 
4. Verify test execution and report generation
5. Document results and any issues encountered
6. Implement fixes if needed and retest

## Future Test Candidates

Additional projects to test:
- **Vitest projects**: Find popular projects using Vitest
- **pytest projects**: Python projects with pytest
- **Go test projects**: Go projects with standard testing
- **Different configurations**: Various Jest/Vitest configurations