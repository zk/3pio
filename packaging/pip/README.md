# 3pio

A context-competent test runner for coding agents. This package provides the 3pio CLI as a native Go binary with zero runtime dependencies.

## Installation

```bash
pip install 3pio
```

## Usage

```bash
# Run pytest tests
3pio pytest

# Run Jest tests (if Node.js is available)
3pio npx jest

# Run Vitest tests (if Node.js is available)
3pio npx vitest run

# Run any command with structured output capture
3pio python -m unittest
```

## What's Different

This package installs a native Go binary that provides:

- **Zero runtime dependencies** - No Python runtime dependencies beyond installation
- **Fast startup** - ~50ms vs ~200ms for pure Python implementations
- **Lower memory usage** - ~10MB vs typical Python process overhead
- **Cross-platform** - Single binary works on macOS, Linux, and Windows

## Supported Test Frameworks

- **pytest** - Python testing framework (primary use case)
- **Jest** - JavaScript testing framework (requires Node.js)
- **Vitest** - Fast Vite-native unit testing framework (requires Node.js)
- **unittest** - Built-in Python testing framework

## How It Works

During installation, this package automatically downloads the appropriate native binary for your platform from GitHub releases. The binary includes embedded adapters for each supported test framework.

## Architecture

1. Spawns your test runner with a silent reporter/adapter
2. Captures all output via IPC (Inter-Process Communication)
3. Generates structured reports in `.3pio/runs/[timestamp]-[name]/`
4. Maintains compatibility with existing test runner behavior

## Python Integration

When used with pytest, 3pio provides enhanced output capture and structured reporting:

```bash
# Standard pytest run with structured output
3pio pytest tests/

# Run with specific markers
3pio pytest -m "integration"

# Run with coverage (pytest-cov required)
3pio pytest --cov=src tests/
```

The generated reports in `.3pio/runs/` contain:
- Individual test case results with timing
- Console output per test file
- Failure details and stack traces
- Summary statistics

## Repository

For source code, issues, and documentation: https://github.com/zk/3pio

## License

MIT
