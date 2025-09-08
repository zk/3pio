# Project Plan: 3pio - The AI Test Protocol Droid

### 1. Core Problem

Current test libraries are optimized for human developers who can watch real-time execution and interpret verbose, colorful output. AI coding agents require a different approach: one that is context-efficient, machine-readable, persistent, and queryable using standard tools. 3pio acts as a "protocol droid," translating traditional test framework output into a format optimized for AI agents.

### 2. Key Goals & Features (Version 1)

* **Persistent Test Sessions:** Test results are saved to the file system, surviving across conversational turns and development sessions.
* **Context-Efficient Output:** 3pio provides a persistent, structured, file-based record for deep analysis, while preserving the raw test runner output for real-time observation.
* **Adapter Architecture:** 3pio wraps existing test runners (e.g., Jest, Vitest) without requiring changes to existing test files.
* **Structured, Searchable Logs:** All persistent output is structured in Markdown and delimited text files, allowing an AI agent to parse results and search for details using standard shell commands (cat, grep, sed).
* **Personality:** All communication is helpful, anxious, and precise, providing character without corrupting the structured data payload.

### 3. High-Level Architecture

3pio's architecture is a focused pipeline that executes tests and translates their results into two distinct, structured outputs.

[Test Runner (Jest/Vitest)] -> [3pio Adapter] -> [Dual Output Stream]

The dual output consists of:

1. **Real-Time stdout Stream:** A preamble from 3pio followed by the raw, direct output from the test runner.
2. **Persistent File Log:** For a complete, detailed, and permanent record, generated in near real-time.

The agent interacts with 3pio primarily through a Command Line Interface (CLI).

### 4. Technology Stack

* **Language:** TypeScript
* **Runtime:** Node.js
* **Build Tool:** esbuild
* **Command Execution:** zx
* **Rationale:** TypeScript is the optimal choice due to its native integration with the JavaScript ecosystem. This allows 3pio to interface directly with test runners like Jest and Vitest using their custom reporter APIs. esbuild is selected for its exceptional speed, ensuring a fast development cycle. To robustly handle complex user-provided shell commands, including pipes and redirects, zx will be used. This avoids the need to build a brittle and insecure custom shell parser.

### 5. Distribution

* **Package Manager:** npm
* **Package Name:** @3pio/core (planned)
* **Installation:** The CLI will be installed globally via a single command. This package contains the main 3pio executable and all supported test runner adapters.
  npm install -g @3pio/core

* **Execution:** Once installed, the 3pio command will be available in the user's path.

### 6. Component Breakdown

#### 6.1. CLI

* **Command:** 3pio run \<underlying\_test\_command\> (e.g., 3pio run vitest)
* **Function:** First, performs a "dry run" to determine the list of test files. Then, prints a preamble, intelligently injects the required reporter flags into the user's command, executes the modified command using zx, pipes its output directly to stdout/stderr, and orchestrates the generation of the persistent report files.

##### 6.1.1. Preamble Generation Strategy (Dry Run)

To display the list of test files in the preamble *before* execution, the CLI will first perform a "dry run" to gather the file paths. This is done by invoking the test runner with a specific flag or command that lists tests without running them.

* **Test Runner Detection for npm Scripts:** If the command is an npm script (e.g., npm test), the CLI will first inspect the project's package.json. It will parse the scripts section to find the underlying command and check the project's dependencies and devDependencies to determine which test runner (e.g., Jest or Vitest) is being used. This detection step determines which of the following dry run strategies to apply and which adapter to inject for the main test run.
* **For Jest:** The CLI will append the --listTests flag to the user's command. This outputs a JSON array of test file paths, which will be parsed.
* **For Vitest:** The CLI will replace the run keyword (if present) in the user's command with list. This outputs a newline-separated list of test file paths.

After the dry run is complete and the preamble has been printed, the CLI will then execute the user's original command (with the 3pio reporter injected).

##### 6.1.2. Command Handling Strategy

To provide a seamless user experience, 3pio will use the zx library to execute the user's command. This allows for full support of standard shell syntax, including pipes (|), redirects (\>), and command chaining (&&). The 3pio CLI will pass the entire command string to zx for parsing and execution, ensuring that commands behave exactly as they would in a standard terminal while allowing 3pio to capture the necessary output and results.

##### 6.1.3. Error Handling and Exit Codes

To ensure robust and predictable behavior, especially in automated environments, the CLI will adhere to the following rules:

* **Exit Code Mirroring:** The 3pio process will capture the exit code from the underlying test runner process and will exit with the same code. For example, if jest fails tests and exits with 1, 3pio will also exit with 1\. This ensures compatibility with CI/CD pipelines and other scripts.
* **Startup Failures:** If 3pio fails at any point before the main test runner process successfully starts (e.g., the dry run command fails, the test runner command is not found), it will **not** generate any report files or directories. It will print a clear error message to stderr and exit with a non-zero status code.

#### 6.2. Test Runner Adapters

* **Responsibility:** The adapters are bundled within the main @3pio/core package. They are activated via command-line flags injected by the 3pio CLI wrapper, not through user configuration files. They interface with specific test runners to capture structured data and transmit it back to the main CLI process via the IPC mechanism.
* **Behavior:** The adapters are designed to be **"silent"**. They do not write any output to stdout or stderr. Their sole purpose is to capture test result data from the runner's API and transmit it via the IPC channel. This ensures that 3pio does not interfere with the console output from the user's default or pre-configured reporters.
* **Initial Targets:** Jest, Vitest.
* **Usage (Automatic via CLI):** The 3pio command will automatically add the correct flag to the user's command. This provides a "zero-config" experience.
* **Injection Logic:** To remain unobtrusive, the 3pio adapter is always appended to the end of the reporter chain.
  * **For Jest:** The CLI will inject the --reporters flag. It will detect if the user has already specified reporters via the command line. If so, it appends @3pio/core/jest. If not, it sets the reporters to \['default', '@3pio/core/jest'\] to ensure both the standard console output and the 3pio report are generated. For example, 3pio run jest --watch would be transformed and executed as jest --watch --reporters default @3pio/core/jest.
  * **For Vitest:** The logic is similar. The CLI injects the --reporter flag, appending @3pio/core/vitest to any existing reporters or setting it alongside the default reporter if none are provided by the user. For example, 3pio run vitest --ui would be transformed and executed as vitest --ui --reporter default --reporter @3pio/core/vitest.

#### 6.3. Real-Time stdout Output

This stream provides context to the agent at the start of the run, followed by the familiar, real-time output from the test runner itself.

* **Example 1: Short List (\<= 10 files)**
  Greetings\! I will now execute the test command:
  \`vitest run src/auth\`

  Full report: .3pio/runs/20250907T210400Z/test-run.md

  The following 3 test files will be run:
  - src/tests/auth/login.test.js
  - src/tests/auth/logout.test.js
  - src/tests/auth/signup.test.js

  Beginning test execution now...

* **Example 2: Medium List (\> 10 files)**
  Greetings\! I will now execute the test command:
  \`vitest run src/\`

  Full report: .3pio/runs/20250907T210400Z/test-run.md

  The following 15 test files will be run:
  - src/tests/auth/login.test.js
  - src/tests/auth/logout.test.js
  - src/tests/auth/signup.test.js
  - src/tests/cart/add.test.js
  - src/tests/cart/remove.test.js
  - src/tests/cart/update.test.js
  - src/tests/checkout/payment.test.js
  - src/tests/checkout/shipping.test.js
  - src/tests/products/details.test.js
  - src/tests/products/list.test.js
  - ...and 5 more.

  Beginning test execution now...

* **Example 3: Long List (\> 25 files)**
  Greetings\! I will now execute the test command:
  \`npm test\`

  Full report: .3pio/runs/20250907T210400Z/test-run.md

  Running 32 total test files.

  Breakdown by directory:
  - src/tests/auth/ (4 files)
  - src/tests/cart/ (8 files)
  - src/tests/checkout/ (12 files)
    - payment.test.js
    - shipping.test.js
    - tax-calculation.test.js
    - ...and 9 more.
  - src/tests/products/ (8 files)

  Beginning test execution now...

* **Raw Output:** After the preamble, 3pio directly pipes the stdout and stderr from the underlying test runner.

#### 6.4. Persistent File Output

This is the complete, official record of the test run, designed for detailed analysis.

* **Directory Structure:** A new directory is created for each run to ensure isolation.
  * /.3pio/runs/20250907T210400Z/
* **Initial Report State:** Immediately after the dry run is complete and the list of test files is known, the CLI will create the test-run.md file. The file will be populated with the header and the "Test Files" table, with every discovered test file listed with an initial status of PENDING. This ensures that a record of the intended run exists even if the main process crashes immediately upon starting.
* **File Structure:**
  * **test-run.md**: A Markdown file that acts as the main entry point and summary. The file's contents are updated periodically during the run and finalized upon completion, managed by the debounced write strategy (see below).
    \# 3pio Test Run Summary
    - \*\*Timestamp:\*\* 2025-09-07T21:04:00Z
    - \*\*Status:\*\* Complete (updated 2025-09-07T21:05:15Z)
    - \*\*Arguments:\*\* \`vitest run src/\`

    \#\# Summary (updated 2025-09-07T21:05:15Z)
    - \*\*Total Files:\*\* 3
    - \*\*Files Passed:\*\* 2
    - \*\*Files Failed:\*\* 1

    \#\# Test Files
    | Status | File | Log File |
    | --- | --- | --- |
    | PASS | \`src/tests/auth/login.test.js\` | \[details\](./logs/login.test.js.log) |
    | FAIL | \`src/tests/auth/logout.test.js\` | \[details\](./logs/logout.test.js.log) |
    | PASS | \`src/tests/cart/add.test.js\` | \[details\](./logs/add.test.js.log) |

  * **Individual Log Files (e.g., logs/src\_tests\_auth\_logout.test.js.log)**: A log file for each test, created as soon as the test file completes. The filename is a sanitized version of the test file's relative path to prevent collisions. It contains a simple preamble with metadata followed by the raw stdout and stderr captured during that test file's execution.
    File: src/tests/auth/logout.test.js
    Timestamp: 2025-09-07T21:04:05Z
    This file represents output from a test run for the listed test file. See \`../test-run.md\`.
    ---
     FAIL  src/tests/auth/logout.test.js
      ❯ User Authentication \> should correctly log out a user
        AssertionError: expected true to be false
         at /path/to/project/src/tests/auth/logout.test.js:25:32

#### 6.5. Real-Time Update Strategy (Debounced Writes)

To ensure high performance and prevent file corruption from frequent writes, the test-run.md file is not updated for every single test result. Instead, a debounced write strategy is used:

1. **In-Memory State:** A complete representation of the test-run.md report is held in memory.
2. **Immediate Update:** When a test result is received via IPC, only the fast in-memory state is updated.
3. **Batched File Writes:** The in-memory state is written to the actual test-run.md file on disk periodically (e.g., every 250ms). This batches potentially hundreds of updates into a single, efficient file write.
4. **Finalization:** A final, guaranteed write occurs when the entire test run is complete to ensure the report on disk is 100% accurate.

#### 6.6. Inter-Process Communication (IPC)

A file-based IPC mechanism is used for the adapter to send a stream of structured events back to the main 3pio CLI process. This allows for real-time streaming of log output.

1. **IPC File Creation:** The main CLI process creates a dedicated IPC directory at /.3pio/ipc/. Inside this directory, it creates a unique, temporary event log file for the current run (e.g., /.3pio/ipc/20250907T210400Z.jsonl).
2. **Path Communication:** The absolute path to this IPC file is passed as a reporter option or an environment variable to the child process.
3. **Event Writing:** The adapter, running within the test runner, receives this file path. It will write different event types to the IPC file as they occur. It hooks into the runner's stdout/stderr streams and immediately writes chunk events. When a test file completes, it writes the final result event.
4. **Event Watching:** The main 3pio CLI process watches the IPC file for changes. When a new line is appended, it reads the line, parses the JSON event, and takes the appropriate action (e.g., appends a chunk to a log file, updates the in-memory state for the summary report).
5. **Cleanup:** After the test run is complete, the main CLI process is responsible for deleting the temporary IPC file.

##### 6.6.1. IPC Event Schema

The following JSON schema defines the events that will be written to the ipc.jsonl file.

* **For streaming stdout output:**
  {
    "eventType": "stdoutChunk",
    "payload": {
      "filePath": "src/tests/auth/login.test.js",
      "chunk": "..."
    }
  }

* **For streaming stderr output:**
  {
    "eventType": "stderrChunk",
    "payload": {
      "filePath": "src/tests/auth/login.test.js",
      "chunk": "..."
    }
  }

* **For the final test file result:**
  {
    "eventType": "testFileResult",
    "payload": {
      "filePath": "src/tests/auth/login.test.js",
      "status": "PASS"
    }
  }

### 7. Agent Workflow Example

1. **Agent Executes Tests:** The agent initiates the process by running a command like 3pio run vitest.
2. **Agent Parses Preamble:** The agent immediately reads the initial stdout from 3pio to parse the path to the main report file, test-run.md, and stores it for later use.
3. **Agent Monitors for Completion:** The agent waits for the 3pio child process to exit. This signals that the test run is complete and the final report has been written to disk.
4. **Agent Reads the Summary Report:** Once the process is complete, the agent reads the full contents of the test-run.md file.
5. **Agent Analyzes Results:** The agent parses the Markdown table under the \#\# Test Files section. It iterates through the rows, looking for any test file with a FAIL status.
6. **Agent Retrieves Failure Details:** For each failed test, the agent extracts the relative path to the individual log file from the third column of the table (e.g., ./logs/src\_tests\_auth\_logout.test.js.log). It then reads the contents of that specific log file to get the raw stdout and stderr, including the detailed error message and stack trace.
7. **Agent Begins Debugging:** Using the isolated and complete error information, the agent has the full context required to begin its analysis, form a hypothesis about the cause of the failure, and attempt to modify the source code to fix the bug.

### 8. Future Work

* **Semantic Query Engine:** A natural language interface that would allow for more intuitive querying of test results. Instead of relying on grep, an agent could use a command like 3pio query "why are the auth tests failing?". This would involve creating embeddings of test names, file contents, and error messages, storing them in a vector database, and translating the natural language query into a similarity search.
* **Interactive Mode:** An enhanced run mode (3pio run --interactive) that, after an initial run, would present the agent with a structured prompt to perform common follow-up actions, such as "Rerun all failed tests," "Rerun a specific file," or "Exit." This would create a faster and more efficient feedback loop for the agent.
* **IDE Integration:** A companion VS Code extension that could parse the .3pio directory and render the test-run.md report in a rich, interactive UI for the human developer overseeing the agent. This would provide a visual bridge between the agent's work and the developer's environment, showing test status, providing clickable links to log files, and offering buttons to rerun tests.
* **Report Garbage Collection:** A mechanism to automatically clean up old run directories to prevent them from consuming excessive disk space. This could be a separate command like 3pio runs clean and could be configured to keep the last N runs or runs newer than a certain date.
* **Broader JavaScript Support:** Adding support for other popular JavaScript test runners like Mocha.
* **Multi-Language Support:** Extending the 3pio model beyond the JavaScript ecosystem to support other languages and their native testing frameworks (e.g., Pytest for Python, Cargo Test for Rust, JUnit for Java).
* **Watch Mode Support:** Providing robust support for interactive "watch" modes (e.g., jest --watch). This would involve a different lifecycle management strategy, where the test-run.md file is cleared and regenerated for each subsequent run within the same 3pio session.
