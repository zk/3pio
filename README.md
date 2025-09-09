# 3pio

Your agent's context-concious test reporter for Jest and Vitest.

3pio is an AI-first test runner that translates traditional test output into a format optimized for AI agents - providing persistent, structured, file-based records that are context-efficient and searchable.

## Installation

```bash
npm install @heyzk/3pio
```

## Usage

```bash
# Run with automatic test runner detection
3pio npm test
3pio npm run test

# Run with specific test runners
3pio jest
3pio vitest
3pio npx vitest run

# Run with specific test files
3pio jest tests/unit/*.test.js
3pio vitest tests/integration/
```

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

### Dynamic Test Discovery
3pio supports both static and dynamic test discovery:
- **Static Discovery**: When test files are explicitly provided or can be determined upfront
- **Dynamic Discovery**: Automatically discovers and tracks test files as they run (e.g., when using `npm run test`)

This means 3pio works seamlessly with any test configuration, including:
- Custom test scripts
- Dynamically generated tests
- Test runners that don't support listing files upfront

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

## Development

```bash
# Build
npm run build

# Test
npm test

# Development mode
npm run dev
```

## License

MIT
