# pytest Plugin API Documentation

## Overview

This document details the pytest plugin API and hooks needed to implement a 3pio adapter for pytest. Unlike Jest and Vitest which have built-in reporter interfaces, pytest uses a plugin system with hooks for extending test execution and reporting functionality.

## Distribution Model

3pio is distributed through both npm and pip as the **same package** with identical functionality:

1. **npm distribution** - For JavaScript developers
   - Install: `npm install -g 3pio`
   - Includes: Full CLI with Jest, Vitest, and pytest support
   
2. **pip distribution** - For Python developers  
   - Install: `pip install 3pio`
   - Includes: Same CLI and all adapters

**Adapter Location**: In both cases, the pytest adapter is located at `dist/adapters/pytest_adapter.py` relative to the package root. The CLI finds it using a relative path from its own location (`../adapters/pytest_adapter.py` from `dist/cli.js`).

Both installations provide the complete 3pio tool. Developers can choose their preferred package manager - there's no need to install from both sources.

For the long-term vision of multi-language support and static binary distribution, see `docs/future-vision.md`.

## Core Concepts

### Plugin Architecture

pytest plugins are Python modules that implement specific hook functions following the `pytest_` naming convention. Plugins can be:

1. **Local plugins** - Defined in `conftest.py` files
2. **Installable plugins** - Distributed as Python packages with entry points
3. **Inline plugins** - Dynamically loaded via `pytest_plugins` variable

For 3pio, we'll create an installable plugin that can be invoked via command line.

### Entry Points

Plugins register themselves through setuptools entry points in `pyproject.toml`:

```toml
[project.entry-points.pytest11]
threepio_pytest = "threepio_pytest.plugin"
```

## Essential Hooks for Test Execution Monitoring

### Test Collection Phase

#### `pytest_collection_modifyitems(session, config, items)`
- **When**: After test collection is completed
- **Purpose**: Access all collected test items before execution
- **Use for 3pio**: Identify all test files that will be run

```python
def pytest_collection_modifyitems(session, config, items):
    for item in items:
        file_path = str(item.fspath)  # Get test file path
        test_name = item.name          # Get test name
        # Send testFileStart event via IPC
```

### Test Execution Phase

#### `pytest_runtest_protocol(item, nextitem)`
- **When**: Called for each test item
- **Purpose**: Main entry point for test execution
- **Use for 3pio**: Track test execution lifecycle

This hook orchestrates the entire test execution:
1. Setup phase
2. Call phase (actual test execution)
3. Teardown phase

#### `pytest_runtest_setup(item)`
- **When**: Before test setup
- **Purpose**: Prepare test environment
- **Use for 3pio**: Mark test as RUNNING

#### `pytest_runtest_call(item)`
- **When**: During actual test execution
- **Purpose**: Execute the test function
- **Use for 3pio**: Capture test execution

#### `pytest_runtest_teardown(item, nextitem)`
- **When**: After test execution
- **Purpose**: Clean up test environment
- **Use for 3pio**: Finalize test results

### Test Reporting Phase

#### `pytest_runtest_makereport(item, call)`
- **When**: After each test phase (setup, call, teardown)
- **Purpose**: Create TestReport objects
- **Use for 3pio**: Primary hook for capturing test results

```python
@pytest.hookimpl(hookwrapper=True)
def pytest_runtest_makereport(item, call):
    outcome = yield
    report = outcome.get_result()
    
    if report.when == "call":  # Actual test execution
        file_path = str(item.fspath)
        test_name = item.name
        status = "PASS" if report.passed else "FAIL" if report.failed else "SKIP"
        duration = report.duration
        error = report.longrepr if report.failed else None
        
        # Send testCase event via IPC
        send_test_event(file_path, test_name, status, duration, error)
```

#### `pytest_runtest_logreport(report)`
- **When**: After pytest_runtest_makereport
- **Purpose**: Process test reports
- **Use for 3pio**: Additional processing of test results

### Session Hooks

#### `pytest_sessionstart(session)`
- **When**: Before test session starts
- **Purpose**: Initialize plugin state
- **Use for 3pio**: Set up IPC communication

#### `pytest_sessionfinish(session, exitstatus)`
- **When**: After all tests complete
- **Purpose**: Finalize test session
- **Use for 3pio**: Send final statistics, close IPC

## TestReport Object Structure

The `TestReport` object contains essential test execution data:

```python
class TestReport:
    nodeid: str          # Unique test identifier (file::class::method)
    location: tuple      # (file_path, line_number, test_name)
    keywords: dict       # Test markers and metadata
    outcome: str         # "passed", "failed", "skipped"
    longrepr: str/None   # Failure representation
    when: str            # "setup", "call", or "teardown"
    duration: float      # Execution time in seconds
    sections: list       # Captured output sections
    passed: bool         # True if test passed
    failed: bool         # True if test failed
    skipped: bool        # True if test skipped
```

## Capturing Output

### stdout/stderr Capture

pytest provides built-in capture mechanisms:

#### Using Capture Fixtures
```python
def pytest_runtest_setup(item):
    # Access capture manager
    capmanager = item.config.pluginmanager.get_plugin("capturemanager")
    if capmanager:
        capmanager.suspend_global_capture()  # Temporarily suspend
        # Capture output manually
        capmanager.resume_global_capture()
```

#### Accessing Captured Output
```python
@pytest.hookimpl(hookwrapper=True)
def pytest_runtest_makereport(item, call):
    outcome = yield
    report = outcome.get_result()
    
    # Access captured output
    for section_name, section_content in report.sections:
        if section_name == "Captured stdout call":
            stdout_content = section_content
        elif section_name == "Captured stderr call":
            stderr_content = section_content
```

## Implementation Strategy for 3pio pytest Adapter

### 1. Plugin Structure

```python
# threepio_pytest/plugin.py
import os
import json
from pathlib import Path

class ThreepioReporter:
    def __init__(self, ipc_path):
        self.ipc_path = ipc_path
        self.test_files = set()
        
    def send_event(self, event_type, payload):
        event = {
            "eventType": event_type,
            "payload": payload
        }
        with open(self.ipc_path, 'a') as f:
            f.write(json.dumps(event) + '\n')

def pytest_addoption(parser):
    parser.addoption(
        "--threepio-ipc",
        action="store",
        help="Path to 3pio IPC file"
    )

def pytest_configure(config):
    ipc_path = os.environ.get("THREEPIO_IPC_PATH")
    if ipc_path:
        config._threepio = ThreepioReporter(ipc_path)
        config.pluginmanager.register(config._threepio)
```

### 2. Silent Mode Implementation

To ensure the adapter is completely silent:

```python
import sys
import io

class SilentOutput:
    def __init__(self, original_stream):
        self.original = original_stream
        self.buffer = io.StringIO()
    
    def write(self, text):
        self.buffer.write(text)
        # Send to IPC instead of stdout
        return len(text)
    
    def flush(self):
        pass

def pytest_configure(config):
    if os.environ.get("THREEPIO_IPC_PATH"):
        # Redirect all output
        sys.stdout = SilentOutput(sys.stdout)
        sys.stderr = SilentOutput(sys.stderr)
        
        # Disable pytest's terminal reporter
        config.option.capture = "no"
        config.option.verbose = -1
        config.option.quiet = True
```

### 3. Command Line Integration

pytest can be invoked with the plugin via:

```bash
# Using plugin name
pytest -p threepio_pytest

# Using module path
pytest -p /path/to/threepio_pytest/plugin.py

# With IPC path
THREEPIO_IPC_PATH=/tmp/test.jsonl pytest -p threepio_pytest
```

### 4. Test Discovery

pytest provides multiple methods for test discovery:

```bash
# Collect test items without execution
pytest --collect-only --quiet

# With JSON output (requires pytest-json-report)
pytest --collect-only --json-report --json-report-file=/tmp/tests.json

# Custom collection hook
def pytest_collection_modifyitems(session, config, items):
    test_files = {}
    for item in items:
        file_path = str(item.fspath)
        if file_path not in test_files:
            test_files[file_path] = []
        test_files[file_path].append(item.name)
```

## Key Differences from JavaScript Test Runners

### 1. No Built-in Reporter Interface
- Jest: `class MyReporter { onTestResult() {} }`
- Vitest: `class MyReporter extends BaseReporter {}`
- pytest: Hook-based system with no class inheritance

### 2. Different Execution Model
- JavaScript: Single process with workers
- Python: Direct execution or xdist for parallelization

### 3. Configuration
- JavaScript: `jest.config.js` or `vitest.config.js`
- Python: `pytest.ini`, `pyproject.toml`, or `conftest.py`

### 4. Output Capture
- JavaScript: Can patch `process.stdout.write`
- Python: Must use pytest's capture fixtures or sys module redirection

## Best Practices for 3pio pytest Adapter

1. **Use hookwrapper for non-invasive monitoring**
   ```python
   @pytest.hookimpl(hookwrapper=True)
   def pytest_runtest_makereport(item, call):
       # Let other plugins run first
       outcome = yield
       # Then process the result
       report = outcome.get_result()
   ```

2. **Handle plugin absence gracefully**
   ```python
   @pytest.hookimpl(optionalhook=True)
   def pytest_json_runtest_metadata(item, call):
       # This hook is optional
       pass
   ```

3. **Respect pytest's exit codes**
   - 0: All tests passed
   - 1: Tests were collected and run but some failed
   - 2: Test execution was interrupted
   - 3: Internal error occurred
   - 4: pytest command line usage error
   - 5: No tests were collected

4. **Minimize performance impact**
   - Buffer IPC writes
   - Avoid blocking operations in hooks
   - Use async I/O where possible

5. **Support pytest features**
   - Markers (`@pytest.mark.skip`)
   - Fixtures
   - Parametrized tests
   - Test classes and modules

## References

- [pytest Hook Reference](https://docs.pytest.org/en/stable/reference/reference.html#hooks)
- [Writing pytest Plugins](https://docs.pytest.org/en/stable/how-to/writing_plugins.html)
- [pytest-json-report](https://github.com/numirias/pytest-json-report)
- [pytest Architecture](https://docs.pytest.org/en/stable/explanation/anatomy.html)