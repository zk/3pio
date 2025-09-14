# How to write good architecture documentation

Architecture documentation is important for loading the context with relevant information that helps you, the agent, work well in the codebase.

Typically you'll be asked to do things like add features, fix bugs, and answer questions about the codebase, for which good architecture documentation is key.


## Where?

Architecture documentation should live at `./docs/architecture.md` and contain the following sections


## Sections

### Overview

High-level executive-style overview the entire architecture of the project. What the project does and why it exists. What problems does it solve and who does it solve them for? It should also contain a high-level explanation of the technologies used to solve these problems. For example:

```markdown
3pio is a context-friendly test runner for frameworks like Jest, Vitest, and pytest. It translates traditional test runner output into structured, persistent, file-based records optimized for AI agents.

It uses a project's existing test runner to run tests via a main process, and depending on the specific test runner it inject adapters or capture output from the test process to write a heirarchy of test results on the filesystem in a way that is easy for coding agents to understand and work with.
```

### System Components

Should list the system components of the project and explain how they work together. It's important to be terse in this section. Generally a one paragraph plain-engligh explanation followed by a section for each component. Each component section should have a list that outlines it's responsibilities and connections to other components.

#### Example

```markdown
The system consists of six primary components:

### 1. CLI Entry Point (`cmd/3pio/main.go`)
- Parses command-line arguments
- Initializes file-based logger for debug output
- Creates and configures the Orchestrator
- Handles version and help commands
- Passes control to Orchestrator for test execution

### 2. Orchestrator (`internal/orchestrator/`)
The central controller managing the entire test execution lifecycle:
- Generates unique run IDs (format: `{timestamp}-{adjective}-{character}`, e.g., `20250911T194308-sneaky-yoda`)
- Detects test runner using Runner Manager
- Creates run directory structure (`.3pio/runs/[runID]/`)
- Initializes IPC and Report managers
- Extracts and prepares embedded adapters
- Spawns test process with adapter injection
- Captures stdout/stderr through pipes
- Processes IPC events concurrently
- Handles signals (SIGINT/SIGTERM) gracefully
- Mirrors test runner exit codes

### 3. Runner Manager (`internal/runner/`)
Manages test runner detection and configuration:
- Registry of supported test runners (Jest, Vitest, pytest)
- Detects runner from command arguments
- Parses package.json for npm/yarn/pnpm commands
- Builds modified commands with adapter injection
- Extracts test files from arguments
- Handles various invocation patterns
```

### Data Flow

### Key Design Decisions

### Error Handling
