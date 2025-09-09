# Known Issues and Gotchas

This document describes known issues, limitations, and workarounds for 3pio.

## Test Runner Detection

- Commands invoked via `npx`, `yarn`, or `pnpm` require special handling to detect the actual test runner
- The system checks both the package manager and the subsequent test runner argument

## Vitest-Specific Behaviors

- **Important**: `vitest list` doesn't just list files - it runs tests in watch mode
- When specific test files are provided as arguments, they are extracted directly rather than using dry run

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

### Reporter Flag Order Quirk

Jest has a critical quirk where the `--reporters` flag **must come AFTER** any test file paths in the command line:

- ✅ **Correct**: `jest file.test.js --reporters /path/to/reporter`
- ❌ **Wrong**: `jest --reporters /path/to/reporter file.test.js`

When `--reporters` comes before test file paths, Jest incorrectly interprets the test files as reporter module paths, causing errors like:
```
Error: Could not resolve a module for a custom reporter.
  Module name: test/system/mcp-tools/click.test.js
```

This is especially important when using npm scripts with the `--` separator:
- ✅ **Correct**: `npm test -- file.test.js --reporters /path/to/reporter`
- ❌ **Wrong**: `npm test -- --reporters /path/to/reporter file.test.js`

3pio automatically handles this by always placing the `--reporters` flag at the end of the command.

## Environment Variables

- `THREEPIO_IPC_PATH` must be explicitly passed to child processes
- Adapter paths must use absolute paths to avoid resolution issues


## File System Limitations

- IPC files are written to `.3pio/ipc/` which must be writable
- Large test suites may generate significant disk I/O for IPC communication
- Report files use debounced writes to minimize file system operations