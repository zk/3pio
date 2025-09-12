# Proposal: Simplify 3pio Architecture by Removing Adapter Output Capture

## Problem Statement

The current 3pio architecture has redundant output capture mechanisms:
1. **Go process manager** captures stdout/stderr at the process level → `output.log`
2. **Test adapters** also attempt to capture stdout/stderr via runtime patching → IPC events

This redundancy creates confusion and fails to capture output in certain failure scenarios (e.g., pytest collection phase errors).

## Current Architecture Issues

### The agno Test Case
When testing the agno repository with pytest:
- Tests failed during collection phase (before execution)
- Go process captured nothing (empty `output.log`)
- Adapter captured nothing (empty IPC file)
- Users get no feedback about what went wrong

### Root Cause
- Pytest writes collection errors to stderr and exits
- Exit happens before adapter hooks can engage
- But Go process is also not capturing the stderr (this is a bug)

## Proposed Architecture

### Clear Separation of Concerns

**Go Process Manager Responsibilities:**
- Capture ALL stdout/stderr from child process → `output.log`
- This is the authoritative record of console output
- Works regardless of adapter state
- Handles all output including early failures

**Adapter Responsibilities (Simplified):**
- Send ONLY test metadata via IPC:
  - `testCase` - test results (pass/fail/skip)
  - `testFileResult` - file-level summaries
  - `testFileStart` - test file boundaries
- NO stdout/stderr capture needed
- NO runtime patching of write functions

## Benefits

1. **Simpler Adapters**
   - Remove all stdout/stderr patching code
   - Fewer moving parts, less to break
   - Easier to maintain and debug

2. **More Reliable**
   - Go process capture cannot be bypassed
   - Works even if adapter completely fails to load
   - Captures early-phase failures (collection, import, syntax errors)

3. **Better Performance**
   - No overhead from patching Python/JavaScript internals
   - No double-buffering of output

4. **Cleaner Architecture**
   - Single source of truth for output (Go process)
   - Adapters focus solely on structured test metadata
   - Clear separation between output capture and test parsing

## Implementation Steps

### Phase 1: Fix Go Process Output Capture
1. Debug why Go process isn't capturing stderr in failure cases
2. Ensure ALL process output goes to `output.log`
3. Test with failing pytest collection scenarios

### Phase 2: Simplify Adapters
1. Remove stdout/stderr patching from Jest adapter
2. Remove stdout/stderr patching from Vitest adapter  
3. Remove output capture from pytest adapter
4. Remove `stdoutChunk`/`stderrChunk` IPC events

### Phase 3: Update Report Generation
1. Use `output.log` as sole source for console output
2. Use IPC events only for test structure/results
3. Document that output attribution to specific tests is best-effort

## Migration Path

1. First ensure Go capture works reliably (fixes immediate problem)
2. Keep adapter output capture temporarily for backward compatibility
3. Add feature flag to disable adapter output capture
4. After validation, remove adapter output capture code entirely

## Alternative Considered

Keep dual capture but make it more robust:
- Pro: Might enable better per-test output attribution
- Con: Complex, redundant, more failure modes
- Decision: Not worth the complexity

## Conclusion

The current dual-capture approach is fundamentally flawed. By having the Go process be the sole capturer of output, we:
- Simplify the architecture significantly
- Handle all failure modes reliably  
- Reduce maintenance burden
- Improve performance

The adapters should focus on what they do best: understanding test structure and results, not capturing output.