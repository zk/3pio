# Writing Test Adapters

Test adapters are code we inject into the users's chosen test runner (e.g. `vitest --reporter /our/custom/reporter.js`).

These are bundled into the go binary and extracted at each test run. Additionally configuration information is written to the adapter (like IPC file location) at the extraction step.
