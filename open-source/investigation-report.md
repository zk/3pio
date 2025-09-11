# 3pio Testing Investigation Report

## Date: 2025-09-10

## Executive Summary
During testing of 3pio with open source projects, we encountered issues with the `ms` project (a small time conversion utility by Vercel). The tests failed to execute properly through 3pio, revealing compatibility and detection issues.

## Project Tested
- **Project**: ms (https://github.com/vercel/ms)
- **Test Framework**: Jest
- **Package Manager**: pnpm (primary), npm (fallback)
- **Version**: 4.0.0

## Issues Identified

### 1. Package Manager Incompatibility
**Problem**: The ms project uses pnpm as its package manager, but 3pio attempted to run tests through npm.

**Details**:
- The package.json test script: `"test": "pnpm run test:nodejs && pnpm run test:edge"`
- Error: `sh: pnpm: command not found`
- 3pio correctly detected npm test command but couldn't handle the pnpm dependency

### 2. TypeScript Configuration Requirements
**Problem**: When attempting to run Jest directly, the project required ts-node for its TypeScript configuration file.

**Details**:
- Jest config file: `jest.config.ts` (TypeScript)
- Initial error: `'ts-node' is required for the TypeScript configuration files`
- Solution: Had to install ts-node as a dev dependency

### 3. Test Detection Failure
**Problem**: Even after resolving dependencies, 3pio reported 0 test files detected.

**Evidence**:
```
## Summary
- Total Files: 0
- Files Completed: 0
- Files Passed: 0
- Files Failed: 0
```

**Possible Causes**:
1. Tests may be in a non-standard location
2. Jest adapter may not be capturing test discovery events properly
3. TypeScript compilation may be interfering with test detection

## Attempted Solutions

1. **Direct Jest Invocation**: Tried `3pio npx jest --env node` instead of `npm test`
   - Result: Bypassed pnpm issue but encountered ts-node requirement

2. **Installing Missing Dependencies**: Added ts-node to resolve TypeScript config parsing
   - Result: Jest ran but 3pio still reported 0 tests

## Recommendations for 3pio Improvement

1. **Enhanced Package Manager Support**
   - Detect and handle pnpm, yarn, and other package managers
   - Consider translating npm commands to detected package manager

2. **Better Error Reporting**
   - Provide clearer messages when test detection fails
   - Include diagnostic information about what was attempted

3. **TypeScript Support**
   - Consider bundling ts-node or providing guidance for TypeScript projects
   - Detect TypeScript config files and warn users about requirements

4. **Test Discovery Debugging**
   - Add a debug mode to show which files Jest is examining
   - Log adapter communication to help diagnose detection issues

## Log Files Generated
- `ms-jest.log` - Initial attempt with npm test
- `ms-jest-direct.log` - Direct Jest invocation attempt
- `ms-jest-with-ts-node.log` - After installing ts-node

## Conclusion
While 3pio successfully invoked the test runner, it failed to properly detect and report test execution. This appears to be a gap in handling TypeScript-based Jest projects and projects using alternative package managers.

## Next Steps
1. Test with simpler JavaScript-only Jest projects
2. Find projects using standard npm scripts without pnpm
3. Test with Vitest, pytest, and Go projects as originally planned
4. Consider contributing fixes to 3pio for identified issues