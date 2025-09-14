package report

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/zk/3pio/internal/ipc"
	"github.com/zk/3pio/internal/logger"
	"github.com/zk/3pio/internal/runner"
)

// Manager handles report generation and file I/O
type Manager struct {
	runDir          string
	state           *ipc.TestRunState
	outputParser    runner.OutputParser
	logger          Logger
	detectedRunner  string // e.g., "vitest", "jest", "go test", "pytest"
	modifiedCommand string // The actual command executed with adapter

	// Group manager for hierarchical test organization
	groupManager *GroupManager

	// Track if we created our own FileLogger that needs closing
	ownedFileLogger *logger.FileLogger

	// File handles for incremental writing
	fileHandles map[string]*os.File
	fileBuffers map[string][]string
	debouncers  map[string]*time.Timer
	outputFile  *os.File

	// Separate stdout/stderr buffers for structured reports
	stdoutBuffers map[string][]string
	stderrBuffers map[string][]string

	mu           sync.RWMutex
	debounceTime time.Duration
	maxWaitTime  time.Duration
}

// Logger interface for debug logging
type Logger interface {
	Debug(format string, args ...interface{})
	Error(format string, args ...interface{})
	Info(format string, args ...interface{})
}

// NewManager creates a new report manager
func NewManager(runDir string, parser runner.OutputParser, lg Logger, detectedRunner string, modifiedCommand string) (*Manager, error) {
	if lg == nil {
		lg = &noopLogger{}
	}

	// Create run directory
	if err := os.MkdirAll(runDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create run directory: %w", err)
	}

	// Reports directory no longer needed - using group-based directories

	// Open output.log file
	outputPath := filepath.Join(runDir, "output.log")
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create output.log: %w", err)
	}

	// Initialize GroupManager for hierarchical test organization
	// Cast the logger to FileLogger if possible, otherwise use a wrapper
	var fileLogger *logger.FileLogger
	var ownedFileLogger *logger.FileLogger
	if fl, ok := lg.(*logger.FileLogger); ok {
		fileLogger = fl
	} else {
		// In test environments, don't create a real FileLogger
		// Check if this is a test logger (by type name)
		typeName := fmt.Sprintf("%T", lg)
		if strings.Contains(strings.ToLower(typeName), "mock") || strings.Contains(strings.ToLower(typeName), "noop") || strings.Contains(strings.ToLower(typeName), "test") {
			// Use nil logger for tests to avoid file handle issues
			fileLogger = nil // GroupManager should handle nil logger
		} else {
			// Create a new file logger for production use
			var err error
			fileLogger, err = logger.NewFileLogger()
			if err != nil {
				return nil, fmt.Errorf("failed to create file logger for group manager: %w", err)
			}
			ownedFileLogger = fileLogger // Track that we created this logger
		}
	}

	groupManager := NewGroupManager(runDir, "", fileLogger)

	return &Manager{
		runDir:          runDir,
		outputParser:    parser,
		logger:          lg,
		detectedRunner:  detectedRunner,
		modifiedCommand: modifiedCommand,
		groupManager:    groupManager,
		ownedFileLogger: ownedFileLogger,
		fileHandles:     make(map[string]*os.File),
		fileBuffers:     make(map[string][]string),
		debouncers:      make(map[string]*time.Timer),
		outputFile:      outputFile,
		stdoutBuffers:   make(map[string][]string),
		stderrBuffers:   make(map[string][]string),
		debounceTime:    100 * time.Millisecond,
		maxWaitTime:     500 * time.Millisecond,
	}, nil
}

// UpdateModifiedCommand updates the modified command after adapter extraction
func (m *Manager) UpdateModifiedCommand(command string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.modifiedCommand = command
}

// Initialize sets up the initial test run state
func (m *Manager) Initialize(testFiles []string, args string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	m.state = &ipc.TestRunState{
		Timestamp:      now,
		Status:         "RUNNING",
		UpdatedAt:      now,
		Arguments:      args,
		TotalFiles:     len(testFiles),
		FilesCompleted: 0,
		FilesPassed:    0,
		FilesFailed:    0,
		FilesSkipped:   0,
		TestFiles:      make([]ipc.TestFile, 0),
	}

	// Register known test files (static discovery)
	for _, file := range testFiles {
		m.registerTestFileInternal(file)
	}

	// Write output.log header
	if err := m.writeOutputLogHeader(args); err != nil {
		return fmt.Errorf("failed to write output log header: %w", err)
	}

	// Write initial state
	return m.writeState()
}

// HandleEvent processes an IPC event
func (m *Manager) HandleEvent(event ipc.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch e := event.(type) {

	case ipc.CollectionErrorEvent:
		return m.handleCollectionError(e)

	// Group events - forward to GroupManager
	case ipc.GroupDiscoveredEvent:
		if m.groupManager != nil {
			return m.groupManager.ProcessGroupDiscovered(e)
		}

	case ipc.GroupStartEvent:
		if m.groupManager != nil {
			return m.groupManager.ProcessGroupStart(e)
		}

	case ipc.GroupResultEvent:
		if m.groupManager != nil {
			return m.groupManager.ProcessGroupResult(e)
		}

	case ipc.GroupTestCaseEvent:
		if m.groupManager != nil {
			return m.groupManager.ProcessTestCase(e)
		}

	case ipc.GroupStdoutChunkEvent:
		if m.groupManager != nil {
			return m.groupManager.ProcessGroupStdout(e)
		}

	case ipc.GroupStderrChunkEvent:
		if m.groupManager != nil {
			return m.groupManager.ProcessGroupStderr(e)
		}

	case ipc.RunCompleteEvent:
		if m.groupManager != nil {
			return m.groupManager.ProcessRunComplete(e)
		}

	default:
		m.logger.Debug("Unknown event type: %T", event)
	}

	return nil
}

// handleCollectionError handles collection error events (pytest specific)
func (m *Manager) handleCollectionError(event ipc.CollectionErrorEvent) error {
	filePath := event.Payload.FilePath

	// Ensure file is registered
	m.ensureTestFileRegisteredInternal(filePath)

	// Find the test file and set execution error
	normalizedPath := m.normalizePath(filePath)
	for i := range m.state.TestFiles {
		if m.normalizePath(m.state.TestFiles[i].File) == normalizedPath {
			// Set execution error and update status to indicate error
			m.state.TestFiles[i].ExecutionError = fmt.Sprintf("Collection failed: %s", event.Payload.Error)
			m.state.TestFiles[i].Updated = time.Now()

			// Mark file as having execution error (will map to ERRORED in status)
			// Keep existing status logic but the presence of ExecutionError will trigger ERRORED in report
			break
		}
	}

	return m.scheduleWrite()
}

// SetExecutionError sets an execution error for a test file
// This is used for errors that occur during test execution (not test failures)
func (m *Manager) SetExecutionError(filePath string, errorMsg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Ensure file is registered
	m.ensureTestFileRegisteredInternal(filePath)

	// Find the test file and set execution error
	normalizedPath := m.normalizePath(filePath)
	for i := range m.state.TestFiles {
		if m.normalizePath(m.state.TestFiles[i].File) == normalizedPath {
			m.state.TestFiles[i].ExecutionError = errorMsg
			m.state.TestFiles[i].Updated = time.Now()
			break
		}
	}

	return m.scheduleWrite()
}

// ensureTestFileRegisteredInternal ensures a test file is registered (internal, assumes lock held)
func (m *Manager) ensureTestFileRegisteredInternal(filePath string) {
	// Normalize the incoming file path
	normalizedPath := m.normalizePath(filePath)

	// Check if already registered (compare normalized paths)
	for _, tf := range m.state.TestFiles {
		if m.normalizePath(tf.File) == normalizedPath {
			return
		}
	}

	// Register new file
	m.registerTestFileInternal(filePath)
	m.state.TotalFiles++
}

// registerTestFileInternal registers a test file (internal, assumes lock held)
func (m *Manager) registerTestFileInternal(filePath string) {
	// File-based reports are no longer created
	// Groups are created on-demand by the group manager
}

// scheduleWrite schedules a debounced state write
func (m *Manager) scheduleWrite() error {
	// For now, write immediately
	// TODO: Implement debouncing for state writes
	return m.writeState()
}

// writeState writes the current state to test-run.md
func (m *Manager) writeState() error {
	m.state.UpdatedAt = time.Now()

	// Generate markdown report
	report := m.generateMarkdownReport()

	// Write to file
	reportPath := filepath.Join(m.runDir, "test-run.md")
	return os.WriteFile(reportPath, []byte(report), 0644)
}

// writeOutputLogHeader writes the header for output.log
func (m *Manager) writeOutputLogHeader(args string) error {
	header := fmt.Sprintf(`# 3pio Test Output Log
# Timestamp: %s
# Command: %s
# This file contains all stdout/stderr output from the test run.
# ---

`, time.Now().Format(time.RFC3339), args)

	_, err := m.outputFile.WriteString(header)
	return err
}

// generateMarkdownReport generates the markdown report
func (m *Manager) generateMarkdownReport() string {
	var sb strings.Builder

	// Extract run ID from runDir
	runID := filepath.Base(m.runDir)

	// Map internal status to spec status
	var statusText string
	// Check if all tests are actually completed
	pendingFiles := m.state.TotalFiles - m.state.FilesCompleted
	if m.state.Status == "COMPLETE" && pendingFiles > 0 {
		// Override to RUNNING if there are still pending files
		statusText = "RUNNING"
	} else {
		switch m.state.Status {
		case "RUNNING":
			statusText = "RUNNING"
		case "COMPLETE":
			statusText = "COMPLETED"
		case "ERROR":
			statusText = "ERRORED"
		default:
			statusText = "PENDING"
		}
	}

	// YAML frontmatter
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("run_id: %s\n", runID))
	sb.WriteString(fmt.Sprintf("run_path: %s\n", m.runDir))
	sb.WriteString(fmt.Sprintf("detected_runner: %s\n", m.detectedRunner))
	sb.WriteString(fmt.Sprintf("modified_command: %s\n", m.modifiedCommand))
	sb.WriteString(fmt.Sprintf("created: %s\n", m.state.Timestamp.UTC().Format("2006-01-02T15:04:05.000Z")))
	sb.WriteString(fmt.Sprintf("updated: %s\n", m.state.UpdatedAt.UTC().Format("2006-01-02T15:04:05.000Z")))
	sb.WriteString(fmt.Sprintf("status: %s\n", statusText))
	sb.WriteString("---\n\n")

	// Header
	sb.WriteString("# 3pio Test Run\n\n")
	sb.WriteString(fmt.Sprintf("- Test command: `%s`\n", m.state.Arguments))
	sb.WriteString("- Run stdout/stderr: `./output.log`\n\n")

	// Error details if status is ERRORED
	if statusText == "ERRORED" && m.state.ErrorDetails != "" {
		sb.WriteString("## Error\n\n")
		sb.WriteString("```\n")
		sb.WriteString(m.state.ErrorDetails)
		sb.WriteString("\n```\n\n")
	}

	// Always use group-based reporting
	if m.groupManager != nil {
		// Generate hierarchical summary and results using group data
		m.generateGroupBasedReport(&sb, statusText)
	} else {
		// No test results to report
		sb.WriteString("## Test Results\n\n")
		sb.WriteString("No test results available.\n")
	}

	return sb.String()
}

// generateGroupBasedReport generates summary and results using hierarchical group data
func (m *Manager) generateGroupBasedReport(sb *strings.Builder, statusText string) {
	// Summary section with group-based statistics
	if statusText != "ERRORED" {
		sb.WriteString("## Summary\n\n")
		sb.WriteString("\n")

		// Calculate statistics from group data
		rootGroups := m.groupManager.GetRootGroups()
		totalFiles := len(rootGroups)
		completedFiles := 0
		passedFiles := 0
		failedFiles := 0
		skippedFiles := 0
		var totalDuration float64

		for _, group := range rootGroups {
			if group.IsComplete() {
				completedFiles++
				switch group.Status {
				case TestStatusPass:
					passedFiles++
				case TestStatusFail:
					failedFiles++
				case TestStatusSkip:
					skippedFiles++
				}
			}
			totalDuration += group.Duration.Seconds()
		}

		pendingFiles := totalFiles - completedFiles

		sb.WriteString(fmt.Sprintf("- Total files: %d\n", totalFiles))
		sb.WriteString(fmt.Sprintf("- Files completed: %d\n", completedFiles))
		sb.WriteString(fmt.Sprintf("- Files passed: %d\n", passedFiles))
		sb.WriteString(fmt.Sprintf("- Files failed: %d\n", failedFiles))
		sb.WriteString(fmt.Sprintf("- Files skipped: %d\n", skippedFiles))
		sb.WriteString(fmt.Sprintf("- Files pending: %d\n", pendingFiles))
		sb.WriteString(fmt.Sprintf("- Total duration: %.2fs\n\n", totalDuration))
	}

	// Test results section with hierarchical structure
	if len(m.groupManager.GetRootGroups()) > 0 {
		sb.WriteString("## Test Results\n\n")
		for _, group := range m.groupManager.GetRootGroups() {
			m.generateGroupReportSection(sb, group, 0)
		}
	}
}

// generateGroupReportSection generates a hierarchical report section for a group
func (m *Manager) generateGroupReportSection(sb *strings.Builder, group *TestGroup, indent int) {
	indentStr := strings.Repeat("  ", indent)

	// File-level groups (root groups)
	if len(group.ParentNames) == 0 {
		statusIcon := getGroupStatusIcon(group.Status)
		filename := filepath.Base(group.Name)
		durationStr := fmt.Sprintf("%.2fs", group.Duration.Seconds())

		sb.WriteString(fmt.Sprintf("%s**%s** - %s %s", indentStr, filename, statusIcon, durationStr))

		// Add test counts
		if group.Stats.TotalTests > 0 {
			sb.WriteString(fmt.Sprintf(" (%d tests: %d passed, %d failed, %d skipped)",
				group.Stats.TotalTests, group.Stats.PassedTests, group.Stats.FailedTests, group.Stats.SkippedTests))
		}
		sb.WriteString("\n")

		// Add report link for failed files
		// Report links are now handled by the group's own report path
		// which is generated in the group-based directory structure
	} else {
		// Suite-level groups
		statusIcon := getGroupStatusIcon(group.Status)
		sb.WriteString(fmt.Sprintf("%s**%s** - %s", indentStr, group.Name, statusIcon))

		if group.Stats.TotalTests > 0 {
			sb.WriteString(fmt.Sprintf(" (%d tests)", group.Stats.TotalTests))
		}
		sb.WriteString("\n")
	}

	// Show individual test cases for this group
	for _, testCase := range group.TestCases {
		testIcon := getTestCaseStatusIcon(testCase.Status)
		testIndent := strings.Repeat("  ", indent+1)
		durationStr := ""
		if testCase.Duration > 0 {
			durationStr = fmt.Sprintf(" (%.2fs)", testCase.Duration.Seconds())
		}
		sb.WriteString(fmt.Sprintf("%s%s %s%s\n", testIndent, testIcon, testCase.Name, durationStr))

		// Show error for failed tests
		if testCase.Status == TestStatusFail && testCase.Error != nil && testCase.Error.Message != "" {
			errorIndent := strings.Repeat("  ", indent+2)
			sb.WriteString(fmt.Sprintf("%s```\n", errorIndent))
			sb.WriteString(fmt.Sprintf("%s%s\n", errorIndent, testCase.Error.Message))
			sb.WriteString(fmt.Sprintf("%s```\n", errorIndent))
		}
	}

	// Recursively display child groups
	for _, child := range group.Subgroups {
		m.generateGroupReportSection(sb, child, indent+1)
	}

	// Add spacing after root groups
	if len(group.ParentNames) == 0 {
		sb.WriteString("\n")
	}
}

// getGroupStatusIcon returns a status icon for groups in markdown reports
func getGroupStatusIcon(status TestStatus) string {
	switch status {
	case TestStatusPass:
		return "PASS"
	case TestStatusFail:
		return "FAIL"
	case TestStatusSkip:
		return "SKIP"
	case TestStatusRunning:
		return "RUNNING"
	default:
		return "PENDING"
	}
}

// getTestCaseStatusIcon returns a status icon for individual test cases in markdown reports
func getTestCaseStatusIcon(status TestStatus) string {
	switch status {
	case TestStatusPass:
		return "✓"
	case TestStatusFail:
		return "x"
	case TestStatusSkip:
		return "o"
	default:
		return "-"
	}
}

// Finalize completes the test run and closes all resources
func (m *Manager) Finalize(exitCode int, errorDetails ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Always close resources, even if already finalized
	// This ensures cleanup happens even on repeated calls

	// Flush all pending group reports
	if m.groupManager != nil {
		m.groupManager.Flush()
	}

	// Close output.log
	if m.outputFile != nil {
		_ = m.outputFile.Close()
		m.outputFile = nil
	}

	// Close our owned FileLogger if we created one
	if m.ownedFileLogger != nil {
		_ = m.ownedFileLogger.Close()
		m.ownedFileLogger = nil
	}

	// Update final status if we have state and it's not already finalized
	if m.state != nil && m.state.Status != "COMPLETE" && m.state.Status != "ERROR" {
		// Only set ERROR status for actual command errors, not test failures
		if len(errorDetails) > 0 && errorDetails[0] != "" {
			m.state.Status = "ERROR"
			m.state.ErrorDetails = errorDetails[0]
		} else {
			m.state.Status = "COMPLETE"
		}

		// Write final state
		return m.writeState()
	}

	return nil
}

// normalizePath normalizes a file path for comparison
func (m *Manager) normalizePath(filePath string) string {
	// Try to get absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		// If we can't get absolute path, use the original
		return filePath
	}
	return absPath
}

// GetRootGroups returns root groups from the group manager for console display
func (m *Manager) GetRootGroups() []*TestGroup {
	if m.groupManager == nil {
		return nil
	}
	return m.groupManager.GetRootGroups()
}

// GetGroup returns a specific group by ID from the group manager
func (m *Manager) GetGroup(groupID string) (*TestGroup, bool) {
	if m.groupManager == nil {
		return nil, false
	}
	return m.groupManager.GetGroup(groupID)
}

// noopLogger is a default logger that does nothing
type noopLogger struct{}

func (n *noopLogger) Debug(format string, args ...interface{}) {}
func (n *noopLogger) Error(format string, args ...interface{}) {}
func (n *noopLogger) Info(format string, args ...interface{})  {}
