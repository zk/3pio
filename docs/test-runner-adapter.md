# Component Design: Test Runner Adapters

* **Version:** 1.0
* **Owner:** Adapters Team
* **Status:** Final

## 1. General Responsibilities

The Test Runner Adapters are modules that run *inside* the test runner's process. Their sole purpose is to capture test lifecycle events and raw output, and transmit this data as structured events back to the CLI Orchestrator via the IPC channel.

All adapters must adhere to the following principles:

* **Silent Operation:** Adapters **must not** write to stdout or stderr. They are designed to be "silent" reporters that do not interfere with the user's default console output.
* **Event Transmission:** Adapters must use the IPCManager's writer to send events adhering to the official IPC Event Schema.
* **Configuration:** Adapters will discover the path to the IPC event file by reading the THREEPIO\_IPC\_PATH environment variable set by the CLI Orchestrator.

## 2. Stream Tapping Strategy

To enable real-time streaming of log output, adapters will patch the global process.stdout.write and process.stderr.write functions within the test runner's process.

* **Lifecycle:**
  1. When the adapter's onTestFileStart (or equivalent) hook is called, it replaces the native .write functions with its own wrappers.
  2. These wrappers capture the output chunk and immediately send a stdoutChunk or stderrChunk event via the IPC writer.
  3. When the onTestFileResult hook is called, the adapter restores the original .write functions to their native state.

## 3. Specific Adapter Implementations

### 3.1. Jest Adapter (@3pio/core/jest)

* **Implementation:** class JestAdapter extends DefaultReporter (from @jest/reporters). This ensures it can coexist with Jest's default reporter.
* **Key Hooks:**
  * onRunStart(): Initializes the IPC writer.
  * onTestStart(test): Begins stream tapping for test.path.
  * onTestResult(test, testResult): Ends stream tapping and sends the final testFileResult event with the pass/fail status from testResult.
  * onRunComplete(): Can be used for final cleanup if necessary.

### 3.2. Vitest Adapter (@3pio/core/vitest)

* **Implementation:** class VitestAdapter implements Reporter (from vitest/node).
* **Key Hooks:**
  * onRunStart(files): Initializes the IPC writer.
  * onTestFileStart(file): Begins stream tapping for file.filepath.
  * onTestFileResult(file): Ends stream tapping and sends the final testFileResult event with the status derived from file.result.state.
  * onRunComplete(files): Can be used for final cleanup.

## 4. Failure Modes

* **Missing Environment Variable:** The THREEPIO\_IPC\_PATH environment variable is not set, so the adapter does not know where to send events.
* **Test Runner API Changes:** A new version of Jest or Vitest introduces a breaking change to their reporter API, causing the adapter to crash or fail to capture events.
* **Adapter Crash:** An unhandled exception within the adapter's code causes it to crash, which may or may not crash the entire test runner process.
* **Stream Patching Conflicts:** Another reporter in the user's configuration also tries to patch process.stdout.write, leading to unpredictable behavior or lost output.

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