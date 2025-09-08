# 3pio Documentation Overview

3pio is an AI-first test runner adapter that acts as a "protocol droid" for test frameworks like Jest and Vitest. It translates traditional test runner output into structured, persistent, file-based records optimized for AI agents.

## Key Features

- **Persistent Test Sessions**: Results saved to filesystem, surviving across development sessions
- **Context-Efficient Output**: Structured Markdown reports alongside real-time console output  
- **Zero-Config Experience**: Wraps existing test runners without requiring test file changes
- **AI-Optimized**: Machine-readable logs searchable with standard shell tools (grep, cat, sed)

## Architecture

3pio uses a four-component architecture with file-based IPC communication:

1. **CLI Orchestrator** - Main entry point managing test lifecycle
2. **Report Manager** - Handles all report file I/O with debounced writes
3. **IPC Manager** - File-based communication between adapters and CLI
4. **Test Runner Adapters** - Silent reporters running inside test processes

## Documentation Files

### Core Design Documents

- **[System Architecture](./system-architecture.md)** - High-level architecture overview with component breakdown and data flow diagrams
- **[Project Plan](./project-plan.md)** - Comprehensive project specification including goals, technology stack, and future roadmap

### Component Documentation

- **[CLI Orchestrator](./cli-orchestrator.md)** - Main entry point design covering argument parsing, test runner detection, and process management
- **[Report Manager](./report-manager.md)** - File I/O component handling debounced writes and report state management
- **[IPC Manager](./ipc-manager.md)** - File-based event communication system between processes
- **[Test Runner Adapters](./test-runner-adapter.md)** - Silent reporter implementations for Jest and Vitest integration

## Quick Start

```bash
npm install -g @heyzk/3pio
3pio run vitest
3pio run jest --watch
3pio run npm test
```

Reports are generated at `.3pio/runs/[timestamp]/test-run.md` with individual test logs in the `logs/` subdirectory.