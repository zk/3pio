# Analysis: Missing Individual Test Logs for Mastra Monorepo

## Issue
When running 3pio with the Mastra monorepo (`pnpm test`), no individual test log files were created in `.3pio/runs/*/logs/` directory, even though tests ran successfully.

## Investigation Findings

### 1. Empty IPC File
- The IPC file at `.3pio/ipc/20250911T085108-feisty-han-solo.jsonl` is **0 bytes**
- This means the Vitest adapter is not sending any test events to 3pio
- Without IPC events, 3pio cannot create individual log files

### 2. Vitest Adapter Issue with Monorepos
The Vitest adapter appears to not be properly injected or activated when running in a monorepo context:
- The adapter code includes `testFileStart` event sending logic
- However, no events are being written to the IPC file
- The test output shows Vitest is running with both the 3pio adapter AND the default reporter

### 3. Current Behavior
- **output.log**: Successfully captures all stdout/stderr (37KB of data)
- **test-run.md**: Generated but only contains error details from stderr
- **logs/**: Directory created but empty (no individual test files)

### 4. Root Cause
The issue appears to be that when Vitest runs in a monorepo with workspaces:
1. Vitest spawns separate processes for each package
2. The adapter may not be properly inheriting the `THREEPIO_IPC_PATH` environment variable
3. Or the adapter is not being loaded correctly in the child processes

### 5. Evidence
From the output.log we can see:
- Tests are running successfully across multiple packages (@mastra/core, @mastra/mcp, etc.)
- The command includes the adapter: `--reporter /Users/zk/code/3pio/open-source/mastra/.3pio/adapters/43a28ae8/vitest.js`
- But also includes `--reporter default` which may be interfering

## Recommendations

### Immediate Fix Needed
1. **Environment Variable Propagation**: Ensure `THREEPIO_IPC_PATH` is passed to all Vitest child processes in monorepo setups
2. **Adapter Loading**: Verify the adapter is being loaded in each workspace's test process
3. **Reporter Conflict**: Investigate if having both the 3pio adapter and default reporter causes issues

### Testing Requirements
- Add integration test specifically for monorepo projects with workspaces
- Test with projects using:
  - pnpm workspaces
  - npm workspaces  
  - yarn workspaces
  - Vitest's deprecated `test.workspace` config (as seen in Mastra)

### Workaround for Users
Currently, users can still access all test output via:
- `output.log` - Contains complete test output
- `test-run.md` - Contains summary and errors

But they won't get the benefit of:
- Individual test file logs with clear test case boundaries
- Structured test results in the report

## Conclusion
3pio's core functionality works (test execution and output capture), but the Vitest adapter fails to send IPC events in monorepo contexts, preventing individual log file creation. This is a significant issue for monorepo projects which are common in modern JavaScript development.