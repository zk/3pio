# Vitest Version Requirements

## Minimum Version: 3.0.0

3pio requires **Vitest 3.0 or higher** for proper operation. The adapter uses modern Vitest 3+ reporter API methods that are not available in earlier versions.

## Why Vitest 3.0+?

The decision to require Vitest 3.0+ was made to:

1. **Maintain clean, maintainable code** - Supporting multiple Vitest versions would require complex fallback logic and version detection
2. **Avoid duplicate test events** - Earlier implementations had issues with duplicate test event emission when trying to support both old and new APIs
3. **Use modern reporter APIs** - Vitest 3 provides cleaner reporter methods like `onTestCaseResult` and `onTestModuleEnd`
4. **Simplify maintenance** - A single code path is easier to test and debug

## Version Check

The Vitest adapter automatically checks the installed Vitest version on startup. If Vitest < 3.0 is detected:

1. An error message is displayed to the console
2. The adapter exits with code 1
3. The user is prompted to upgrade Vitest

## Upgrading Vitest

To upgrade to the latest Vitest version:

```bash
npm install --save-dev vitest@latest
# or
yarn add -D vitest@latest
# or
pnpm add -D vitest@latest
```

## Breaking Changes

If you're upgrading from Vitest 1.x or 2.x to 3.x, please review the [Vitest migration guide](https://vitest.dev/guide/migration) for any breaking changes that might affect your test suite.

## Technical Details

The adapter specifically uses these Vitest 3+ APIs:
- `onTestCaseResult` - For individual test result reporting
- `onTestModuleEnd` - For file completion events
- `onTestRunStart` - For test run initialization
- `onTestFileStart` - For file start events

These methods provide cleaner integration points and eliminate the need for complex workarounds that were required in earlier versions.