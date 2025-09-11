# ES Module Compatibility Issue Investigation

## Problem
3pio Jest adapter fails when used with ES module projects (projects with `"type": "module"` in package.json).

## Error Details
```
ReferenceError: An error occurred while adding the reporter at path "/path/to/jest.js".
module is not defined
    at file:///path/to/jest.js:125:1
```

## Root Cause
- The `ms` project has `"type": "module"` in package.json, making it an ES module project
- Our Jest adapter is built as CommonJS (uses `module.exports`)
- In ES module context, CommonJS constructs like `module.exports` are not available
- Jest tries to load our adapter but fails because `module` is undefined

## Solution Options
1. **Dual Format Adapter**: Build adapter to work in both CommonJS and ES module contexts
2. **Runtime Detection**: Detect module type and use appropriate export mechanism
3. **File Extension**: Use `.mjs` extension to force ES module loading
4. **Jest Configuration**: Configure Jest to handle the adapter differently for ES module projects

## Test Case
- Project: `ms` (https://github.com/vercel/ms)
- Configuration: `"type": "module"` in package.json
- Jest config: TypeScript-based with ts-jest preset
- Tests: Actually pass (4 passed, 167 total) but adapter loading fails

## Impact
- Tests execute successfully but 3pio reporting fails
- Error occurs during Jest reporter loading phase
- Affects any ES module project using Jest with 3pio

## Next Steps
Need to modify the Jest adapter to support ES module projects while maintaining CommonJS compatibility.