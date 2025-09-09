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

# Run with specific test runners
3pio jest
3pio vitest
3pio npx vitest run
```

## Supported Test Runners

### Jest
- All versions supported

### Vitest
- **Requires Vitest 3.0+** - The reporter uses Vitest 3.x lifecycle hooks (`onFinished`)
- Older versions of Vitest are not supported due to API changes

## Output

3pio generates structured reports in `.3pio/runs/[timestamp]/`:
- `test-run.md` - Main report with test summary and results
- `output.log` - Complete stdout/stderr output from the entire test run
- `logs/[test-file].log - stdout/stderr output for specific test file run by test case

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
