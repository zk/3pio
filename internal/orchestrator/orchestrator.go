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
		config.Logger = &consoleLogger{}
	}

	return &Orchestrator{
		runnerManager:  runner.NewManager(),
		logger:         config.Logger,
		command:        config.Command,
		displayedFiles: make(map[string]bool),
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

	// Create report manager
	o.reportManager, err = report.NewManager(o.runDir, parser, o.logger)
	if err != nil {
		return fmt.Errorf("failed to create report manager: %w", err)
	}

	// Initialize report
	args := strings.Join(o.command, " ")
	if err := o.reportManager.Initialize(testFiles, args); err != nil {
		return fmt.Errorf("failed to initialize report: %w", err)
	}

	// Extract adapter to temp directory
	adapterPath, err := o.extractAdapter(runnerDef.GetAdapterFileName())
	if err != nil {
		return fmt.Errorf("failed to extract adapter: %w", err)
	}

	// Build command with adapter injection
	testCommandSlice := runnerDef.BuildCommand(o.command, adapterPath)

	o.logger.Debug("Executing command: %v", testCommandSlice)
	o.logger.Debug("Adapter path: %s", adapterPath)
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

	// Capture stdout (only to file, don't echo to console)
	wg.Add(1)
	go func() {
		defer wg.Done()
		o.captureOutput(stdoutPipe, outputFile)
	}()

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
	}

	// Format results summary
	var resultParts []string
	if o.failedFiles > 0 {
		resultParts = append(resultParts, fmt.Sprintf("%d failed", o.failedFiles))
	}
	if o.passedFiles > 0 {
		resultParts = append(resultParts, fmt.Sprintf(" %d passed", o.passedFiles))
	}
	if o.totalFiles > 0 {
		resultParts = append(resultParts, fmt.Sprintf(" %d total", o.totalFiles))
	}

	if len(resultParts) > 0 {
		fmt.Printf("Results: %s\n", strings.Join(resultParts, ","))
	}

	// Calculate and display elapsed time
	elapsed := time.Since(o.startTime).Seconds()
	fmt.Printf("Time:        %.3fs\n", elapsed)

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
	case ipc.TestFileStartEvent:
		// Normalize path for deduplication - use absolute path as key
		normalizedPath := o.normalizePath(e.Payload.FilePath)
		displayKey := normalizedPath + ":start"

		// Skip if already displayed this start event for this file
		if o.displayedFiles[displayKey] {
			return
		}
		o.displayedFiles[displayKey] = true

		relativePath := o.getRelativePath(normalizedPath)
		fmt.Printf("RUNNING  %s\n", relativePath)

	case ipc.TestFileResultEvent:
		// Normalize path for deduplication - use absolute path as key
		normalizedPath := o.normalizePath(e.Payload.FilePath)

		// Skip if already displayed this result event for this file
		if o.displayedFiles[normalizedPath+":result"] {
			return
		}
		o.displayedFiles[normalizedPath+":result"] = true

		relativePath := o.getRelativePath(normalizedPath)

		// Format status with proper spacing
		var status string
		switch e.Payload.Status {
		case ipc.TestStatusPass:
			status = "PASS    "
			o.passedFiles++
		case ipc.TestStatusFail:
			status = "FAIL    "
			o.failedFiles++
		default:
			status = "SKIP    "
		}

		fmt.Printf("%s %s\n", status, relativePath)

		// Display failed test details using actual data
		if e.Payload.Status == ipc.TestStatusFail {
			// Show actual failed tests if available
			if len(e.Payload.FailedTests) > 0 {
				for _, failedTest := range e.Payload.FailedTests {
					duration := ""
					if failedTest.Duration > 0 {
						duration = fmt.Sprintf(" (%.0f ms)", failedTest.Duration)
					}
					fmt.Printf("    ✕ %s%s\n", failedTest.Name, duration)
				}
			}

			// Use the actual file name for the log reference
			logFileName := filepath.Base(normalizedPath)
			fmt.Printf("  See .3pio/runs/%s/logs/%s.log\n", o.runID, strings.TrimSuffix(logFileName, filepath.Ext(logFileName)))
			fmt.Println("    ")
		}

		o.totalFiles++
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
	embeddedPath, err := adapters.GetAdapterPath(adapterName)
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

// consoleLogger is a simple console logger
type consoleLogger struct{}

func (c *consoleLogger) Debug(format string, args ...interface{}) {
	if os.Getenv("THREEPIO_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}

func (c *consoleLogger) Error(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[ERROR] "+format+"\n", args...)
}

func (c *consoleLogger) Info(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}
