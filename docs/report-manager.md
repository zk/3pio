# Component Design: Report Manager

## 1. Core Purpose

The Report Manager encapsulates all file system logic related to the creation and real-time updating of the persistent test reports. It acts as the single source of truth for report-related I/O, ensuring that file access is safe, performant, and free of race conditions. It is controlled exclusively by the CLI Orchestrator.

## 2. Internal State

The manager holds the entire state of the test-run.md file in memory to facilitate fast updates.

* **TestRunState Interface:**
  ```typescript
  interface TestRunState {
    timestamp: string;
    status: 'RUNNING' | 'COMPLETE' | 'ERROR';
    updatedAt: string;
    arguments: string;
    totalFiles: number;
    filesCompleted: number;
    filesPassed: number;
    filesFailed: number;
    filesSkipped: number;
    testFiles: Array<{
      status: 'PENDING' | 'RUNNING' | 'PASS' | 'FAIL' | 'SKIP';
      file: string;
      logFile?: string;
    }>;
  }
  ```

* **Unified Output Log:** The manager maintains a single `output.log` file handle that captures all stdout/stderr output in real-time. Individual test log files are created during post-processing in `finalize()`.

## 3. Public API

The manager exposes a simple API for the CLI Orchestrator to use.

* **constructor(runId: string, testCommand: string)**
  * Initializes the manager with run ID and test command.
  * Sets up directory paths and initial state structure.
  * Creates debounced write function using lodash.debounce (250ms delay, 1000ms maxWait).
* **async initialize(testFiles: string[]): Promise<void>**
  * Creates the run directory (.3pio/runs/[runId]/) and opens output.log for writing.
  * Populates the initial in-memory TestRunState with all testFiles set to PENDING status.
  * Performs an immediate, initial write of test-run.md to disk.
* **async handleEvent(event: IPCEvent): Promise<void>**
  * The central method for processing events from the IPC channel.
  * If eventType is stdoutChunk or stderrChunk, it appends the chunk to the unified output.log file.
  * If eventType is testFileResult, it updates the status of the corresponding test file in the in-memory state, re-calculates counters, and schedules a debounced write of test-run.md.
  * Dynamically adds test files that weren't discovered during dry run (common with Vitest).
* **async finalize(exitCode: number): Promise<void>**
  * Closes the output.log file handle.
  * Parses the unified output.log into individual test file logs using `parseOutputIntoTestLogs()`.
  * Cancels any pending debounced writes and performs one final write of test-run.md.
  * Updates final status to 'COMPLETE'.

## 4. Debounced Write Mechanism

To ensure high performance, the manager does not write to test-run.md on every single status update.

1. **Library:** Uses lodash.debounce with 250ms delay and 1000ms maxWait.
2. **Flow:**
   * A `writeTestRunReport()` method renders the in-memory TestRunState into a Markdown string and writes it to test-run.md.
   * This method is wrapped in a debounce function during constructor initialization.
   * The `handleEvent` method calls this debounced function every time the state changes.
   * This effectively batches numerous state changes into a single file write, drastically reducing I/O load.
3. **Output Log Processing:**
   * Real-time output goes to a unified `output.log` file.
   * During `finalize()`, `parseOutputIntoTestLogs()` parses this file to create individual test file logs.
   * Uses pattern matching to identify Vitest output format: `stdout | filename.test.js > content`.

## 5. Failure Modes

* **File System Permissions:** The process does not have permission to create the .3pio directory or write files within it.
* **Disk Full:** The disk runs out of space while writing a log file or the summary report.
* **Invalid Event Data:** It receives an event with a filePath that was not part of the initial list of test files (handled gracefully by dynamic addition to state).
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
