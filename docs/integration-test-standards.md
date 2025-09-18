# Integration Test Standards

This document defines the standard integration test suite that should be implemented for each supported test runner in 3pio.

## Core Test Categories

### 1. Basic Functionality
- Full test run - Execute all tests with no arguments, verify complete execution
- Specific file execution - Run tests from specific file(s), verify targeted execution
- Pattern matching - Run tests matching a pattern/filter, verify correct test selection
- Exit code mirroring - Verify exit codes match underlying test runner (0 for pass, non-zero for fail)

### 2. Console Output
- Minimal per-group summary lines after group completion, including FAIL/PASS/SKIP counts and a report path using `$trun_dir/reports/<sanitized>/index.md`.
- Do not list individual failed test names inline in the console; details live in report files.
- Always print a final `Results:` summary line reflecting overall counts.
- Outcome messages: "Splendid! All tests passed successfully", "Tests completed with some skipped", "All tests were skipped", or "Test failures! <exclamation>".
- Preamble includes: `current_time`, `cwd`, `test_command`, `trun_dir`, and `full_report`.
- Path expectations: console report directory names end with a sanitized suffix (e.g., `string_test_js`); `$trun_dir` maps to the actual run directory on disk.

### 3. Report Generation
- Main report creation - Verify `test-run.md` is generated with correct structure and content
- Output log creation - Verify `output.log` captures all stdout/stderr from test run
- Hierarchical reports - Verify group/file-based report structure in `reports/` directory
- YAML frontmatter - Verify correct metadata (status, counts, timing, command)
- Report content accuracy - Test names, statuses, error messages, and durations properly captured

### 4. Error Handling
- Configuration errors - Handle missing dependencies, invalid config files gracefully
- Test failures - Proper reporting of failing tests with error details and stack traces
- Syntax errors - Handle test files with syntax errors without crashing
- Missing test files - Gracefully handle non-existent test files with appropriate error messages
- Empty test suites - Handle projects with no tests, still create basic structure

### 5. Process Management
- SIGINT handling - Graceful shutdown on Ctrl+C with partial results preserved
- SIGTERM handling - Clean termination on kill signal with state saved
- Quick termination - Handle immediate process kill without hanging
- Partial results - Verify reports exist and are valid even when interrupted

### 6. Command Variations
- Package manager variants - Support `npm test`, `yarn test`, `pnpm test` commands
- Separator handling - Correctly parse `npm test -- file.test.js` format
- Watch mode rejection - Properly detect and reject watch/interactive modes
- Verbose/quiet modes - Handle different verbosity flags without breaking parsing
- Coverage mode - Detect and reject coverage flags (coverage mode is unsupported)

### 7. Complex Project Structures
- Monorepo support - Multiple packages with shared config, single IPC path
- Nested test directories - Deep directory structures with proper path resolution
- Long file/test names - Handle filesystem limits gracefully with truncation
- Special characters - Unicode, spaces, and special chars in paths/names

### 8. IPC & Adapter Management
- IPC file creation - Verify JSONL event stream with correct format
- Adapter injection - Proper reporter/plugin injection into test command
- Adapter cleanup - Temporary adapter files removed after run
- Event completeness - All test discovery, execution, and result events captured
- Event ordering - Events maintain logical order (discovery → start → result)

### 9. Console Output Capture
- Stdout capture - Test console.log output captured and associated with tests
- Stderr capture - Error output properly captured and preserved
- ANSI color handling - Color codes preserved in output.log
- Progress indicators - Handle progress bars/spinners without corruption
- Buffering behavior - Large output volumes handled without loss

### 10. Performance & Scale
- Large test suites - Handle 100+ test files without degradation
- Long-running tests - Tests with extended execution times handled properly
- Parallel execution - Concurrent test execution with proper event correlation
- Memory management - Handle memory-intensive tests without OOM
- File handle limits - Respect system limits for open files

### 11. State Management
- Run directory creation - Unique timestamped directories for each run
- Memorable names - Human-readable run directory names
- State persistence - Reports incrementally written during execution
- Cleanup on failure - Proper cleanup when initialization fails
- Concurrent runs - Multiple 3pio instances can run simultaneously

## Test Implementation Guidelines

### Test Structure
Each integration test should:
1. Set up a clean test environment (remove `.3pio` directory)
2. Execute 3pio with specific arguments
3. Verify expected files and directories are created
4. Check file contents for correctness
5. Clean up test artifacts (unless debugging)

### Assertions to Include
- File existence checks (test-run.md, output.log, reports/)
- Content validation (headers, sections, test results)
- Exit code verification
- Timing information presence
- Error message clarity

### Error Scenarios
Each test runner should have tests for:
- Zero tests found
- All tests passing
- Mixed pass/fail results
- All tests failing
- Configuration errors
- Runtime errors

### Platform Considerations
- Path separator handling (forward vs backslash)
- Line ending differences (LF vs CRLF)
- Process signal availability
- File system case sensitivity
- Permission differences

### Windows CI Requirements

**Critical Windows-Specific Checks**:

#### 1. Binary Extension Handling
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

#### 2. Path Separator Handling
**Issue**: Windows uses backslashes (`\`) while Unix uses forward slashes (`/`) for paths.

**Requirements**:
- Always use `filepath.Join()` for constructing paths
- Use `filepath.Separator` when building path strings
- Never hardcode path separators

#### 3. PowerShell vs Bash in CI
**Issue**: GitHub Actions uses PowerShell on Windows, which has different syntax than bash.

**Requirements**:
- CI workflow must have separate steps for Windows and Unix systems
- Windows steps should use `shell: pwsh` explicitly
- Use PowerShell-specific commands for Windows (e.g., `Test-Path` instead of `test -d`)

#### CI Workflow Validation Checklist
Before merging any PR, ensure:

**Go Code Checks**:
- [ ] All references to `build/3pio` check for Windows and use `build/3pio.exe`
- [ ] All path constructions use `filepath.Join()` instead of string concatenation
- [ ] Runtime OS detection is used where platform-specific behavior is needed

**GitHub Actions Workflow Checks**:
- [ ] Windows-specific steps are properly marked with `if: matrix.os == 'windows-latest'`
- [ ] Windows steps use PowerShell syntax and commands
- [ ] Binary paths in CI scripts account for `.exe` extension on Windows

#### Common Windows Failure Patterns
1. **Binary Not Found**: Missing `.exe` extension for Windows binary
2. **Path Construction Errors**: Hardcoded Unix-style paths or incorrect path separator usage
3. **Command Execution Failures**: Missing `.exe` extension when executing the binary

#### Automated Checks
Consider adding a linter rule or pre-commit hook to check for:
- Hardcoded paths with `/` separators
- Direct references to `build/3pio` without OS checks
- Missing `runtime` import in test files

## Validation Criteria

A test runner implementation is considered complete when:
1. All core test categories have coverage
2. Exit codes correctly mirror the underlying test runner
3. Reports are generated for all scenarios (success, failure, error)
4. Interruption handling works gracefully
5. Console output is fully captured
6. IPC events are complete and ordered
7. Memory and file handles are managed properly
8. Error messages are clear and actionable

## Best Practices

### Test Isolation
- Each test should clean its environment before and after
- Tests should not depend on execution order
- Shared fixtures should be read-only

### Test Maintenance
- Document expected behavior in test comments
- Use descriptive test names
- Group related tests logically
- Keep fixtures minimal and focused
