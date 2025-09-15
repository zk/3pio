# Simplify Output Handling - Use output.log Directly

## Objective
Eliminate the complex temporary file mechanism and use output.log directly for ALL test runners (Jest, Vitest, pytest, Go test, cargo test, nextest).

## Current Problems
1. **Dual File System**: Creating both temp file and output.log with same content
2. **Complex Conditionals**: Different paths for different runners
3. **Race Conditions**: File closing issues with pipes
4. **Code Duplication**: TeeReader copying from temp to output.log

## Implementation Plan

### Step 1: Remove Temporary File Creation
- **Location**: orchestrator.go lines 350-355
- **Action**: Delete temp file creation code
- **Keep**: output.log as the single source of truth

### Step 2: Change output.log to Write Mode
- **Location**: orchestrator.go line 339
- **Change FROM**: `os.O_WRONLY|os.O_APPEND`
- **Change TO**: `os.O_CREATE|os.O_WRONLY|os.O_TRUNC`
- **Reason**: Start fresh, not append to existing

### Step 3: Redirect Command Output to output.log
- **Location**: orchestrator.go lines 382, 387
- **Change FROM**: `cmd.Stdout = tempFile`
- **Change TO**: `cmd.Stdout = outputFile`
- **Apply to**: ALL runners universally

### Step 4: Tail output.log Instead of Temp File
- **Location**: orchestrator.go lines 417-421
- **Change FROM**: Open tempPath for reading
- **Change TO**: Open outputPath for reading
- **Use**: Same TailReader mechanism

### Step 5: Remove TeeReader Complexity
- **Location**: orchestrator.go line 470 and similar
- **Change FROM**: `io.TeeReader(fileReader, outputFile)`
- **Change TO**: Direct reading from fileReader
- **Reason**: Already reading from output.log

### Step 6: Clean Up Unnecessary Code
- Remove temp file cleanup (lines 611-619)
- Remove isCargoTest, isGoTest variables
- Remove complex runner type detection
- Simplify to just check if Go test needs separate stderr

### Step 7: Simplify Runner Detection
```go
// Simple approach:
usesSeparateStderr := false
if isNativeRunner {
    if _, ok := nativeDef.(*definitions.GoTestDefinition); ok {
        usesSeparateStderr = true
    }
}

// Universal output handling:
cmd.Stdout = outputFile
if usesSeparateStderr {
    stderrPipe, _ = cmd.StderrPipe()
} else {
    cmd.Stderr = outputFile
}
```

## Benefits
1. **50% Less Code**: Remove ~100 lines of conditionals
2. **No Race Conditions**: Single file, no synchronization issues
3. **Universal Solution**: Works for ALL test runners
4. **Simpler Debugging**: One output file to inspect
5. **Better Performance**: No redundant file operations

## Testing Plan
1. Test with Jest - verify output captured
2. Test with Vitest - verify output captured
3. Test with pytest - verify output captured
4. Test with Go test - verify stderr handled correctly
5. Test with cargo test - verify combined output works
6. Test with nextest - verify no race conditions

## Risk Assessment
- **Low Risk**: Simplifying existing working mechanism
- **Main Change**: Where command output goes (direct to output.log)
- **Fallback**: Git history has working version if needed

## Implementation Order
1. First backup current orchestrator.go
2. Make changes incrementally
3. Test after each major change
4. Verify all test runners still work