# Output Handling in 3pio

## Overview

3pio uses a dual capture strategy to ensure complete output collection from test runners. This document explains how output is captured, processed, and stored across different test frameworks and execution models.

## The Challenge

Modern test runners present several output handling challenges:

### Parallel Execution
- **Jest** (`--maxWorkers`): Spawns Node.js worker processes
- **Vitest** (parallel by default): Uses worker threads or processes
- **pytest-xdist** (`-n` flag): Distributes tests across Python processes

### Output Buffering
Test runners typically:
- Capture stdout/stderr internally
- Associate output with specific test cases
- Buffer successful test output (never emit it)
- Only display output for failed tests
- Format output through reporter APIs, not raw stdout

## Dual Capture Strategy

3pio implements capture at two levels:

### 1. Process-Level Capture (Go Orchestrator)
- Captures main process output (startup, summary)
- Uses pipes to intercept stdout/stderr
- Writes everything to `output.log`
- Essential for startup failures and crashes

### 2. Adapter-Level Capture (Test Reporters)
- Hooks into internal reporter APIs
- Sees output from all workers (aggregated)
- Associates output with specific tests
- Sends via IPC events with file context

This redundancy is **necessary**, not a design flaw. Different scenarios require different capture mechanisms.

## Framework-Specific Behaviors

### Jest Console Handling

**Key Findings**:
1. `testResult.console` is always undefined despite API documentation
2. Direct `process.stdout.write()` bypasses Jest completely
3. `console.log()` methods are intercepted and formatted by Jest
4. 3pio does NOT include the default reporter to avoid duplication

**Implementation**:
- Patch both console methods AND stream writers
- Capture during test execution, not from testResult
- Store and associate with current test file

### Vitest Output Handling

**Characteristics**:
- Default reporter included for better UX
- Global capture with context switching
- Dynamic test discovery for unknown files
- Recursive test case extraction from task tree

**Implementation**:
- Start global capture in `onInit`
- Switch context in `onTestFileStart`
- Process recursively in `onTestFileResult`

### pytest Output Handling

**Characteristics**:
- Plugin architecture, not reporter-based
- Collection phase can fail before tests run
- Uses capsys fixture for capture
- Works with pytest's assertion rewriting

**Implementation**:
- Hook into `pytest_configure()` early
- Use `pytest_runtest_logreport()` for results
- Capture via capsys fixture
- Handle collection errors specially

## Collection Phase Handling

Some test runners have distinct collection/discovery phases:

### pytest Collection
- Imports all test modules upfront
- Can fail with import/syntax errors
- Sends specialized IPC events:
  - `collectionStart`
  - `collectionError`
  - `collectionFinish`

### Jest/Vitest Collection
- No separate collection phase
- Import errors reported as test failures
- Handled through normal test failure mechanisms

## Testing Output Capture

### Scenarios to Test

1. **Single process, sequential tests**
   - Both capture methods see all output
   
2. **Multiple workers, all tests pass**
   - Output may be suppressed entirely
   - Only adapter sees buffered output
   
3. **Multiple workers, some tests fail**
   - Failed test output should appear
   - Adapter captures and attributes correctly
   
4. **Worker process crashes**
   - Process-level capture essential
   - Adapter may not receive events
   
5. **Collection/startup failures**
   - Occur before workers spawn
   - Process-level capture critical

## Output Storage

### File Structure
```
.3pio/runs/[runID]/
├── output.log              # Complete stdout/stderr from process
└── logs/                   # Individual test file logs
    ├── math.test.js.log   # Output from math tests
    └── string.test.js.log # Output from string tests
```

### Incremental Writing
- Log files created immediately when tests registered
- Output written as it arrives, not batched
- Partial results available if interrupted
- File handles managed efficiently

## Best Practices

### For Adapter Writers
1. Capture at the earliest possible point
2. Handle both direct writes and console methods
3. Associate output with correct test file
4. Send output chunks immediately via IPC

### For Performance
1. Don't buffer large amounts in memory
2. Use debounced writes for efficiency
3. Stream process for large outputs
4. Clean up resources promptly

## Known Issues

### Coverage Mode Interference
When running with coverage (`--coverage`), 3pio may fail to capture individual test results because coverage reporters take precedence.

**Workaround**: Run without coverage during development:
```bash
# Instead of
3pio npm test:ci  # (includes --coverage)

# Use
3pio npm test -- --no-coverage
```

### Worker Process Output
Output from crashed worker processes may be lost if only relying on adapter capture. Process-level capture provides a safety net.

## Design Principles

1. **Defense in Depth**: Both process and adapter capture provide redundancy
2. **Framework Integration**: Adapters must use framework-specific APIs
3. **Graceful Degradation**: If adapter fails, process capture is fallback
4. **Output Attribution**: Best effort to associate output with specific tests

The dual capture approach is essential for handling the full range of test execution scenarios across different frameworks and parallelization strategies.