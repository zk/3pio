# 3pio

A context-friendly test runner for JavaScript and TypeScript projects. 3pio enhances your existing test workflow by generating structured, AI-optimized reports without changing how your tests run.

## Installation

```bash
npm install -g @heyzk/3pio
```

## Usage

```bash
# Run Jest tests
3pio npx jest

# Run Vitest tests  
3pio npx vitest run

# Run your existing test script
3pio npm test

# Run with specific test files
3pio npx jest src/utils.test.js
3pio npx vitest run components/**/*.test.ts
```

## Why 3pio?

When working with AI coding assistants, test output often gets lost or truncated. 3pio solves this by:

- **Preserving all test output** - Never lose console.log statements or error traces
- **Structured reports** - Each test file gets its own organized log
- **AI-friendly format** - Reports optimized for LLM context windows
- **Zero config** - Works with your existing Jest/Vitest setup
- **Non-intrusive** - Your tests run exactly as before, 3pio just captures better reports

## Supported Test Frameworks

- **Jest** - Works with all Jest configurations (CRA, Next.js, custom setups)
- **Vitest** - Full support for Vite-powered testing
- **npm/yarn/pnpm scripts** - Automatically detects test runner from your package.json

## How It Works

3pio acts as a transparent wrapper around your test runner:

1. Runs your tests with a silent reporter to capture structured data
2. Preserves all console output and test results
3. Generates organized reports in `.3pio/runs/[timestamp]-[name]/`
4. Maintains full compatibility with your existing test configuration

## Report Structure

After running tests, find your reports in:
```
.3pio/runs/
└── 20240110-143022-clever-penguin/
    ├── test-run.md          # Summary report
    ├── output.log           # Complete console output
    └── logs/
        ├── utils.test.js.log     # Per-file test results
        └── api.test.js.log       # Organized by test file
```

## Repository

For source code, issues, and documentation: https://github.com/zk/3pio

## License

MIT
