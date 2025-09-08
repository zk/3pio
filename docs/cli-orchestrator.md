# Component Design: CLI Orchestrator

## 1. Core Purpose

The CLI Orchestrator is the main entry point and central controller for the 3pio application. It is responsible for parsing user input, managing the entire lifecycle of a test run, coordinating the other components, and handling all top-level process management.

## 2. Sequence of Operations

The orchestrator executes the following sequence for a typical 3pio run command:

1. **Parse Arguments:** Use commander.js to parse the user's command (e.g., vitest --ui).
2. **Detect Test Runner:** If the command is an abstraction like npm test, inspect package.json to determine the underlying runner (Jest or Vitest).
3. **Perform Dry Run:** Execute the appropriate dry run command (jest --listTests or vitest list) to get the list of test files to be run.
4. **Initialize Run:**
   * Create the unique run directory (e.g., /.3pio/runs/20250907T131600Z/).
   * Create the unique IPC event file (e.g., /.3pio/ipc/20250907T131600Z.jsonl).
5. **Initialize Report:** Instantiate the ReportManager and call `await reportManager.initialize(testFiles)` to create the initial test-run.md with all tests marked PENDING.
6. **Print Preamble:** Generate and print the formatted preamble to the console.
7. **Start IPC Listening:** Use the IPCManager to start watching the IPC event file for new events. Process events sequentially through a queue to avoid concurrent file writes. Delegate received events to `await reportManager.handleEvent(event)`.
8. **Execute Main Command:**
   * Programmatically modify the user's command to inject the correct adapter flags with absolute paths.
   * Use zx with explicit environment: `$({ env })`sh -c ${modifiedCommand}``
   * Pipe the child process's stdout and stderr directly to the user's console.
   * Handle zx exceptions to extract exit codes from failed processes.
9. **Await Completion:** Wait for the zx process to exit.
10. **Finalize and Clean Up:**
    * Call `await reportManager.finalize(exitCode)` to ensure the final report is written to disk and individual logs are parsed.
    * Call `await ipcManager.cleanup()` to stop watching and clean up resources.
    * Exit the 3pio process with the same exit code as the child process.

## 3. Key Dependencies

* **commander.js:** For robust command-line argument parsing.
* **zx:** For executing the user's command, providing reliable handling of shell syntax like pipes and redirects.
* **ReportManager:** To delegate all report file I/O.
* **IPCManager:** To listen for events from the test runner adapter.

## 4. Configuration and Environment

The orchestrator is responsible for passing the path to the unique IPC event file to the adapter running in the child process. 

* **Environment Variable:** THREEPIO_IPC_PATH
* **Implementation:** 
  1. Sets `process.env.THREEPIO_IPC_PATH = this.ipcPath` in the main process.
  2. Explicitly passes environment to child process: `$({ env: { ...process.env, THREEPIO_IPC_PATH: this.ipcPath } })`
  3. Uses absolute adapter paths: `path.join(__dirname, 'jest.js')` or `path.join(__dirname, 'vitest.js')`
* **Command Modification:** Appends reporter flags like `--reporters default --reporters ${adapterPath}` for Jest or `--reporter default --reporter ${adapterPath}` for Vitest.

## 5. Failure Modes

* **Invalid User Command:** The user provides an unknown command or invalid flags.
* **Test Runner Detection Fails:** The orchestrator cannot determine the test runner from package.json for an npm script.
* **Dry Run Fails:** The dry run command (e.g., jest --listTests) exits with an error.
* **Child Process Fails to Spawn:** The test runner command (e.g., vitest) does not exist or cannot be executed.
* **IPC Channel Fails:** The IPC event file cannot be created or watched.
* **3pio Process is Force-Killed:** The user sends a SIGKILL to the main 3pio process, preventing graceful cleanup.

## 6. Testing Strategy

* **Unit Tests:**
  * Test the argument parsing logic to ensure flags and commands are correctly identified.
  * Test the test runner detection logic with mock package.json files.
  * Test the command modification logic to ensure adapter flags are injected correctly for various user inputs.
* **Integration Tests:**
  * Write tests that execute the full sequence against mock ReportManager and IPCManager components to verify the coordination logic.
  * Test the failure modes by simulating errors (e.g., a dry run command that throws an error) and asserting that the orchestrator exits gracefully with the correct error code and message, and that no report files are created.
* **End-to-End Tests:**
  * Run the compiled 3pio binary against a small, sample Jest project and a sample Vitest project.
  * Assert that the correct preamble is printed, the test runner's output is piped, the final report files (test-run.md and .log files) are generated correctly, and the final exit code is mirrored.
