# 3pio Test Report: Supabase

**Project**: supabase
**Framework(s)**: Mixed (Vitest, Jest, custom)
**Test Date**: 2025-09-15
**3pio Version**: (from build)

## Project Analysis
- Project type: Multi-language monorepo with TypeScript/JavaScript
- Test framework(s): Mixed - different packages use different test runners
- Test command(s): `npm test` (orchestrates via turbo)

## 3pio Test Results
### Command: `/Users/zk/code/3pio/build/3pio npm test`
- **Status**: NOT DETECTED
- **Exit Code**: Error
- **Detection**: Could not detect test runner
- **Output**: "Could not detect test runner from command: npm test"

### Issues Encountered
- The npm test command in the root uses turbo to orchestrate tests across multiple packages
- 3pio cannot detect the test runner when it's wrapped by turbo or other build tools
- Individual packages likely use Jest or Vitest but are not directly accessible from root

### Recommendations
1. Add support for detecting test runners through turbo/nx/lerna monorepo tools
2. Allow 3pio to run on individual packages within monorepos
3. Consider parsing turbo.json or similar config files to understand test orchestration
4. Users could work around this by running 3pio directly in individual packages