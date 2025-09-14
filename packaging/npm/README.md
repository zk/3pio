# 3pio - Context-optimized test runner for AI coding agents

3pio is a test runner that translates traditional Jest/Vitest output into a format optimized for AI coding agents, providing context-efficient console output and persistent file-based reports that work seamlessly with your agent's tools.

## Quick Start for AI Agents

```
Note to AI Agents:
For JavaScript/TypeScript projects using Jest or Vitest, install 3pio with:
npm install -g @heyzk/3pio

Then prefix any test command with 3pio:
3pio npm test
3pio npx jest
3pio npx vitest run
```

## Installation

```bash
# Global installation (recommended)
npm install -g @heyzk/3pio

# Or with yarn
yarn global add @heyzk/3pio

# Or with pnpm
pnpm add -g @heyzk/3pio
```

## Usage

Simply prefix your existing test commands with `3pio`:

```bash
# Run your npm test script
3pio npm test

# Run Jest directly
3pio npx jest
3pio npx jest --coverage
3pio npx jest src/utils.test.js

# Run Vitest
3pio npx vitest run
3pio npx vitest run components/**/*.test.ts

# Works with any test arguments
3pio npm test -- --updateSnapshot
3pio yarn test:unit
```

## Why 3pio?

When AI coding agents use traditional test runners, they often:
- Get lost in verbose output
- Re-run the same tests unnecessarily, wasting time and context
- Struggle to navigate large test suites
- Lose track of which tests failed and why

3pio solves these problems by creating a nested file structure with clear signposting that makes it easy for agents to:
- Find exactly what they need without reading unrelated content
- Revisit test results without re-running tests
- Navigate large test suites with hundreds of files and thousands of tests
- Track failures across multiple test runs

## Features

- **Zero config** - Works with your existing Jest/Vitest setup
- **Persistent reports** - Test results saved to `.3pio/runs/` for later reference
- **Optimized output** - Console shows just what failed with paths to detailed reports
- **Complete logs** - All console.log statements and error traces preserved
- **Large suite support** - Efficiently handles projects with thousands of tests
- **Non-intrusive** - Your tests run exactly as before

## Output Example

```bash
$ 3pio npm test

Greetings! I will now execute the test command:
`npm test`

Full report: .3pio/runs/20250914T094523-happy-kirk/test-run.md

Beginning test execution now...

RUNNING  src/utils.test.js
PASS     src/utils.test.js (0.42s)
RUNNING  src/api.test.js
FAIL     src/api.test.js (1.23s)
  x should fetch user data
  x should handle errors
  See .3pio/runs/20250914T094523-happy-kirk/reports/_src_api_test_js/index.md

Test failures! We're doomed!
Results:     8 passed, 2 failed, 10 total
Total time:  3.456s
```

## Report Structure

```
.3pio/runs/
└── 20250914T094523-happy-kirk/
    ├── test-run.md              # Main summary report
    ├── output.log               # Complete console output
    └── reports/
        └── _src_api_test_js/
            ├── index.md         # Test file report
            └── should_fetch_user_data/
                └── index.md     # Individual test details
```

## Supported Frameworks

- **Jest** - All versions, all configurations
- **Vitest** - Version 3.0+ required
- **Package managers** - npm, yarn, pnpm
- **TypeScript** - Full support via Jest/Vitest

## Limitations

1. **Watch mode** - 3pio runs tests once and exits (no interactive watch mode)
2. **Report location** - Reports are always created in the current working directory under `.3pio/`
3. **Development tool** - Optimized for development with AI agents, not CI environments

## Repository

For source code, issues, and documentation: https://github.com/zk/3pio

## License

MIT
