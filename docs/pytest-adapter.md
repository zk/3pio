# Pytest Adapter Documentation

This document describes the specific behaviors and implementation details of the 3pio pytest adapter.

## Overview

The 3pio pytest adapter is a custom pytest plugin that integrates with pytest's hook system to capture test execution events and send them via IPC (Inter-Process Communication) to the 3pio orchestrator. The adapter is designed to be completely silent, capturing all test output without interfering with test execution.

## Test Status Mapping

The pytest adapter recognizes and correctly reports the following test statuses:

### Standard Test Outcomes

- **PASS**: Test executed successfully
- **FAIL**: Test assertion failed or raised an unexpected exception
- **SKIP**: Test was skipped via marker or runtime skip
- **XFAIL**: Test failed as expected (expected failure)
- **XPASS**: Test passed unexpectedly (expected to fail but passed)

## Skip Handling

The pytest adapter captures skipped tests from multiple sources and phases:

### Skip Mechanisms

1. **Marker-based skips** (`@pytest.mark.skip`)
   - Evaluated during the `setup` phase
   - Test function is never executed
   - Skip reason extracted from marker

2. **Conditional skips** (`@pytest.mark.skipif`)
   - Evaluated during the `setup` phase
   - Condition checked before test execution
   - Skip reason includes the condition

3. **Runtime skips** (`pytest.skip()`)
   - Triggered during the `call` phase
   - Can occur anywhere in test function
   - Dynamic skip based on runtime conditions

### Skip Phases

Tests can be skipped in two distinct phases:

- **setup phase**: Marker-based skips are evaluated before test execution
- **call phase**: Runtime skips occur during test execution

The adapter tracks which phase a skip occurred in and includes this information in the IPC event as `skipPhase`.

### Skip Deduplication

To prevent duplicate skip events, the adapter maintains a `processed_skips` set that tracks `(file_path, test_name)` tuples. This ensures each skipped test is only reported once, even if pytest reports it in multiple phases.

## XFail and XPass Handling

Expected failures are distinct from regular failures and skips:

### XFail (Expected Failure)

Tests marked with `@pytest.mark.xfail` that fail are reported as `XFAIL`:

```python
@pytest.mark.xfail(reason="Known bug in feature X")
def test_broken_feature():
    assert broken_function() == "expected"  # This fails as expected
```

- Status: `XFAIL`
- The test failure is expected and doesn't indicate a regression
- XFail reason is captured and included in the report

### XPass (Unexpected Pass)

Tests marked with `@pytest.mark.xfail` that pass are reported as `XPASS`:

```python
@pytest.mark.xfail(reason="Flaky test")
def test_sometimes_works():
    assert random.random() > 0.5  # Sometimes passes unexpectedly
```

- Status: `XPASS`
- The test was expected to fail but passed
- May indicate the expected failure has been fixed

### XFail vs Skip

Important distinction:
- **XFail**: Test is executed and expected to fail
- **Skip**: Test is not executed at all

The adapter ensures xfail tests are never confused with skipped tests by checking for xfail markers before processing skip events.

## Parallel Execution (pytest-xdist)

The adapter fully supports parallel test execution with pytest-xdist:

### Worker Detection

The adapter detects when it's running in a worker process by checking for the `PYTEST_XDIST_WORKER` environment variable:

```python
def is_xdist_worker() -> bool:
    return os.environ.get('PYTEST_XDIST_WORKER') is not None
```

### Worker Behavior

- **Controller process**: Reports test events normally via IPC
- **Worker processes**: Stay completely silent, no IPC communication
- **Result**: No duplicate test reports in parallel execution

### Supported Patterns

The adapter correctly handles all common xdist patterns:
- `pytest -n auto` - Automatic worker count
- `pytest -n 4` - Specific worker count
- `pytest -n logical` - Logical CPU count
- `pytest --dist loadscope` - Load distribution by scope

## Output Capture

The adapter implements comprehensive output capture:

### Capture Strategy

1. **stdout/stderr patching**: Replaces `sys.stdout` and `sys.stderr` with custom streams
2. **Silent operation**: Captured output is not displayed in the terminal
3. **IPC transmission**: Output is sent to the orchestrator via IPC events

### Capture Lifecycle

1. Capture starts in `pytest_configure` hook
2. Test file context switches tracked via `pytest_runtest_protocol`
3. Output associated with the current test file
4. Capture continues throughout entire test session

## Event Flow

The typical event flow for a test file:

1. **Collection Phase**
   - `collectionStart` event
   - Test discovery
   - `collectionFinish` event with test count

2. **Test Execution**
   - `testGroupDiscovered` for file and any test classes
   - `testGroupStart` when file execution begins
   - For each test:
     - Skip evaluation (setup phase)
     - Test execution (call phase)
     - `testCase` event with result
   - `testGroupResult` when file completes

3. **Session Finish**
   - Final statistics aggregation
   - All group results finalized

## Configuration

The adapter is injected automatically by 3pio and configured via:

- **IPC Path**: File path for JSON Lines communication
- **Log Level**: Debugging verbosity (DEBUG, INFO, WARN, ERROR)

These values are injected into the adapter code at runtime by replacing placeholder tokens.

## Debugging

Debug logs are written to `.3pio/debug.log` and include:

- Adapter initialization
- Worker process detection
- Event processing
- Error conditions

To enable verbose logging, set the log level to DEBUG in the adapter code injection.

## Known Limitations

1. **Interactive plugins**: Plugins requiring user input are not supported
2. **Custom reporters**: Other pytest reporters may conflict with output capture
3. **Watch mode**: Continuous test watching is not supported

## Best Practices

When using 3pio with pytest:

1. **Avoid coverage flags**: Use separate coverage runs
2. **No watch mode**: Don't use `--watch` or similar flags
3. **Prefer markers**: Use skip markers over runtime skips when possible
4. **Clear test names**: Use descriptive test names for better reports

## Technical Implementation

### Key Components

- **ThreepioReporter**: Main reporter class managing test lifecycle
- **Event handlers**: pytest hooks for various test phases
- **IPC communication**: JSON Lines format for structured events
- **Output streams**: Custom stdout/stderr replacement

### Hook Integration

The adapter integrates with these pytest hooks:

- `pytest_configure`: Initialize adapter
- `pytest_runtest_protocol`: Track test file context
- `pytest_runtest_logreport`: Capture test results
- `pytest_sessionfinish`: Finalize reports
- `pytest_unconfigure`: Cleanup

### State Management

The adapter maintains state for:
- Test results per file
- Group hierarchy
- Output buffers
- Skip deduplication
- Worker mode detection