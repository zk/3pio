# Log Level Configuration via Environment Variable

**Date**: 2025-09-15 23:25
**Status**: Planning
**Priority**: Medium

## Objective

Implement configurable log levels for 3pio via the `THREEPIO_LOG_LEVEL` environment variable, with log level settings injected into extracted adapters during runtime.

## Background

Currently, 3pio uses a fixed WARN log level for production performance. Debug logging requires code changes. We need configurable log levels that work across:

- CLI orchestrator (Go)
- Extracted adapters (JavaScript/Python) running in separate processes
- Complex process hierarchies where environment variables don't propagate reliably

## Current Architecture Analysis

**Environment Variable Handling**: System uses `THREEPIO_IPC_PATH` passed to child processes via `cmd.Env` in orchestrator.go:328, but this doesn't work reliably for adapters in complex process hierarchies.

**Adapter Extraction**: `extractAdapter()` in `internal/adapters/embedded.go` already has template injection using regex replacement for IPC paths. This is the reliable mechanism for configuration.

**Logger Architecture**: `FileLogger` in `internal/logger/file_logger.go` has configurable log levels but defaults to WARN. Needs dynamic configuration from environment.

## Design

### Environment Variable

- **Variable**: `THREEPIO_LOG_LEVEL`
- **Values**: `DEBUG`, `INFO`, `WARN`, `ERROR` (case-insensitive)
- **Default**: `WARN` (maintains current production-optimized behavior)
- **Scope**: CLI reads from environment, injects into adapters via code injection

### Code Injection Strategy

**Rationale**: Since adapters run in separate processes where environment variables don't propagate reliably, we inject the log level directly into adapter source code during extraction (similar to existing IPC path injection).

**Template Markers**:
- **JavaScript**: `/*__LOG_LEVEL__*/"WARN"/*__LOG_LEVEL__*/`
- **Python**: `#__LOG_LEVEL__#"WARN"#__LOG_LEVEL__#`

### Implementation Flow

1. CLI reads `THREEPIO_LOG_LEVEL` environment variable
2. FileLogger configured with specified level
3. During adapter extraction, log level injected into adapter code templates
4. Adapters use injected log level configuration for their own logging

## Technical Approach

### Dual Template Injection

Extend existing `extractAdapter()` function to handle both:
- IPC path injection (existing)
- Log level injection (new)

### Adapter Logging

Adapters currently have their own logging mechanisms. Standardize to respect injected log level:
- JavaScript: Use injected level for console/debug output
- Python: Use injected level for logging module configuration

## Implementation Checklist

### Core Implementation
- [ ] Add log level parsing to FileLogger constructor
- [ ] Modify `NewFileLogger()` to read `THREEPIO_LOG_LEVEL` environment variable
- [ ] Add log level validation and default handling
- [ ] Update adapter extraction to support dual template injection (IPC + log level)
- [ ] Add log level template markers to JavaScript adapters (jest.js, vitest.js)
- [ ] Add log level template markers to Python adapter (pytest_adapter.py)
- [ ] Implement log level injection in `extractAdapter()` function

### Adapter Updates
- [ ] Update Jest adapter to use injected log level for debugging output
- [ ] Update Vitest adapter to use injected log level for debugging output
- [ ] Update pytest adapter to use injected log level for Python logging
- [ ] Ensure adapter logging respects injected level configuration

### Testing
- [ ] Write unit tests for log level parsing and validation
- [ ] Write unit tests for dual template injection mechanism
- [ ] Test adapter extraction with different log levels
- [ ] Integration test with Jest using DEBUG level
- [ ] Integration test with Vitest using DEBUG level
- [ ] Integration test with pytest using DEBUG level
- [ ] Test default behavior (WARN level) unchanged
- [ ] Test invalid log level handling

### Documentation
- [ ] Update CLAUDE.md with new environment variable
- [ ] Update architecture documentation for log level injection
- [ ] Update debugging documentation with log level usage
- [ ] Add examples of log level configuration to README

## Benefits

- **Reliability**: Works regardless of environment variable propagation issues
- **Consistency**: Same log level across CLI and all adapters
- **Backward Compatibility**: Defaults maintain current WARN behavior
- **Debugging**: Enables fine-grained debug output when needed
- **Performance**: Debug logging minimal overhead when disabled

## Implementation Notes

### Multiple Injection Points
Need to handle both IPC path and log level injection in same extraction process:
```go
// Existing: IPC path injection
pattern := regexp.MustCompile(`/\*__IPC_PATH__\*/".*?"/\*__IPC_PATH__\*/`)
contentStr = pattern.ReplaceAllString(contentStr, escapedPath)

// New: Log level injection
logPattern := regexp.MustCompile(`/\*__LOG_LEVEL__\*/".*?"/\*__LOG_LEVEL__\*/`)
contentStr = logPattern.ReplaceAllString(contentStr, escapedLogLevel)
```

### Adapter Logging Standards
Currently adapters log via different mechanisms:
- Jest: Console output and internal debugging
- Vitest: Logger.create() system
- pytest: Python logging module

Standardize these to respect injected log level.

### Performance Considerations
Debug logging should have minimal overhead when disabled:
- Use conditional logging based on injected level
- Avoid expensive string formatting for disabled levels
- Maintain current production performance at WARN level

## Success Criteria

1. `THREEPIO_LOG_LEVEL=DEBUG` enables detailed logging in CLI and all adapters
2. Default behavior (WARN level) unchanged for backward compatibility
3. Log level configuration works reliably across all test runners
4. Performance impact negligible when debug logging disabled
5. Comprehensive test coverage for all log levels and adapters