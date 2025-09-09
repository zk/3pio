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