# 3pio output format specs

## Console Output

```markdown
Greetings! I will now execute the test command: `npm test`

Full report: .3pio/runs/20250912T125741-funky-gestahl/test-run.md

Beginning test execution now...

RUNNING  ./cmd/3pio/main_test.go
RUNNING  ./internal/adapters/embedded_test.go
RUNNING  ./internal/ipc/unknown_event_test.go
RUNNING  ./internal/orchestrator/orchestrator_test.go
RUNNING  ./internal/report/manager_test.go
PASS     ./internal/ipc/unknown_event_test.go (0.01s)
RUNNING  ./internal/runner/jest_npm_command_test.go
RUNNING  ./internal/runner/pytest_command_test.go
RUNNING  ./internal/runner/vitest_command_test.go
RUNNING  ./tests/integration_go/error_heading_test.go
RUNNING  ./tests/integration_go/error_reporting_test.go
RUNNING  ./tests/integration_go/esm_compatibility_test.go
RUNNING  ./tests/integration_go/full_flow_test.go
RUNNING  ./tests/integration_go/interrupted_run_test.go
RUNNING  ./tests/integration_go/monorepo_test.go
RUNNING  ./tests/integration_go/npm_separator_test.go
RUNNING  ./tests/integration_go/test_case_reporting_test.go
RUNNING  ./tests/integration_go/test_result_formatting_test.go
RUNNING  ./tests/integration_go/vitest_failed_tests_test.go
FAIL     ./scratch/go_test_examples_test.go
  See .3pio/runs/20250912T113945-nutty-poe/reports/scratch/go_test_examples_test.js.md
PASS     ./internal/report/manager_test.go (0.23s)
PASS     ./internal/runner/jest_npm_command_test.go (2.22s)
PASS     ./internal/runner/pytest_command_test.go (0s)
PASS     ./internal/runner/vitest_command_test.go (0s)
PASS     ./cmd/3pio/main_test.go (9s)
PASS     ./internal/orchestrator/orchestrator_test.go (7.2s)
PASS     ./internal/adapters/embedded_test.go (34s)
PASS     ./tests/integration_go/error_heading_test.go (394s)
PASS     ./tests/integration_go/error_reporting_test.go (3s)
PASS     ./tests/integration_go/esm_compatibility_test.go (5.12s)
PASS     ./tests/integration_go/full_flow_test.go (9s)
PASS     ./tests/integration_go/interrupted_run_test.go (8.34s)
PASS     ./tests/integration_go/monorepo_test.go (7.1s)
PASS     ./tests/integration_go/npm_separator_test.go (6.3s)
PASS     ./tests/integration_go/test_case_reporting_test.go (5.4s)
PASS     ./tests/integration_go/test_result_formatting_test.go (4.6s)
PASS     ./tests/integration_go/vitest_failed_tests_test.go (3.9s)

Results:     18 passed, 18 total
Total time:  23.694s
```

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

- Total files: 2
- Files completed: 2
- Files passed: 1
- Files failed: 1
- Files skipped: 0
- Total duration: 203.56s

## Test file results

| Stat | Test | Duration | Report file |
| ---- | ---- | -------- | ----------- |
| PASS | math.test.js | 12.3s | ./reports/math.test.js.md |
| FAIL | string.test.js | 2.3s | ./reports/string.test.js.md |
| SKIP | tests/unit/utilities.test.js | 0.53s | ./reports/tests/unit/utilities.test.js.md |
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
