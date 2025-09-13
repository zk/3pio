package orchestrator

import (
	"bufio"
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

	runID    string
	runDir   string
	ipcPath  string
	command  []string
	exitCode int

	// Console output state
	startTime      time.Time
	passedFiles    int
	failedFiles    int
	totalFiles     int
	displayedFiles map[string]bool // Track which files we've already displayed
	lastCollected  int             // Track last collection count to avoid duplicates
	fileStartTimes map[string]time.Time // Track start time for each file
	fileFailedTests map[string][]string // Track failed test names by file

	// Error capture
	stderrCapture strings.Builder
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

	return &Orchestrator{
		runnerManager:  runner.NewManager(),
		logger:         config.Logger,
		command:        config.Command,
		displayedFiles: make(map[string]bool),
		fileStartTimes: make(map[string]time.Time),
		fileFailedTests: make(map[string][]string),
	}, nil
}

// Run executes the test command with 3pio instrumentation
func (o *Orchestrator) Run() error {
	// Generate run ID
	o.runID = generateRunID()
	o.runDir = filepath.Join(".3pio", "runs", o.runID)

	// Print greeting and command
	testCommand := strings.Join(o.command, " ")
	fmt.Println()
	fmt.Println("Greetings! I will now execute the test command:")
	fmt.Printf("`%s`\n", testCommand)
	fmt.Println()

	// Print report path
	reportPath := filepath.Join(o.runDir, "test-run.md")
	fmt.Printf("Full report: %s\n", reportPath)
	fmt.Println()
	fmt.Println("Beginning test execution now...")
	fmt.Println()

	// Detect test runner
	runnerDef, err := o.runnerManager.Detect(o.command)
	if err != nil {
		return fmt.Errorf("failed to detect test runner: %w", err)
	}

	// Get test files (may be empty for dynamic discovery)
	testFiles, err := runnerDef.GetTestFiles(o.command)
	if err != nil {
		o.logger.Debug("Could not get test files upfront: %v", err)
		testFiles = []string{} // Use dynamic discovery
	}

	// Setup IPC
	ipcDir, err := ipc.EnsureIPCDirectory()
	if err != nil {
		return fmt.Errorf("failed to setup IPC directory: %w", err)
	}
	o.ipcPath = filepath.Join(ipcDir, fmt.Sprintf("%s.jsonl", o.runID))

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
	case "":
		detectedRunner = "go test"
	default:
		detectedRunner = "unknown"
	}

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

	// Initialize report
	args := strings.Join(o.command, " ")
	if err := o.reportManager.Initialize(testFiles, args); err != nil {
		return fmt.Errorf("failed to initialize report: %w", err)
	}

	// Check if this is a native runner (like Go test)
	var testCommandSlice []string
	var isNativeRunner bool
	var nativeDef interface{}
	
	// Check if adapter is needed (empty adapter name means native runner)
	adapterFileName := runnerDef.GetAdapterFileName()
	if adapterFileName == "" {
		// Native runner - no adapter needed (e.g., Go test)
		isNativeRunner = true
		// Try to get the native definition using type assertion
		if wrapper, ok := runnerDef.(*definitions.GoTestWrapper); ok {
			nativeDef = wrapper.GoTestDefinition
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

	// Set environment
	cmd.Env = append(os.Environ(), fmt.Sprintf("THREEPIO_IPC_PATH=%s", o.ipcPath))

	// Connect stdin to allow interactive prompts
	cmd.Stdin = os.Stdin

	// Capture output (append to existing file with header)
	outputPath := filepath.Join(o.runDir, "output.log")
	outputFile, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open output file: %w", err)
	}
	defer func() { _ = outputFile.Close() }()

	// Create pipes for stdout and stderr
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		o.exitCode = 1 // Set error exit code
		return fmt.Errorf("failed to start test command: %w", err)
	}

	// Record start time for duration calculation
	o.startTime = time.Now()

	// Process events and output concurrently
	var wg sync.WaitGroup
	eventsDone := make(chan struct{})

	// Process IPC events in background
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(eventsDone)
		o.processEvents()
	}()

	// Capture stdout
	if isNativeRunner {
		// For native runners, process the JSON output and generate IPC events
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Process output through the native definition
			if nd, ok := nativeDef.(interface {
				ProcessOutput(io.Reader, string) error
			}); ok {
				// Create a tee reader to capture output to file and process it
				teeReader := io.TeeReader(stdoutPipe, outputFile)
				if err := nd.ProcessOutput(teeReader, o.ipcPath); err != nil {
					o.logger.Error("Failed to process native output: %v", err)
				}
			} else {
				// Fallback to just capturing
				o.captureOutput(stdoutPipe, outputFile)
			}
		}()
	} else {
		// For adapter-based runners, just capture to file
		wg.Add(1)
		go func() {
			defer wg.Done()
			o.captureOutput(stdoutPipe, outputFile)
		}()
	}

	// Capture stderr (to file and error capture buffer)
	wg.Add(1)
	go func() {
		defer wg.Done()
		o.captureOutput(stderrPipe, outputFile, &o.stderrCapture)
	}()

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
			} else {
				o.exitCode = 1
			}
		}
	case sig := <-sigChan:
		o.logger.Info("Received signal: %v", sig)
		_ = cmd.Process.Kill()
		o.exitCode = 130 // Standard exit code for SIGINT
	}

	// Stop watching for events (this closes the Events channel and allows processEvents to exit)
	_ = o.ipcManager.Cleanup()

	// Wait for event processing to complete (channel is closed, range will exit)
	<-eventsDone
	o.logger.Debug("Event processing completed")

	// Wait for output capture to complete
	wg.Wait()
	o.logger.Debug("Output capture completed")

	// All goroutines should be finished at this point
	// (they were waited for via outputDone)

	// Finalize report
	var errorDetails string
	if commandErr != nil {
		// Only include error details for actual command errors (not test failures)
		// If tests were processed, this is just a test failure, not a command error
		if o.totalFiles == 0 {
			errorDetails = commandErr.Error()

			// Include stderr content if available for command errors
			stderrContent := strings.TrimSpace(o.stderrCapture.String())
			if stderrContent != "" {
				errorDetails = stderrContent
			}
		}
	}
	if err := o.reportManager.Finalize(o.exitCode, errorDetails); err != nil {
		o.logger.Error("Failed to finalize report: %v", err)
	}

	// Print completion message with TypeScript-style summary
	fmt.Println()

	// Print error details if command failed and we have error details
	if commandErr != nil && errorDetails != "" {
		fmt.Printf("Error: %s\n", errorDetails)
		fmt.Println()
	}

	// Add random failure exclamation if tests failed
	if o.failedFiles > 0 {
		exclamations := []string{
			"This is madness!",
			"We're doomed!",
			"Are you sure this thing is safe?",
		}
		randomExclamation := exclamations[time.Now().UnixNano()%int64(len(exclamations))]
		fmt.Printf("Test failures! %s\n", randomExclamation)
	} else if o.passedFiles > 0 {
		// Success message
		fmt.Println("Splendid! All tests passed successfully")
	}

	// Format results summary in new format
	fmt.Printf("Results:     %d passed, %d total\n", o.passedFiles, o.totalFiles)

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
		// Handle console output for different event types
		o.handleConsoleOutput(event)

		// Pass event to report manager
		if err := o.reportManager.HandleEvent(event); err != nil {
			o.logger.Error("Failed to handle event: %v", err)
		}
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

// sanitizePathForFilesystem sanitizes a file path to preserve directory structure
// while preventing directory traversal and filesystem issues
func sanitizePathForFilesystem(filePath string) string {
	// Clean the path to normalize it
	cleanPath := filepath.Clean(filePath)

	// Remove any leading slash or dot-slash
	cleanPath = strings.TrimPrefix(cleanPath, "/")
	cleanPath = strings.TrimPrefix(cleanPath, "./")

	// Handle paths that start with ".." by replacing with "_UP"
	parts := strings.Split(cleanPath, string(filepath.Separator))
	for i, part := range parts {
		if part == ".." {
			parts[i] = "_UP"
		}
	}
	cleanPath = strings.Join(parts, string(filepath.Separator))

	// For JS/TS files, keep the extension as part of the name
	// For other files like Go, remove the extension
	ext := filepath.Ext(cleanPath)
	if ext != ".js" && ext != ".jsx" && ext != ".ts" && ext != ".tsx" && ext != ".mjs" && ext != ".cjs" {
		cleanPath = strings.TrimSuffix(cleanPath, ext)
	}

	return cleanPath
}

// getRelativePath converts a file path to a relative path starting with ./
func (o *Orchestrator) getRelativePath(filePath string) string {
	cwd, _ := os.Getwd()
	relativePath := "./" + filepath.Base(filePath) // Use basename as fallback
	if relPath, err := filepath.Rel(cwd, filePath); err == nil {
		relativePath = "./" + relPath
	}
	return relativePath
}

// handleConsoleOutput displays real-time console output for test events
func (o *Orchestrator) handleConsoleOutput(event ipc.Event) {
	switch e := event.(type) {
	case ipc.CollectionStartEvent:
		// Display collection start message
		fmt.Println("Collecting tests...")

	case ipc.CollectionFinishEvent:
		// Display collection complete message with file count (avoid duplicates)
		if e.Payload.Collected > 0 && e.Payload.Collected != o.lastCollected {
			fmt.Printf("Found %d test files\n\n", e.Payload.Collected)
			o.lastCollected = e.Payload.Collected
		}

	case ipc.GroupStartEvent:
		// Track group start time for duration calculation
		groupID := report.GenerateGroupID(e.Payload.GroupName, e.Payload.ParentNames)
		o.fileStartTimes[groupID] = time.Now()

		// Display RUNNING status for the group
		o.displayGroupRunning(e.Payload.GroupName, e.Payload.ParentNames)

	case ipc.GroupResultEvent:
		// Display hierarchical output when a group completes
		status := convertStringToTestStatus(e.Payload.Status)
		o.displayGroupResult(e.Payload.GroupName, e.Payload.ParentNames, status)

	// Legacy file-based events removed - only group events are supported

	case ipc.GroupTestCaseEvent:
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
				o.fileFailedTests[normalizedPath] = append(o.fileFailedTests[normalizedPath], testName)
			}
		}

	// Legacy TestFileResultEvent removed - using group events instead
	}
}

// captureOutput captures and duplicates output streams
func (o *Orchestrator) captureOutput(input io.Reader, outputs ...io.Writer) {
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text() + "\n"
		for _, output := range outputs {
			_, _ = output.Write([]byte(line))
		}
	}

	if err := scanner.Err(); err != nil {
		o.logger.Error("Error reading output: %v", err)
	}
}

// extractAdapter extracts the adapter file to a temporary directory
func (o *Orchestrator) extractAdapter(adapterName string) (string, error) {
	// Always use embedded adapters in production
	// Pass IPC path and run ID for injection
	embeddedPath, err := adapters.GetAdapterPath(adapterName, o.ipcPath, o.runID)
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
	// Build hierarchical path
	var parts []string
	if len(parentNames) == 0 {
		// Root group (file)
		parts = append(parts, o.getRelativePath(groupName))
	} else {
		// Add parent names first
		for i, parentName := range parentNames {
			if i == 0 {
				parts = append(parts, o.getRelativePath(parentName))
			} else {
				parts = append(parts, parentName)
			}
		}
		// Add this group's name
		parts = append(parts, groupName)
	}

	hierarchicalPath := strings.Join(parts, " > ")
	fmt.Printf("%-8s %s\n", "RUNNING", hierarchicalPath)
}

// displayGroupResult displays the result of a completed group
func (o *Orchestrator) displayGroupResult(groupName string, parentNames []string, status ipc.TestStatus) {
	groupID := report.GenerateGroupID(groupName, parentNames)

	// Get the group from the report manager
	group, exists := o.reportManager.GetGroup(groupID)
	if !exists {
		return
	}

	// Only display top-level groups (files) to avoid duplicates
	if len(parentNames) == 0 {
		o.displayGroupHierarchy(group, 0)
	}
}

// displayGroupHierarchy displays a group and its children with hierarchical indentation
func (o *Orchestrator) displayGroupHierarchy(group *report.TestGroup, indent int) {
	// Build the hierarchical path for display
	hierarchicalPath := o.buildHierarchicalPath(group)

	// Display group status with hierarchical path
	statusStr := getGroupStatusString(convertReportStatusToIPC(group.Status))

	// Get duration for this group
	durationStr := ""
	groupID := group.ID
	if startTime, ok := o.fileStartTimes[groupID]; ok {
		duration := time.Since(startTime).Seconds()
		if duration > 0.01 { // Only show if > 10ms
			durationStr = fmt.Sprintf(" (%.2fs)", duration)
		}
		delete(o.fileStartTimes, groupID) // Clean up
	}

	// Display the group result line
	fmt.Printf("%-8s %s%s\n", statusStr, hierarchicalPath, durationStr)

	// Show report path for failed groups
	if group.Status == report.TestStatusFail {
		relativePath := o.getRelativePath(group.Name)
		reportPath := fmt.Sprintf(".3pio/runs/%s/reports/%s.md", o.runID, sanitizePathForFilesystem(relativePath))
		fmt.Printf("  See %s\n", reportPath)
	}

	// Display child test failures inline (matching the plan's format)
	if group.Status == report.TestStatusFail {
		for _, testCase := range group.TestCases {
			if testCase.Status == report.TestStatusFail {
				fmt.Printf("  x %s\n", testCase.Name)
				if testCase.Error != nil && testCase.Error.Message != "" {
					// Show first line of error
					lines := strings.Split(strings.TrimSpace(testCase.Error.Message), "\n")
					if len(lines) > 0 {
						fmt.Printf("    %s\n", lines[0])
					}
				}
			}
		}
	}
}

// displayLegacyFileResult removed - using group-based display instead

// getTestCaseConsoleIcon returns an icon for individual test cases in console output
func getTestCaseConsoleIcon(status ipc.TestStatus) string {
	switch status {
	case ipc.TestStatusPass:
		return "✓"
	case ipc.TestStatusFail:
		return "x"
	case ipc.TestStatusSkip:
		return "o"
	default:
		return "-" // Running or unknown
	}
}

// getTestFileConsoleIcon returns an icon for file-level status in console output
func getTestFileConsoleIcon(status ipc.TestStatus) string {
	switch status {
	case ipc.TestStatusPass:
		return "✓"
	case ipc.TestStatusFail:
		return "x"
	case ipc.TestStatusSkip:
		return "o"
	default:
		return "-"
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
	default:
		return "PENDING"
	}
}

// buildHierarchicalPath builds the full hierarchical path for a group
func (o *Orchestrator) buildHierarchicalPath(group *report.TestGroup) string {
	// Build the full path from parent names + group name
	var parts []string

	// Start with file path (first parent or the group itself if root)
	if len(group.ParentNames) == 0 {
		// This is a root group (file)
		parts = append(parts, o.getRelativePath(group.Name))
	} else {
		// Add all parent names
		for i, parentName := range group.ParentNames {
			if i == 0 {
				// First parent is usually the file path
				parts = append(parts, o.getRelativePath(parentName))
			} else {
				// Other parents are group names
				parts = append(parts, parentName)
			}
		}
		// Add this group's name
		parts = append(parts, group.Name)
	}

	// Join with " > " separator as specified in the plan
	return strings.Join(parts, " > ")
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
	default:
		return ipc.TestStatusPending
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
		"fuzzy", "bubbly", "snappy", "zippy", "perky",
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
