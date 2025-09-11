# Known Issues and Gotchas

This document describes known issues, limitations, and workarounds for 3pio.

## Dynamic Test Discovery

3pio supports two modes of test discovery:

### Static Discovery
- Used when test files can be determined upfront (e.g., explicit file arguments, Jest's `--listTests`)
- Shows list of files before running tests
- Pre-creates all log files

### Dynamic Discovery
- Used when test files cannot be determined upfront (e.g., `npm run test` with Vitest)
- Files are discovered and registered as they send their first event
- Shows "Test files will be discovered and reported as they run"
- Log files created on demand

The system automatically chooses the appropriate mode based on the test runner and command.

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

Jest's handling of console output has several important characteristics that affect how 3pio captures test output:

1. **testResult.console is always undefined** - Despite being documented in the Jest Reporter API, this property is not populated in practice (verified with Jest 29.x)
2. **Direct stdout/stderr writes bypass Jest** - Using `process.stdout.write()` or `process.stderr.write()` directly will output immediately without Jest's formatting
3. **Console methods are intercepted** - Methods like `console.log()` are captured and formatted with stack traces by Jest
4. **3pio does NOT include the default reporter** - When 3pio runs Jest with `--reporters`, it intentionally excludes the default reporter. This means:
   - No duplicate output in the console
   - All test output is captured via IPC events with file path associations
   - Individual test log files are created from IPC events, not by parsing output.log
   - This is a deliberate design choice to prevent output duplication and maintain clean console output

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

## Coverage Mode Limitations

### Coverage Reporting Interference

When test runners are executed with coverage enabled (e.g., `--coverage` flag), 3pio may fail to capture individual test results:

#### Affected Commands
- `vitest --coverage` or `vitest run --coverage`
- `jest --coverage` or `jest --collectCoverage`
- `pytest --cov=module`

#### Symptoms
- Tests run successfully but 3pio only captures the final summary
- `test-run.md` shows 0 files despite tests executing
- `output.log` contains test results but individual test tracking is lost

#### Why This Happens
Coverage instrumentation changes how test runners output results. The coverage reporter often takes precedence over custom reporters, preventing 3pio's adapter from receiving test events.

#### Workaround
Run tests without coverage during development:
```bash
# Instead of
3pio npm test:ci  # (which includes --coverage)

# Use
3pio npm test -- --run --no-coverage
```

**Note**: This primarily affects CI/CD workflows where coverage is mandatory. For day-to-day development, developers typically run tests without coverage for faster feedback.

## Environment Variables

- `THREEPIO_IPC_PATH` must be explicitly passed to child processes
- Adapter paths must use absolute paths to avoid resolution issues


## File System Limitations

- IPC files are written to `.3pio/ipc/` which must be writable
- Large test suites may generate significant disk I/O for IPC communication
- Report files use debounced writes to minimize file system operations