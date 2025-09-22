# Pytest Adapter Documentation

## Overview

Custom pytest plugin that integrates with pytest's hook system to capture test events and send them via IPC to the 3pio orchestrator. Designed to be completely silent.

## Test Status Mapping

- **PASS**: Test executed successfully
- **FAIL**: Test assertion failed or raised exception
- **SKIP**: Test was skipped (marker or runtime)
- **XFAIL**: Test failed as expected
- **XPASS**: Test passed unexpectedly

## Skip Handling

### Skip Types
1. **Marker skips** (`@pytest.mark.skip`) - Evaluated during setup
2. **Conditional skips** (`@pytest.mark.skipif`) - Condition checked before execution
3. **Runtime skips** (`pytest.skip()`) - Triggered during execution

### Deduplication
Maintains `processed_skips` set with `(file_path, test_name)` tuples to prevent duplicate events.

## XFail/XPass

- **XFail**: Test executed and fails as expected (not a regression)
- **XPass**: Test expected to fail but passes (may indicate fix)
- Different from Skip: XFail tests are executed, Skip tests are not

## Parallel Execution (pytest-xdist)

### Worker Detection
Checks `PYTEST_XDIST_WORKER` environment variable.

### Behavior
- Controller process: Reports events via IPC
- Worker processes: Stay silent, no IPC
- Result: No duplicate reports

### Supported
- `pytest -n auto/4/logical`
- `pytest --dist loadscope`

## Output Capture

1. Patches `sys.stdout` and `sys.stderr`
2. Captures silently (no terminal output)
3. Sends to orchestrator via IPC events
4. Associates output with current test file

## Event Flow

1. **Collection**: Discovery and test count
2. **Execution**: Group discovery, test runs, result events
3. **Finish**: Statistics aggregation

## Configuration

Injected automatically with:
- IPC Path: JSON Lines communication file
- Log Level: DEBUG, INFO, WARN, ERROR

## Debugging

Logs to `.3pio/debug.log`:
- Initialization
- Worker detection
- Event processing
- Errors

## Limitations

1. No interactive plugins
2. Other reporters may conflict
3. Watch mode not supported

## Best Practices

- Avoid coverage flags
- No watch mode
- Prefer markers over runtime skips
- Use descriptive test names

## Technical Details

### Key Components
- **ThreepioReporter**: Main reporter class
- **Event handlers**: pytest hooks
- **IPC**: JSON Lines format
- **Output streams**: Custom stdout/stderr

### Main Hooks
- `pytest_configure`: Initialize
- `pytest_runtest_protocol`: Track context
- `pytest_runtest_logreport`: Capture results
- `pytest_sessionfinish`: Finalize
- `pytest_unconfigure`: Cleanup

### State Management
- Test results per file
- Group hierarchy
- Output buffers
- Skip deduplication
- Worker mode detection