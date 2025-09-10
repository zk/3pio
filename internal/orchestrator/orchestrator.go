package orchestrator

import (
	"bufio"
	"fmt"
	"io"
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
	
	runID         string
	runDir        string
	ipcPath       string
	command       []string
	exitCode      int
	
	mu            sync.Mutex
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
		runnerManager: runner.NewManager(),
		logger:        config.Logger,
		command:       config.Command,
	}, nil
}

// Run executes the test command with 3pio instrumentation
func (o *Orchestrator) Run() error {
	// Generate run ID
	o.runID = generateRunID()
	o.runDir = filepath.Join(".3pio", "runs", o.runID)
	
	// Print preamble
	fmt.Printf("\n📊 Test results will be saved to: %s\n", o.runDir)
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
	testCommand := runnerDef.BuildCommand(o.command, adapterPath)
	
	o.logger.Debug("Executing command: %v", testCommand)
	o.logger.Debug("Adapter path: %s", adapterPath)
	o.logger.Debug("IPC path: %s", o.ipcPath)
	
	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	// Create command
	cmd := exec.Command(testCommand[0], testCommand[1:]...)
	
	// Set environment
	cmd.Env = append(os.Environ(), fmt.Sprintf("THREEPIO_IPC_PATH=%s", o.ipcPath))
	
	// Connect stdin to allow interactive prompts
	cmd.Stdin = os.Stdin
	
	// Capture output
	outputPath := filepath.Join(o.runDir, "output.log")
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()
	
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
		return fmt.Errorf("failed to start test command: %w", err)
	}
	
	// Process events and output concurrently
	var wg sync.WaitGroup
	eventsDone := make(chan struct{})
	
	// Process IPC events in background
	go func() {
		defer close(eventsDone)
		o.processEvents()
	}()
	
	// Capture stdout
	wg.Add(1)
	go func() {
		defer wg.Done()
		o.captureOutput(stdoutPipe, outputFile, os.Stdout)
	}()
	
	// Capture stderr
	wg.Add(1)
	go func() {
		defer wg.Done()
		o.captureOutput(stderrPipe, outputFile, os.Stderr)
	}()
	
	// Wait for command completion or signal
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	
	select {
	case err := <-done:
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				o.exitCode = exitErr.ExitCode()
			} else {
				o.exitCode = 1
			}
		}
	case sig := <-sigChan:
		o.logger.Info("Received signal: %v", sig)
		cmd.Process.Kill()
		o.exitCode = 130 // Standard exit code for SIGINT
	}
	
	// Wait for output capture to complete with timeout
	outputDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(outputDone)
	}()
	
	// Wait for output capture or timeout
	select {
	case <-outputDone:
		o.logger.Debug("Output capture completed")
	case <-time.After(2 * time.Second):
		o.logger.Debug("Output capture timeout - proceeding")
	}
	
	// Stop watching for events
	o.ipcManager.Cleanup()
	
	// Wait briefly for event processing to complete
	select {
	case <-eventsDone:
		o.logger.Debug("Event processing completed")
	case <-time.After(1 * time.Second):
		o.logger.Debug("Event processing timeout - proceeding")
	}
	
	// Finalize report
	if err := o.reportManager.Finalize(o.exitCode); err != nil {
		o.logger.Error("Failed to finalize report: %v", err)
	}
	
	// Print completion message
	fmt.Println()
	if o.exitCode == 0 {
		fmt.Printf("✅ Tests completed successfully. Results saved to: %s\n", o.runDir)
	} else {
		fmt.Printf("❌ Tests failed with exit code %d. Results saved to: %s\n", o.exitCode, o.runDir)
	}
	
	return nil
}

// processEvents processes IPC events
func (o *Orchestrator) processEvents() {
	for event := range o.ipcManager.Events {
		if err := o.reportManager.HandleEvent(event); err != nil {
			o.logger.Error("Failed to handle event: %v", err)
		}
	}
}

// captureOutput captures and duplicates output streams
func (o *Orchestrator) captureOutput(input io.Reader, outputs ...io.Writer) {
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text() + "\n"
		for _, output := range outputs {
			output.Write([]byte(line))
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
	
	// Star Wars character names for memorable suffixes
	characters := []string{
		"luke-skywalker", "princess-leia", "han-solo", "chewbacca",
		"darth-vader", "obi-wan", "yoda", "r2d2", "c3po",
		"boba-fett", "jabba", "padme", "anakin", "mace-windu",
		"qui-gon", "palpatine", "kylo-ren", "rey", "finn", "poe",
	}
	
	// Add adjectives for more variety
	adjectives := []string{
		"brave", "clever", "mighty", "swift", "bold",
		"wise", "fierce", "noble", "quick", "strong",
	}
	
	// Random selection (simplified - in production use crypto/rand)
	adjIdx := time.Now().Nanosecond() % len(adjectives)
	charIdx := (time.Now().Nanosecond() / 1000) % len(characters)
	
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