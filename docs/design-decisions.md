# Design Decisions

This document captures key architectural and design decisions made during the development of 3pio, along with the rationale behind them.

## Console Output Design

### Test File List Removal (v0.0.1)

**Decision**: Remove the initial list of test files from console output.

**Previous Behavior**: 
- For small test runs (<10 files), all file paths were listed
- For medium runs (10-25 files), first 10 files were shown with a count of remaining
- For large runs (>25 files), a directory breakdown was displayed

**New Behavior**:
- No upfront file listing
- Only show "Beginning test execution now..."
- Files are still reported as they run (RUNNING/PASS/FAIL status)

**Rationale**:
1. **Context Efficiency for AI Agents**: The primary consumers of 3pio output are AI coding assistants. Showing potentially hundreds of file paths consumes significant context window space while providing minimal actionable value.

2. **Avoiding Duplicate Information**: Test files are already reported in real-time as they execute (RUNNING status) and complete (PASS/FAIL status). The initial list was redundant.

3. **Faster Time to First Test**: Removing the list generation and display logic slightly improves startup time, especially for large test suites.

4. **Cleaner Output**: Less visual noise makes it easier to focus on test execution progress and results.

**Trade-offs**:
- Human users lose the ability to see all files that will be tested upfront
- Cannot estimate test suite size before execution begins
- These trade-offs are acceptable given 3pio's AI-first design philosophy

## Go Package Status Display (2025-09-13)

**Decision**: Display `NO_TESTS` status for Go packages that have no test files.

**Background**: Go's test runner uniquely reports on ALL packages when running `go test ./...`, including those without any test files. Other test runners (Jest, Vitest, pytest) only report on files that match test patterns.

**Status Meanings**:
- `PASS` - Package has tests and they all passed
- `FAIL` - Package has tests and at least one failed
- `SKIP` - Tests were skipped (e.g., with build tags or t.Skip())
- `NO_TESTS` - Package exists but has no test files (Go specific)

**Rationale**:
1. **Clear Distinction**: Differentiates between packages that have tests but skip them vs packages with no tests at all
2. **Coverage Visibility**: Makes it immediately obvious which packages lack test coverage entirely
3. **Go Philosophy Alignment**: Aligns with Go's opinionated stance that all packages should have tests

## Failure Display Format (2025-09-13)

**Decision**: Show up to 3 failed test names in console output with "+N more" indicator for additional failures.

**Example Output**:
```
FAIL     github.com/zk/3pio/tests/integration_go (18.82s)
  x TestMonorepoIPCPathInjection
  x TestReportFileGeneration
  x TestErrorRecovery
  +12 more
  See .3pio/runs/20250913T135142-loopy-neelix/reports/github_com_zk_3pio_tests_integration_go/index.md
```

**Rationale**:
1. **Context Efficiency**: Provides immediate visibility into which specific tests failed without overwhelming the console
2. **Actionable Information**: Developers can quickly identify patterns in failures (e.g., all auth tests failing)
3. **Progressive Disclosure**: Full failure details are available in the report file for those who need them
4. **Consistent Format**: Matches patterns used by other modern test runners

**Implementation Details**:
- Failed test names are collected recursively from the entire test group hierarchy
- Display order matches the order tests were executed
- Report path is always shown last for easy access to full details
- Single failures don't show the "+N more" indicator

## Dynamic Test Discovery

**Decision**: Support both static and dynamic test file discovery modes.

**Rationale**:
- Some test runners (like Vitest with npm run test) cannot reliably provide a test file list upfront
- Dynamic discovery allows 3pio to work with any test runner configuration
- Files are registered as they send their first IPC event

**Implementation**:
- ReportManager accepts optional test file list in initialize()
- ensureTestFileRegistered() method dynamically adds files as discovered
- System automatically chooses mode based on test runner capabilities

## Runtime Adapter Generation with Embedded IPC Path (2025-09-11)

**Decision**: Inject IPC paths directly into adapter code at runtime instead of using environment variables.

**Previous Behavior**:
- Adapters extracted to `.3pio/adapters/[hash]/` directory with content-based hash
- Adapters relied on `THREEPIO_IPC_PATH` environment variable
- Same adapter could be reused across multiple test runs
- Adapter extraction was cached based on content hash

**New Behavior**:
- Adapters extracted to `.3pio/adapters/[runID]/` directory  
- IPC path is hardcoded directly into adapter code using template replacement
- Each test run gets its own unique adapter instance
- No caching, fresh adapter for each run

**Rationale**:
1. **Monorepo Compatibility**: Environment variables don't propagate reliably when test runners spawn child processes in monorepos (e.g., pnpm workspaces). Each package's tests may run in a separate process that doesn't inherit the environment.

2. **100% Reliability**: Hardcoded paths eliminate all environment variable discovery issues, fallback mechanisms, and error handling complexity.

3. **Process Isolation**: Each run gets its own adapter with the correct IPC path baked in, preventing any cross-run contamination.

4. **Simplicity**: No need for complex environment variable propagation logic or fallback mechanisms.

**Implementation Details**:
- Template markers in adapter source: `/*__IPC_PATH__*/"WILL_BE_REPLACED"/*__IPC_PATH__*/` for JavaScript
- Template markers in adapter source: `#__IPC_PATH__#"WILL_BE_REPLACED"#__IPC_PATH__#` for Python
- `strconv.Quote()` used for proper path escaping (handles quotes, backslashes, Unicode)
- Adapters stored in `.3pio/adapters/[runID]/` for easy debugging

**Trade-offs**:
- Slightly more disk usage (one adapter per run instead of shared)
- Cannot reuse adapters across runs (acceptable for dev tool)
- Must regenerate adapter even if nothing changed (negligible performance impact)

**Testing**:
- Unit tests verify path injection with special characters, Unicode, Windows paths
- Integration tests confirm monorepo scenarios work correctly
- Each package in a monorepo uses the same adapter with same IPC path

## IPC Communication via File System

**Decision**: Use file-based IPC instead of sockets or other mechanisms.

**Rationale**:
1. **Simplicity**: File I/O is universally supported and easy to debug
2. **Persistence**: IPC events are automatically persisted for debugging
3. **Compatibility**: Works across all platforms without special permissions
4. **Visibility**: Easy to inspect communication by reading the JSONL file

## Memorable Directory Names

**Decision**: Append Star Wars character names to test run directories.

**Example**: `2025-09-09T111224921Z-revolutionary-chewbacca`

**Rationale**:
1. **Human Recognition**: Easier to remember and reference recent runs
2. **Unique Identification**: Combines timestamp precision with memorable suffix
3. **Debugging Aid**: Developers can quickly identify and discuss specific test runs
4. **Cultural Relevance**: Star Wars references resonate with developer community

## Incremental Log File Writing

**Decision**: Write individual test log files incrementally as output arrives, instead of writing all logs at the end of test execution.

**Implementation Date**: 2025-01-09

**Previous Behavior**:
- All test output was collected in memory during execution
- Individual log files were written only during finalization
- If process was interrupted, no individual log files would exist

**New Behavior**:
- Log files are created immediately when test files are registered
- Output is written incrementally with debouncing for performance
- File handles remain open during execution for efficiency
- Partial results are available even if test run is interrupted

**Rationale**:
1. **Resilience**: Partial results are preserved if test run is interrupted (Ctrl+C, process crash)
2. **Real-time Monitoring**: Users can tail log files during execution to monitor progress
3. **Memory Efficiency**: Output doesn't need to be kept entirely in memory
4. **Better UX for Long Tests**: Immediate feedback available for long-running test suites

**Implementation Details**:
- File handles stored in Map<string, fs.FileHandle>
- Per-file buffers with Map<string, string[]>
- Per-file debounced write functions for batched I/O
- Automatic recovery if logs directory is deleted mid-run
- Test case boundaries marked in output for better organization

**Trade-offs**:
- **Pros**:
  - Resilient to interruptions
  - Real-time monitoring capability
  - Lower memory usage
  - Better debugging experience
- **Cons**:
  - More file handles open during execution
  - Slightly more complex error handling
  - Small performance overhead from I/O operations (mitigated by debouncing)

**Future Considerations**:
- File handle pooling for very large test suites (>100 files) to avoid OS limits
- Configurable debounce timings via environment variables
- Compression for very large log files

## Unified Report Generation Architecture (2025-09-12)

**Decision**: Replace incremental buffering with complete file regeneration using a single unified report generation function.

**Previous Behavior**:
- Test files started with legacy format headers ("## Test case results", "--- Test: ..." boundaries)
- Different phases used different report generation logic
- Incremental buffering with file handles and periodic flushes
- Inconsistent formatting between file start and completion phases

**New Behavior**:
- Single `generateIndividualFileReport()` function used for all report generation
- `updateIndividualFileReport()` regenerates entire file content when test state changes
- Consistent clean format across all phases: file start, test execution, and completion
- No incremental buffering - complete file rewrite on each state change

**Report Format**:
```yaml
---
test_file: /path/to/test.ts
created: 2025-09-12T10:10:49.389Z
updated: 2025-09-12T10:10:56.937Z
status: RUNNING|COMPLETED|ERRORED
---

# Test results for `test.ts`

✓ passing test (2ms)
✕ failing test (5ms)
```error block```
```

**Rationale**:
1. **Consistency**: Single function ensures identical formatting across all test phases
2. **Simplicity**: Eliminates complex buffering logic and file handle management
3. **Reliability**: No risk of corrupted reports from partial writes or buffer issues
4. **AI-Friendly**: Clean, predictable format optimized for context efficiency
5. **Maintainability**: Single source of truth for report format reduces code complexity

**Implementation Details**:
- Removed `appendToFileBuffer()`, `flushFileBuffer()`, and `scheduleFileWrite()` functions
- Eliminated `fileHandles` and `fileBuffers` maps
- `handleTestCase()` now calls `updateIndividualFileReport()` for immediate regeneration
- `Finalize()` regenerates all reports one final time for consistency

**Performance Considerations**:
- File regeneration is fast for typical test report sizes (<1MB)
- Trade-off: Slightly more I/O for significantly better consistency and reliability
- Most individual test files have <100 test cases, making regeneration negligible

**Trade-offs**:
- **Pros**:
  - 100% consistent formatting across all phases
  - Eliminates buffer-related complexity and bugs
  - Simplifies debugging and maintenance
  - AI agents see predictable format always
- **Cons**:
  - More file I/O operations during test execution
  - Cannot preserve partial writes in buffer during crashes (acceptable trade-off)