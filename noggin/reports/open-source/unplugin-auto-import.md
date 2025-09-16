# 3pio Test Report: Unplugin Auto Import

**Project**: unplugin-auto-import
**Framework(s)**: Vitest
**Test Date**: 2025-09-15
**3pio Version**: (from build)

## Project Analysis
- Project type: TypeScript unplugin library
- Test framework(s): Vitest
- Test command(s): `npx vitest run`

## 3pio Test Results
### Command: `/Users/zk/code/3pio/build/3pio npx vitest run`
- **Status**: FAILURE
- **Exit Code**: 1
- **Detection**: Framework detected correctly: YES
- **Output**: 3 tests total, 2 passed, 1 failed

### Issues Encountered
- One test file failed: transform.test.ts
- Clean execution from 3pio perspective
- Exit code correctly mirrored (1)

### Test Details
- Passed:
  - dts.test.ts
  - search.test.ts
- Failed:
  - transform.test.ts

### Recommendations
- 3pio handled the test execution correctly
- Exit code properly reflected test failure
- Report generation worked as expected
- The test failure appears to be a genuine test failure in the project, not a 3pio issue