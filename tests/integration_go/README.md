# Integration Test Organization

This directory contains integration tests for 3pio, organized by test category and runner.

## Naming Convention

Test files follow the pattern: `{category}_{runner}_test.go` or `{category}_test.go` for cross-runner tests.

### Categories

Based on `docs/integration-test-standards.md`:

- **basic_** - Basic functionality (full runs, specific files, pattern matching, exit codes)
- **report_** - Report generation (structure, content, YAML frontmatter, hierarchical reports)
- **error_** - Error handling (config errors, test failures, syntax errors, missing files)
- **process_** - Process management (SIGINT/SIGTERM, graceful shutdown, partial results)
- **command_** - Command variations (npm/yarn/pnpm, separators, verbose/quiet, watch/coverage rejection)
- **structure_** - Complex project structures (monorepo, nested dirs, long names, special chars)
- **ipc_** - IPC & adapter management (event stream, adapter injection, cleanup)
- **output_** - Console output capture (stdout/stderr, ANSI colors, progress bars, buffering)
- **performance_** - Performance & scale (large suites, long-running tests, parallel execution)
- **state_** - State management (run directories, persistence, cleanup, concurrent runs)
- **platform_** - Platform-specific tests (Windows, path separators, binary extensions)

### Test Runners

- **jest** - Jest-specific tests
- **vitest** - Vitest-specific tests
- **pytest** - pytest-specific tests
- **rust** - Rust/cargo test specific tests
- **go** - Go test specific tests (if applicable)
- (no suffix) - Cross-runner tests that apply to multiple runners

## File Organization

```
tests/integration_go/
├── README.md                    # This file
├── helpers_test.go              # Shared test helpers and utilities
│
├── basic_test.go               # Cross-runner basic functionality
├── basic_jest_test.go          # Jest-specific basic tests
├── basic_vitest_test.go        # Vitest-specific basic tests
├── basic_pytest_test.go        # pytest-specific basic tests
├── basic_rust_test.go          # Rust-specific basic tests
│
├── report_generation_test.go   # Report generation tests
├── report_formatting_test.go   # Report formatting tests
├── report_failure_test.go      # Failure reporting tests
│
├── error_console_test.go       # Console error reporting
├── error_format_test.go        # Error message formatting
├── error_vitest_test.go        # Vitest-specific error handling
├── error_pytest_test.go        # pytest-specific error handling
│
├── process_interruption_test.go # Process interruption handling
├── process_pytest_test.go      # pytest-specific process tests
│
├── command_npm_test.go         # npm command variations
├── command_watch_test.go       # Watch mode rejection
├── command_coverage_test.go    # Coverage mode rejection
├── command_variations_test.go  # Other command variations
│
├── structure_monorepo_test.go  # Monorepo support
├── structure_esm_test.go       # ESM compatibility
├── structure_pytest_test.go    # pytest complex structures
│
├── output_path_test.go         # Path consistency in output
├── output_edge_test.go         # Edge cases in output capture
│
├── performance_scale_test.go   # Large-scale performance tests
│
├── state_concurrent_test.go    # Concurrent run tests
│
└── platform_windows_test.go    # Windows-specific tests
```

## Running Tests

```bash
# Run all integration tests
go test ./tests/integration_go

# Run specific category
go test ./tests/integration_go -run "^TestBasic"

# Run tests for specific runner
go test ./tests/integration_go -run "Jest"

# Run with verbose output
go test -v ./tests/integration_go

# Run single test
go test -v ./tests/integration_go -run "^TestBasicJestFullRun$"
```

## Test Utilities

Common test utilities are provided in `helpers_test.go`:

- `getBinaryPath()` - Returns platform-appropriate binary path with .exe on Windows
- `runTest()` - Executes 3pio with given arguments and returns output
- `cleanTestDir()` - Removes .3pio directory for clean test environment
- `assertFileExists()` - Verifies file exists with helpful error message
- `assertFileContains()` - Verifies file contains expected content
- `assertExitCode()` - Verifies command exit code matches expected

## Adding New Tests

1. Determine the appropriate category for your test
2. Choose runner-specific or cross-runner file
3. Follow existing test patterns in that category
4. Use shared helpers from `helpers_test.go`
5. Ensure test cleans up after itself
6. Add appropriate build tags if platform-specific

## Platform Considerations

- Always use `filepath.Join()` for path construction
- Use `getBinaryPath()` helper for binary references
- Check `runtime.GOOS` for platform-specific behavior
- Test on all platforms before merging

## CI Integration

These tests run automatically on:
- Linux (Ubuntu latest)
- macOS (latest)
- Windows (latest)

Tests must pass on all platforms before merge.