# Component Design: Test Runner Adapters

## 1. General Responsibilities

The Test Runner Adapters are modules that run *inside* the test runner's process. Their sole purpose is to capture test lifecycle events and raw output, and transmit this data as structured events back to the CLI Orchestrator via the IPC channel.

All adapters must adhere to the following principles:

* **Silent Operation:** Adapters **must not** write to stdout or stderr for normal operations. They are designed to be "silent" reporters that do not interfere with the user's default console output. (Debug logging to /tmp/vitest-debug.log is acceptable for development/troubleshooting.)
* **Event Transmission:** Adapters must use `IPCManager.sendEvent()` static method to send events adhering to the official IPC Event Schema.
* **Configuration:** Adapters discover the IPC file path by reading the THREEPIO_IPC_PATH environment variable set by the CLI Orchestrator.
* **Error Resilience:** All IPC operations must be wrapped in try/catch or .catch() to prevent adapter failures from crashing the test runner.

## 2. Stream Tapping Strategy

To enable real-time streaming of log output, adapters will patch the global process.stdout.write and process.stderr.write functions within the test runner's process. This approach is necessary because Jest's `testResult.console` property is not populated (see [Jest Console Handling](./jest-console-handling.md) for details).

* **Lifecycle:**
  1. Adapters store references to original stdout/stderr write functions in constructor.
  2. startCapture() replaces the native .write functions with wrappers that:
     - Capture output chunks and send stdoutChunk/stderrChunk events via IPCManager.sendEvent()
     - Pass through the original output to maintain normal console behavior
  3. stopCapture() restores the original write functions.
  4. Jest adapter: Capture lifecycle tied to onTestStart/onTestResult hooks.
  5. Vitest adapter: Capture started in onInit() and maintained throughout run, with currentTestFile context switching.

* **Error Handling:** All IPC send operations use .catch(() => {}) for silent failure to avoid disrupting test execution.

## 3. Specific Adapter Implementations

### 3.1. Jest Adapter (ThreePioJestReporter)

* **Implementation:** `class ThreePioJestReporter implements Reporter` (from @jest/reporters). This ensures it can coexist with Jest's default reporter.
* **Key Hooks:**
  * onRunStart(): Initializes connection and validates THREEPIO_IPC_PATH environment variable.
  * onTestStart(test): Sets currentTestFile and begins stream tapping for test.path.
  * onTestResult(test, testResult, aggregatedResult): Ends stream tapping and sends testFileResult event with status derived from testResult.numFailingTests and testResult.skipped. Note: `testResult.console` is not used as it's always undefined (see [Jest Console Handling](./jest-console-handling.md)).
  * onRunComplete(testContexts, results): Ensures capture is stopped and performs cleanup.
  * getLastError(): Required by Reporter interface (no-op implementation).

### 3.2. Vitest Adapter (ThreePioVitestReporter)

* **Implementation:** `class ThreePioVitestReporter implements Reporter` (from vitest).
* **Key Hooks:**
  * onInit(ctx): Initializes connection, validates THREEPIO_IPC_PATH, and starts capturing immediately.
  * onPathsCollected(paths): Called when test files are discovered.
  * onCollected(files): Called when test files are collected.
  * onTestFileStart(file): Sets currentTestFile and ensures capture is started for file.filepath.
  * onTestFileResult(file): Ends stream tapping and sends testFileResult event with status derived from file.result.state or file.mode.
  * onFinished(files, errors): Ensures capture is stopped, includes fallback logic to send results for files that may not have triggered onTestFileResult.

**Note:** The Vitest adapter includes extensive debug logging to `/tmp/vitest-debug.log` for troubleshooting integration issues.

## 4. Failure Modes

* **Missing Environment Variable:** The THREEPIO\_IPC\_PATH environment variable is not set, so the adapter does not know where to send events.
* **Test Runner API Changes:** Breaking changes in Jest/Vitest reporter APIs between versions causing adapter to crash or fail to capture events.
* **Adapter Crash:** An unhandled exception within the adapter's code causes it to crash, which may or may not crash the entire test runner process.
* **Stream Patching Conflicts:** Another reporter in the user's configuration also tries to patch process.stdout.write, leading to unpredictable behavior or lost output.
* **Context Switching Issues:** In Vitest, output may occur without clear test file context, handled by using 'global' as fallback filepath.

## 5. Testing Strategy

* **Unit Tests:**
  * Test the logic that reads the environment variable and initializes the IPC writer.
  * Test the stream tapping logic by using mock objects for process.stdout and asserting that the IPC writer is called with the correct chunk data.
* **Integration Tests:**
  * This is the most critical testing layer for the adapters.
  * Create small, self-contained Jest and Vitest projects.
  * Write tests that programmatically invoke the test runners with the 3pio adapter configured.
  * These tests will monitor the contents of the IPC file and assert that the correct sequence and content of events are written for various scenarios (passing tests, failing tests, tests with console logs).
  * This validates that the adapter correctly hooks into the real test runner's lifecycle and captures the data as expected.
