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
2. **Report Manager** - Handles hierarchical group-based report generation
3. **Group Manager** - Manages test hierarchy with universal group abstractions
4. **IPC Manager** - File-based communication between adapters and CLI
5. **Test Runner Adapters** - Silent reporters running inside test processes

Data flows from test runners → adapters → group events → group hierarchy → reports.

## Documentation Structure

### Architecture Documentation (`architecture/`)

Documentation for understanding and working on the 3pio codebase - system design, implementation details, and development guides.

- **[Architecture](./architecture/architecture.md)** - Complete system architecture, components, and data flow
- **[Test Runner Adapters](./architecture/test-runner-adapters.md)** - Adapter implementation, writing guides, and framework support
- **[Output Handling](./architecture/output-handling.md)** - Console capture strategies and parallel execution handling
- **[Debugging](./architecture/debugging.md)** - Troubleshooting guide and debug logs

### Implementation & Support

- **[Rust Support](./rust-support.md)** - Complete Rust test runner implementation with cargo test and cargo-nextest
- **[Make Support](./make-support.md)** - Makefile parsing and test command extraction (future feature)
- **[Test Organization](./test-organization.md)** - Path sanitization, group counting, and hierarchy handling

### Standards & Guidelines

- **[Integration Test Standards](./integration-test-standards.md)** - Testing requirements and Windows CI guidelines
- **[Design Decisions](./design-decisions.md)** - Key architectural choices and their rationale

### Operations & Troubleshooting

- **[Known Issues](./known-issues.md)** - Current limitations and workarounds
- **[Future Roadmap](./future-roadmap.md)** - Planned enhancements and long-term vision

## Output Structure

```
.3pio/
├── runs/
│   └── 2025-09-09T111224921Z-revolutionary-chewbacca/
│       ├── test-run.md                         # Main report with group hierarchy
│       ├── output.log                          # Complete stdout/stderr capture
│       └── reports/                            # Hierarchical group reports
│           ├── src_components_button_test_js/  # File group directory
│           │   ├── index.md                    # File-level tests
│           │   └── button_rendering/           # Nested describe block
│           │       └── with_props/            # Nested test suite
│           │           └── index.md           # Nested test results
│           └── test_math_py/                   # Python file group
│               └── testmathoperations/         # Class-based test directory
│                   └── index.md                # Class test methods
├── ipc/
│   └── *.jsonl                                # Inter-process communication events
└── debug.log                                   # System debug information
```

## Key Concepts

- **Universal Group Abstractions**: Hierarchical test organization (files → describes → suites → tests)
- **Dynamic Test Discovery**: Tests discovered during execution (standard approach for all runners)
- **Group Events**: Group discovery, start, test cases with hierarchy, and group results
- **Hierarchical Reports**: Nested directory structure mirroring test organization
- **Debounced Writes**: Performance optimization for frequent file updates
- **Silent Adapters**: Test reporters that communicate only via IPC, no console output
- **Deterministic IDs**: SHA256-based group identification for consistent cross-run references
