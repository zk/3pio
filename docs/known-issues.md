# Known Issues and Gotchas

This document describes known issues, limitations, and workarounds for 3pio.

## Test Runner Detection

- Commands invoked via `npx`, `yarn`, or `pnpm` require special handling to detect the actual test runner
- The system checks both the package manager and the subsequent test runner argument

## Vitest-Specific Behaviors

- **Important**: `vitest list` doesn't just list files - it runs tests in watch mode
- When specific test files are provided as arguments, they are extracted directly rather than using dry run
- Duplicate output may appear (from both default reporter and 3pio adapter) - this is expected behavior

## Jest-Specific Behaviors

### Console Output Handling

Jest's handling of console output has several important characteristics that affect how 3pio captures test output:

1. **testResult.console is always undefined** - Despite being documented in the Jest Reporter API, this property is not populated in practice (verified with Jest 29.x)
2. **Direct stdout/stderr writes bypass Jest** - Using `process.stdout.write()` or `process.stderr.write()` directly will output immediately without Jest's formatting
3. **Console methods are intercepted** - Methods like `console.log()` are captured and formatted with stack traces by Jest

For detailed investigation and implications, see [Jest Console Handling](./jest-console-handling.md).

## Environment Variables

- `THREEPIO_IPC_PATH` must be explicitly passed to child processes
- Adapter paths must use absolute paths to avoid resolution issues

## Output Duplication

When running tests with 3pio, you may see duplicate output:
- Jest/Vitest's default reporter continues to run alongside 3pio's adapter
- This is intentional to preserve the user's normal testing experience
- The 3pio adapter operates silently, capturing output without displaying it

## File System Limitations

- IPC files are written to `.3pio/ipc/` which must be writable
- Large test suites may generate significant disk I/O for IPC communication
- Report files use debounced writes to minimize file system operations