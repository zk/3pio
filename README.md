# 3pio

An AI-first test runner adapter that acts as a "protocol droid" for test frameworks. 3pio translates traditional test runner output (Jest, Vitest) into a format optimized for AI agents - providing persistent, structured, file-based records that are context-efficient and searchable.

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