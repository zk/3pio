# Component Design: Report Manager

## 1. Core Purpose

The Report Manager encapsulates all file system logic related to the creation and real-time updating of the persistent test reports. It acts as the single source of truth for report-related I/O, ensuring that file access is safe, performant, and free of race conditions. It is controlled exclusively by the CLI Orchestrator.

## 2. Internal State

The manager holds the entire state of the test-run.md file in memory to facilitate fast updates.

* **RunState Interface:**
  interface RunState {
    runId: string;
    startTime: string;
    arguments: string;
    summary: {
      total: number;
      passed: number;
      failed: number;
      pending: number;
    };
    testFiles: Map\<string, {
      status: 'PENDING' | 'PASS' | 'FAIL';
      logPath: string;
    }\>;
  }

* **Log Buffers:** It also manages the creation and writing of individual .log files.

## 3. Public API

The manager exposes a simple API for the CLI Orchestrator to use.

* **initialize(runId: string, testFiles: string\[\], args: string): void**
  * Creates the run directory (/.3pio/runs/\[runId\]/) and the logs subdirectory.
  * Populates the initial in-memory RunState with all testFiles set to a PENDING status.
  * Performs an immediate, initial write of test-run.md to disk.
* **handleEvent(event: IPCEvent): void**
  * The central method for processing events from the IPC channel.
  * If eventType is stdoutChunk or stderrChunk, it appends the chunk to the appropriate .log file.
  * If eventType is testFileResult, it updates the status of the corresponding test file in the in-memory RunState, re-calculates the summary, and schedules a debounced write of test-run.md.
* **finalize(): Promise\<void\>**
  * Cancels any pending debounced writes.
  * Performs one final, synchronous write of the complete test-run.md file to disk to ensure the final state is 100% accurate.

## 4. Debounced Write Mechanism

To ensure high performance, the manager does not write to test-run.md on every single status update.

1. **Library:** A standard library like lodash.debounce will be used.
2. **Flow:**
   * A writeReportToDisk() method is created, which renders the in-memory RunState into a Markdown string and writes it to the file.
   * This method is wrapped in a debounce function with a timeout (e.g., 250ms).
   * The handleEvent method calls this debounced function every time the state changes.
   * This effectively batches numerous state changes into a single file write, drastically reducing I/O load.

## 5. Failure Modes

* **File System Permissions:** The process does not have permission to create the .3pio directory or write files within it.
* **Disk Full:** The disk runs out of space while writing a log file or the summary report.
* **Invalid Event Data:** It receives an event with a filePath that was not part of the initial list of test files.
* **Process Crash:** The main 3pio process crashes after some logs have been written but before the final debounced write of test-run.md can complete, leaving the summary report in an incomplete state.

## 6. Testing Strategy

* **Unit Tests:**
  * Test the initialize method to ensure it correctly creates the directory structure and the initial test-run.md file with all tests PENDING.
  * Test the handleEvent method by passing mock IPC events and asserting that the in-memory state is updated correctly.
  * Test the Markdown rendering logic to ensure the in-memory state is correctly converted to a string.
  * Use mock timers (jest.useFakeTimers()) to test the debounced write mechanism, ensuring that writeReportToDisk is called only after the specified delay.
* **Integration Tests:**
  * Test the component against a real file system (in a temporary directory).
  * Simulate a stream of IPC events and verify that both the individual .log files and the final test-run.md are written correctly and contain the expected content.
  * Test the finalize method to ensure it correctly performs the final write.
