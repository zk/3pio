# Known Issues and Failure Modes

This document describes known issues, limitations, and failure modes discovered during development and testing of 3pio.

## Test Runner Detection

### npx/yarn/pnpm Command Detection
**Issue**: The CLI initially failed to detect test runners when invoked via package managers like `npx`, `yarn`, or `pnpm`.

**Symptoms**: 
- Commands like `npx vitest` would timeout
- The system wouldn't recognize the test runner type

**Root Cause**: The command detection logic only checked the first argument, missing cases where the actual test runner was the second argument.

**Solution**: Extended detection to check both the package manager and the subsequent test runner:
```typescript
if ((command === 'npx' || command === 'yarn' || command === 'pnpm') && args[1]) {
  if (args[1] === 'jest') {
    return { name: 'jest', command: args.slice(0, 2).join(' ') };
  }
  if (args[1] === 'vitest') {
    return { name: 'vitest', command: args.slice(0, 2).join(' ') };
  }
}
```

## Vitest-Specific Issues

### vitest list Command Behavior
**Issue**: The `vitest list` command doesn't just list test files - it actually runs the tests in watch mode.

**Symptoms**:
- Dry run would hang indefinitely
- Tests would execute when only file discovery was expected

**Root Cause**: Vitest's `list` command is designed differently than expected. It runs tests with special output formatting rather than simply listing files.

**Solution**: For Vitest, detect test files from command arguments instead of relying on a dry run:
```typescript
// For Vitest with specific test files, extract them from arguments
const testFileExtensions = ['.test.js', '.test.ts', '.test.mjs', '.test.jsx', '.test.tsx', '.spec.js', '.spec.ts', '.spec.mjs'];
const providedFiles = args.filter(arg => 
  !arg.startsWith('-') && testFileExtensions.some(ext => arg.includes(ext))
);
```

### Duplicate Output in Vitest
**Issue**: When running Vitest with the 3pio adapter, output appears twice - once from the default reporter and once from the 3pio adapter.

**Symptoms**:
- Test results are printed twice to the console
- Each test output line appears duplicated

**Root Cause**: Both Vitest's default reporter and the 3pio adapter are active simultaneously. The 3pio adapter captures output for logging but also allows it to pass through to the original stdout/stderr.

**Status**: This is expected behavior to maintain real-time feedback while also capturing output for reports. Users can disable the default reporter if desired using Vitest configuration.

## Environment Variable Issues

### THREEPIO_IPC_PATH Not Available to Child Process
**Issue**: The IPC path environment variable wasn't being passed to the child process running the tests.

**Symptoms**:
- Adapters would fail with "THREEPIO_IPC_PATH environment variable not set"
- No IPC communication would occur

**Root Cause**: The environment variable was set in the parent process but not explicitly passed to the child process spawned by zx.

**Solution**: Explicitly pass the environment variable when spawning the child process:
```typescript
const testProcess = $({
  env: {
    ...process.env,
    THREEPIO_IPC_PATH: ipcFilePath
  }
})`${fullCommand}`;
```

## Adapter Path Resolution

### Relative Path Issues in Reporter Configuration
**Issue**: Relative paths to adapters would fail to resolve correctly when tests were run from different directories.

**Symptoms**:
- "Cannot find module" errors for the adapter
- Tests would fail to start

**Root Cause**: Relative paths in reporter configuration are resolved from the test runner's working directory, not from the 3pio installation.

**Solution**: Always use absolute paths when specifying adapter locations:
```typescript
const adapterPath = path.resolve(__dirname, `../dist/${adapter.name}.js`);
```

## Recommendations for Users

1. **When using Vitest**: Be aware that the `list` command runs tests. If you need to discover test files without running them, parse the file system or use Vitest's programmatic API.

2. **When seeing duplicate output**: This is expected behavior. To suppress the default reporter, configure your test runner to use only the 3pio reporter.

3. **When tests timeout**: Check that:
   - The test runner is correctly detected (use `--dry-run` to verify)
   - The adapter is properly configured
   - Environment variables are being passed correctly

4. **For custom test runners**: Ensure your test runner is supported or implement a custom adapter following the adapter interface specification.