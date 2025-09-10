<p align="center">
    <img src="./assets/3pio_logo.jpg" width="256" height="256" />
</p>

# 3pio - Your agent's context-concious test runner 

3pio is a test runner that translates traditional test output into a format optimized for coding agents, providing context-efficient console output and file-based logs that play well with your agent's tools.

## Installation / Usage

```bash
npm install -g @heyzk/3pio
3pio [your test command]

# or 

npx @heyzk/3pio [your test command]

# examples:

3pio npm test
3pio npm test -- test/unit
3pio npx jest
3pio npx vitest run
```

**Note:** 3pio writes it's files to project root directory at `.3pio/`, which you can safely add to your `.gitignore`.

## Supported Test Runners

### Jest
- All versions supported

### Vitest
- **Requires Vitest 3.0+** - The reporter uses Vitest 3.x lifecycle hooks (`onFinished`)
- Older versions of Vitest are not supported due to API changes

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

## License

MIT
