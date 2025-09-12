# 3pio output format specs

## test-run.md

```markdown
---
run_id: 20250912T001847-funky-mccoy
run_path: /Users/zk/code/3pio/.3pio/runs/20250912T001212-snappy-cyan
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

| Stat | Test | Report file |
| ---- | ---- | ----------- |
| PASS | math.test.js | ./reports/math.test.js.md |
| FAIL | string.test.js | ./reports/string.test.js.md |
| SKIP | tests/unit/utilities.test.js | ./reports/tests/unit/utilities.test.js.md |
```


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
