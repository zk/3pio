# Vitest Version Requirements

## Minimum Version: 2.0.0

3pio supports **Vitest 2.x and 3.x**. The adapter detects the installed Vitest major version at runtime and uses the corresponding reporter API surface.

## Why 2.x and 3.x?

- 3.x provides modern reporter APIs (e.g., `onTestCaseResult`, `onTestModuleEnd`).
- 2.x is still widely used and exposes a different reporter surface; we normalize its events during `onFinished`.
- We avoid duplicate events by cleanly branching per major version.

## Version Check

The Vitest adapter automatically checks the installed Vitest version on startup. If Vitest < 2.0 is detected, the adapter exits with an error and prompts to upgrade.

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

If you're upgrading between major versions, review the [Vitest migration guide](https://vitest.dev/guide/migration) for any breaking changes that might affect your test suite.

## Technical Details

Adapter APIs used:
- Vitest 3.x: `onTestCaseResult`, `onTestModuleEnd`, `onTestRunStart`, `onTestFileStart`
- Vitest 2.x: `onFinished` with post-run task traversal to emit `testCase` and file results

We normalize both paths to a single internal IPC event model.
