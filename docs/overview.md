# 3pio Documentation Overview

3pio is a context-friendly test runner for frameworks like Jest, Vitest, Mocha, Cypress, and pytest — and native runners like Go test and Rust (cargo test/nextest). It translates traditional test runner output into structured, persistent, file-based records optimized for AI agents.

## Project Goals

- Make running tests context-efficient for agents, existing test runners' output ergonomics are for humans.
- First class DX for devs and agents. Since 3pio is a tool for agents, dev DX is mainly around installation and maintenance.
  - Easy to install with your preferred package manager. This means we support many package managers, but they all install the same artifacts.
  - Easy to use for agents: simple invocation (prefix existing test command with `3pio`). This means supporting and testing against a wide range of cli invocations.

## Key Features

- **Persistent Test Sessions**: Results saved to filesystem, surviving across development sessions
- **Context-Efficient Output**: Structured Markdown reports with individual test case tracking
- **Zero-Config Experience**: Wraps existing test runners without requiring test file changes
- **Agent-Optimized**: Machine-readable logs searchable with standard shell tools (grep, cat, sed)

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
