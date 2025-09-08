# 3pio: System Architecture Design

## 1. Introduction

This document outlines the system architecture for 3pio, an AI-first test runner adapter. The design is based on the detailed project plan and is intended to provide a clear blueprint for implementation. The architecture prioritizes performance, reliability, and a seamless "zero-config" user experience.

## 2. High-Level Architecture

The system is composed of four primary components that work together to execute a test run, capture results, and generate a persistent, structured report.

1. **CLI Orchestrator:** The main entry point and "brain" of the application. It manages the entire lifecycle of a test run.
2. **Report Manager:** A dedicated service that encapsulates all file system interactions related to the report generation.
3. **IPC Manager:** A focused module that manages the file-based communication channel between the test runner adapters and the CLI.
4. **Test Runner Adapters:** The modules that run *inside* the test runner processes (Jest/Vitest) to capture and transmit data.

## 3. Component Breakdown

### 3.1. CLI Orchestrator (src/cli.ts)

The orchestrator is responsible for managing the high-level flow of the 3pio command.

* **Responsibilities:**
  * **Argument Parsing:** Uses commander.js to parse the run command and capture the underlying test command and its arguments.
  * **Test Runner Detection:** Implements the logic to inspect package.json to determine which test runner to use for abstract commands like npm test.
  * **Dry Run Execution:** Performs the pre-flight check (jest --listTests or vitest list) to get the list of test files required for the preamble.
  * **Setup:** Creates the unique run directory (e.g., /.3pio/runs/RUN\_ID) and the IPC event file (/.3pio/ipc/RUN\_ID.jsonl).
  * **Preamble Generation:** Prints the formatted preamble to the console.
  * **Report Initialization:** Instantiates the ReportManager and calls its initialize() method to create the initial test-run.md file with all tests marked as PENDING.
  * **Process Spawning:** Uses zx to execute the user's command after programmatically injecting the correct adapter flags. The zx process's stdout and stderr are piped directly to the user's console to preserve the raw output.
  * **IPC Listening:** Uses the IPCManager to listen for events from the adapter. When an event is received, it delegates the handling to the ReportManager.
  * **Cleanup:** When the zx process exits, it calls reportManager.finalize(), cleans up the IPC file, and then exits with the mirrored exit code from the test runner.

### 3.2. Report Manager (src/ReportManager.ts)

This component is the single source of truth for all report-related file I/O, ensuring that file access is safe and performant.

* **Responsibilities:**
  * **State Management:** Holds the complete state of the test-run.md file in memory as a JavaScript object. This includes the overall summary and the status of each test file in the results table.
  * **Initialization:** The initialize(testFiles) method creates the run directory, the logs subdirectory, and the initial test-run.md from the list of files gathered during the dry run.
  * **Event Handling:** Has a central handleEvent(event) method that updates the in-memory state based on IPC events.
    * On stdoutChunk or stderrChunk, it appends the text chunk to the correct individual .log file.
    * On testFileResult, it updates the status (PASS/FAIL) of the test file in the main report's in-memory state.
  * **Debounced Writes:** After any change to the in-memory state of test-run.md, it schedules a debounced write operation (e.g., using lodash.debounce). This function, when it runs, will render the current in-memory state into a Markdown string and write it to test-run.md, batching potentially hundreds of updates into a single efficient file write. A final, non-debounced write is performed upon run completion.

### 3.3. IPC Manager (src/ipc.ts)

This is a focused, reusable module that manages the file-based communication channel.

* **Responsibilities:**
  * **Writer (for Adapters):** Provides a simple, robust writeEvent(filePath, event) function. This function will serialize the event object to a JSON string and append it as a new line to the specified IPC file.
  * **Reader (for CLI):** Provides a watchEvents(filePath, callback) function. This will use a file watcher (like chokidar) to monitor the IPC file for changes. When the file grows, it will efficiently read only the *new* lines, parse each one as JSON, and invoke the callback with the structured event object. This avoids wastefully re-reading the entire file on every update.

### 3.4. Test Runner Adapters (src/adapters/)

These modules are the data collectors that run *inside* the Jest or Vitest process.

* **Structure:** A base Adapter class can define the interface, with JestAdapter.ts and VitestAdapter.ts as concrete implementations.
* **Responsibilities:**
  * **Initialization:** The adapter is initialized by the test runner. Its constructor will receive the path to the IPC file from the CLI Orchestrator (e.g., via an environment variable).
  * **Stream Tapping:** To enable real-time log streaming, the adapter will patch process.stdout.write and process.stderr.write *within the test runner's process*. When these functions are called, the adapter captures the output chunk.
  * **Event Transmission:**
    * When it captures a stdout or stderr chunk, it immediately sends a stdoutChunk or stderrChunk event via the IPCManager.
    * When the test runner's API signals that a test file has finished (e.g., onTestFileResult in Jest), it sends the final testFileResult event containing the pass/fail status.
  * **Silent Operation:** As per the project plan, these adapters never write to the console themselves. Their only job is to send data through the IPC channel.

## 4. Data Flow (Sequence of Events)

A typical run of 3pio run vitest would proceed as follows:

1. **User** executes the command.
2. **CLI Orchestrator** starts, parses arguments, and detects the runner is vitest.
3. **Orchestrator** performs the dry run (vitest list) to get all test file paths.
4. **Orchestrator** creates the run directory and the IPC file.
5. **Orchestrator** instantiates the **Report Manager**.
6. **Report Manager** creates the initial test-run.md with all tests marked PENDING.
7. **Orchestrator** prints the formatted preamble to the console.
8. **Orchestrator** uses zx to spawn the final vitest command with the --reporter @3pio/core/vitest flag injected. It begins piping the raw stdout/stderr from this process to the console.
9. Inside the vitest process, the **Vitest Adapter** is initialized with the path to the IPC file.
10. The **Adapter** patches process.stdout.write and begins capturing output.
11. As a test file runs and logs to the console, the **Adapter** captures the chunk and uses the **IPC Manager** to append a stdoutChunk event to the IPC file.
12. The **Orchestrator**, listening via the **IPC Manager**, receives the event.
13. The **Orchestrator** passes the event to the **Report Manager**.
14. The **Report Manager** appends the chunk to the correct .log file in the logs subdirectory.
15. When the test file finishes, the **Adapter** sends the testFileResult event.
16. The **Report Manager** receives this event, updates its in-memory state for test-run.md (e.g., changing the status from PENDING to PASS), and schedules a debounced write.
17. This process repeats for all test files.
18. When the vitest process exits, the **Orchestrator** calls reportManager.finalize() to perform one last, guaranteed write of test-run.md.
19. The **Orchestrator** cleans up the IPC file and exits with the same exit code as vitest.
