# Integration Test Standards

## Core Test Categories

### 1. Basic Functionality
- Full test run with no arguments
- Specific file execution
- Pattern matching
- Exit code mirroring

### 2. Console Output
- Per-group summary lines with report paths
- Final results summary with counts
- Preamble with metadata (time, cwd, command, paths)
- Proper outcome messages

### 3. Report Generation
- `test-run.md` with correct structure
- `output.log` with all stdout/stderr
- Hierarchical reports in `reports/` directory
- YAML frontmatter with metadata
- Accurate test results and error messages

### 4. Error Handling
- Configuration errors
- Test failures with stack traces
- Syntax errors
- Missing test files
- Empty test suites

### 5. Process Management
- SIGINT/SIGTERM handling
- Partial results preservation
- Clean shutdown

### 6. Command Variations
- Package manager support (npm/yarn/pnpm)
- Separator handling (`--`)
- Watch mode rejection
- Coverage mode rejection

### 7. Complex Structures
- Monorepo support
- Nested test directories
- Long names and special characters

### 8. IPC & Adapters
- JSONL event stream
- Adapter injection and cleanup
- Event completeness and ordering

### 9. Output Capture
- Stdout/stderr capture
- ANSI color preservation
- Large output handling

### 10. Performance & Scale
- 100+ test files
- Long-running tests
- Parallel execution
- Memory and file handle management

### 11. State Management
- Unique timestamped run directories
- Incremental report writing
- Concurrent run support

## Windows CI Requirements

### Binary Extension
- Append `.exe` for Windows executables
- Use `runtime.GOOS` detection

### Path Handling
- Use `filepath.Join()` for all paths
- Never hardcode separators

### CI Workflow
- Separate Windows/Unix steps
- Use PowerShell for Windows
- Account for `.exe` in scripts

## Validation Criteria

Complete when:
1. All categories covered
2. Exit codes match runner
3. Reports generated for all scenarios
4. Interruption handling works
5. Console output captured
6. IPC events complete
7. Resources managed properly
8. Clear error messages

## Best Practices

- Clean environment before/after tests
- No execution order dependencies
- Descriptive test names
- Minimal focused fixtures