# Component Design: CLI Orchestrator

## 1. Core Purpose

The CLI Orchestrator is the main entry point and central controller for the 3pio application. It manages the entire test run lifecycle, from command parsing to report finalization, while coordinating between test runners, adapters, and report generation.

## 2. Key Responsibilities

### Command Processing
- Parse user commands using commander.js
- Detect test runner using TestRunnerManager
- Support both explicit runners (jest, vitest) and abstract commands (npm test)

### Test Discovery
- Static discovery via TestRunnerDefinition.getTestFiles()
- Dynamic discovery for commands that don't provide file lists upfront
- Gracefully handle cases where no files are discovered initially

### Run Management
- Generate unique run IDs with ISO8601 timestamps and human-memorable names
- Create run directories and IPC communication channels
- Initialize Report and IPC managers
- Execute test commands with adapter injection

### Output Handling
- Minimal console output for context efficiency
- Capture all stdout/stderr to output.log
- Process IPC events from adapters
- Mirror test runner exit codes

## 3. Sequence of Operations

1. **Parse Arguments:** Extract run command and arguments from user input
2. **Detect Test Runner:** Use TestRunnerManager.detect() to identify Jest or Vitest
3. **Test Discovery:** Call TestRunnerDefinition.getTestFiles() (may return empty for dynamic mode)
4. **Generate Run ID:** Create timestamp + human memorable name
5. **Initialize Infrastructure:**
   - Create run directory at `.3pio/runs/[runId]/`
   - Create IPC file at `.3pio/ipc/[runId].jsonl`
   - Set THREEPIO_IPC_PATH environment variable
6. **Initialize Managers:**
   - Create IPCManager with IPC file path
   - Create ReportManager with run ID, command, and OutputParser
   - Initialize report with discovered test files (or empty for dynamic)
7. **Print Minimal Preamble:**
   - Show report path
   - Display "Beginning test execution now..."
8. **Start IPC Monitoring:** Begin watching for adapter events
9. **Execute Test Command:**
   - Use TestRunnerDefinition.buildMainCommand() to inject adapter
   - Spawn process with zx, capturing output to output.log
   - Let stdout/stderr flow to console naturally
10. **Process Events:** Handle IPC events as they arrive, updating report state
11. **Finalize:**
    - Wait 1 second for final adapter events
    - Call reportManager.finalize() to write final reports
    - Clean up IPC resources
    - Exit with test runner's exit code

## 4. Component Integration

### TestRunnerManager
- Provides test runner detection via `detect(args, packageJson)`
- Returns TestRunnerDefinition for command building
- Returns OutputParser for report generation

### TestRunnerDefinition
- Interface for runner-specific behavior
- Methods: matches(), getTestFiles(), buildMainCommand(), getAdapterFileName()
- Implementations: JestDefinition, VitestDefinition

### ReportManager
- Handles all report file I/O
- Processes IPC events to update test state
- Supports dynamic test file registration
- Manages test case level reporting

### IPCManager
- File-based communication with adapters
- Event watching and processing
- Cleanup and resource management

## 5. Configuration

### Environment Variables
- `THREEPIO_IPC_PATH`: Path to IPC communication file
- `THREEPIO_DEBUG`: Enable debug logging when set to "1"

### Adapter Injection
- Jest: `--reporters [absolutePath]` (no default reporter - clean single output)
- Vitest: `--reporter default --reporter [absolutePath]` (includes default for user visibility)
- Paths resolved to absolute using `path.join(__dirname, adapter)`
- Design choice: Jest omits default to avoid duplicate output, Vitest includes it for better user experience

### Run ID Generation
Combines two components for unique, memorable identifiers:
- ISO8601 timestamp with special characters removed
- Memorable name using adjectives and Star Wars character names
- Format: `[timestamp]-[adjective]-[character]`
- Example: `2025-09-09T111224921Z-revolutionary-chewbacca`

## 6. Error Handling

### Detection Failures
- Unknown test runner: Show supported runners and exit
- Package.json not found: Continue if possible, error if needed

### Execution Failures
- Test runner not found: Exit with error message
- IPC creation failure: Log error and exit
- Adapter communication failure: Continue but log warnings

### Graceful Shutdown
- Always finalize reports if possible
- Clean up IPC resources
- Mirror test runner exit codes

## 7. Logging System

### Logger Integration
- Structured logging with timestamp, level, component, and data
- Debug logs at `.3pio/debug.log`
- Lifecycle events for major operations
- Decision logging for test runner detection

### Log Levels
- `DEBUG`: Detailed debugging information (only with THREEPIO_DEBUG=1)
- `INFO`: General operational information
- `WARN`: Warning conditions that don't prevent operation
- `ERROR`: Error conditions requiring attention

## 8. Testing Strategy

### Unit Tests
- Argument parsing with various command formats
- Test runner detection logic with mock package.json
- Run ID generation format and uniqueness
- Command modification for adapter injection

### Integration Tests
- Full flow with mock TestRunnerManager and ReportManager
- IPC event processing pipeline
- Error handling and cleanup scenarios
- Dynamic vs static test discovery modes

### End-to-End Tests
- Complete runs against sample Jest/Vitest projects
- Verify console output format
- Check report generation accuracy
- Validate exit code mirroring

## 9. Future Considerations

- Support for additional test runners (Mocha, Jasmine)
- Parallel test execution tracking
- Real-time progress indicators
- Custom reporter configurations
- Watch mode support
