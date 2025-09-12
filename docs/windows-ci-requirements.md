# Windows CI Requirements

## Overview

This document outlines the specific requirements and checks needed for Windows CI builds to prevent common failures that have occurred multiple times in the past.

## Critical Windows-Specific Checks

### 1. Binary Extension Handling

**Issue**: Windows executables require the `.exe` extension, but Unix systems don't use file extensions for executables.

**Requirements**:
- All test files that reference the 3pio binary must check `runtime.GOOS` and append `.exe` for Windows
- Use `filepath.Join()` for path construction to handle platform-specific separators

**Example Implementation**:
```go
binaryName := "3pio"
if runtime.GOOS == "windows" {
    binaryName = "3pio.exe"
}
binaryPath := filepath.Join("../../build", binaryName)
```

**Files to Check**:
- `tests/integration_go/test_result_formatting_test.go`
- `tests/integration_go/error_reporting_test.go`
- `tests/integration_go/test_case_reporting_test.go`
- Any new integration test files

### 2. Path Separator Handling

**Issue**: Windows uses backslashes (`\`) while Unix uses forward slashes (`/`) for paths.

**Requirements**:
- Always use `filepath.Join()` for constructing paths
- Use `filepath.Separator` when building path strings
- Never hardcode path separators

### 3. PowerShell vs Bash in CI

**Issue**: GitHub Actions uses PowerShell on Windows, which has different syntax than bash.

**Requirements**:
- CI workflow must have separate steps for Windows and Unix systems
- Windows steps should use `shell: pwsh` explicitly
- Use PowerShell-specific commands for Windows (e.g., `Test-Path` instead of `test -d`)

## CI Workflow Validation Checklist

Before merging any PR, ensure:

### Go Code Checks
- [ ] All references to `build/3pio` check for Windows and use `build/3pio.exe`
- [ ] All path constructions use `filepath.Join()` instead of string concatenation
- [ ] Runtime OS detection is used where platform-specific behavior is needed

### GitHub Actions Workflow Checks
- [ ] Windows-specific steps are properly marked with `if: matrix.os == 'windows-latest'`
- [ ] Windows steps use PowerShell syntax and commands
- [ ] Binary paths in CI scripts account for `.exe` extension on Windows

## Common Failure Patterns

### Pattern 1: Binary Not Found
```
3pio binary not found at D:\a\3pio\3pio\build\3pio. Run 'make build' first.
```
**Root Cause**: Missing `.exe` extension for Windows binary.

### Pattern 2: Path Construction Errors
```
cannot find file: tests\fixtures\basic-jest
```
**Root Cause**: Hardcoded Unix-style paths or incorrect path separator usage.

### Pattern 3: Command Execution Failures
```
'3pio' is not recognized as an internal or external command
```
**Root Cause**: Missing `.exe` extension when executing the binary.

## Testing Requirements

### Local Windows Testing
Before pushing changes that affect cross-platform compatibility:
1. Test on Windows if possible
2. Or use GitHub Actions draft PR to validate Windows CI

### Automated Checks
Consider adding a linter rule or pre-commit hook to check for:
- Hardcoded paths with `/` separators
- Direct references to `build/3pio` without OS checks
- Missing `runtime` import in test files

## Historical Context

This is the third documented Windows CI failure due to binary extension issues:
1. Initial implementation didn't account for Windows
2. Second fix missed some test files
3. Current fix (this PR) addresses all known instances

To prevent future occurrences, all new integration tests must follow the patterns documented here.

## References

- [Go filepath package documentation](https://pkg.go.dev/path/filepath)
- [GitHub Actions Windows runners](https://docs.github.com/en/actions/using-github-hosted-runners/about-github-hosted-runners#supported-runners-and-hardware-resources)
- [Cross-platform Go development best practices](https://go.dev/doc/code#Testing)