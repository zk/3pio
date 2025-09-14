# 3pio output format specs

## Console Output

```markdown
Greetings! I will now execute the test command:
`go test ./...`

Full report: .3pio/runs/20250913T203231-batty-neelix/test-run.md

Beginning test execution now...

RUNNING  github.com/zk/3pio/cmd/3pio
RUNNING  github.com/zk/3pio/internal/adapters
RUNNING  github.com/zk/3pio/internal/ipc
RUNNING  github.com/zk/3pio/internal/logger
NO_TESTS github.com/zk/3pio/internal/logger
FAIL     github.com/zk/3pio/tests/integration_go (29.57s)
  x TestFailureDisplayFormat
  x shows_report_path
  + 12 more
  see .3pio/runs/20250912T113945-nutty-poe/reports/go_test_examples_test_js/index.md

PASS     github.com/zk/3pio/internal/runner (10.77s)
PASS     github.com/zk/3pio/internal/orchestrator (4.38s)
PASS     github.com/zk/3pio/cmd/3pio (8.08s)
PASS     github.com/zk/3pio/internal/runner/definitions (6.42s)
Test failures! We're doomed!
Results:     7 passed, 1 failed, 1 skipped, 9 total
Total time:  42.411s
```

### Results Summary Format

The results line dynamically adjusts based on test outcomes:

- **All passed, no skips**: `Results:     N passed, N total`
- **With failures**: `Results:     N passed, M failed, X total`
- **With skipped tests**: `Results:     N passed, M failed, S skipped, X total`

Where:
- `passed`: Number of test groups that passed all tests
- `failed`: Number of test groups with at least one failure
- `skipped`: Number of test groups that were skipped or had no tests (includes Go packages with `NO_TESTS`)
- `total`: Sum of passed + failed + skipped

### Success Messages

The console displays different messages based on outcomes:
- **All tests passed** (no failures, no skips): "Splendid! All tests passed successfully"
- **Tests with skips** (no failures, some skips): "Tests completed with some skipped"
- **Only skipped tests**: "All tests were skipped"
- **Test failures**: "Test failures! [random exclamation]"

**Note on Group Names in Console Output:**
The console output displays raw group names as provided by each test runner:
- **Jest/Vitest**: Full file paths as discovered (e.g., `/path/to/project/math.test.js`)
- **Go**: Package import paths (e.g., `github.com/zk/3pio/cmd/3pio`)
- **pytest**: File paths relative to test root (e.g., `test_math.py`, `tests/unit/test_string.py`)

The group names are displayed exactly as provided by the test runner adapters without normalization.

Notes:

- Outputs the full command passed to

## test-run.md

```markdown
---
run_id: 20250912T001847-funky-mccoy
run_path: /Users/zk/code/3pio/.3pio/runs/20250912T001212-snappy-cyan
detected_runner: vitest
modified_command: `npm test --reporter path/to/vitest/reporter`
created: 2025-02-15T12:30:00.000Z
updated: 2025-02-15T12:31:11.000Z
status: PENDING | RUNNING | COMPLETED | ERRORED
---

# 3pio Test Run

- Test command: `npx vitest run`
- Run stdout/stderr: `./output.log`

## Summary

- Total test cases: 2
- Test cases completed: 2
- Tests cases passed: 1
- Test cases failed: 1
- Test cases skipped: 0
- Total duration: 203.56s

## Test group results

| Status | Name | Tests | Duration | Report |
|--------|------|-------|----------|--------|
| PASS | math.test.js | 5 passed | 12.3s | ./reports/math_test_js/index.md |
| FAIL | string.test.js | 3 passed, 1 failed | 2.3s | ./reports/string_test_js/index.md |
| SKIP | tests/unit/utilities.test.js | 0 tests | 0.53s | ./reports/tests_unit_utilities_test_js/index.md |
```

Notes:

- detected_runner ex: `vitest`, `jest`, `go test`, `pytest`
- modified_command: The modified command we create the runner process with. Helps debug issues with positional arguments in the field.


## Individual Test Files

```markdown
---
test_file: /Users/zk/code/3pio/tests/fixtures/basic-vitest/string.test.js
created: 2025-02-15T12:30:00.000Z
updated: 2025-02-15T12:31:11.000Z
status: PENDING | RUNNING | COMPLETED | ERRORED
---

# Test results for `string.test.js`

## Test case results

- ✓ should concatenate strings (1ms)
- ✓ should concatenate strings (1ms)
- ✕ should fail this test (3ms)
```
expected 'foo' to be 'bar' // Object.is equality
```
○ should skip this test
✓ should pass through other methods unchanged (0ms)
✓ should handle proxy creation errors gracefully (0ms)
✓ should wrap createRunAsync to return run proxy (0ms)
✓ should wrap createRun to return run proxy (0ms)
✓ should inject tracing context into run start method (0ms)
✓ should preserve user-provided tracingContext in run start (0ms)
✓ should pass through other run methods unchanged (0ms)
✓ should work in nested workflow step scenario (0ms)
✓ should work with workflow calling another workflow (0ms)
✓ should preserve type safety (0ms)
✓ should handle mixed wrapped and unwrapped usage (0ms)

## Error

Error: Something went wrong in thirdFunction!
    at thirdFunction (file:///path/to/your/script.js:2:9)
    at secondFunction (file:///path/to/your/script.js:6:3)
    at firstFunction (file:///path/to/your/script.js:10:3)
    at file:///path/to/your/script.js:14:3


## stdout/stderr
```

Notes:
- Error should only be present if an error was encountered while running this test. Test failures alone do not constitute an error
- stdout/stderr should only be present if stdout/stderr was collected and is not empty
- This file should be updated as the test file transitions state
  - YAML frontmatter: updated and status
