# Future Work

This document tracks planned improvements and known limitations that should be addressed in future versions of 3pio.

## Go Test Runner Package-Level Abstraction

**Issue**: Go's test runner operates at the package level, not the file level, which doesn't match the abstractions used by JavaScript/Python test runners.

**Current State**: 
- Go test JSON output only includes package and test name, not source file information
- We attempt to map tests to files by parsing source files, but this is imperfect
- All tests in a package may get incorrectly attributed to the first test file
- File-level timing can be inaccurate when multiple test files exist in a package

**Proposed Solution**:
1. Recognize Go's package-level abstraction as the primary unit of organization
2. Consider reporting Go tests at the package level by default
3. Optionally provide file-level breakdown when possible, but clearly indicate it's a best-effort approximation
4. Alternative: Run `go test` separately for each test file (e.g., `go test -run TestName path/to/specific_test.go`) to get true file-level isolation

**Impact**: This would make Go test reporting more accurate and aligned with Go's actual testing model, while still providing useful information for AI agents consuming the reports.

## Other Future Work

(Add additional future work items here as they are identified)