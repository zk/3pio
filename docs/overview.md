# 3pio Documentation Overview

3pio is a context-friendly test runner for frameworks like Jest, Vitest, and pytest. It translates traditional test runner output into structured, persistent, file-based records optimized for AI agents.

## Key Features

- **Persistent Test Sessions**: Results saved to filesystem, surviving across development sessions
- **Context-Efficient Output**: Structured Markdown reports with individual test case tracking
- **Zero-Config Experience**: Wraps existing test runners without requiring test file changes
- **Agent-Optimized**: Machine-readable logs searchable with standard shell tools (grep, cat, sed)

## Quick Start

### Installation

```bash
# Install via npm (JavaScript projects)
npm install -g @heyzk/3pio

# Install via pip (Python projects)
pip install threepio_test_runner
```

### Usage

```bash
# Run with your existing test commands
3pio npm test
3pio npx jest
3pio npx vitest run
3pio pytest

# Find your reports in .3pio/runs/
cat .3pio/runs/*/test-run.md
```

## Architecture

1. **CLI Orchestrator** - Main entry point managing test lifecycle
2. **Report Manager** - Handles all report file I/O with debounced writes
3. **IPC Manager** - File-based communication between adapters and CLI
4. **Test Runner Adapters** - Silent reporters running inside test processes

Data flows from test runners → adapters → IPC files → CLI → reports.

## Documentation Structure

### Architecture Documentation (`architecture/`)

Documentation for understanding and working on the 3pio codebase - system design, implementation details, and development guides.

- **[Architecture](./architecture/architecture.md)** - Complete system architecture, components, and data flow
- **[Test Runner Adapters](./architecture/test-runner-adapters.md)** - Adapter implementation, embedding, and framework support
- **[Output Handling](./architecture/output-handling.md)** - Console capture strategies and parallel execution handling
- **[Writing Adapters](./architecture/writing-adapters.md)** - Guide for adding new test runner support
- **[Debugging](./architecture/debugging.md)** - Troubleshooting guide and debug logs

### Planning & Design

- **[Design Decisions](./design-decisions.md)** - Key architectural choices and their rationale
- **[Future Vision](./future-vision.md)** - Universal test reporting across all languages
- **[Future Plans](./future-plans.md)** - SBOM, signing, and cleanup features

### Implementation Details

- **[pytest Plugin API](./implementation-details/pytest-plugin-api.md)** - pytest adapter implementation specifics

### Operations & Troubleshooting

- **[Known Issues](./known-issues.md)** - Current limitations and workarounds

## Output Structure

```
.3pio/
├── runs/
│   └── 2025-09-09T111224921Z-revolutionary-chewbacca/
│       ├── test-run.md           # Main report with all test results
│       ├── output.log             # Complete stdout/stderr capture
│       └── logs/                  # Per-file test logs
│           ├── math.test.js.log
│           └── string.test.js.log
├── ipc/
│   └── *.jsonl                   # Inter-process communication events
└── debug.log                      # System debug information

```

## Key Concepts

- **Test Discovery Modes**: Static (files known upfront) vs Dynamic (files discovered during execution)
- **IPC Events**: Test case results, file completion, and console output chunks
- **Debounced Writes**: Performance optimization for frequent file updates
- **Silent Adapters**: Test reporters that communicate only via IPC, no console output
