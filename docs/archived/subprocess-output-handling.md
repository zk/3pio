# Subprocess Output Handling in Test Runners

## Overview

Modern test runners often spawn multiple subprocesses or worker threads to run tests in parallel. This creates complexity for 3pio's output capture strategy, as test output may be buffered internally and never reach the main process's stdout/stderr.

## The Challenge

### Parallel Execution Models

1. **Jest** (`--maxWorkers`)
   - Spawns Node.js worker processes
   - Each worker runs a subset of test files
   - Output is buffered per worker
   - Only displayed for failed tests by default

2. **Vitest** (parallel by default)
   - Uses worker threads or processes
   - Buffers output per test file
   - Aggregates results in main process

3. **pytest-xdist** (`-n` flag)
   - Distributes tests across multiple Python processes
   - Each worker captures output independently
   - Master process aggregates results

### Output Buffering Behavior

Test runners typically:
- Capture stdout/stderr from each test internally
- Associate output with specific test cases
- Buffer successful test output (never emit it)
- Only display output for failed tests
- Format output through reporter APIs, not raw stdout

## Why This Matters for 3pio

### Process-Level Capture Limitations

If 3pio only captures at the Go process level:
- ✅ Captures main process output (startup, summary)
- ❌ Misses buffered test output
- ❌ Misses per-test console logs
- ❌ Misses worker process output

### Adapter-Level Capture Necessity

Test framework adapters MUST capture output because:
- They hook into internal reporter APIs
- These APIs have access to buffered output
- They see output from all workers (aggregated)
- They can associate output with specific tests

## Current Architecture

### Dual Capture Strategy
1. **Go Process Manager**: Captures process-level stdout/stderr → `output.log`
2. **Test Adapters**: Capture test-specific output via framework APIs → IPC events

This redundancy is actually **necessary**, not a design flaw.

## Testing Requirements

### Scenarios to Test

1. **Single process, sequential tests**
   - Baseline case
   - Both capture methods should see all output

2. **Multiple workers, all tests pass**
   - Output may be suppressed entirely
   - Only adapter sees buffered output

3. **Multiple workers, some tests fail**
   - Failed test output should appear
   - Adapter should capture and attribute correctly

4. **Worker process crashes**
   - Process-level capture essential
   - Adapter may not receive events

5. **Collection/startup failures**
   - Occur before workers spawn
   - Process-level capture critical

### Test Implementation Strategy

Create fixtures that:
- Force parallel execution
- Generate output in different scenarios
- Verify both capture mechanisms work
- Test worker crash scenarios

## Known Issues

### pytest Collection Failures
- **Problem**: Errors during test collection phase aren't captured
- **Cause**: Adapter hooks engage too late
- **Solution**: Hook into `pytest_configure()` and `pytest_collectreport()`

### Jest Worker Crashes
- **Problem**: If a Jest worker crashes, its output may be lost
- **Investigation Needed**: How Jest handles worker failures

### Vitest Thread Termination
- **Problem**: Abrupt thread termination might lose buffered output
- **Investigation Needed**: Thread cleanup behavior

## Future Improvements

1. **Adapter Health Checks**
   - Verify adapter can capture before running tests
   - Fall back to process-only capture if adapter fails

2. **Hybrid Output Attribution**
   - Use timestamps to correlate process and adapter output
   - Merge intelligently in reports

3. **Worker-Aware Adapters**
   - Detect parallel execution mode
   - Adjust capture strategy accordingly

4. **Stress Testing**
   - Large number of workers
   - High output volume
   - Worker failures and timeouts

## Design Principles

1. **Defense in Depth**: Both process and adapter capture provide redundancy
2. **Framework Integration**: Adapters must use framework-specific APIs
3. **Graceful Degradation**: If adapter fails, process capture is fallback
4. **Output Attribution**: Best effort to associate output with specific tests

## Conclusion

The dual capture approach (process + adapter) is essential for handling modern test runners with parallel execution. This is not redundancy to be eliminated, but necessary complexity to handle the full range of test execution scenarios.