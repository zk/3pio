# Runtime Adapter Generation with Embedded IPC Path

## Brilliant Solution!

Since 3pio already writes adapters to `.3pio/adapters/[hash]/` at runtime, we can modify the adapter code on-the-fly to hardcode the IPC path directly into it. This completely eliminates the need for environment variable propagation.

## Current Flow
1. Go binary has embedded adapter files (jest.js, vitest.js, pytest_adapter.py)
2. `extractAdapter()` writes these to `.3pio/adapters/[hash]/`
3. Adapter relies on `process.env.THREEPIO_IPC_PATH`

## Proposed Solution

Modify `extractAdapter()` in `internal/adapters/embedded.go` to inject the IPC path:

```go
// extractAdapter extracts an embedded adapter with IPC path injected
func extractAdapter(name, ipcPath, runID string) (string, error) {
    var content []byte
    var filename string
    
    switch name {
    case "vitest.js":
        content = vitestAdapter
        filename = "vitest.js"
    // ... other cases
    }
    
    // Replace template markers with actual IPC path
    contentStr := string(content)
    
    // For JavaScript adapters, use JSON.stringify for proper escaping
    if name == "vitest.js" || name == "jest.js" {
        escapedPath := strconv.Quote(ipcPath) // Go's strconv.Quote is similar to JSON.stringify
        contentStr = regexp.MustCompile(`/\*__IPC_PATH__\*/".*?"/\*__IPC_PATH__\*/`).
            ReplaceAllString(contentStr, escapedPath)
    }
    
    // For Python adapter, use Python string escaping
    if name == "pytest_adapter.py" {
        // Python uses similar escaping to JSON for basic strings
        escapedPath := strconv.Quote(ipcPath)
        contentStr = regexp.MustCompile(`#__IPC_PATH__#".*?"#__IPC_PATH__#`).
            ReplaceAllString(contentStr, escapedPath)
    }
    
    content = []byte(contentStr)
    
    // Use run ID for adapter directory (e.g., "20250911T085108-feisty-han-solo")
    adapterDir := filepath.Join(".3pio", "adapters", runID)
    if err := os.MkdirAll(adapterDir, 0755); err != nil {
        return "", fmt.Errorf("failed to create adapter directory: %w", err)
    }
    
    // Write adapter file
    adapterPath := filepath.Join(adapterDir, filename)
    // ... rest of function
}
```

## Implementation Changes Needed

### 1. Modify `GetAdapterPath()` to accept IPC path and run ID
```go
// internal/adapters/embedded.go
func GetAdapterPath(name, ipcPath, runID string) (string, error) {
    // No caching needed since each run gets its own adapter
    return extractAdapter(name, ipcPath, runID)
}
```

### 2. Update orchestrator to pass IPC path and run ID
```go
// internal/orchestrator/orchestrator.go
func (o *Orchestrator) prepareAdapter() error {
    // Get adapter path with IPC path and run ID injected
    // runID is something like "20250911T085108-feisty-han-solo"
    adapterPath, err := adapters.GetAdapterPath(o.adapterFile, o.ipcPath, o.runID)
    if err != nil {
        return fmt.Errorf("failed to get adapter: %w", err)
    }
    o.adapterPath = adapterPath
    return nil
}
```

### 3. Adapter Template Markers
Modify the adapter source files to use template markers:

**JavaScript adapters (jest.js, vitest.js):**
```javascript
// Instead of:
const ipcPath = process.env.THREEPIO_IPC_PATH;

// Use:
const ipcPath = /*__IPC_PATH__*/"WILL_BE_REPLACED"/*__IPC_PATH__*/;
```

**Python adapter (pytest_adapter.py):**
```python
# Instead of:
ipc_path = os.environ.get('THREEPIO_IPC_PATH')

# Use:
ipc_path = #__IPC_PATH__#"WILL_BE_REPLACED"#__IPC_PATH__#
```

Also remove the error checks for missing environment variables since the path will always be present.

## Benefits

1. **100% Reliable** - No environment variable issues
2. **Works in Monorepos** - Each spawned process gets adapter with correct path
3. **No Discovery Needed** - Path is hardcoded at generation time
4. **Process-Safe** - Each run gets its own adapter instance
5. **Clean** - No hacky workarounds or fallback mechanisms
6. **Self-Contained** - Adapter lives in `.3pio/adapters/[runID]/`
7. **No Cache Pollution** - Each run has its own adapter, no hash collisions
8. **Dev Tool Focus** - Optimized for development/agent use, not CI/CD performance

## Failure Modes

### 1. Path Escaping Issues
**Scenario**: IPC path contains characters that break JavaScript string literals.
- **Example**: Windows path with backslashes: `C:\Users\test\.3pio\ipc\test.jsonl`
- **Example**: Path with quotes: `/home/user's files/.3pio/ipc/test.jsonl`
- **Detection**: JavaScript syntax error when adapter loads
- **Mitigation**: 
  - Properly escape all paths before injection (backslashes, quotes, newlines)
  - Convert Windows paths to forward slashes for JavaScript
  - Test with paths containing special characters

### 2. Disk Space Exhaustion
**Scenario**: Many test runs accumulate adapter directories without cleanup.
- **Example**: CI server runs hundreds of tests daily, each creating adapter directory
- **Detection**: Disk full errors, test failures
- **Mitigation**: 
  - Implement cleanup of old run directories (configurable retention)
  - Reuse adapter if IPC path hasn't changed (optional optimization)
  - Monitor disk usage in .3pio directory

### 3. Permission Errors
**Scenario**: Cannot write adapter to run directory due to permissions.
- **Example**: Running in restricted container, read-only filesystem
- **Detection**: File creation fails with permission denied
- **Mitigation**: 
  - Fail fast with clear error message
  - Document required permissions in known-issues.md
  - No fallback - if we can't write adapters, 3pio cannot function

### 4. Partial Injection
**Scenario**: Template markers are not all replaced correctly.
- **Example**: Regex pattern doesn't match due to code changes
- **Detection**: Adapter fails to load or sends no events
- **Mitigation**: 
  - Use consistent, unique template markers
  - Fail fast if markers remain after replacement
  - Test replacement logic thoroughly

### 5. Python Adapter Complications
**Scenario**: Python string injection has different escaping rules than JavaScript.
- **Example**: Python raw strings, triple quotes, different escape sequences
- **Detection**: Python syntax errors or incorrect path interpretation
- **Mitigation**: 
  - Handle Python and JavaScript injection separately
  - Test Python path injection with various path formats
  - Use Python-specific escaping rules

## Testing Strategy

### Unit Tests

**Test: Basic IPC Path Injection**
- Given: A vitest adapter source and an IPC path
- When: Extracting adapter with injection
- Then: 
  - Adapter file exists in correct run directory
  - IPC path is hardcoded in the adapter
  - No environment variable references remain
  - File is valid JavaScript (can be parsed)

**Test: Special Characters in Path**
- Given: IPC paths with various special characters
  - Spaces: `/home/user/my files/.3pio/ipc/test.jsonl`
  - Quotes: `/home/user's/.3pio/ipc/test.jsonl`
  - Backslashes (Windows): `C:\Users\test\.3pio\ipc\test.jsonl`
  - Unicode: `/home/用户/.3pio/ipc/test.jsonl`
- When: Injecting each path
- Then: 
  - Paths are properly escaped
  - Adapter remains valid JavaScript
  - Path can be read back correctly

### Integration Tests

**Test: Monorepo with Multiple Packages**
- Setup: Create test monorepo with 3 packages, each with tests
- Execute: Run `pnpm test` with 3pio
- Verify:
  - Each package's tests receive adapter with correct IPC path
  - All test events are recorded in single IPC file
  - Individual log files are created for each test file

**Test: Concurrent Test Runs**
- Setup: Start two 3pio test runs simultaneously
- Execute: Both runs create and use adapters
- Verify:
  - Each run has its own adapter directory
  - No file conflicts or corruption
  - Both runs complete successfully
  - IPC events are written to correct files

**Test: Long-Running Tests with Process Spawning**
- Setup: Test that spawns child processes that also run tests
- Execute: Run parent test with 3pio
- Verify:
  - Child processes use adapter with correct IPC path
  - Events from all processes are captured
  - No environment variable errors in child processes

## Implementation Priority

All test scenarios are equally important and should be implemented:
1. **Monorepo with Multiple Packages** - Solves the original problem
2. **Special Character Path Handling** - Ensures robustness 
3. **Concurrent Test Runs** - Validates unique adapter isolation

## Summary

This plan modifies 3pio to inject IPC paths directly into adapter code at runtime, eliminating environment variable propagation issues in monorepos. The solution uses template markers for clean injection, JSON.stringify-style escaping for safety, and stores adapters in `.3pio/adapters/[runID]/`. It fails fast on errors with no fallbacks, focusing on being a reliable development tool for agents rather than a high-performance CI tool.