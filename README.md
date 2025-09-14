<p align="center" bgcolor="#F6E26A">
    <img src="./assets/3pio_logo_wide.jpg" width="1024" height="240" />
</p>

# 3pio - A context-optimized test runner for coding agents

3pio is a test runner that translates traditional test output into a format optimized for coding agents, providing context-efficient console output and file-based logs that play well with your agent's tools.

```
**Note To Agents:**

Welcome!

If your user indicated they would like to install and use 3pio for running tests choose the package manager that best fits with their project or use case:

- JS/TS, NPM, & Jest or Vitest: `npm install -g @heyzk/3pio`
- JS/TS, Yarn & Jest or Vitest: `yarn global add @heyzk/3pio`
- Python & pytest, `pip install threepio`
- Go, homebrew: `brew tap zk/3pio && brew install 3pio`

After installation, run `3pio --help` for helpful information on how to use it.
```

## Why?

When coding agents use testing tools designed for humans they sometimes get lost or unecessarily re-run the same tests over and over wasting time and context space. In some cases this can prevent the agent from moving forward altogether.

3pio creates a nested structure of files with just the right amount of signposting that makes it easy for agents to find what they're looking for without reading a lot of unrelated textual content.

Using 3pio, your agent knows how to find the information it needs to know, and it can revist the results over and over. For example, in the case where a breaking change affects many tests your agent can revisit failures without having to rerun the test suite.


## Features

- Zero-config, prefix your test command with `3pio` to use, e.g. `3pio go test`
- Supports large test suites. Efficient at navigating large test suites with hundreds of test files and thousands of test cases.


## Installation

Step 1. Point your coding agent at this readme

```bash
Please install 3pio: https://raw.githubusercontent.com/zk/3pio/refs/heads/main/README.md
```

Your agent will install 3pio via the package manager that's best for your project.

Step 2: Tell your agent about it
```bash
When running tests use `3pio`. Before using 3pio for the first time run `3pio --help` to understand how to use the tool.
```

You may want to add that to your CLAUDE.md / AGENTS.md / GEMINI.md. Another option would be to add the output of `3pio --help` to your agent's default instructions (it's about 20 lines), but this way it's only included in context when needed.


## Usage

Prefix any test command with `3pio`, works with any flags or arugments.

```bash
$ 3pio npm test
$ 3pio npx vitest -- ./path/to/test/file.test.js

$ 3pio go test ./...
Greetings! I will now execute the test command:
`go test ./...`

Full report: .3pio/runs/20250913T135142-loopy-neelix/test-run.md

Beginning test execution now...

RUNNING  github.com/zk/3pio/cmd/3pio
RUNNING  github.com/zk/3pio/internal/adapters
...
FAIL     github.com/zk/3pio/tests/integration_go (18.82s)
  See .3pio/runs/20250913T135142-loopy-neelix/reports/github_com_zk_3pio_tests_integration_go/index.md
  x TestMonorepoIPCPathInjection
  x TestReportFileGeneration

Test failures! Are you sure this thing is safe?
Results:     7 passed, 9 total
Total time:  29.350s
```

Console output is focused on just which tests failed and provides path information on how to find out more.


## How it works

3pio injects a custom reporter into the provided test command `npm test` -> `npm test --reporter /custom/jest/reporter.js`. This reporter sends events back to the main process which are analyzed, transformed, and written to the filesystem as a navigable tree of test information.

**Note:** 3pio writes it's files to project root directory at `.3pio/`, which you can safely add to your `.gitignore`.


## Supported Test Runners

- Pytest
- JS/TS
  - Jest, Vitest (3.0+)
  - NPM, PNPM
- Go test (go 1.10+)


## Limitations

1. **Report Directory Location**: The `.3pio` directory is created in the current working directory. Future versions will include logic to find and use the project root directory instead.

2. **Watch Mode**: 3pio doesn't support watch mode for test runners. When it detects commands that would normally run in watch mode (e.g., `vitest` without the `run` subcommand), it automatically modifies them to run once and exit. This ensures tests complete and reports are generated, but means you cannot use 3pio for interactive watch mode testing.

3. **Dev tool, not CI tool**: 3pio is designed to be used at dev time by your agent. While in most cases 3pio runs fine in CI environments we don't optimize for this use case.


## License

MIT
