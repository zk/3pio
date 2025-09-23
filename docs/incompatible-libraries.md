# Incompatible Libraries

This document lists libraries that have been tested with 3pio but are currently incompatible due to unsupported test runners or other technical limitations.

## Unsupported Test Runners

These libraries use test runners that 3pio does not currently support:

### JavaScript/TypeScript

#### typescript
- **Repository**: https://github.com/microsoft/TypeScript
- **Date Tested**: 2025-09-21
- **Test Runner**: Custom "hereby" task runner with parallel Mocha workers
- **Issue**: Uses parallel test execution architecture that 3pio cannot intercept
- **Test Details**:
  - Total tests: 98,877 passing, 3 failing
  - Test files: 19,584
  - Parallel workers: 9
  - Duration: ~5 minutes
- **Architecture Problem**:
  ```
  npm test → hereby runtests-parallel → 9 parallel workers → each runs Mocha independently
  ```
  3pio's reporter injection only affects the main process, not the child workers.
- **Partial Workaround**: Individual test files can be run directly with `npx mocha built/local/run.js --grep "pattern"` and 3pio tracks them correctly (verified with 210 scanner tests)
- **Notes**: TypeScript's test architecture is optimized for speed through parallelization, making it fundamentally incompatible with 3pio's current single-process adapter model

#### lodash
- **Repository**: https://github.com/lodash/lodash
- **Date Tested**: 2025-09-20
- **Test Runner**: Custom QUnit-based test runner
- **Issue**: Uses a custom test implementation built on QUnit, which is not in 3pio's supported test runner list
- **Test Details**:
  - Main tests: 6794 tests
  - FP tests: 327 tests
  - Total: 7121 tests
- **Notes**: The test suite runs successfully standalone but cannot be instrumented by 3pio due to the custom test runner implementation

## Future Support

Libraries listed here may become compatible if:
1. 3pio adds support for their test runner
2. The library migrates to a supported test runner
3. A custom adapter is developed for the specific test framework

## Contributing

If you've tested a library with 3pio and found it incompatible, please add it to this list with:
- Repository URL
- Date tested
- Test runner used
- Specific incompatibility issue
- Any relevant test statistics or notes