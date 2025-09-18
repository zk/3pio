package orchestrator

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/zk/3pio/internal/adapters"
	"github.com/zk/3pio/internal/ipc"
	"github.com/zk/3pio/internal/logger"
	"github.com/zk/3pio/internal/report"
	"github.com/zk/3pio/internal/runner"
	"github.com/zk/3pio/internal/runner/definitions"
)

// Orchestrator manages the test execution lifecycle
type Orchestrator struct {
	runnerManager *runner.Manager
	reportManager *report.Manager
	ipcManager    *ipc.Manager
	logger        Logger

	runID          string
	runDir         string
	ipcPath        string
	command        []string
	exitCode       int
	detectedRunner string // Track which test runner was detected

	// Console output state
	startTime        time.Time
	passedGroups     int
	failedGroups     int
	skippedGroups    int
	totalGroups      int
	passedTests      int                  // Track actual test cases
	failedTests      int                  // Track actual test cases
	skippedTests     int                  // Track actual test cases
	totalTests       int                  // Track actual test cases
	displayedGroups  map[string]bool      // Track which groups we've already displayed
	lastCollected    int                  // Track last collection count to avoid duplicates
	groupStartTimes  map[string]time.Time // Track start time for each group
	groupFailedTests map[string][]string  // Track failed test names by group
	completedGroups  map[string]bool      // Track which groups have shown their final PASS/FAIL status
	noTestGroups     map[string]bool      // Track packages with no test files (Go specific)

	// Error capture
	stderrCapture strings.Builder

	// Cargo test support
	cargoProcessExited chan<- struct{}
}

// TailReader implements io.Reader that tails a file until signaled to stop
type TailReader struct {
	file          io.ReadCloser
	processExited <-chan struct{}
	logger        Logger
}

func (t *TailReader) Read(p []byte) (n int, err error) {
	for {
		n, err = t.file.Read(p)
		if n > 0 {
			return n, nil
		}

		// Check if process has exited
		select {
		case <-t.processExited:
			// Process exited, return EOF
			return 0, io.EOF
		default:
			// Process still running, wait for more data
			if err == io.EOF {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			return n, err
		}
	}
}

// Logger interface for logging
type Logger interface {
	Debug(format string, args ...interface{})
	Error(format string, args ...interface{})
	Info(format string, args ...interface{})
}

// Config holds orchestrator configuration
type Config struct {
	Command []string
	Logger  Logger
}

// New creates a new orchestrator
func New(config Config) (*Orchestrator, error) {
	if config.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	// Cast logger to FileLogger for runner manager
	// In tests, we use TestLogger which doesn't need file operations
	var runnerMgr *runner.Manager
	if fileLogger, ok := config.Logger.(*logger.FileLogger); ok {
		runnerMgr = runner.NewManager(fileLogger)
	} else if testLogger, ok := config.Logger.(*logger.TestLogger); ok {
		// For tests, create a temporary FileLogger for the runner manager
		// The test logger will still capture all logs via the orchestrator's logger field
		tempLogger, _ := logger.NewFileLogger()
		runnerMgr = runner.NewManager(tempLogger)
		_ = testLogger // avoid unused variable warning
	} else {
		return nil, fmt.Errorf("logger must be a *logger.FileLogger or *logger.TestLogger")
	}

	return &Orchestrator{
		runnerManager:    runnerMgr,
		logger:           config.Logger,
		command:          config.Command,
		displayedGroups:  make(map[string]bool),
		groupStartTimes:  make(map[string]time.Time),
		groupFailedTests: make(map[string][]string),
		completedGroups:  make(map[string]bool),
		noTestGroups:     make(map[string]bool),
	}, nil
}

// Close closes the orchestrator and cleans up resources
func (o *Orchestrator) Close() error {
	if o.runnerManager != nil {
		return o.runnerManager.Close()
	}
	return nil
}

// Run executes the test command with 3pio instrumentation
func (o *Orchestrator) Run() error {
	// Ensure cleanup on exit
	defer func() {
		_ = o.Close()
	}()

	// Generate run ID
	o.runID = generateRunID()
	o.runDir = filepath.Join(".3pio", "runs", o.runID)

	// Setup IPC in the run directory (do this early so it's available even if runner detection fails)
	o.ipcPath = filepath.Join(o.runDir, "ipc.jsonl")

	// Print test run header with metadata
	testCommand := strings.Join(o.command, " ")
	currentTime := time.Now().Format(time.RFC3339)
	trunDir := o.runDir
	fullReport := "$trun_dir/test-run.md"

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "unknown"
	}

	fmt.Println("---")
	fmt.Printf("current_time: %s\n", currentTime)
	fmt.Printf("cwd: %s\n", cwd)
	fmt.Printf("test_command: `%s`\n", testCommand)
	fmt.Printf("trun_dir: %s\n", trunDir)
	fmt.Printf("full_report: %s\n", fullReport)
	fmt.Println("---")
	fmt.Println()
	fmt.Println("Test execution starting, no output until test results.")
	fmt.Println()

	// Detect test runner
	runnerDef, err := o.runnerManager.Detect(o.command)
	if err != nil {
		return fmt.Errorf("failed to detect test runner: %w", err)
	}

	// Create IPC manager
	o.ipcManager, err = ipc.NewManager(o.ipcPath, o.logger)
	if err != nil {
		return fmt.Errorf("failed to create IPC manager: %w", err)
	}
	// Cleanup will be called explicitly later, not deferred

	// Start watching for events
	if err := o.ipcManager.WatchEvents(); err != nil {
		return fmt.Errorf("failed to start IPC watcher: %w", err)
	}

	// Get output parser for the runner
	parser := o.runnerManager.GetParser(runnerDef.GetAdapterFileName())

	// Determine runner name from adapter file
	adapterFile := runnerDef.GetAdapterFileName()
	var detectedRunner string
	switch adapterFile {
	case "jest.js":
		detectedRunner = "jest"
	case "vitest.js":
		detectedRunner = "vitest"
	case "pytest_adapter.py":
		detectedRunner = "pytest"
	case "cypress.js":
		detectedRunner = "cypress"
	case "mocha.js":
		detectedRunner = "mocha"
	case "":
		// Native runner - determine which one based on the underlying definition
		if nativeRunner, ok := runnerDef.(runner.NativeRunner); ok {
			nativeDef := nativeRunner.GetNativeDefinition()
			o.logger.Debug("Native definition type: %T", nativeDef)
			switch nativeDef.(type) {
			case *definitions.GoTestDefinition:
				detectedRunner = "go test"
				o.logger.Debug("Detected as go test")
			case *definitions.CargoTestDefinition:
				detectedRunner = "cargo test"
				o.logger.Debug("Detected as cargo test")
			case *definitions.NextestDefinition:
				detectedRunner = "cargo nextest"
				o.logger.Debug("Detected as cargo nextest")
			default:
				detectedRunner = fmt.Sprintf("unknown native (%T)", nativeDef)
				o.logger.Debug("Unknown native type: %T", nativeDef)
			}
		} else {
			detectedRunner = "unknown native"
			o.logger.Debug("Not a native runner")
		}
	default:
		detectedRunner = "unknown"
	}

	// Store the detected runner
	o.detectedRunner = detectedRunner

	// Build modified command for logging
	var modifiedCommand string
	if adapterFile == "" {
		// Native runner
		testCommandSlice := runnerDef.BuildCommand(o.command, "")
		modifiedCommand = strings.Join(testCommandSlice, " ")
	} else {
		// Will be set after adapter extraction
		modifiedCommand = "" // Placeholder, will update after adapter extraction
	}

	// Create report manager
	o.reportManager, err = report.NewManager(o.runDir, parser, o.logger, detectedRunner, modifiedCommand)
	if err != nil {
		return fmt.Errorf("failed to create report manager: %w", err)
	}
	// Ensure report manager is finalized even on early return
	defer func() {
		if o.reportManager != nil {
			_ = o.reportManager.Finalize(o.exitCode, "")
		}
	}()

	// Initialize report
	args := strings.Join(o.command, " ")
	if err := o.reportManager.Initialize(args); err != nil {
		return fmt.Errorf("failed to initialize report: %w", err)
	}

	// Check if this is a native runner (like Go test)
	var testCommandSlice []string
	var isNativeRunner bool
	var nativeDef interface{}

	// Check if adapter is needed (empty adapter name means native runner)
	adapterFileName := runnerDef.GetAdapterFileName()
	if adapterFileName == "" {
		// Native runner - no adapter needed (e.g., Go test, cargo test, nextest)
		isNativeRunner = true
		// Try to get the native definition using type assertion
		switch wrapper := runnerDef.(type) {
		case *definitions.GoTestWrapper:
			nativeDef = wrapper.GoTestDefinition
		case *definitions.CargoTestWrapper:
			nativeDef = wrapper.CargoTestDefinition
		case *definitions.NextestWrapper:
			nativeDef = wrapper.NextestDefinition
		}
		testCommandSlice = runnerDef.BuildCommand(o.command, "")
		o.logger.Debug("Using native runner for: %v", testCommandSlice)
	} else {
		// Traditional adapter-based runner
		adapterPath, err := o.extractAdapter(adapterFileName)
		if err != nil {
			return fmt.Errorf("failed to extract adapter: %w", err)
		}
		testCommandSlice = runnerDef.BuildCommand(o.command, adapterPath)
		o.logger.Debug("Adapter path: %s", adapterPath)

		// Update modified command now that we have the actual command
		modifiedCommand = strings.Join(testCommandSlice, " ")
		o.reportManager.UpdateModifiedCommand(modifiedCommand)
	}

	o.logger.Debug("Executing command: %v", testCommandSlice)
	o.logger.Debug("IPC path: %s", o.ipcPath)

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Create command
	cmd := exec.Command(testCommandSlice[0], testCommandSlice[1:]...)

	// Set working directory to current directory (where 3pio was invoked)
	if wd, err := os.Getwd(); err == nil {
		cmd.Dir = wd
		o.logger.Debug("Set working directory to: %s", wd)
	} else {
		o.logger.Error("Failed to get current working directory: %v", err)
	}

	// Set environment
	cmd.Env = append(os.Environ(), fmt.Sprintf("THREEPIO_IPC_PATH=%s", o.ipcPath))

	// Add RUSTC_BOOTSTRAP=1 for cargo test to enable JSON output
	if len(o.command) >= 2 && o.command[0] == "cargo" && o.command[1] == "test" {
		cmd.Env = append(cmd.Env, "RUSTC_BOOTSTRAP=1")
		o.logger.Debug("Added RUSTC_BOOTSTRAP=1 for cargo test JSON output")
	}

	// Add NEXTEST_EXPERIMENTAL_LIBTEST_JSON=1 for cargo nextest to enable JSON output
	if len(o.command) >= 2 && o.command[0] == "cargo" && o.command[1] == "nextest" {
		cmd.Env = append(cmd.Env, "NEXTEST_EXPERIMENTAL_LIBTEST_JSON=1")
		o.logger.Debug("Added NEXTEST_EXPERIMENTAL_LIBTEST_JSON=1 for cargo nextest JSON output")
	}

	// Connect stdin to allow interactive prompts
	cmd.Stdin = os.Stdin

	// Create output.log for capturing all command output
	outputPath := filepath.Join(o.runDir, "output.log")
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	// Ensure outputFile is closed on all exit paths, but track if we closed it explicitly
	outputFileClosed := false
	defer func() {
		if !outputFileClosed {
			_ = outputFile.Sync() // Ensure file is flushed on Windows
			_ = outputFile.Close()
		}
	}()

	// Universal approach: ALL runners write directly to output.log
	// This eliminates race conditions, pipe buffer limitations, and redundant files
	o.logger.Debug("Using output.log directly for all runners")

	// Determine if stderr should be kept separate (only for Go test)
	var keepStderrSeparate bool
	if isNativeRunner {
		if _, ok := nativeDef.(*definitions.GoTestDefinition); ok {
			keepStderrSeparate = true
			o.logger.Debug("Go test detected - keeping stderr separate")
		}
	}
	if !keepStderrSeparate {
		o.logger.Debug("Combining stdout and stderr into output.log")
	}

	// Configure command output redirection directly to output.log
	cmd.Stdout = outputFile

	var stderrPipe io.ReadCloser
	if keepStderrSeparate {
		// Keep stderr separate (for Go test only)
		stderrPipe, err = cmd.StderrPipe()
		if err != nil {
			return fmt.Errorf("failed to create stderr pipe: %w", err)
		}
		o.logger.Debug("Keeping stderr separate for Go test")
	} else {
		// Redirect both stdout and stderr to output.log
		cmd.Stderr = outputFile
	}

	// Debug: Log the exact command being executed
	o.logger.Debug("Starting command: %s %v", cmd.Path, cmd.Args)
	o.logger.Debug("Working directory: %s", cmd.Dir)
	o.logger.Debug("Environment variables count: %d", len(cmd.Env))

	// Start the command
	if err := cmd.Start(); err != nil {
		o.exitCode = 1 // Set error exit code
		return fmt.Errorf("failed to start test command: %w", err)
	}

	// Record start time for duration calculation
	o.startTime = time.Now()

	// Open output.log for reading (tail -f style) only for native runners
	var tailReader *os.File
	if isNativeRunner {
		tailReader, err = os.Open(outputPath)
		if err != nil {
			return fmt.Errorf("failed to open output.log for reading: %w", err)
		}
		o.logger.Debug("Opened output.log for tailing: %s", outputPath)
	}

	// Process events and output concurrently
	var wg sync.WaitGroup
	eventsDone := make(chan struct{})

	// Process IPC events in background
	// Note: NOT part of wg since we wait for it separately via eventsDone
	go func() {
		defer close(eventsDone)
		o.processEvents()
	}()

	// Universal output handling for ALL runners
	// Create a channel to signal when the process exits
	processExited := make(chan struct{})
	o.cargoProcessExited = processExited // Used by TailReader

	// Process output from output.log for native runners
	if isNativeRunner && tailReader != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				_ = tailReader.Close()
				o.logger.Debug("Closed tail reader for native runner")
			}()

			// Create a custom reader that polls the file until process exits
			fileReader := &TailReader{
				file:          tailReader,
				processExited: processExited,
				logger:        o.logger,
			}

			// Process output through native definition (no TeeReader needed)
			if nd, ok := nativeDef.(interface {
				ProcessOutput(io.Reader, string) error
			}); ok {
				o.logger.Debug("Processing output for native runner")
				if err := nd.ProcessOutput(fileReader, o.ipcPath); err != nil {
					o.logger.Error("Failed to process native output: %v", err)
				}
			}
		}()
	}

	// Handle stderr separately if needed (Go test only)
	if stderrPipe != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Capture stderr to the error buffer
			_, _ = io.Copy(&o.stderrCapture, stderrPipe)
		}()
	}

	// Wait for command completion or signal
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	var commandErr error
	select {
	case err := <-done:
		commandErr = err
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				o.exitCode = exitErr.ExitCode()
				o.logger.Debug("Command completed with exit code: %d", exitErr.ExitCode())
			} else {
				o.exitCode = 1
				o.logger.Debug("Command completed with error: %v", err)
			}
		} else {
			o.logger.Debug("Command completed successfully")
		}
		// Command finished, signal cargo reader if it exists
		if o.cargoProcessExited != nil {
			close(o.cargoProcessExited)
			o.logger.Debug("Signaled cargo reader that process has exited")
		}
		// Wait for readers to finish processing remaining data
		o.logger.Debug("Command completed, waiting for readers to finish...")
	case sig := <-sigChan:
		o.logger.Info("Received signal: %v", sig)
		_ = cmd.Process.Kill()
		o.exitCode = 130 // Standard exit code for SIGINT
		// Signal cargo reader if it exists (same as normal completion)
		if o.cargoProcessExited != nil {
			close(o.cargoProcessExited)
			o.logger.Debug("Signaled cargo reader that process was interrupted")
		}
	}

	// Wait for output capture to complete
	wg.Wait()
	o.logger.Debug("Output capture completed")

	// Stop watching for events (this closes the Events channel and allows processEvents to exit)
	_ = o.ipcManager.Cleanup()

	// Wait for event processing to complete (channel is closed, range will exit)
	<-eventsDone
	o.logger.Debug("Event processing completed")

	o.logger.Debug("Output capture completed")

	// NOW it's safe to close the output file after all goroutines are done
	// On Windows, we need to ensure the file is fully flushed before closing
	outputFileClosed = true
	if err := outputFile.Sync(); err != nil {
		o.logger.Debug("Failed to sync output file: %v", err)
	}
	if err := outputFile.Close(); err != nil {
		o.logger.Error("Failed to close output file: %v", err)
	}

	// All goroutines should be finished at this point
	// (they were waited for via outputDone)

	// Finalize report
	var errorDetails string
	var shouldShowError bool
	if commandErr != nil {
		// Check if this is a configuration/startup error vs test failures
		// Configuration errors happen when we have very few or no test groups
		// or when the exit code suggests a setup problem
		isConfigError := o.totalGroups == 0 ||
			(o.exitCode != 0 && o.exitCode != 1 && o.totalGroups < 2) ||
			(o.passedGroups == 0 && o.failedGroups == 0 && o.exitCode != 0)

		if isConfigError {
			errorDetails = commandErr.Error()

			// Include stderr content if available for command errors
			stderrContent := strings.TrimSpace(o.stderrCapture.String())
			if stderrContent != "" {
				errorDetails = stderrContent
			}

			// For config/setup errors (non-zero exit with no tests run),
			// show the actual output instead of generic "exit status N"
			if (errorDetails == "exit status 1" || errorDetails == "exit status 2") && o.totalGroups == 0 {
				// Read first part of output.log to show actual error
				if outputContent, err := os.ReadFile(outputPath); err == nil {
					lines := strings.Split(string(outputContent), "\n")
					// Show first non-empty lines (up to 10 lines)
					var errorLines []string
					for i := 0; i < len(lines) && len(errorLines) < 10; i++ {
						if trimmed := strings.TrimSpace(lines[i]); trimmed != "" {
							errorLines = append(errorLines, lines[i])
						}
					}
					if len(errorLines) > 0 {
						errorDetails = strings.Join(errorLines, "\n")
						shouldShowError = true
					}
				}
			} else {
				shouldShowError = true
			}
		}
	}
	if err := o.reportManager.Finalize(o.exitCode, errorDetails); err != nil {
		o.logger.Error("Failed to finalize report: %v", err)
	}

	// If we didn't get GroupResult events, compute stats and display results from the report manager
	if o.totalGroups == 0 {
		o.computeStatsFromReportManager()
		o.displayFinalResults()
	}

	// Print completion message with TypeScript-style summary
	fmt.Println()

	// Print error details if command failed and we have error details
	if (commandErr != nil && errorDetails != "" && shouldShowError) ||
		(commandErr != nil && o.totalGroups == 0 && errorDetails != "") {
		fmt.Printf("Error: %s\n", errorDetails)
		fmt.Println()
	}

	// Add random failure exclamation if tests failed
	if o.failedGroups > 0 {
		exclamations := []string{
			"This is madness!",
			"We're doomed!",
			"Are you sure this thing is safe?",
		}
		randomExclamation := exclamations[time.Now().UnixNano()%int64(len(exclamations))]
		fmt.Printf("Test failures! %s\n", randomExclamation)
		// Test details are shown inline with each failing group
	} else if o.passedGroups > 0 && o.skippedGroups == 0 {
		// All tests that ran passed (no skips)
		fmt.Println("Splendid! All tests passed successfully")
	} else if o.passedGroups > 0 && o.skippedGroups > 0 {
		// Some tests passed, some were skipped
		fmt.Println("Tests completed with some skipped")
	} else if o.skippedGroups > 0 && o.passedGroups == 0 {
		// Only skipped tests
		fmt.Println("All tests were skipped")
	}

	// Format results summary
	// Show test case counts when we have actual test counts with skipped tests
	// Otherwise show group counts (for compatibility with runners that don't report individual tests)
	if o.totalTests > 0 && (o.skippedTests > 0 || strings.HasPrefix(o.detectedRunner, "cargo")) {
		// Show test case counts
		if o.skippedTests > 0 {
			fmt.Printf("Results:     %d passed, %d failed, %d skipped, %d total\n",
				o.passedTests, o.failedTests, o.skippedTests, o.totalTests)
		} else if o.failedTests > 0 {
			fmt.Printf("Results:     %d passed, %d failed, %d total\n",
				o.passedTests, o.failedTests, o.totalTests)
		} else {
			fmt.Printf("Results:     %d passed, %d total\n", o.passedTests, o.totalTests)
		}
	} else {
		// Show group counts for other runners or when no test-level detail available
		if o.skippedGroups > 0 {
			fmt.Printf("Results:     %d passed, %d failed, %d skipped, %d total\n",
				o.passedGroups, o.failedGroups, o.skippedGroups, o.totalGroups)
		} else if o.failedGroups > 0 {
			fmt.Printf("Results:     %d passed, %d failed, %d total\n",
				o.passedGroups, o.failedGroups, o.totalGroups)
		} else {
			fmt.Printf("Results:     %d passed, %d total\n", o.passedGroups, o.totalGroups)
		}
	}

	// Calculate and display elapsed time
	elapsed := time.Since(o.startTime).Seconds()
	fmt.Printf("Total time:  %.3fs\n", elapsed)

	// Return command error if there was one
	if commandErr != nil {
		return fmt.Errorf("test command failed: %w", commandErr)
	}

	return nil
}

// processEvents processes IPC events and displays console output
func (o *Orchestrator) processEvents() {
	for event := range o.ipcManager.Events {
		// Pass event to report manager FIRST to update state
		if err := o.reportManager.HandleEvent(event); err != nil {
			o.logger.Error("Failed to handle event: %v", err)
		}

		// Then handle console output for different event types
		o.handleConsoleOutput(event)
	}
}

// normalizePath normalizes a file path for console output deduplication
func (o *Orchestrator) normalizePath(filePath string) string {
	// Try to get absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		// If absolute path fails, use the original path
		return filePath
	}
	return absPath
}

// normalizePathForReportManager normalizes paths the same way the report manager's GroupManager does
func (o *Orchestrator) normalizePathForReportManager(name string) string {
	// If it's not a file path (e.g., test names, suite names), return as-is
	if !strings.HasPrefix(name, "/") && !strings.HasPrefix(name, "./") && !strings.Contains(name, "/") {
		return name
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(name)
	if err != nil {
		// If we can't get absolute path, return original
		return name
	}

	// Always attempt to resolve symlinks for absolute paths
	// This is crucial for macOS where /tmp is a symlink to /private/tmp
	resolved, err := filepath.EvalSymlinks(absPath)
	if err == nil {
		return resolved
	}

	// If symlink resolution fails, return the absolute path
	return absPath
}

// makeRelativePath normalizes paths to relative paths (matching report manager)
func (o *Orchestrator) makeRelativePath(name string) string {
	// Only convert if it looks like an absolute file path
	if !strings.HasPrefix(name, "/") && !strings.HasPrefix(name, "./") {
		// Not a path, return as-is (e.g., test names, suite names)
		return name
	}

	// Try to make relative to current working directory
	if cwd, err := os.Getwd(); err == nil {
		if relPath, err := filepath.Rel(cwd, name); err == nil {
			// Ensure relative paths start with ./
			if !strings.HasPrefix(relPath, ".") && !strings.HasPrefix(relPath, "/") {
				relPath = "./" + relPath
			}
			return relPath
		}
	}

	// If conversion fails, return original
	return name
}

// handleConsoleOutput displays real-time console output for test events
func (o *Orchestrator) handleConsoleOutput(event ipc.Event) {
	switch e := event.(type) {
	case ipc.CollectionStartEvent:
		// Skip collection messages - we now show "Test execution starting" instead
		// This reduces console noise and provides cleaner output

	case ipc.CollectionFinishEvent:
		// Skip collection complete messages - we now show minimal output
		// Track the collected count internally but don't display it
		if e.Payload.Collected > 0 {
			o.lastCollected = e.Payload.Collected
		}

	case ipc.GroupStartEvent:
		// Track group start time for duration calculation
		groupID := report.GenerateGroupID(e.Payload.GroupName, e.Payload.ParentNames)
		o.groupStartTimes[groupID] = time.Now()

		// Display RUNNING status for the group - disabled to reduce console noise
		// o.displayGroupRunning(e.Payload.GroupName, e.Payload.ParentNames)

	case ipc.GroupResultEvent:
		// Check if this is a NOTESTS status (Go packages with no test files)
		if e.Payload.Status == "NOTESTS" {
			o.noTestGroups[e.Payload.GroupName] = true
			// Convert to SKIP for internal handling
			e.Payload.Status = "SKIP"
		}

		// For cargo test, mark groups with 0 tests as NO_TEST
		if strings.HasPrefix(o.detectedRunner, "cargo") {
			totalTests := e.Payload.Totals.Passed + e.Payload.Totals.Failed + e.Payload.Totals.Skipped
			if totalTests == 0 && len(e.Payload.ParentNames) == 0 { // Only for top-level groups
				o.noTestGroups[e.Payload.GroupName] = true
			}
		}

		// Display hierarchical output when a group completes
		status := convertStringToTestStatus(e.Payload.Status)
		o.displayGroupResult(e.Payload.GroupName, e.Payload.ParentNames, status, e.Payload.Duration)

		// Update group counters for top-level groups
		if len(e.Payload.ParentNames) == 0 {
			o.totalGroups++
			switch e.Payload.Status {
			case "PASS":
				o.passedGroups++
			case "FAIL":
				o.failedGroups++
			case "SKIP":
				o.skippedGroups++
			}
		}

	case ipc.GroupTestCaseEvent:
		// Track test case counts
		o.totalTests++
		switch e.Payload.Status {
		case "PASS":
			o.passedTests++
		case "FAIL":
			o.failedTests++
		case "SKIP":
			o.skippedTests++
		}

		// Track failed tests for hierarchical display
		if e.Payload.Status == "FAIL" {
			// Use the first parent name as file path (should be the file)
			if len(e.Payload.ParentNames) > 0 {
				normalizedPath := o.normalizePath(e.Payload.ParentNames[0])
				testName := e.Payload.TestName
				// Use parent names to build full hierarchy (skip file path)
				if len(e.Payload.ParentNames) > 1 {
					suiteNames := e.Payload.ParentNames[1:]
					testName = strings.Join(suiteNames, " > ") + " > " + testName
				}
				o.groupFailedTests[normalizedPath] = append(o.groupFailedTests[normalizedPath], testName)
			}
		}

	}
}

// extractAdapter extracts the adapter file to a temporary directory
func (o *Orchestrator) extractAdapter(adapterName string) (string, error) {
	// Read log level from environment variable for adapter injection
	logLevel := os.Getenv("THREEPIO_LOG_LEVEL")
	if logLevel == "" {
		logLevel = "WARN" // Default to WARN if not set
	}

	// Always use embedded adapters in production
	// Pass IPC path, run directory, and log level for injection
	embeddedPath, err := adapters.GetAdapterPath(adapterName, o.ipcPath, o.runDir, logLevel)
	if err != nil {
		return "", fmt.Errorf("failed to extract embedded adapter %s: %w", adapterName, err)
	}

	o.logger.Debug("Using embedded adapter: %s", embeddedPath)
	return embeddedPath, nil
}

// GetExitCode returns the exit code from the test run
func (o *Orchestrator) GetExitCode() int {
	return o.exitCode
}

// displayGroupRunning displays RUNNING status for a group that just started
func (o *Orchestrator) displayGroupRunning(groupName string, parentNames []string) {
	// Only display RUNNING status for top-level groups (files)
	// Subgroups will only be shown if they have failures
	if len(parentNames) > 0 {
		return
	}

	// Normalize the path for consistent display
	// normalizedGroupName := o.makeRelativePath(groupName)
	// No longer printing RUNNING status to console
}

// displayGroupResult displays the result of a completed group
func (o *Orchestrator) displayGroupResult(groupName string, parentNames []string, status ipc.TestStatus, duration float64) {
	o.logger.Debug("displayGroupResult called: group=%s, parentNames=%v, status=%s, duration=%f",
		groupName, parentNames, status, duration)

	// Normalize paths the same way the report manager does - absolute paths with symlink resolution
	normalizedGroupName := o.normalizePathForReportManager(groupName)
	normalizedParentNames := make([]string, len(parentNames))
	for i, name := range parentNames {
		normalizedParentNames[i] = o.normalizePathForReportManager(name)
	}

	groupID := report.GenerateGroupID(normalizedGroupName, normalizedParentNames)

	// Get the group from the report manager
	group, exists := o.reportManager.GetGroup(groupID)
	if !exists {
		o.logger.Debug("Group not found in report manager: %s", groupID)
		return
	}

	// Only display top-level groups (files)
	if len(parentNames) == 0 {
		// Check if we've already displayed the FINAL result for this file
		// Don't count intermediate PASS results as final if tests are still running
		if o.completedGroups[groupName] {
			o.logger.Debug("Group already completed: %s", groupName)
			return
		}

		// Only mark as completed if this is truly the final status
		// (all tests are done or it's a failure)
		if group.IsComplete() || status == ipc.TestStatusFail {
			o.completedGroups[groupName] = true
		}

		o.logger.Debug("Calling displayGroupHierarchy for: %s", groupName)
		o.displayGroupHierarchy(group, 0, duration)
	}
}

// displayGroupHierarchy displays a group and its children with hierarchical indentation
func (o *Orchestrator) displayGroupHierarchy(group *report.TestGroup, indent int, eventDuration float64) {
	// Only display top-level groups (files) in main output
	// Subgroups will only be shown if they have failures
	if len(group.ParentNames) > 0 {
		return // Don't display subgroups at this level
	}

	// Debug logging
	o.logger.Debug("displayGroupHierarchy: group=%s, hasTestCases=%v, testCases=%d, subgroups=%d",
		group.Name, group.HasTestCases(), len(group.TestCases), len(group.Subgroups))

	// For groups without test cases, only show if they failed
	if !group.HasTestCases() {
		// Check if this is a package with no test files or failed group
		isNoTests := o.noTestGroups[group.Name]
		isFailed := group.Status == report.TestStatusFail

		o.logger.Debug("Group %s has no test cases, isNoTests=%v, isFailed=%v", group.Name, isNoTests, isFailed)

		// Only show if failed or has no tests (don't show successful groups)
		if isFailed || isNoTests {
			// Build status string
			var statusParts []string
			if isFailed {
				statusParts = append(statusParts, "FAIL")
			}
			if isNoTests {
				statusParts = append(statusParts, "NO_TESTS")
			}

			// Make the path relative before sanitizing for report path
			groupName := group.Name
			if strings.HasPrefix(groupName, "/") {
				// Derive the test execution directory from runDir
				if absRunDir, err := filepath.Abs(o.runDir); err == nil {
					// Go up from runDir to find the project root (parent of .3pio)
					testExecDir := filepath.Dir(filepath.Dir(absRunDir)) // Go up twice: [id] -> runs -> .3pio
					testExecDir = filepath.Dir(testExecDir)              // Go up once more: .3pio -> project root

					// Resolve symlinks for consistent comparison
					if resolved, err := filepath.EvalSymlinks(testExecDir); err == nil {
						testExecDir = resolved
					}
					if resolvedGroup, err := filepath.EvalSymlinks(groupName); err == nil {
						groupName = resolvedGroup
					}

					// Try to make the path relative
					if relPath, err := filepath.Rel(testExecDir, groupName); err == nil {
						if !strings.HasPrefix(relPath, "..") {
							groupName = relPath
						}
					}
				}
			}

			// Build report path using $trun_dir placeholder
			reportPath := fmt.Sprintf("$trun_dir/reports/%s/index.md", report.SanitizeGroupName(groupName))

			// Print all on one line
			fmt.Printf("%s %s\n", strings.Join(statusParts, " "), reportPath)
		}
		return
	}

	// Display group status using raw groupName
	o.logger.Debug("displayGroupHierarchy: group.Status=%s for %s",
		group.Status, group.Name)

	// Update stats to ensure they're current
	group.UpdateStats()

	// Only display groups that have failures or no tests
	// Use recursive stats to include subgroups
	hasNoTestsAtAll := group.Stats.TotalTestsRecursive == 0 && group.Stats.SkippedTestsRecursive == 0

	o.logger.Debug("Group %s: FailedTestsRecursive=%d, TotalTestsRecursive=%d, PassedTestsRecursive=%d, SkippedTestsRecursive=%d",
		group.Name, group.Stats.FailedTestsRecursive, group.Stats.TotalTestsRecursive, group.Stats.PassedTestsRecursive, group.Stats.SkippedTestsRecursive)

	if group.Stats.FailedTestsRecursive > 0 || hasNoTestsAtAll {
		// Build status string with fail/pass/skip counts
		var statusParts []string

		// Check if this is a NO_TESTS case first
		if hasNoTestsAtAll && o.noTestGroups[group.Name] {
			// For NO_TESTS groups, just show NO_TESTS without counts
			statusParts = append(statusParts, "NO_TESTS")
		} else {
			// Add FAIL count if there are failures
			if group.Stats.FailedTestsRecursive > 0 {
				statusParts = append(statusParts, fmt.Sprintf("FAIL(%d)", group.Stats.FailedTestsRecursive))
			}

			// Add PASS count only if > 0
			if group.Stats.PassedTestsRecursive > 0 {
				statusParts = append(statusParts, fmt.Sprintf("PASS(%d)", group.Stats.PassedTestsRecursive))
			}

			// Add SKIP count only if > 0
			if group.Stats.SkippedTestsRecursive > 0 {
				statusParts = append(statusParts, fmt.Sprintf("SKIP(%d)", group.Stats.SkippedTestsRecursive))
			}
		}

		// Make the path relative before sanitizing for report path
		groupName := group.Name
		if strings.HasPrefix(groupName, "/") {
			// Derive the test execution directory from runDir
			if absRunDir, err := filepath.Abs(o.runDir); err == nil {
				// Go up from runDir to find the project root (parent of .3pio)
				testExecDir := filepath.Dir(filepath.Dir(absRunDir)) // Go up twice: [id] -> runs -> .3pio
				testExecDir = filepath.Dir(testExecDir)              // Go up once more: .3pio -> project root

				// Resolve symlinks for consistent comparison
				if resolved, err := filepath.EvalSymlinks(testExecDir); err == nil {
					testExecDir = resolved
				}
				if resolvedGroup, err := filepath.EvalSymlinks(groupName); err == nil {
					groupName = resolvedGroup
				}

				// Try to make the path relative
				if relPath, err := filepath.Rel(testExecDir, groupName); err == nil {
					if !strings.HasPrefix(relPath, "..") {
						groupName = relPath
					}
				}
			}
		}

		// Build report path using $trun_dir placeholder
		reportPath := fmt.Sprintf("$trun_dir/reports/%s/index.md", report.SanitizeGroupName(groupName))

		// Print all on one line
		fmt.Printf("%s %s\n", strings.Join(statusParts, " "), reportPath)
	}
}

// formatElapsedTime returns a human-friendly elapsed time since startTime
func (o *Orchestrator) formatElapsedTime() string {
	// Handle zero start time defensively
	if o.startTime.IsZero() {
		return "[T+ 0s]"
	}
	elapsed := time.Since(o.startTime)
	if elapsed < 0 {
		elapsed = 0
	}
	// Truncate to whole seconds to match tests
	elapsed = elapsed.Truncate(time.Second)
	return fmt.Sprintf("[T+ %s]", elapsed.String())
}

// collectFailedTests recursively collects all failed test names from a group hierarchy
func (o *Orchestrator) collectFailedTests(group *report.TestGroup) []string {
	var failedTests []string

	// Collect failed tests from this group
	for _, testCase := range group.TestCases {
		if testCase.Status == report.TestStatusFail {
			failedTests = append(failedTests, testCase.Name)
		}
	}

	// Recursively collect from subgroups
	for _, subgroup := range group.Subgroups {
		subgroupFailures := o.collectFailedTests(subgroup)
		failedTests = append(failedTests, subgroupFailures...)
	}

	return failedTests
}

// formatElapsedTime formats the elapsed time from start in a progressive display
func (o *Orchestrator) formatElapsedTime() string {
	elapsed := time.Since(o.startTime)
	totalSeconds := int(elapsed.Seconds())

	if totalSeconds < 60 {
		return fmt.Sprintf("[T+ %ds]", totalSeconds)
	} else if totalSeconds < 3600 {
		minutes := totalSeconds / 60
		seconds := totalSeconds % 60
		return fmt.Sprintf("[T+ %dm%ds]", minutes, seconds)
	} else {
		hours := totalSeconds / 3600
		minutes := (totalSeconds % 3600) / 60
		seconds := totalSeconds % 60
		return fmt.Sprintf("[T+ %dh%dm%ds]", hours, minutes, seconds)
	}
}

// getGroupStatusString returns a status string for groups in console output
func getGroupStatusString(status ipc.TestStatus) string {
	switch status {
	case ipc.TestStatusPass:
		return "PASS"
	case ipc.TestStatusFail:
		return "FAIL"
	case ipc.TestStatusSkip:
		return "SKIP"
	case ipc.TestStatusRunning:
		return "RUNNING"
	case ipc.TestStatusNoTests:
		return "NO_TESTS"
	default:
		return "PENDING"
	}
}

// convertStringToTestStatus converts a string status to ipc.TestStatus
func convertStringToTestStatus(status string) ipc.TestStatus {
	switch status {
	case "PASS":
		return ipc.TestStatusPass
	case "FAIL":
		return ipc.TestStatusFail
	case "SKIP":
		return ipc.TestStatusSkip
	case "NOTESTS", "NO_TESTS":
		// Special status for packages with no test files
		return ipc.TestStatusNoTests
	case "PENDING":
		return ipc.TestStatusPending
	case "RUNNING":
		return ipc.TestStatusRunning
	default:
		return ipc.TestStatusPending
	}
}

// convertReportStatusToIPC converts report.TestStatus to ipc.TestStatus
func convertReportStatusToIPC(status report.TestStatus) ipc.TestStatus {
	switch status {
	case report.TestStatusPass:
		return ipc.TestStatusPass
	case report.TestStatusFail:
		return ipc.TestStatusFail
	case report.TestStatusSkip:
		return ipc.TestStatusSkip
	case report.TestStatusPending:
		return ipc.TestStatusPending
	case report.TestStatusRunning:
		return ipc.TestStatusRunning
	case report.TestStatusNoTests:
		return ipc.TestStatusNoTests
	default:
		return ipc.TestStatusPending
	}
}

// computeStatsFromReportManager computes test statistics from the report manager
// This is a fallback when GroupResult events are not sent
func (o *Orchestrator) computeStatsFromReportManager() {
	// Get all root groups (files) from the report manager
	rootGroups := o.reportManager.GetRootGroups()
	for _, group := range rootGroups {
		// Only count groups that have test cases
		if group.HasTestCases() {
			o.totalGroups++
			switch group.Status {
			case report.TestStatusPass:
				o.passedGroups++
			case report.TestStatusFail:
				o.failedGroups++
			case report.TestStatusSkip:
				o.skippedGroups++
			}
		}
	}
}

// displayFinalResults displays the final test results when GroupResult events are not sent
func (o *Orchestrator) displayFinalResults() {
	// Get all root groups (files) from the report manager
	rootGroups := o.reportManager.GetRootGroups()
	for _, group := range rootGroups {
		// Display all groups (including those without test cases, which might be NO_TESTS)
		o.displayGroupHierarchy(group, 0, -1) // No duration available in this context (-1 indicates no duration)
	}
}

// generateRunID generates a unique run identifier
func generateRunID() string {
	timestamp := time.Now().Format("20060102T150405")

	// Character names from various sci-fi universes for memorable suffixes
	characters := []string{
		// Star Wars
		"luke-skywalker", "princess-leia", "han-solo", "chewbacca",
		"darth-vader", "obi-wan", "yoda", "r2d2", "c3po",
		"boba-fett", "jabba", "padme", "anakin", "mace-windu",
		"qui-gon", "palpatine", "kylo-ren", "rey", "finn", "poe",
		// Star Trek
		"kirk", "spock", "mccoy", "scotty", "uhura", "sulu", "chekov",
		"picard", "riker", "data", "worf", "geordi", "troi", "beverly",
		"janeway", "chakotay", "tuvok", "torres", "paris", "kim", "neelix",
		"sisko", "kira", "odo", "dax", "bashir", "obrien", "nog",
		"archer", "tpol", "tucker", "reed", "phlox", "hoshi", "travis",
		// Chrono Trigger
		"crono", "marle", "lucca", "robo", "frog", "ayla", "magus",
		"gato", "dalton", "lavos", "schala", "janus", "gaspar", "melchior",
		// Final Fantasy 6
		"terra", "locke", "edgar", "sabin", "celes", "cyan", "shadow",
		"setzer", "strago", "relm", "mog", "gau", "umaro", "gogo",
		"kefka", "leo", "banon", "gestahl", "rachel", "interceptor",
	}

	// Funny adjectives for memorable run names
	adjectives := []string{
		"grumpy", "sneaky", "giggly", "wonky", "dizzy",
		"cranky", "bouncy", "quirky", "sleepy", "dopey",
		"sassy", "goofy", "wacky", "silly", "funky",
		"nutty", "zany", "loopy", "kooky", "batty",
		"fuzzy", "bubbly", "snappy", "zippy", "rowdy",
		"cheeky", "spunky", "feisty", "frisky", "peppy",
	}

	// Use proper cross-platform random number generation
	// Seed with current time for different results each run
	source := rand.NewSource(time.Now().UnixNano())
	rng := rand.New(source)

	adjIdx := rng.Intn(len(adjectives))
	charIdx := rng.Intn(len(characters))

	return fmt.Sprintf("%s-%s-%s", timestamp, adjectives[adjIdx], characters[charIdx])
}
