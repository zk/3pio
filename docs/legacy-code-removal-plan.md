# Legacy Code Removal Plan

## Status: ALL PHASES COMPLETED ✅

### Completion Summary
- **Phase 1 (Critical Bug Fix)**: ✅ Completed - Fixed hang issue with large Go repositories
- **Phase 2 (File Cleanup)**: ✅ Completed - Removed all legacy file-tracking code
- **Phase 3 (Report Manager)**: ✅ Completed - Removed file-based initialization

### Impact
- **Performance**: Eliminated 242+ subprocess calls for grpc-go (from minutes/hang to 0.025s)
- **Code Quality**: Removed ~250 lines of legacy code
- **Maintainability**: Cleaner, simpler codebase aligned with universal abstractions
- **Architecture**: Complete migration to group-based test organization model

## Overview
This document tracks legacy code from the pre-universal-abstractions era that needs to be removed or refactored. The universal abstractions migration introduced a group-based hierarchical test organization model, but some file-centric legacy code remains.

## Critical Issues

### 1. Go Test Runner - buildTestToFileMap() Performance Bug
**Location**: `/internal/runner/definitions/gotest.go:738-790`
**Severity**: HIGH - Causes hangs on large repositories with go.mod errors
**Issue**:
- Runs `go test -list` for EACH package in a repository (242+ subprocess calls for grpc-go)
- Each call can hang indefinitely with go.mod parse errors
- No timeout on subprocess execution
- Provides no real value - all tests in a package map to the same "representative file"

**Impact**:
- 3pio appears to hang when testing large Go repositories
- High CPU usage (188%+) when stuck in loop
- Poor user experience

**Solution**:
- Remove entire `buildTestToFileMap()` function
- Remove `listTestsInPackage()` function (lines 793-816)
- Remove `testToFileMap` field from GoTestDefinition struct
- Use existing `getFilePathForPackage()` fallback instead
- The universal abstractions already handle Go packages as groups

## Legacy Code Audit

### Go Test Runner (`/internal/runner/definitions/gotest.go`)

#### File-Centric Legacy Code
1. **testToFileMap** (line 24) - Maps test names to file paths
   - Status: REMOVE - Not needed with package-level groups

2. **buildTestToFileMap()** (lines 738-790)
   - Status: REMOVE - Expensive and provides no value

3. **listTestsInPackage()** (lines 793-816)
   - Status: REMOVE - Called by buildTestToFileMap, causes hangs

4. **getFilePathForTest()** (lines 681-700)
   - Status: REFACTOR - Simplify to always use package-level mapping

5. **File-level tracking fields** (lines 29-33):
   - `fileStarted`, `fileStartTimes`, `fileTestCounts`, `fileTestsDone`, `fileStatuses`
   - Status: REVIEW - May be partially needed for internal tracking

6. **fileGroups** (line 48)
   - Status: REVIEW - Check if used by universal abstractions

#### Comments Referencing File-Level Organization
- Lines 741-743: Comment acknowledging Go's package-level limitation
- Line 777: "This is a limitation of Go's test runner"
- Status: UPDATE - Reflect new group-based approach

### Jest Adapter (`/internal/adapters/jest.js`)
- Status: MIGRATED ✅
- Current adapter uses group events (`testGroupDiscovered`, `testGroupStart`, `testGroupResult`)
- Legacy file: `/internal/adapters/jest-old.js` still exists with file-centric events
- Action: DELETE `jest-old.js`

### Vitest Adapter (`/internal/adapters/vitest.js`)
- Status: MIGRATED ✅
- Uses group events (`testGroupDiscovered`, `testGroupStart`, `testGroupResult`)
- No legacy code found

### Pytest Adapter (`/internal/adapters/pytest_adapter.py`)
- Status: MIGRATED ✅
- Contains comments: "testFileStart event removed - using group events instead" (line 359)
- Contains comments: "testFileResult event removed - using group events instead" (line 504)
- No legacy code found

### Report Manager (`/internal/report/`)
- Status: PARTIALLY MIGRATED ⚠️
- Still uses `testFiles` parameter in `Initialize()` method (line 126)
- Has `TotalFiles` field in run state (line 136)
- May need refactoring to use group-based initialization

### IPC Event Types
- Status: CLEANUP NEEDED ⚠️
- Deprecated event types no longer used by current adapters:
  - `testFileStart` - removed from all adapters
  - `testFileResult` - removed from all adapters
  - `testFileResultWithDuration` - removed from all adapters
- Legacy file `/internal/adapters/jest-old.js` still uses these

### Console Output Formatter (`/internal/output/`)
- Status: MIGRATED ✅
- No file-centric code found
- Properly displays group hierarchies

## Summary of Legacy Code Found

### Critical Performance Issues
1. **Go Test Runner** - `buildTestToFileMap()` causing hangs on large repos
   - Runs subprocess for each package (242+ calls for grpc-go)
   - No timeout on subprocess execution
   - Provides no value with group-based model

### Obsolete Files
1. `/internal/adapters/jest-old.js` - Old Jest adapter using file-centric events

### Partially Migrated Components
1. **Report Manager** - Still has file-centric initialization parameters
2. **Go Test Runner** - Still has file-tracking fields that may be unused

### Already Migrated Components ✅
1. Current Jest adapter
2. Vitest adapter
3. Pytest adapter
4. Console output formatter

## Removal Priority

### Phase 1: Critical Bug Fixes (Immediate) ✅ COMPLETED
1. **Fix Go test hang issue**:
   - ✅ Removed `buildTestToFileMap()` function (lines 738-790)
   - ✅ Removed `listTestsInPackage()` function (lines 793-816)
   - ✅ Removed call to `buildTestToFileMap()` in `GetTestFiles()` (line 226)
   - ✅ Removed `testToFileMap` field from struct (line 24)
   - ✅ Updated `getFilePathForTest()` to always use fallback
   - ✅ Fixed unused import (`bytes`)
   - ✅ Updated tests to reflect removal

**Result**: 3pio now fails quickly (0.025s) on go.mod parse errors instead of hanging

### Phase 2: File Cleanup (Next Sprint) ✅ COMPLETED
1. ✅ Deleted `/internal/adapters/jest-old.js`
2. ✅ Removed unused file-tracking fields from GoTestDefinition:
   - ✅ `fileStarted` (line 28)
   - ✅ `fileStartTimes` (line 29)
   - ✅ `fileTestCounts` (line 30)
   - ✅ `fileTestsDone` (line 31)
   - ✅ `fileStatuses` (line 32)
   - ✅ `fileGroups` (line 47) - confirmed not needed for universal abstractions
   - ✅ `FileGroupInfo` type definition
3. ✅ Cleaned up comments referencing file-level limitations
4. ✅ Removed all usage of file-tracking in `handleTestRun()` and `handleTestResult()`

**Result**: Cleaner codebase with no legacy file-tracking code

### Phase 3: Report Manager Refactoring ✅ COMPLETED
1. ✅ Updated `Initialize()` to remove `testFiles` parameter
2. ✅ Removed `TotalFiles` field from TestRunState
3. ✅ Removed all file registration methods (ensureTestFileRegisteredInternal, registerTestFileInternal)
4. ✅ Updated orchestrator to remove GetTestFiles() calls
5. ✅ Updated all tests to use new Initialize() signature

**Result**: Complete migration to group-based reporting without file-centric initialization

## Testing Requirements

Before removing legacy code:
1. Ensure all existing tests pass
2. Test with large repositories (grpc-go, kubernetes)
3. Verify group-based reports are generated correctly
4. Check that no functionality is lost

## Migration Notes

The universal abstractions migration (see `/docs/universal-abstractions-migration-plan.md`) established:
- Groups as the primary abstraction (not files)
- Hierarchical test organization
- Package-level groups for Go (acknowledging its limitations)
- File paths become group names where appropriate

## Related Issues
- Performance issue with large Go repositories
- Hang when go.mod has parse errors
- High CPU usage during test discovery

## Success Criteria
1. No hangs when testing large repositories
2. Faster test discovery (no unnecessary subprocess calls)
3. Cleaner, more maintainable codebase
4. All tests continue to pass
5. No loss of functionality