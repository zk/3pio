# Integration Tests

This directory contains integration tests for 3pio that verify the complete test runner flow and output generation.

## Test Fixtures

Test fixtures are located in `tests/fixtures/` with the following naming convention:
- `{name}-jest` - Jest variant of the fixture
- `{name}-vitest` - Vitest variant of the fixture

Available fixtures:
- `basic-{jest,vitest}` - Standard test projects with passing and failing tests
- `empty-{jest,vitest}` - Edge case with empty test suites
- `long-names-{jest,vitest}` - Tests with very long names for formatting verification
- `npm-separator-{jest,vitest}` - For testing npm command with `--` separator

## Default Verifications

Each integration test must verify the following items unless the specific test requirements preclude them:

### 1. File Existence Checks
All tests must verify the existence of:
- `test-run.md` - Main test report file
- `output.log` - Complete stdout/stderr capture
- `logs/*.log` - Individual test file logs

### 2. Content Verification

#### test-run.md
Must contain:
- Header with "# 3pio Test Run"
- Timestamp section
- Summary section with file counts
- Individual file sections with test results
- Test case details with ✓/✕/○ symbols
- Links to log files

#### output.log
Must contain:
- Header with "# 3pio Test Output Log"
- Timestamp
- Command that was run
- Separator line "# ---"
- Captured console output from test run

#### logs/*.log files
Each must contain:
- Header with "# File: {filename}"
- Timestamp
- Description line
- Separator line "# ---"
- Test-specific console output (if any)

## Test Files

### test-case-reporting.test.ts
Comprehensive test of the test case reporting feature. Verifies:
- All default file checks
- Detailed markdown format in test-run.md
- Individual test case results with symbols
- Error messages for failed tests
- Duration formatting
- Edge cases (empty tests, long names)

### npm-separator.test.ts
Tests the npm command with `--` separator handling. Currently only verifies:
- Command format preservation
- Basic output presence
**TODO**: Add default file and content verifications

### full-flow.test.ts
Tests the complete CLI flow with mocked components. Currently verifies:
- Basic flow execution
- Error recovery
**TODO**: Add default file and content verifications

## Writing New Integration Tests

When creating new integration tests:

1. Use the helper functions for common operations:
```typescript
const getLatestRunDir = (projectPath: string): string => {
  const runsDir = path.join(projectPath, '.3pio', 'runs');
  const runDirs = fs.readdirSync(runsDir);
  const latestRun = runDirs.sort().pop()!;
  return path.join(runsDir, latestRun);
};

const cleanProjectOutput = (projectPath: string): void => {
  const threePioDir = path.join(projectPath, '.3pio');
  if (fs.existsSync(threePioDir)) {
    fs.rmSync(threePioDir, { recursive: true, force: true });
  }
};
```

2. Always clean the `.3pio` directory before each test

3. Include all default verifications unless explicitly not applicable

4. Use descriptive test names that explain what is being verified

5. Group related tests using `describe` blocks

## Running Integration Tests

```bash
# Run all integration tests
npm test tests/integration

# Run specific test file
npm test tests/integration/test-case-reporting.test.ts

# Run with coverage
npm test -- --coverage tests/integration
```