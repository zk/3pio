# 3pio

A context-friendly test runner for Python projects. 3pio enhances your existing test workflow by generating structured, AI-optimized reports without changing how your tests run.

## Installation

```bash
pip install threepio-test-runner
```

## Usage

```bash
# Run pytest tests
3pio pytest

# Run specific test file
3pio pytest tests/test_utils.py

# Run with pytest options
3pio pytest -v -s tests/

# Run unittest tests
3pio python -m unittest

# Run with coverage
3pio pytest --cov=myproject tests/
```

## Why 3pio?

When working with AI coding assistants, test output often gets lost or truncated. 3pio solves this by:

- **Preserving all test output** - Never lose print statements or error traces
- **Structured reports** - Each test file gets its own organized log
- **AI-friendly format** - Reports optimized for LLM context windows
- **Zero config** - Works with your existing pytest/unittest setup
- **Non-intrusive** - Your tests run exactly as before, 3pio just captures better reports

## Supported Test Frameworks

- **pytest** - Full support for pytest and its plugins
- **unittest** - Python's built-in testing framework
- **nose2** - Works with nose2 test runner
- **Any Python test runner** - Captures output from any test command

## How It Works

3pio acts as a transparent wrapper around your test runner:

1. Runs your tests with a custom plugin to capture structured data
2. Preserves all console output and test results
3. Generates organized reports in `.3pio/runs/[timestamp]-[name]/`
4. Maintains full compatibility with your existing test configuration

## Report Structure

After running tests, find your reports in:
```
.3pio/runs/
└── 20240110-143022-clever-penguin/
    ├── test-run.md              # Summary report
    ├── output.log               # Complete console output
    └── logs/
        ├── test_utils.py.log    # Per-file test results
        └── test_api.py.log      # Organized by test file
```

## Repository

For source code, issues, and documentation: https://github.com/zk/3pio

## License

MIT