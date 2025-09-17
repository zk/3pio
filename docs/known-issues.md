# Known Issues and Gotchas

This document describes known issues, limitations, and workarounds for 3pio.

## Error Display Behavior

### Configuration and Startup Errors
**Important**: As of the latest version, 3pio now displays configuration and startup errors directly in the console output for immediate user feedback. This includes:
- Missing test files errors
- Syntax errors in test files
- Test runner configuration issues
- Missing dependencies
- Invalid command arguments

The orchestrator automatically detects these types of errors and displays the relevant error messages from the test runner's output, ensuring users don't have to dig through log files to understand what went wrong.

## Test Discovery

3pio uses **dynamic test discovery** as the standard approach:

### Dynamic Discovery (Current Standard)
- Test files are discovered during execution, not beforehand
- Files are registered as they send their first event
- Shows "Test files will be discovered and reported as they run"
- Log files and reports created on demand as tests execute
- Works consistently across all test runners (Jest, Vitest, pytest, Go test, Cargo test)

### Static Discovery (Legacy - To Be Removed)
- Legacy code exists for pre-execution test discovery but is no longer used
- Some stubs remain in the codebase but will be removed in future updates
- All test runners now use dynamic discovery for consistency

**Note**: The `GetTestFiles()` method intentionally returns an empty array for all runners to enable dynamic discovery.

## Test Runner Detection

- Commands invoked via `npx`, `yarn`, or `pnpm` require special handling to detect the actual test runner
- The system checks both the package manager and the subsequent test runner argument

## Vitest-Specific Behaviors

### Test Discovery Limitations

- **`vitest list` is unreliable**: The `vitest list` command doesn't just list files - it runs tests in watch mode, making it unsuitable for dry runs
- **Dynamic discovery mode**: When running `npm run test` with Vitest, 3pio uses dynamic discovery mode since test files cannot be determined upfront
- **Explicit files work**: When specific test files are provided as arguments, they are extracted directly and tracked from the start

### npm run Command Handling

When using `npm run test` with Vitest, the reporter arguments must be properly separated with `--`:
- 3pio automatically adds the `--` separator before reporter arguments
- This ensures Vitest receives the reporter configuration correctly

## Jest-Specific Behaviors

### Console Output Handling

**Important**: Jest and Vitest have intentionally different reporter configurations in 3pio:

#### Jest Configuration
Jest's handling of console output has several important characteristics:

1. **testResult.console is always undefined** - Despite being documented in the Jest Reporter API, this property is not populated in practice (verified with Jest 29.x)
2. **Direct stdout/stderr writes bypass Jest** - Using `process.stdout.write()` or `process.stderr.write()` directly will output immediately without Jest's formatting
3. **Console methods are intercepted** - Methods like `console.log()` are captured and formatted with stack traces by Jest
4. **Jest does NOT include the default reporter** - When 3pio runs Jest with `--reporters`, it intentionally excludes the default reporter. This means:
   - No duplicate output in the console
   - All test output is captured via IPC events with file path associations
   - Individual test log files are created from IPC events, not by parsing output.log
   - Clean, minimal console output with only 3pio's formatted results

#### Vitest Configuration
**Vitest DOES include the default reporter** - This is an intentional difference from Jest:
   - Provides better user experience with familiar Vitest output
   - Users see progress indicators and test results in real-time
   - 3pio captures output in parallel without interfering with the default reporter
   - This dual output approach works well with Vitest's architecture

For detailed investigation and implications, see [Jest Console Handling](./jest-console-handling.md).

### Reporter Flag Order Critical Requirement

Jest has a critical parsing issue with the `--reporters` flag that requires careful handling:

#### The Problem
The `--reporters` flag in Jest uses a greedy parsing algorithm - it treats ALL subsequent arguments as reporter module names until it encounters another flag starting with `--`. This means:

- ❌ **Wrong**: `jest --reporters /path/to/reporter file.test.js` 
  - Jest interprets `file.test.js` as a reporter module name
- ❌ **Wrong**: `npm test -- --reporters /path/to/reporter --coverage --verbose`
  - Jest interprets `--coverage` and `--verbose` as reporter module names
  
#### The Solution
The `--reporters` flag **must come LAST** in the command line, after all other Jest options and test file paths:

- ✅ **Correct**: `jest file.test.js --reporters /path/to/reporter`
- ✅ **Correct**: `jest --coverage --verbose --reporters /path/to/reporter`
- ✅ **Correct**: `npm test -- --coverage --verbose --reporters /path/to/reporter`

#### Examples with Package Managers

When using npm scripts with the `--` separator:
- ✅ **Correct**: `npm test -- --coverage --bail --reporters /path/to/reporter`
- ✅ **Correct**: `npm test -- file.test.js --reporters /path/to/reporter`
- ❌ **Wrong**: `npm test -- --reporters /path/to/reporter --coverage`
- ❌ **Wrong**: `npm test -- --reporters /path/to/reporter file.test.js`

#### How 3pio Handles This

3pio automatically ensures correct placement by:
1. For package manager commands with existing `--` separator: appending `--reporters` at the very end
2. For package manager commands without `--`: adding `-- --reporters /adapter/path` at the end
3. For direct Jest invocations: placing `--reporters` immediately after the `jest` command but using `--` to separate any test file paths

This prevents errors like:
```
Error: Could not resolve a module for a custom reporter.
  Module name: --coverage
```

## Coverage Mode - UNSUPPORTED

### ⚠️ Coverage Mode is Not Supported

**3pio does not support running tests with coverage enabled.** Coverage mode interferes with 3pio's ability to capture test results and should not be used together.

#### Affected Commands (Will Not Work Properly)
- `vitest --coverage` or `vitest run --coverage`
- `jest --coverage` or `jest --collectCoverage`
- `pytest --cov=module`
- Any test command with coverage flags

#### Why Coverage is Incompatible
- Coverage instrumentation changes how test runners output results
- Coverage reporters take precedence over 3pio's custom reporters
- This prevents 3pio's adapters from receiving test events
- Results in 0 test files being tracked despite tests executing

#### Symptoms When Coverage is Enabled
- `test-run.md` shows 0 files despite tests running
- `output.log` contains test output but no structured test tracking
- Individual test results are not captured

#### Recommendation
Run tests without coverage when using 3pio:
```bash
# Instead of
3pio npm run test:coverage
3pio jest --coverage
3pio vitest --coverage

# Use
3pio npm test
3pio jest
3pio vitest run
```

**Note**: If you need both test results tracking (via 3pio) and coverage data, run them separately:
1. Use 3pio for test execution and result tracking
2. Run coverage separately without 3pio for coverage metrics

## Environment Variables

- `THREEPIO_IPC_PATH` must be explicitly passed to child processes
- Adapter paths must use absolute paths to avoid resolution issues

## IPC Concurrency

### Concurrent Adapter Writes

When multiple test adapters run in parallel, they write to the same IPC file without explicit locking:

- **Reliance on OS guarantees**: 3pio depends on operating system atomic append guarantees
  - Linux/macOS: 4KB atomic writes (PIPE_BUF)
  - Windows: ~4KB atomic writes (undocumented but reliable in practice)
- **JSON Lines format**: Each event is a single line, typically 100-1000 bytes
- **Risk**: Theoretical possibility of interleaved writes if messages exceed 4KB
- **In practice**: Safe due to small message sizes staying well under OS limits
- **Trade-off**: Accepting minimal risk for simplicity vs adding locking complexity

This is not expected to cause issues in normal operation but is documented for transparency.


## Rust/Cargo Nextest Specific Behaviors

### Test Count Discrepancy with Nextest

**Important**: When using cargo nextest with 3pio, you may notice different test counts compared to running nextest directly.

#### The Behavior
- **Direct nextest run** (human format): Reports ~2733 tests started
- **3pio with nextest** (JSON format): Reports 2756 tests
- **Discrepancy**: ~22-23 additional tests reported by 3pio

#### Root Cause
This is **NOT a bug in 3pio**. The discrepancy occurs because:

1. **Nextest behaves differently between output formats**:
   - Human format (default): May consolidate or group certain tests
   - JSON format (`--message-format libtest-json`): Reports all individual tests

2. **Duplicate test names across modules**: Many Rust projects have identically-named tests in different modules. For example, in Zed:
   - `test_parse_remote_url_given_https_url` exists in 7 different provider modules
   - 72 test names are duplicated across different modules
   - All 2756 tests are unique when considering their full module paths

#### What This Means
- 3pio correctly processes ALL unique tests that nextest reports in JSON format
- The test counts are accurate - they just represent different granularities
- All tests are properly tracked with their full module paths
- Exit codes and pass/fail results remain consistent

#### No Action Required
This is expected behavior when nextest switches to JSON output format. Your tests are running correctly and 3pio is capturing all results accurately.

For a detailed investigation of this behavior, see [Test Count Discrepancy Investigation](/noggin/investigations/test-count-discrepancy-zed.md).

## File System Limitations

### Write Permissions Required

3pio requires write access to the project directory to function properly:

- **Required directories**: `.3pio/runs/`, `.3pio/ipc/`, `.3pio/runs/[runID]/adapters/`
- **Files created**: Test adapters (in `[runID]/adapters/`), IPC communication files, test reports, and log files
- **Adapter extraction**: Each test run extracts adapters to `.3pio/runs/[runID]/adapters/` for isolation
- **Common failure scenarios**:
  - Running in read-only containers
  - CI/CD environments with restricted permissions
  - Network-mounted filesystems with limited access
  - Docker containers without volume mounts

#### Symptoms of Permission Issues
- Error: `failed to create adapter directory: permission denied`
- Error: `cannot write IPC file: read-only file system`
- Tests run but no reports are generated
- Adapter injection fails silently

#### Workarounds
- Ensure the working directory has write permissions before running 3pio
- In containers, mount a writable volume for the project directory
- Consider using `TMPDIR` environment variable to redirect `.3pio` to a writable location (future feature)
- For CI/CD, ensure the build agent has appropriate filesystem permissions

### Other File System Considerations

- IPC files are written to `.3pio/ipc/` which must be writable
- Large test suites may generate significant disk I/O for IPC communication
- Report files use debounced writes to minimize file system operations

## Cross-Platform Compatibility

### Windows-Specific Considerations

#### File Locking Behavior
Windows has stricter file locking semantics than Unix systems:
- **Output file handling**: Files must be explicitly synced (`file.Sync()`) before closing to ensure Windows releases handles properly
- **Tail reading**: Only native runners (Go test, Cargo test) use tail readers to avoid file locking conflicts with Jest/Vitest/pytest
- **Cleanup timing**: Test cleanup may need small delays on Windows to ensure file handles are fully released

#### Path Separators
- Report paths use forward slashes internally but may display with backslashes in Windows console output
- Test detection and report generation handle both forward and backward slashes appropriately
- File paths in test output respect the platform's native separator

#### Console Output
- Unicode characters (like ×) are replaced with ASCII equivalents (x) for better Windows terminal compatibility
- Test failure indicators use simple ASCII characters to ensure consistent display across all platforms

### Platform-Specific Test Behavior

#### Pytest on Windows
- Error output may be less detailed in console compared to Unix systems
- Full error details are always available in the generated reports
- Exit codes are properly mirrored regardless of output verbosity

#### PowerShell Execution
- 3pio works correctly when invoked from PowerShell
- Command quoting is handled automatically for PowerShell compatibility
- Binary detection includes `.exe` extension on Windows