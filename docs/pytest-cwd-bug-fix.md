# Pytest Working Directory Bug Fix

## Issue
When running Flask tests with 3pio, some tests were not being captured in the IPC events. Specifically, 4-6 tests that changed the working directory during execution were missing from the 3pio reports.

## Root Cause
The pytest adapter was using a relative IPC path (`.3pio/runs/{runID}/ipc.jsonl`) received from the environment variable. When certain tests changed the working directory during execution (e.g., Flask's `test_dotenv_optional`), the adapter could no longer find the IPC file at the relative path.

### Sequence of Events
1. 3pio starts pytest from `/tmp/3pio-open-source/flask`
2. IPC file created at `.3pio/runs/{runID}/ipc.jsonl` (relative path)
3. Pytest adapter initializes, IPC file is accessible
4. Test `test_dotenv_optional` changes working directory to `/tmp/3pio-open-source/flask/tests/test_apps`
5. Adapter tries to write to `.3pio/runs/{runID}/ipc.jsonl` but now resolves to `/tmp/3pio-open-source/flask/tests/test_apps/.3pio/runs/{runID}/ipc.jsonl`
6. File not found error, test event not captured

## Solution
Modified the pytest adapter to convert the IPC path to an absolute path during initialization:

```python
def __init__(self, ipc_path: str):
    # Store the absolute path to the IPC file to handle directory changes
    self.ipc_path = os.path.abspath(ipc_path)
    # ... rest of initialization
```

This ensures that even if tests change the working directory, the adapter can still write to the correct IPC file location.

## Affected Tests
The following Flask tests were affected because they change the working directory:
- `test_dotenv_optional`
- `test_disable_dotenv_from_env`
- `test_load_dotenv`
- `test_scriptinfo`
- Parameterized tests with special characters in names

## Verification
After the fix, all 490 Flask tests are successfully captured by 3pio (previously only 486 were captured).

## Files Changed
- `/Users/edie/code/3pio/internal/adapters/pytest_adapter.py` - Modified to use absolute IPC path