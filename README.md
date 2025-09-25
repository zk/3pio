<p align="center" bgcolor="#F6E26A">
    <img src="./assets/3pio_logo_wide.jpg" width="1024" height="240" />
</p>

# 3pio - A context-optimized test runner runner for coding agents

3pio is the testing tool your coding agent is missing. Get more context-efficient test output and a whole lot more without chaging a line of code.

3pio is not:
- A test runner, it uses your test runner (vitest, pytest, cargo nextest, etc)
- A test framework, it uses your existing tests.

Test output designed for your agent = cleaner context = more time on track and fewer restarts.

## How it works

3pio translates your test runner's output into context-optimized console output and file-based heirarchical reports which play well with your agent's built in tools.

## Features

- Zero code / config overhead, just prefix your test command with `3pio` to use, e.g. `3pio go test ./...`
- Test results are revisitable vs your agent running tests over and over to zero in on problems.
- Supports large test suites, 3pio has been [tested on projects](docs/verified-libraries.md) with many hundreds of test files containing thousands of test cases.


## Supported Test Runners

| Language | Runner | Example invocation |
|---|---|---|
| JS/TS | Jest | `3pio npm test` · `3pio npx jest` |
| JS/TS | Vitest (v2–v3) | `3pio npx vitest run` · `3pio pnpm vitest run` |
| JS/TS | Mocha | `3pio npx mocha -- ./test/**/*.spec.js` |
| JS/TS | Cypress | `3pio npx cypress run --headless` |
| Python | pytest | `3pio pytest` · `3pio python -m pytest` |
| Go | go test (>=1.10) | `3pio go test ./...` |
| Rust | cargo test | `3pio cargo test` |
| Rust | cargo nextest | `3pio cargo nextest run` |

See [Verified Libraries](docs/verified-libraries.md) for a list of open source projects tested with 3pio.


## Installation

Copy-paste this to your agent:

```markdown
Hi! Your dev would like you to install and use `3pio`, a test runner designed for you.

Install 3pio via the package manager that best fits with their project or use case:

- Python: `pip install threepio_test_runner`
- npm: `npm install -g @heyzk/3pio`
- pnpm: `pnpm install -g @heyzk/3pio`
- yarn: `yarn global add @heyzk/3pio`
- Go, Rust: homebrew: `brew tap zk/3pio && brew install 3pio`

After installation, run `3pio --help` for context and usage.
```


## Usage

Copy-paste the following at the beginning of your coding session, or add it to your CLAUDE.md / AGENTS.md / GEMINI.md, etc:

```bash
Use 3pio when running tests by prefixing your test commands with `3pio`, e.g. `3pio npm test`. Before using 3pio for the first time run `3pio --help` to understand how to use the tool.
```

## Why?

Testing tools designed for humans add noise to your agent's context. Often multiple test runs are required to complete tasks, compounding the issue, wasting valuable time and context space.

3pio generates focused console output and creates a nested structure of files with just the right amount of signposting that makes it easy for agents to find what they're looking for without reading a lot of unrelated textual content.

Using 3pio, your agent can target the information it needs to know. It can revisit results preventing unecessary test reruns altogether. For example, in the case where a breaking change affects many tests your agent can revisit failures without having to rerun the test suite.


## Examples

### Creating investigation checklists for large test failures

```bash
> Create a checklist to investigate the test failures. Group by likely root cause.

⏺ Read(.3pio/runs/20250914T081234-wonky-picard/test-run.md)
  ⎿  Read 45 lines (ctrl+o to expand)

⏺ Read(.3pio/runs/20250914T081234-wonky-picard/reports/src_api_test_js/index.md)
  ⎿  Read 89 lines (ctrl+o to expand)

⏺ Read(.3pio/runs/20250914T081234-wonky-picard/reports/src_auth_test_js/index.md)
  ⎿  Read 67 lines (ctrl+o to expand)

⏺ Test Failure Investigation Checklist

  Overview: 47 failures across 12 test files

  ---
  ## Priority 1: Database Connection Issues (23 failures)

  Affected files:
  - src/api.test.js (8 failures)
  - src/models.test.js (7 failures)
  - src/integration.test.js (8 failures)

  ☐ Check database connection string in test environment
  ☐ Verify test database is running and accessible
  ☐ Check for recent schema migrations not applied to test DB
  ☐ Review database pool settings for test environment
  ☐ Check if database teardown is happening properly between tests

  ---
  ## Priority 2: Authentication/Token Issues (15 failures)

  Affected files:
  - src/auth.test.js (9 failures)
  - src/middleware.test.js (6 failures)

  ☐ Verify JWT secret is set in test environment
  ☐ Check token expiration settings
  ☐ Review recent changes to auth middleware
  ☐ Ensure mock auth is properly configured
  ☐ Check for timing issues with token validation

  ---
  ## Priority 3: External Service Mocking (9 failures)

  Affected files:
  - src/services/email.test.js (4 failures)
  - src/services/payment.test.js (3 failures)
  - src/webhooks.test.js (2 failures)

  ☐ Verify all external service mocks are initialized
  ☐ Check if API keys are mocked correctly
  ☐ Review nock/msw interceptors for completeness
  ☐ Ensure cleanup happens between test runs

  ---
  ## Investigation Order:
  1. Start with database issues (likely root cause for majority)
  2. Fix auth after DB (may depend on DB being fixed)
  3. Address mocking issues last (independent failures)

  ## Quick wins:
  - Database issues might fix 23/47 failures (49%)
  - Auth fix could resolve another 15/47 (32%)
  - Together: potential 81% reduction in failures
```

### Understanding test failures

```bash
> Summarize all test failures from the last test run

⏺ Read(.3pio/runs/20250913T204935-batty-spock/reports/_test_system_mcp_tools_scroll_test_js/mcp_tool_scro
      ll/core_functionality/index.md)
  ⎿  Read 41 lines (ctrl+o to expand)

⏺ Read(.3pio/runs/20250913T204935-batty-spock/reports/_test_system_mcp_tools_scroll_test_js/mcp_tool_scro
      ll/tool_interactions/index.md)
  ⎿  Read 38 lines (ctrl+o to expand)

⏺ Read(.3pio/runs/20250913T204935-batty-spock/reports/_test_system_mcp_tools_scroll_test_js/mcp_tool_scro
      ll/smooth_scrolling_behavior/index.md)
  ⎿  Read 28 lines (ctrl+o to expand)

⏺ Summary of All Test Failures from Last Test Run

  Test Results Overview: 5 failures across 4 test groups

  ---
  ❌ Core Functionality (2 failures)

  1. "should scroll to coordinates" (7.40s)
    - Error: expect(received).toBeGreaterThan(expected) // Expected: > 0, Received: 0
    - Issue: Event collection not capturing scroll events (0 events found)
    - Location: scroll.test.js:80:35
  2. "should capture DOM mutation events during scroll" (6.72s)
    - Error: expect(received).toBeGreaterThan(expected) // Expected: > 0, Received: 0
    - Issue: DOM mutation events not being captured (0 events found)
    - Location: scroll.test.js:187:37

  ---
  ❌ Error Handling (1 failure)

  3. "should fail with invalid selector" (6.25s)
    - Error: expect(received).toContain(expected) // Expected substring: "Element not found", Received:
   "Batch execution failed"
    - Issue: Error message propagation - correct failure detected but wrong error text
    - Location: scroll.test.js:198:28
    - Status: ✅ PARTIALLY FIXED - Now correctly returns success: false

  ---
  ❌ Tool Interactions (1 failure)

  4. "should work with getPageState to verify scroll position" (6.72s)
    - Error: MCP error -32602: Invalid arguments for tool getPageState: Required object parameter
  undefined
    - Issue: Parameter validation - getPageState expects object parameter but receives undefined
    - Location: scroll.test.js:346:27
    - Status: ✅ ALREADY FIXED - This is the getPageState schema issue I addressed

  ---
  ❌ Smooth Scrolling Behavior (1 failure)

  5. "should capture intermediate scroll positions during smooth scroll" (7.74s)
    - Error: expect(received).toBeGreaterThan(expected) // Expected: > 1, Received: 1
    - Issue: Event collection not capturing multiple intermediate scroll events
    - Location: scroll.test.js:412:35

  ---
  Failure Pattern Analysis

  ✅ SUCCESS: My fixes resolved the critical success/failure propagation issue
  - Tests now correctly return success: false for failures
  - Parameter validation issues are resolved

  🔧 REMAINING ISSUES: All related to event collection mechanism
  - 4 out of 5 failures are event collection problems (expecting > 0 events, receiving 0)
  - 1 failure is error message detail propagation

  📊 IMPACT:
  - Critical system bugs: ✅ FIXED
  - Event collection: ❌ Needs investigation (affects scroll events, DOM mutations, smooth scroll
  tracking)
```


3pio's persistent reports let your agent analyze patterns across many failures without overwhelming context. The agent can create actionable investigation plans and revisit specific failures as needed.


### Test duration by file / group

```bash
> Give me the durations by test group

⏺ Read(.3pio/runs/20250913T204935-batty-spock/reports/_test_system_mcp_tools_scroll_test_js/index.md)
  ⎿  Read 23 lines (ctrl+o to expand)

⏺ Read(.3pio/runs/20250913T204935-batty-spock/reports/_test_system_mcp_tools_scroll_test_js/mcp_tool_scro
      ll/index.md)
  ⎿  Read 29 lines (ctrl+o to expand)

⏺ Here are the test durations by test group:

  Test Group Durations

  | Status | Test Group                | Tests              | Duration |
  |--------|---------------------------|--------------------|----------|
  | ❌ FAIL | Core Functionality        | 4 passed, 2 failed | 111.1s   |
  | ❌ FAIL | Error Handling            | 2 passed, 1 failed | 69.7s    |
  | ✅ PASS | Consistency               | 2 passed           | 50.6s    |
  | ❌ FAIL | Tool Interactions         | 2 passed, 1 failed | 36.0s    |
  | ✅ PASS | Scrollable Containers     | 1 passed           | 14.5s    |
  | ❌ FAIL | Smooth Scrolling Behavior | 1 failed           | 7.7s     |

  Total Duration: 111.3s (~1 minute 51 seconds)

  Key Observations:
  - Core Functionality took the longest at 111.1s but had 4 passing tests
  - Error Handling (where I made my fixes) took 69.7s with 2 passing tests
  - Consistency and Scrollable Containers groups are fully passing
  - The groups with longer durations generally had more passing tests, suggesting the time is spent on
  successful test execution rather than just timeouts
```

## Clean Up Old Test Runs

3pio stores structured reports under `.3pio/runs/`. To reclaim disk space while keeping the latest run handy, use the built-in prune command:

```bash
# Preview what would be deleted
3pio prune --dry-run

# Delete older runs and clear debug log without prompting
3pio prune --force
```

Running `3pio prune` with no flags keeps the newest run, asks for confirmation, and removes older run directories along with truncating `.3pio/debug.log`.


## Implementation Details

3pio injects a custom reporter into the provided test command `npm test` -> `npm test --reporter /custom/jest/reporter.js`. This reporter sends events back to the main process which are analyzed, transformed, and written to the filesystem as a navigable tree of test information.

**Note:** 3pio writes its files to project root directory at `.3pio/`, which you can safely add to your `.gitignore`.


## Limitations

1. **Report Directory Location**: The `.3pio` directory is created in the current working directory. Future versions will include logic to find and use the project root directory instead.

2. **Watch Mode**: 3pio doesn't support watch mode for test runners. When it detects commands that would normally run in watch mode (e.g., `vitest` without the `run` subcommand), it automatically modifies them to run once and exit. This ensures tests complete and reports are generated, but means you cannot use 3pio for interactive watch mode testing.

3. **Dev tool, not CI tool**: 3pio is designed to be used at dev time by your agent. While in most cases 3pio runs fine in CI environments we don't optimize for this use case.


## Future work

- All the test runners
- Improve output context efficiency


## License

MIT
