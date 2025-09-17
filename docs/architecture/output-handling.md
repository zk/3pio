# Output Handling and Formats in 3pio

## Overview

3pio captures all output at the **process level** through the orchestrator. Adapters do not capture output - they only send structured test events via IPC. This document explains how output is captured, processed, stored, and formatted across different output types.

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

## Single Capture Strategy

3pio captures all output at the **process level only**:

### Process-Level Capture (Go Orchestrator)
- Captures ALL output from the test process and its children
- Uses pipes to intercept stdout/stderr
- Writes everything to `output.log`
- Handles startup failures, crashes, and all test output

### Adapter Role (IPC Events Only)
- Adapters do NOT capture output
- They only send structured test events via IPC
- Events include: test discovery, start, results, and grouping
- Output association happens in the orchestrator based on timing

**Note**: Previous documentation referred to "dual capture" but this is outdated. All output capture happens at the process level.

## Framework-Specific Behaviors

### Jest Console Handling

**Key Findings**:
1. `testResult.console` is always undefined despite API documentation
2. Direct `process.stdout.write()` bypasses Jest completely
3. `console.log()` methods are intercepted and formatted by Jest
4. **Jest does NOT include the default reporter** - Prevents output duplication

**Behavior**:
- Jest adapter sends test events via IPC only
- All console output captured by process-level pipes
- Clean console output with only 3pio's formatted results

### Vitest Output Handling

**Characteristics**:
- **Vitest DOES include the default reporter** - Better UX with familiar output
- Global capture with context switching
- Dynamic test discovery for unknown files
- Recursive test case extraction from task tree

**Behavior**:
- Vitest adapter sends test events via IPC only
- Default reporter output captured by process-level pipes
- Dual output visible: default reporter + 3pio reports

### Reporter Configuration Rationale

The different reporter configurations are intentional:
- **Jest**: Excludes default reporter for clean, deduplicated output
- **Vitest**: Includes default reporter for familiar user experience
- Both approaches are optimized for their respective frameworks' architectures

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

## Output Storage

### File Structure
```
.3pio/runs/[runID]/
├── test-run.md                         # Main report (only non-index.md file)
├── output.log                          # Complete stdout/stderr from process
└── reports/                            # Individual test file reports
    ├── math_test_js/
    │   └── index.md                    # Output from math tests
    └── string_test_js/
        └── index.md                    # Output from string tests
```

### Incremental Writing
- Log files created immediately when tests registered
- Output written as it arrives, not batched
- Partial results available if interrupted
- File handles managed efficiently

## Output Formats

3pio generates three main types of output: console output during execution, main test reports, and individual test file reports.

### Console Output

The console output provides real-time feedback during test execution:

```markdown
Greetings! I will now execute the test command:
`go test ./...`

Full report: .3pio/runs/20250913T203231-batty-neelix/test-run.md

Beginning test execution now...

RUNNING  github.com/zk/3pio/cmd/3pio
RUNNING  github.com/zk/3pio/internal/adapters
RUNNING  github.com/zk/3pio/internal/ipc
RUNNING  github.com/zk/3pio/internal/logger
NO_TESTS github.com/zk/3pio/internal/logger
FAIL     github.com/zk/3pio/tests/integration_go (29.57s)
  x TestFailureDisplayFormat
  x shows_report_path
  + 12 more
  see .3pio/runs/20250912T113945-nutty-poe/reports/go_test_examples_test_js/index.md

PASS     github.com/zk/3pio/internal/runner (10.77s)
PASS     github.com/zk/3pio/internal/orchestrator (4.38s)
PASS     github.com/zk/3pio/cmd/3pio (8.08s)
PASS     github.com/zk/3pio/internal/runner/definitions (6.42s)
Test failures! We're doomed!
Results:     7 passed, 1 failed, 1 skipped, 9 total
Total time:  42.411s
```

#### Results Summary Format

The results line dynamically adjusts based on test outcomes:

- **All passed, no skips**: `Results:     N passed, N total`
- **With failures**: `Results:     N passed, M failed, X total`
- **With skipped tests**: `Results:     N passed, M failed, S skipped, X total`

Where:
- `passed`: Number of test groups that passed all tests
- `failed`: Number of test groups with at least one failure
- `skipped`: Number of test groups that were skipped or had no tests (includes Go packages with `NO_TESTS`)
- `total`: Sum of passed + failed + skipped

#### Success Messages

The console displays different messages based on outcomes:
- **All tests passed** (no failures, no skips): "Splendid! All tests passed successfully"
- **Tests with skips** (no failures, some skips): "Tests completed with some skipped"
- **Only skipped tests**: "All tests were skipped"
- **Test failures**: "Test failures! [random exclamation]"

#### Group Names in Console Output

The console output displays raw group names as provided by each test runner:
- **Jest/Vitest**: Full file paths as discovered (e.g., `/path/to/project/math.test.js`)
- **Go**: Package import paths (e.g., `github.com/zk/3pio/cmd/3pio`)
- **pytest**: File paths relative to test root (e.g., `test_math.py`, `tests/unit/test_string.py`)

The group names are displayed exactly as provided by the test runner adapters without normalization.

### Main Test Report (test-run.md)

The main report provides a comprehensive overview of the entire test run:

```markdown
---
run_id: 20250912T001847-funky-mccoy
run_path: /Users/zk/code/3pio/.3pio/runs/20250912T001212-snappy-cyan
detected_runner: vitest
modified_command: `npm test --reporter path/to/vitest/reporter`
created: 2025-02-15T12:30:00.000Z
updated: 2025-02-15T12:31:11.000Z
status: PENDING | RUNNING | COMPLETED | ERRORED
---

# 3pio Test Run

- Test command: `npx vitest run`
- Run stdout/stderr: `./output.log`

## Summary

- Total test cases: 2
- Test cases completed: 2
- Tests cases passed: 1
- Test cases failed: 1
- Test cases skipped: 0
- Total duration: 203.56s

## Test group results

| Status | Name | Tests | Duration | Report |
|--------|------|-------|----------|--------|
| PASS | math.test.js | 5 passed | 12.3s | ./reports/math_test_js/index.md |
| FAIL | string.test.js | 3 passed, 1 failed | 2.3s | ./reports/string_test_js/index.md |
| SKIP | tests/unit/utilities.test.js | 0 tests | 0.53s | ./reports/tests_unit_utilities_test_js/index.md |
```

**Notes:**
- `detected_runner` examples: `vitest`, `jest`, `go test`, `pytest`, `cargo test`
- `modified_command`: The modified command used to create the runner process. Helps debug issues with positional arguments in the field.

### Individual Test File Reports

Each test file gets its own detailed report with YAML frontmatter and structured test results:

```markdown
---
test_file: /Users/zk/code/3pio/tests/fixtures/basic-vitest/string.test.js
created: 2025-02-15T12:30:00.000Z
updated: 2025-02-15T12:31:11.000Z
status: PENDING | RUNNING | COMPLETED | ERRORED
---

# Test results for `string.test.js`

## Test case results

- ✓ should concatenate strings (1ms)
- ✓ should concatenate strings (1ms)
- ✕ should fail this test (3ms)
```
expected 'foo' to be 'bar' // Object.is equality
```
○ should skip this test
✓ should pass through other methods unchanged (0ms)
✓ should handle proxy creation errors gracefully (0ms)
✓ should wrap createRunAsync to return run proxy (0ms)
✓ should wrap createRun to return run proxy (0ms)
✓ should inject tracing context into run start method (0ms)
✓ should preserve user-provided tracingContext in run start (0ms)
✓ should pass through other run methods unchanged (0ms)
✓ should work in nested workflow step scenario (0ms)
✓ should work with workflow calling another workflow (0ms)
✓ should preserve type safety (0ms)
✓ should handle mixed wrapped and unwrapped usage (0ms)

## Error

Error: Something went wrong in thirdFunction!
    at thirdFunction (file:///path/to/your/script.js:2:9)
    at secondFunction (file:///path/to/your/script.js:6:3)
    at firstFunction (file:///path/to/your/script.js:10:3)
    at file:///path/to/your/script.js:14:3

## stdout/stderr
```

**Notes:**
- Error section only present if an error was encountered while running this test. Test failures alone do not constitute an error
- stdout/stderr section only present if stdout/stderr was collected and is not empty
- Files are updated as the test file transitions state with YAML frontmatter timestamps and status updates

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
All output from worker processes is captured at the process level, including crashes and errors.

## Design Principles

1. **Single Source of Truth**: All output capture happens at the process level
2. **Framework Integration**: Adapters provide structured test events via IPC
3. **Complete Coverage**: Process capture ensures no output is lost
4. **Output Attribution**: Orchestrator correlates output with test events based on timing
5. **Consistent Formatting**: All output formats use structured approach with YAML frontmatter
6. **Incremental Updates**: Reports are updated in real-time as tests execute

The process-level capture approach ensures complete output collection across all test execution scenarios while providing multiple output formats optimized for different use cases.