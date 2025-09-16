# 3pio Test Report: Create React App

**Project**: create-react-app
**Framework(s)**: JavaScript (Jest) - supported by 3pio
**Test Date**: 2025-09-15
**3pio Version**: (from build)

## Project Analysis
- Project type: JavaScript React application bootstrapping tool
- Test framework(s): Jest (via react-scripts test command)
- Test command(s): `npm test` (proxies to react-scripts test) and `npm run test:integration`
- Test suite size: Monorepo with workspace packages and integration tests

## 3pio Test Results
### Command: `CI=1 ../../build/3pio yarn test:integration --silent --runInBand`
- **Status**: TESTED (PARTIAL FAIL)
- **Exit Code**: 1
- **Detection**: Jest detected; adapter injected
- **Results**: 4 passed, 3 failed, 0 skipped
- **Total time**: ~138s

### Failure Summary (key excerpts)
- creates a project on supplying a name as the argument
  - Expected files included `package-lock.json`; received `yarn.lock`
- creates a project in the current directory
  - Expected `package-lock.json`; received `yarn.lock`
- creates a project based on the typescript template
  - Expected no `tsconfig.json`; received `tsconfig.json` and `yarn.lock`

Notes: The repository defaulted to Yarn when invoking the CLI, which differs from tests that assume npm (lockfile mismatch). Forcing npm via `env npm_config_user_agent=npm` or equivalent may align expectations.

### Project Structure
- npm workspace with packages/* structure
- Uses Jest for testing via react-scripts
- Has both unit tests and integration tests
- Complex build system with custom scripts

### Expected Compatibility
- 3pio supports JavaScript Jest and works with CRA integration tests
- react-scripts passthrough at repo root runs no tests; use `yarn test:integration`
- Adapter injection works in CI/non-watch (`CI=1`)

### Recommendations
1. Use `CI=1 yarn test:integration --runInBand` with 3pio for reliable runs
2. To satisfy npm-based expectations, set `npm_config_user_agent=npm` (or remove Yarn from PATH) when running tests
3. For per‑package experimentation, run consumer-app scenarios rather than `packages/react-scripts` (which has no tests)
