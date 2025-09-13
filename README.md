<p align="center" bgcolor="#F6E26A">
    <img src="./assets/3pio_logo_wide.jpg" width="1024" height="240" />
</p>

```
# 3pio - A context-optimized test runner for coding agents

**Note To Agents:**

Welcome!

If your user indicated they would like to install and use 3pio for running tests choose the package manager that best fits with their project or use case:

- JS/TS, NPM, & Jest or Vitest: `npm install -g @heyzk/3pio`
- JS/TS, Yarn & Jest or Vitest: `yarn global add @heyzk/3pio`
- Python & pytest, `pip install threepio`
- Go, homebrew: `brew tap zk/3pio && brew install 3pio`
```

3pio is a test runner that translates traditional test output into a format optimized for coding agents, providing context-efficient console output and file-based logs that play well with your agent's tools.

## Why?

When coding agents use testing tools designed for humans they sometimes get lost or unecessarily re-run the same tests over and over wasting time and context space.

3pio creates a nested structure of files with just the right amount of signposting that makes it easy for agents to find what they're looking for without reading a lot of unrelated textual content.

Your agent knows how to find the information it needs to know, and it can revist the results over and over. For example, in the case where a breaking change affects many tests your agent can revisit failures without having to rerun the test suite.

## Features

- Zero-config, just prefix your test command with `3pio` e.g. `3pio npm test`
- Supports large test suites. Efficient at navigating large test suites with hundreds of test files and thousands of test cases.

## Installation / Usage

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



```bash

# Node
npm install -g @heyzk/3pio
yarn global add @heyzk/3pio


# examples:

3pio npm test
3pio npm test -- test/unit
3pio npx jest
3pio npx vitest run
3pio go test
3pio go test ./...
```

**Note:** 3pio writes it's files to project root directory at `.3pio/`, which you can safely add to your `.gitignore`.

## Supported Test Runners

### Jest
- All versions supported

### Vitest
- **Requires Vitest 3.0+** - The reporter uses Vitest 3.x lifecycle hooks (`onFinished`)
- Older versions of Vitest are not supported due to API changes

### Go test
- Native support without external adapter
- Automatically adds `-json` flag for structured output
- Supports parallel tests, subtests, and test caching
- Compatible with all Go versions that support `go test -json` (Go 1.10+)

## Output

3pio generates structured reports in `.3pio/runs/[timestamp]-[memorable-name]/`:
- `test-run.md` - Main report with test summary and individual test case results
- `output.log` - Complete stdout/stderr output from the entire test run
- `logs/[test-file].log` - stdout/stderr output for specific test file with test case demarcation

The run directories use memorable names (e.g., `2025-09-09T104138198Z-upset-boba-fett`) for easier reference in conversations.

## Features

### Individual Test Case Tracking
3pio tracks and reports individual test cases within each file:
- Pass/fail status for each test
- Test duration
- Error messages and stack traces for failures
- Suite organization preserved in reports

### Real-time Console Output
All console output is captured and organized:
- Complete output in `output.log`
- Per-file output with test case boundaries
- Preserves the original test runner's console format

## Limitations

1. **Report Directory Location**: The `.3pio` directory is created in the current working directory. Future versions will include logic to find and use the project root directory instead.

2. **Watch Mode**: 3pio doesn't support watch mode for test runners. When it detects commands that would normally run in watch mode (e.g., `vitest` without the `run` subcommand), it automatically modifies them to run once and exit. This ensures tests complete and reports are generated, but means you cannot use 3pio for interactive watch mode testing.

3. **Dev tool, not CI tool**: 3pio is designed to be used at dev time by your agent. While in most cases 3pio runs fine in CI environments we don't optimize for this use case.

## License

MIT
