# 3pio Test Runner Specification
*Version 1.0 - Sept 2025*

## Overview

3pio is a context-friendly test runner that translates traditional test runner output into a format optimized for coding agents. This specification defines the behavioral requirements, constraints, and design principles that govern 3pio's operation.

## Design Philosophy

3pio follows these guiding principles in all design decisions:

- **Transparent Wrapping**: Users shouldn't notice 3pio unless they look for its output
- **Context-Efficient Output**: Every byte of output should help coding agents understand test state
- **Continuous State Synchronization**: Never lose test data, even during catastrophic failures

## Core Principles

### Principle 1: Transparent Wrapping

**Rule**: 3pio MUST operate as a transparent wrapper that preserves the exact behavior of the underlying test runner.

**Rationale**: Users should be able to prefix any test command with `3pio` without breaking existing workflows, CI/CD pipelines, or test behavior.

**Requirements**:
- MUST preserve exit codes from the wrapped test runner
- MUST NOT alter test execution order or parallelization
- MUST NOT modify test timeouts or retry behavior
- MUST pass through all command-line arguments unchanged
- MAY add command line arguments that do not violate any of the above requirements

**Examples**:
- ✓ Compliant: `3pio npm test` returns exit code 1 when `npm test` would return 1
- ✗ Violation: 3pio forces sequential test execution when runner uses parallel mode

### Principle 2: Context-Efficient Output

**Rule**: All 3pio output MUST be optimized for minimal context consumption while preserving essential information.

**Rationale**: Coding agents have limited context windows. Verbose output, deeply nested directories, and redundant information reduce the agent's ability to analyze test results effectively. Every byte should provide value.

**Requirements**:
- MUST use concise, meaningful identifiers
- MUST avoid redundant information across files
- MUST flatten directory structures where possible
- MUST summarize repetitive patterns
- MUST prioritize actionable information over diagnostic noise

**Console Output**:
- Show only test counts and failures summary by default
- Use single-line progress indicators
- Combine related errors into grouped displays
- Omit timestamps if not explicitly requested

**Test Reports**:
- Use YAML frontmatter for metadata (more compact than JSON)
- Combine test results by file, not individual test files
- Omit passing test details unless failures reference them
- Use relative paths from project root, not absolute paths

**Directory Structure**:
```
Good:
.3pio/runs/[timestamp]-[name]/
  test-run.md          # Combined summary
  logs/[test-file].log # Flat structure

Bad:
.3pio/runs/[timestamp]-[name]/
  reports/by-suite/by-file/by-test/
    individual-test-result.md
```

**Debug Output**:
- Use log levels to control verbosity
- Batch similar messages with counts
- Include only changed state, not full state dumps
- Compress repetitive stack traces

**Examples**:
- ✓ Compliant: `PASS: 47, FAIL: 3, SKIP: 2` instead of listing all 47 passing tests
- ✓ Compliant: `src/utils/auth.test.js` instead of `/home/user/projects/myapp/src/utils/auth.test.js`
- ✗ Violation: Creating separate directories for each test suite
- ✗ Violation: Including millisecond-precision timestamps for every log line
- ✗ Violation: Repeating full file paths in every test entry

**Compression Strategies**:
- Group consecutive passing tests: "Tests 1-15: PASS"
- Collapse similar error messages: "5 tests failed with: TypeError: Cannot read property 'x'"
- Use memorable names over UUIDs: "brave-panda" vs "a7f3d2e8-9b5c-4a2d-8e1f-3c5b7d9e2f4a"
- Deduplicate stack traces: Show once, reference by ID

### Principle 3: Continuous State Synchronization

**Rule**: 3pio MUST continuously synchronize test execution state to persistent storage throughout the run.

**Rationale**: Testing state is valuable data that must survive process failures, user interruptions, and system issues. Real-time synchronization ensures users never lose visibility into what tests have run and their outcomes.

**Requirements**:
- Synchronize state changes within a reasonable timeframe
- Maintain atomic writes (no partial/corrupt states)
- Support concurrent readers during writes
- Preserve chronological order of events

**Examples**:
- ✓ Compliant: Test results appear in reports within seconds of completion
- ✓ Compliant: Reports readable while tests still running
- ✗ Violation: "Results will be available when complete"
- ✗ Violation: Race conditions causing out-of-order results

### Principle 4: Fail-Fast on Constraint Violations

**Rule**: 3pio MUST immediately fail with a clear error message when it cannot meet its core behavioral requirements.

**Rationale**: Silent degradation or attempting to continue when core principles cannot be upheld leads to confusing test results, wasted debugging time, and loss of user trust. It is better to fail immediately with actionable information than to produce unreliable results.

**Requirements**:
- MUST exit with non-zero code when core principles cannot be met
- MUST output clear error message to stderr explaining the failure
- MUST NOT attempt workarounds that violate core principles
- MUST fail before test execution begins if pre-conditions aren't met

**Examples**:
- ✓ Compliant: Exit immediately if adapter cannot be injected and runner cannot be identified
- ✓ Compliant: Fail if IPC directory cannot be created due to permissions
- ✗ Violation: Continuing without report generation when filesystem is read-only
- ✗ Violation: Silently falling back to console output when IPC fails

**Fail-Fast Triggers**:
- Cannot create `.3pio` directory (filesystem permissions)
- Cannot inject adapter and runner detection fails
- IPC path exceeds system limits
- Test runner binary not found or not executable
- Mutually incompatible flags detected (e.g., requiring both silent operation and console output)

## Behavioral Boundaries

### 3pio Cannot:

1. **Modify test behavior**
   - Change test timeouts
   - Alter assertion behavior
   - Inject global variables
   - Modify test execution order

2. **Require configuration**
   - Demand config files
   - Require package.json modifications
   - Need preset selections for basic operation

3. **Interfere with test output**
   - Break test runner reporters
   - Corrupt test output streams
   - Modify test results

### 3pio Must Always:

1. **Preserve runner semantics**
   - Exit with same code as underlying runner
   - Support all runner CLI arguments
   - Handle runner-specific protocols

2. **Generate reports atomically**
   - Create unique run directories (timestamp + name)
   - Never overwrite existing reports
   - Include all test output in logs

3. **Fail safely**
   - Continue test execution if adapter fails
   - Log errors only to debug.log
   - Preserve partial results
   - Never corrupt user's test environment

## Conflict Resolution

### When adapter injection conflicts with runner operation:
1. Attempt injection with automatic detection
2. If detection fails, try dry run to discover test files
3. If injection causes errors, disable adapter and continue
4. Log all decisions to debug.log with timestamps

### When output capture conflicts with test assertions:
1. Prioritize test correctness over capture completeness
2. Allow test output to pass through unmodified
3. Capture output at process level, not test level
4. If capture fails, continue test execution normally

### When multiple test runners are detected:
1. Check explicit command (jest, vitest, pytest)
2. Check package.json scripts if using npm/yarn/pnpm
3. Fail with clear error listing detected runners
4. Never guess or auto-select runners

## State Management

### Test Report States

Each test file progresses through states:

```
DISCOVERED → STARTED → COMPLETED
           ↘         ↗
             FAILED
```

**State Transitions**:
- DISCOVERED: File found via dry run or test execution
- STARTED: First test in file begins execution
- COMPLETED: All tests in file finished (pass/fail/skip)
- FAILED: Setup error or file-level failure

**Requirements**:
- MUST track state per file, not per test
- MUST preserve last known state on interruption
- MUST include state in report frontmatter

### Report Update Triggers

Reports SHOULD be created and/or updated when:
1. New test file discovered
2. Test file transitions state
3. Test case completes
4. Test group completes
5. Run finishes or is interrupted

Update timing should align with the reasonable timeframe for state synchronization.

## Performance Constraints

- Adapter overhead: Minimal impact on test execution time
- IPC latency: Within reasonable bounds for real-time updates
- Report generation: Fast enough for continuous synchronization
- Memory overhead: Reasonable for adapter operation
- File I/O: May batch writes for efficiency

## Version History

- v1.0 (December 2024): Initial specification
- Core principles established
- Behavioral boundaries defined
- Runner requirements documented

---

*This is a living document. Updates require testing against all supported runners and review of existing adapter implementations.*
