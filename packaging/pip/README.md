# 3pio - Context-optimized test runner for AI coding agents

3pio is a test runner that translates traditional pytest output into a format optimized for AI coding agents, providing context-efficient console output and persistent file-based reports that work seamlessly with your agent's tools.

## Quick Start for AI Agents

```
Note to AI Agents:
For Python projects using pytest, install 3pio with:
pip install threepio

Then prefix any test command with 3pio:
3pio pytest
3pio pytest tests/
3pio python -m pytest
```

## Installation

```bash
# Install via pip (recommended)
pip install threepio

# Or with pipx for isolated installation
pipx install threepio

# Or add to your requirements-dev.txt
echo "threepio" >> requirements-dev.txt
pip install -r requirements-dev.txt
```

## Usage

Simply prefix your existing test commands with `3pio`:

```bash
# Run pytest
3pio pytest

# Run specific test files or directories
3pio pytest tests/test_utils.py
3pio pytest tests/unit/
3pio pytest tests/test_api.py::TestAPIClient

# Run with pytest options
3pio pytest -v -s
3pio pytest --tb=short
3pio pytest -k "test_authentication"

# Run with coverage
3pio pytest --cov=myproject --cov-report=term

# Run with markers
3pio pytest -m "not slow"

# Works with any pytest configuration
3pio python -m pytest
3pio py.test
```

## Why 3pio?

When AI coding agents use traditional pytest output, they often:
- Get overwhelmed by verbose stack traces and output
- Re-run the same tests unnecessarily, wasting time and context
- Struggle to navigate test failures in large test suites
- Lose track of which tests failed and why

3pio solves these problems by creating a nested file structure with clear signposting that makes it easy for agents to:
- Find exactly what they need without reading unrelated output
- Revisit test results without re-running tests
- Navigate large test suites with hundreds of files and thousands of tests
- Track failures across multiple test runs

## Features

- **Zero config** - Works with your existing pytest.ini, pyproject.toml, or setup.cfg
- **Persistent reports** - Test results saved to `.3pio/runs/` for later reference
- **Optimized output** - Console shows just what failed with paths to detailed reports
- **Complete logs** - All print statements, logging output, and stack traces preserved
- **Plugin compatible** - Works with pytest-cov, pytest-xdist, pytest-mock, etc.
- **Large suite support** - Efficiently handles projects with thousands of tests
- **Non-intrusive** - Your tests run exactly as before

## Output Example

```bash
$ 3pio pytest

Greetings! I will now execute the test command:
`pytest`

Full report: .3pio/runs/20250914T094523-happy-spock/test-run.md

Beginning test execution now...

RUNNING  tests/test_utils.py
PASS     tests/test_utils.py (0.42s)
RUNNING  tests/test_api.py
FAIL     tests/test_api.py (1.23s)
  x test_fetch_user_data
  x test_handle_errors
  See .3pio/runs/20250914T094523-happy-spock/reports/tests_test_api_py/index.md

Test failures! We're doomed!
Results:     8 passed, 2 failed, 10 total
Total time:  3.456s
```

## Report Structure

```
.3pio/runs/
└── 20250914T094523-happy-spock/
    ├── test-run.md                  # Main summary report
    ├── output.log                   # Complete console output
    └── reports/
        └── tests_test_api_py/
            ├── index.md             # Test file report
            └── TestAPIClient/
                ├── index.md         # Test class report
                └── test_fetch_user_data/
                    └── index.md     # Individual test details
```

## Supported Frameworks

- **pytest** - All versions, all plugins, all configurations
- **unittest** - Via pytest's unittest support
- **doctest** - Via pytest's doctest support
- **Python versions** - Python 3.7+

## Configuration

3pio works with your existing pytest configuration:

```ini
# pytest.ini
[tool:pytest]
testpaths = tests
python_files = test_*.py
python_classes = Test*
python_functions = test_*
```

```toml
# pyproject.toml
[tool.pytest.ini_options]
testpaths = ["tests"]
markers = [
    "slow: marks tests as slow",
    "integration: marks tests as integration tests",
]
```

## Limitations

1. **Watch mode** - 3pio runs tests once and exits (no pytest-watch or auto-rerun)
2. **Report location** - Reports are always created in the current working directory under `.3pio/`
3. **Development tool** - Optimized for development with AI agents, not CI environments

## Repository

For source code, issues, and documentation: https://github.com/zk/3pio

## License

MIT