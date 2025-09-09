# 3pio: System Architecture Design

## 1. Introduction

This document outlines the system architecture for 3pio, an AI-first test runner adapter. The architecture prioritizes context efficiency, performance, reliability, and a seamless "zero-config" user experience.

## 2. High-Level Architecture

The system is composed of five primary components that work together to execute a test run, capture results, and generate persistent, structured reports.

1. **CLI Orchestrator:** The main entry point that manages the entire lifecycle of a test run.
2. **Test Runner Manager:** Strategy pattern implementation for test runner detection and configuration.
3. **Report Manager:** Handles all file I/O for report generation with debounced writes and dynamic test discovery.
4. **IPC Manager:** File-based communication channel between test runner adapters and the CLI.
5. **Test Runner Adapters:** Silent reporters running inside test processes (Jest/Vitest) to capture and transmit data.

## 3. Component Breakdown

### 3.1. CLI Orchestrator (src/cli.ts)

The orchestrator manages the high-level flow of the 3pio command.

* **Responsibilities:**
  * **Argument Parsing:** Uses commander.js to parse the run command and capture the underlying test command.
  * **Test Runner Detection:** Delegates to TestRunnerManager to identify the test runner from command and package.json.
  * **Test File Discovery:** Uses TestRunnerDefinition.getTestFiles() for static discovery or supports dynamic discovery.
  * **Run ID Generation:** Creates unique identifiers with ISO8601 timestamps plus memorable Star Wars character names.
  * **Setup:** Creates run directory (.3pio/runs/[timestamp]-[name]) and IPC file (.3pio/ipc/[timestamp]-[name].jsonl).
  * **Simplified Preamble:** Outputs only report path and "Beginning test execution now..." message.
  * **Report Initialization:** Creates ReportManager with OutputParser and initializes with optional test file list.
  * **Process Spawning:** Uses zx to execute command with adapter injection, capturing stdout/stderr to output.log.
  * **IPC Listening:** Monitors IPC file for events, delegating to ReportManager for state updates.
  * **Cleanup:** Finalizes reports, cleans up IPC resources, and exits with mirrored exit code.

### 3.2. Report Manager (src/ReportManager.ts)

This component manages all report-related file I/O with performance optimizations.

* **Responsibilities:**
  * **State Management:** Maintains TestRunState in memory with file statuses and test case results.
  * **Dynamic Initialization:** `initialize(testFiles?: string[])` supports both static and dynamic test discovery modes.
  * **Event Handling:** Central `handleEvent(event)` processes IPC events:
    * **testCase:** Updates individual test case status, duration, and error details
    * **testFileStart:** Updates file status to RUNNING
    * **testFileResult:** Updates file status and counters with debounced write
    * **stdoutChunk/stderrChunk:** Appends to output.log and collects in per-file Maps
  * **Dynamic Test Discovery:** `ensureTestFileRegistered(filePath)` adds files discovered during execution
  * **Test Case Tracking:** Maintains test cases per file with suite organization and individual results
  * **Output Collection:**
    * Unified output.log for complete test run capture
    * Per-file output Maps for individual test logs
    * Test case boundary tracking for organized output
  * **Debounced Writes:** Uses lodash.debounce (250ms delay, 1000ms maxWait) for performance
  * **Finalization:** Writes individual test logs, updates final status, and ensures all data is persisted

### 3.3. IPC Manager (src/ipc.ts)

Manages file-based communication between adapters and CLI.

* **Responsibilities:**
  * **Class-based Design:** IPCManager class with instance methods for CLI, static methods for adapters
  * **Writer (for Adapters):** Static `IPCManager.sendEvent(event)` reads THREEPIO_IPC_PATH environment variable
  * **Reader (for CLI):** Instance `watchEvents(callback)` uses chokidar for file monitoring
  * **Event Types:** Supports testCase, testFileStart, testFileResult, stdoutChunk, stderrChunk events
  * **Directory Management:** Static `ensureIPCDirectory()` creates .3pio/ipc directory structure
  * **Cleanup:** `cleanup()` method stops file watching and releases resources
  * **Logging:** Integrated Logger for debugging IPC communication

### 3.4. Test Runner Adapters (src/adapters/)

Silent reporters that run inside Jest or Vitest processes.

* **Structure:**
  * **Jest:** ThreePioJestReporter class implementing Jest reporter interface
  * **Vitest:** ThreePioVitestReporter class implementing Vitest reporter interface
* **Responsibilities:**
  * **Silent Operation:** No console output - all communication via IPC
  * **Test Case Reporting:** Send individual test case results with suite, status, duration, and errors
  * **Stream Capture:** Patch process.stdout.write and process.stderr.write to capture console output
  * **Event Transmission:** Send testCase, testFileStart, testFileResult, stdoutChunk, stderrChunk events
  * **Lifecycle Hooks:**
    * Jest: onRunStart, onTestStart, onTestCaseResult, onTestResult, onRunComplete
    * Vitest: onInit, onPathsCollected, onTestFileStart, onTestFileResult, onFinished
  * **Fallback Processing:** Both adapters handle edge cases where test cases aren't reported individually
  * **Error Resilience:** All IPC operations use .catch() to prevent test runner crashes
  * **Startup Preamble:** Log adapter version and configuration for debugging

## 4. Console Output Capture Strategy

### Design Decision: Reporter Strategy

- **Jest**: 3pio does NOT include Jest's default reporter to avoid duplicate and redundant output
- **Vitest**: 3pio DOES include Vitest's default reporter to provide visual feedback during test execution
- This difference reflects the different architectures and user expectations of each test runner

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

## 5. Test Runner Abstraction

The system uses a strategy pattern for test runner support:

### 5.1. TestRunnerDefinition Interface
* **matches():** Determines if command matches this test runner
* **getTestFiles():** Returns list of test files (or empty for dynamic discovery)
* **buildMainCommand():** Injects adapter into command arguments
* **getAdapterFileName():** Returns adapter file (jest.js or vitest.js)
* **interpretExitCode():** Maps exit codes to success/failure/error

### 5.2. TestRunnerManager
* **Static registry:** TEST_RUNNERS object with Jest and Vitest definitions
* **detect():** Identifies test runner from command and package.json
* **getDefinition():** Returns TestRunnerDefinition for a runner
* **getParser():** Returns OutputParser for a runner

### 5.3. OutputParser
* **Base class:** Abstract OutputParser with common parsing logic
* **Implementations:** JestOutputParser and VitestOutputParser
* **parseTestOutput():** Extracts test boundaries and associates output with files

## 6. Data Flow (Sequence of Events)

A typical run of `3pio npx vitest run` proceeds as follows:

1. **User** executes the command
2. **CLI Orchestrator** parses arguments and uses **TestRunnerManager** to detect Vitest
3. **Orchestrator** calls VitestDefinition.getTestFiles() for test discovery (may return empty for dynamic mode)
4. **Orchestrator** generates run ID with timestamp and Star Wars character name
5. **Orchestrator** creates run directory and IPC file
6. **Report Manager** initializes with optional test file list
7. **Orchestrator** prints minimal preamble (report path and start message)
8. **Orchestrator** spawns vitest with injected adapter using VitestDefinition.buildMainCommand()
9. **Vitest Adapter** initializes, reads THREEPIO_IPC_PATH, patches stdout/stderr
10. **Adapter** sends testFileStart event when test file begins
11. **Adapter** sends testCase events for each individual test with results
12. **Adapter** captures console output and sends stdoutChunk/stderrChunk events
13. **Report Manager** receives events via IPC:
    * Updates test case statuses in memory
    * Collects output in per-file Maps
    * Schedules debounced report writes
14. **Adapter** sends testFileResult when file completes
15. **Report Manager** updates file status and counters
16. Process repeats for all test files (discovered statically or dynamically)
17. **Orchestrator** calls reportManager.finalize():
    * Writes individual test logs from Maps
    * Updates final report with all test cases
    * Closes output.log
18. **Orchestrator** cleans up IPC and exits with test runner's exit code
