# Test Runner Abstraction Architecture

## 1. Overview

The test runner abstraction layer provides a pluggable architecture for supporting multiple test frameworks (Jest, Vitest, pytest, and future runners) without modifying core components. This design follows the strategy pattern with explicit, compile-time known test runners in the Go implementation.

## 2. Core Components

### Runner Definition Interface

The Go implementation defines a contract for test runner implementations:
- **Matches**: Determines if a command uses this test runner
- **GetTestFiles**: Discovers test files (static or dynamic mode)
- **BuildCommand**: Injects adapter into command arguments
- **GetAdapterFileName**: Returns the adapter file name
- **InterpretExitCode**: Maps exit codes to semantic meanings

### Runner Manager

Central registry and detection logic:
- Registry of all test runner implementations
- Detection method that checks each runner
- Returns appropriate definition for command building
- Handles npm/yarn/pnpm script resolution

### Embedded Adapters

Test runner adapters are embedded in the Go binary:
- JavaScript adapters for Jest and Vitest
- Python adapter for pytest
- Extracted to temporary directory at runtime
- Cleaned up after test completion

## 3. Implementation Structure

### Directory Organization

The Go implementation structure:
- `internal/runner/` - Test runner management
  - `manager.go` - Registry and detection
  - `definition.go` - Interface and implementations
  - `*_command_test.go` - Comprehensive tests
- `internal/adapters/` - Embedded adapters
  - `jest.js` - Jest reporter
  - `vitest.js` - Vitest reporter
  - `pytest_adapter.py` - pytest plugin
  - `embedded.go` - Go embed directives

### Registration Pattern

Test runners are explicitly registered in the Manager:
- Compile-time known set of runners
- Type-safe runner handling
- No runtime discovery or plugin loading
- Clear, predictable behavior

## 4. Detection Strategy

### Command Detection

Each runner implements pattern matching for:
- Direct invocation (jest, vitest, pytest)
- Package manager invocation (npx, yarn, pnpm)
- npm scripts (npm test, npm run test)
- Python invocations (python -m pytest)

### Package.json Analysis

For abstract commands (npm test):
- Parse scripts section for runner references
- Check for test runner patterns in commands
- Handle nested script references
- Fallback to explicit runner arguments

### Priority Order

Runners checked in registration order:
1. Jest (most common in JavaScript)
2. Vitest (growing adoption)
3. pytest (Python standard)
4. Future runners as added

## 5. Test File Discovery

### Static Discovery

When test files can be determined upfront:
- Jest: Uses --listTests dry run
- Explicit file arguments in command
- Returns complete file list before execution

### Dynamic Discovery

When files discovered during execution:
- Vitest: list command unreliable (runs in watch mode)
- pytest: Collection phase identifies files
- npm run commands without file lists
- Returns empty array, files tracked via IPC events

## 6. Command Building

### Adapter Injection

Each runner defines how to add its adapter:

**Jest:**
- Uses `--reporters` flag
- Replaces default reporter with 3pio adapter
- Absolute path to extracted adapter file

**Vitest:**
- Uses `--reporter` flag (supports multiple)
- Includes both default and 3pio adapter
- Preserves user experience with progress output

**pytest:**
- Sets PYTHONPATH to include adapter directory
- Uses `-p pytest_adapter` to load plugin
- Works with pytest's plugin architecture

### Argument Preservation

Original command structure maintained:
- User flags preserved in order
- File arguments kept in position
- Environment variables passed through
- Shell features supported

## 7. Implementation Details

### Jest Definition

Handles various Jest invocation patterns:
- Direct: `jest`, `jest test.js`
- Via npx: `npx jest`
- Via npm: `npm test` (when package.json uses jest)
- Preserves all Jest-specific flags

### Vitest Definition

Supports Vitest patterns:
- Direct: `vitest`, `vitest run`
- Via npx: `npx vitest`
- Via npm: `npm test` (when package.json uses vitest)
- Handles both run and watch modes

### pytest Definition

Manages Python test patterns:
- Direct: `pytest`, `py.test`
- Via python: `python -m pytest`
- With options: `pytest -v tests/`
- Supports all pytest plugins and options

## 8. Integration Points

### Orchestrator

Uses Runner Manager for:
- Runner detection from commands
- Test file discovery
- Command modification with adapter
- Exit code interpretation

### Report Manager

Receives events from adapters:
- Test file start/completion
- Individual test case results
- Console output chunks
- Processes into structured reports

### IPC Manager

Facilitates communication:
- Adapters write events to IPC file
- Manager watches for new events
- Events parsed and forwarded
- Channel-based event delivery

## 9. Adding New Test Runners

### Implementation Steps

1. Create runner definition in `internal/runner/definition.go`
2. Implement required interface methods
3. Register in Manager's initialization
4. Create adapter in appropriate language
5. Add adapter to `internal/adapters/`
6. Update embedded.go with embed directive
7. Write comprehensive tests
8. Update documentation

### Required Components

- Definition struct implementing interface
- Detection logic for command patterns
- Command building with adapter injection
- Adapter implementation for test runner
- Unit and integration tests
- Documentation updates

## 10. Benefits

### Maintainability
- Single responsibility per component
- Clear interfaces and contracts
- Isolated test runner logic
- Reduced coupling between components

### Extensibility
- New runners don't modify existing code
- Well-defined extension points
- Consistent patterns across runners
- Type safety in Go

### Performance
- Embedded adapters (no runtime download)
- Efficient detection algorithms
- Minimal overhead in command building
- Concurrent event processing

## 11. Design Decisions

### Go Implementation Choice
- Single binary distribution
- Cross-platform compatibility
- Superior performance
- Built-in concurrency

### Embedded Adapters
- No external dependencies
- Version consistency guaranteed
- Fast extraction at runtime
- Automatic cleanup

### File-Based IPC
- Simple, reliable communication
- Works across all platforms
- Easy to debug and inspect
- No network dependencies

## 12. Future Considerations

### Potential Enhancements
- Support for Mocha, Jasmine
- Ruby RSpec integration
- Custom runner configurations
- Plugin architecture for external runners

### Scalability
Current design supports:
- Additional test runners without refactoring
- Custom runners via interface implementation
- Performance optimization opportunities
- Distributed test execution