# 3pio

A context-competent test runner for coding agents. This package provides the 3pio CLI as a native Go binary with zero runtime dependencies.

## Installation

```bash
npm install -g @heyzk/3pio
```

## Usage

```bash
# Run Jest tests
3pio npx jest

# Run Vitest tests  
3pio npx vitest run

# Run pytest tests
3pio pytest

# Run any npm script
3pio npm test
```

## What's Different

This package installs a native Go binary that provides:

- **Zero runtime dependencies** - No Node.js required at runtime
- **Fast startup** - ~50ms vs ~200ms for Node.js version
- **Lower memory usage** - ~10MB vs ~50MB for Node.js version
- **Cross-platform** - Single binary works on macOS, Linux, and Windows

## Supported Test Frameworks

- **Jest** - JavaScript testing framework
- **Vitest** - Fast Vite-native unit testing framework  
- **pytest** - Python testing framework

## How It Works

During installation, this package automatically downloads the appropriate native binary for your platform from GitHub releases. The binary includes embedded adapters for each supported test framework.

## Architecture

The 3pio binary acts as a "protocol droid" that translates test runner output into structured, AI-friendly reports. It:

1. Spawns your test runner with a silent reporter/adapter
2. Captures all output via IPC (Inter-Process Communication)
3. Generates structured reports in `.3pio/runs/[timestamp]-[name]/`
4. Maintains compatibility with existing test runner behavior

## Repository

For source code, issues, and documentation: https://github.com/zk/3pio

## License

MIT