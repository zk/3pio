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
  * **State Management:** Holds the complete TestRunState in memory including counters, timestamps, and test file statuses.
  * **Initialization:** The `async initialize(testFiles)` method creates the run directory and the initial test-run.md, opens output.log with header for real-time writing.
  * **Event Handling:** Has a central `async handleEvent(event)` method that updates the in-memory state based on IPC events.
    * On stdoutChunk or stderrChunk, it appends the text chunk to the unified output.log file AND collects it in a per-file Map for individual log generation.
    * On testFileStart, it updates the test file status to RUNNING.
    * On testFileResult, it updates the test file status and counters, then schedules a debounced write.
    * Dynamically adds test files not discovered during dry run (common with Vitest).
  * **Dual Logging Strategy:**
    * Maintains single output.log file during execution for complete output record (including non-test output)
    * Simultaneously collects per-file output in memory Map using file paths from IPC events
    * During finalize(), writes individual test logs directly from the collected Map
    * Non-test output (startup messages, warnings, summary) only appears in output.log, not in individual files
  * **Debounced Writes:** Uses lodash.debounce (250ms delay, 1000ms maxWait) to batch test-run.md updates for performance.

### 3.3. IPC Manager (src/ipc.ts)

This is a focused, class-based module that manages the file-based communication channel.

* **Responsibilities:**
  * **Class-based Design:** Uses IPCManager class for CLI orchestrator with instance methods.
  * **Writer (for Adapters):** Provides static `IPCManager.sendEvent(event)` method that reads THREEPIO_IPC_PATH environment variable and uses synchronous file operations for reliability.
  * **Reader (for CLI):** Instance method `watchEvents(callback)` uses chokidar to monitor IPC file changes, tracks read position to process only new events.
  * **Debug Support:** Static sendEvent includes debug logging for troubleshooting adapter integration issues.

### 3.4. Test Runner Adapters (src/adapters/)

These modules are the data collectors that run *inside* the Jest or Vitest process.

* **Structure:** Concrete implementations: ThreePioJestReporter and ThreePioVitestReporter.
* **Responsibilities:**
  * **Initialization:** Adapters read THREEPIO_IPC_PATH environment variable and validate connection.
  * **Stream Tapping:** Patch process.stdout.write and process.stderr.write with pass-through wrappers that capture output and send IPC events.
  * **Event Transmission:** Use static `IPCManager.sendEvent()` method to send testFileStart, stdoutChunk, stderrChunk, and testFileResult events.
  * **Lifecycle Management:**
    * Jest: Capture tied to onTestStart/onTestResult hooks
    * Vitest: Capture started in onInit() and maintained throughout run
  * **Error Resilience:** All IPC operations wrapped in .catch(() => {}) to prevent test runner crashes.
  * **Debug Support:** Vitest adapter includes extensive debug logging for troubleshooting.

## 4. Console Output Capture Strategy

### Design Decision: No Default Reporter

3pio intentionally does NOT include Jest's or Vitest's default reporter when running tests. This is a deliberate design choice to maintain clean test output that is context concious.

### Implications and Solutions

**Challenge:** Without the default reporter, test runners don't format console output with file associations and stack traces.

**Solution:** 3pio uses a dual-capture approach:
1. **IPC Events with File Associations:** Test adapters send stdout/stderr chunks via IPC with the associated test file path
2. **In-Memory Collection:** ReportManager maintains a Map of file paths to output arrays, populated from IPC events
3. **Direct Writing:** Individual log files are written directly from the collected Map, not parsed from output.log

This approach ensures:
- Console output is correctly attributed to test files
- No dependency on specific output formatting
- Works even when test runners use worker processes (like Jest)

## 5. Data Flow (Sequence of Events)

A typical run of 3pio run vitest would proceed as follows:

1. **User** executes the command.
2. **CLI Orchestrator** starts, parses arguments, and detects the runner is vitest.
3. **Orchestrator** performs the dry run (vitest list) to get all test file paths.
4. **Orchestrator** creates the run directory and the IPC file.
5. **Orchestrator** instantiates the **Report Manager**.
6. **Report Manager** creates the initial test-run.md with all tests marked PENDING.
7. **Orchestrator** prints the formatted preamble to the console.
8. **Orchestrator** uses zx to spawn the final vitest command with the --reporter @heyzk/3pio/vitest flag injected. It begins piping the raw stdout/stderr from this process to the console.
9. Inside the vitest process, the **Vitest Adapter** is initialized with the path to the IPC file.
10. The **Adapter** patches process.stdout.write and begins capturing output.
11. As a test file runs and logs to the console, the **Adapter** captures the chunk and uses the **IPC Manager** to append a stdoutChunk event to the IPC file.
12. The **Orchestrator**, listening via the **IPC Manager**, receives the event.
13. The **Orchestrator** passes the event to the **Report Manager**.
14. The **Report Manager** appends the chunk to the unified output.log file AND stores it in the per-file output Map.
15. When the test file finishes, the **Adapter** sends the testFileResult event.
16. The **Report Manager** receives this event, updates its in-memory state for test-run.md (e.g., changing the status from PENDING to PASS), and schedules a debounced write.
17. This process repeats for all test files.
18. When the vitest process exits, the **Orchestrator** calls `await reportManager.finalize(exitCode)` to:
    * Close the output.log file handle
    * Write individual test file logs from the collected per-file output Map (no parsing needed)
    * Perform one last, guaranteed write of test-run.md
19. The **Orchestrator** cleans up the IPC file and exits with the same exit code as vitest.
