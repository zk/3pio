package report

import (
	"fmt"
	"math"
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
	// Cast the logger to FileLogger if possible, otherwise create a new one
	var fileLogger *logger.FileLogger
	if fl, ok := lg.(*logger.FileLogger); ok {
		fileLogger = fl
	} else {
		// Create a new file logger if cast fails
		var err error
		fileLogger, err = logger.NewFileLogger()
		if err != nil {
			return nil, fmt.Errorf("failed to create file logger for group manager: %w", err)
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
	// File-based events are now handled by GroupManager
	case ipc.TestFileStartEvent:
		if m.groupManager != nil {
			// Convert to group event
			return m.groupManager.ProcessGroupDiscovered(ipc.GroupDiscoveredEvent{
				Payload: ipc.GroupDiscoveredPayload{
					GroupName:    e.Payload.FilePath,
					ParentNames:  []string{},
				},
			})
		}
		return nil

	case ipc.TestCaseEvent:
		if m.groupManager != nil {
			// First ensure parent groups are discovered
			if e.Payload.FilePath != "" {
				// Discover file group
				_ = m.groupManager.ProcessGroupDiscovered(ipc.GroupDiscoveredEvent{
					Payload: ipc.GroupDiscoveredPayload{
						GroupName:   e.Payload.FilePath,
						ParentNames: []string{},
					},
				})

				// If there's a suite, discover it too
				if e.Payload.SuiteName != "" {
					_ = m.groupManager.ProcessGroupDiscovered(ipc.GroupDiscoveredEvent{
						Payload: ipc.GroupDiscoveredPayload{
							GroupName:   e.Payload.SuiteName,
							ParentNames: []string{e.Payload.FilePath},
						},
					})
				}
			}

			// Now convert to group test case event
			parentNames := []string{}
			if e.Payload.FilePath != "" {
				parentNames = append(parentNames, e.Payload.FilePath)
			}
			if e.Payload.SuiteName != "" {
				parentNames = append(parentNames, e.Payload.SuiteName)
			}
			// Convert error string to TestError pointer if needed
			var testError *ipc.TestError
			if e.Payload.Error != "" {
				testError = &ipc.TestError{
					Message: e.Payload.Error,
				}
			}
			return m.groupManager.ProcessTestCase(ipc.GroupTestCaseEvent{
				Payload: ipc.TestCasePayload{
					TestName:     e.Payload.TestName,
					ParentNames:  parentNames,
					Status:       string(e.Payload.Status),
					Duration:     e.Payload.Duration,
					Error:        testError,
				},
			})
		}
		return nil

	case ipc.TestFileResultEvent:
		if m.groupManager != nil {
			// Convert to group result event
			return m.groupManager.ProcessGroupResult(ipc.GroupResultEvent{
				Payload: ipc.GroupResultPayload{
					GroupName:    e.Payload.FilePath,
					ParentNames:  []string{},
					Status:       string(e.Payload.Status),
				},
			})
		}
		return nil

	case ipc.StdoutChunkEvent:
		return m.handleStdoutChunk(e)

	case ipc.StderrChunkEvent:
		return m.handleStderrChunk(e)

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

// Removed legacy handleTestFileStart - now handled by group manager
// Legacy function removed
func (m *Manager) handleTestFileStart_REMOVED(event ipc.TestFileStartEvent) error {
	filePath := event.Payload.FilePath

	// Ensure file is registered (dynamic discovery)
	m.ensureTestFileRegisteredInternal(filePath)

	// Update status to RUNNING and update timestamp
	for i := range m.state.TestFiles {
		if m.state.TestFiles[i].File == filePath {
			m.state.TestFiles[i].Status = ipc.TestStatusRunning
			m.state.TestFiles[i].Updated = time.Now()
			break
		}
	}

	// Test-run.md updates now handled by group manager

	return m.scheduleWrite()
}

// Removed legacy handleTestCase - now handled by group manager
// Legacy function removed
func (m *Manager) handleTestCase_REMOVED(event ipc.TestCaseEvent) error {
	filePath := event.Payload.FilePath

	// Ensure file is registered
	m.ensureTestFileRegisteredInternal(filePath)

	// Find the test file and add/update test case
	normalizedPath := m.normalizePath(filePath)
	for i := range m.state.TestFiles {
		if m.normalizePath(m.state.TestFiles[i].File) == normalizedPath {
			// Update timestamp for any test case activity
			m.state.TestFiles[i].Updated = time.Now()

			// Initialize test cases if needed
			if m.state.TestFiles[i].TestCases == nil {
				m.state.TestFiles[i].TestCases = make([]ipc.TestCase, 0)
			}

			// Check if test case already exists (update) or add new
			found := false
			for j := range m.state.TestFiles[i].TestCases {
				tc := &m.state.TestFiles[i].TestCases[j]
				if tc.Name == event.Payload.TestName && tc.Suite == event.Payload.SuiteName {
					// Update existing test case
					tc.Status = event.Payload.Status
					tc.Duration = event.Payload.Duration
					tc.Error = event.Payload.Error
					found = true
					break
				}
			}

			if !found {
				// Add new test case
				m.state.TestFiles[i].TestCases = append(m.state.TestFiles[i].TestCases, ipc.TestCase{
					Name:     event.Payload.TestName,
					Suite:    event.Payload.SuiteName,
					Status:   event.Payload.Status,
					Duration: event.Payload.Duration,
					Error:    event.Payload.Error,
				})

				// Immediately regenerate and update the individual file report
				// Individual file reports removed - using group reports
			} else {
				// Update existing test case with new information
				for j := range m.state.TestFiles[i].TestCases {
					if m.state.TestFiles[i].TestCases[j].Name == event.Payload.TestName &&
						(m.state.TestFiles[i].TestCases[j].Suite == event.Payload.SuiteName ||
							m.state.TestFiles[i].TestCases[j].Suite == "" && event.Payload.SuiteName == "") {
						// Update test case fields
						if event.Payload.Status != "" {
							m.state.TestFiles[i].TestCases[j].Status = event.Payload.Status
						}
						if event.Payload.Duration > 0 {
							m.state.TestFiles[i].TestCases[j].Duration = event.Payload.Duration
						}
						if event.Payload.Error != "" {
							m.state.TestFiles[i].TestCases[j].Error = event.Payload.Error
						}
						break
					}
				}

				// Regenerate and update the individual file report
				// Individual file reports removed - using group reports
			}
			break
		}
	}

	return m.scheduleWrite()
}

// Removed legacy handleTestFileResult - now handled by group manager
// Legacy function removed
func (m *Manager) handleTestFileResult_REMOVED(event ipc.TestFileResultEvent) error {
	filePath := event.Payload.FilePath

	// Update file status and counters
	normalizedPath := m.normalizePath(filePath)
	for i := range m.state.TestFiles {
		if m.normalizePath(m.state.TestFiles[i].File) == normalizedPath {
			m.state.TestFiles[i].Status = event.Payload.Status
			m.state.TestFiles[i].Updated = time.Now()
			m.state.FilesCompleted++

			switch event.Payload.Status {
			case ipc.TestStatusPass:
				m.state.FilesPassed++
			case ipc.TestStatusFail:
				m.state.FilesFailed++
			case ipc.TestStatusSkip:
				m.state.FilesSkipped++
			}

			break
		}
	}

	// Regenerate individual file report
	// Individual file reports removed - using group reports

	// Test-run.md updates now handled by group manager

	// Generate structured individual file report
	if err := m.writeIndividualFileReport(filePath); err != nil {
		m.logger.Error("Failed to write individual file report for %s: %v", filePath, err)
	}

	return m.scheduleWrite()
}

// handleStdoutChunk handles stdout output chunks
func (m *Manager) handleStdoutChunk(event ipc.StdoutChunkEvent) error {
	// Write to output.log
	if _, err := m.outputFile.WriteString(event.Payload.Chunk); err != nil {
		m.logger.Error("Failed to write stdout to output.log: %v", err)
	}

	// Append to stdout buffer for structured reports
	if buffer, ok := m.stdoutBuffers[event.Payload.FilePath]; ok {
		m.stdoutBuffers[event.Payload.FilePath] = append(buffer, event.Payload.Chunk)
		// Regenerate individual file report to include the new stdout content
		// Individual file reports removed - using group reports
	}

	return nil
}

// handleStderrChunk handles stderr output chunks
func (m *Manager) handleStderrChunk(event ipc.StderrChunkEvent) error {
	// Write to output.log
	if _, err := m.outputFile.WriteString(event.Payload.Chunk); err != nil {
		m.logger.Error("Failed to write stderr to output.log: %v", err)
	}

	// Append to stderr buffer for structured reports
	if buffer, ok := m.stderrBuffers[event.Payload.FilePath]; ok {
		m.stderrBuffers[event.Payload.FilePath] = append(buffer, event.Payload.Chunk)
		// Regenerate individual file report to include the new stderr content
		// Individual file reports removed - using group reports
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
	// Legacy file registration removed - handled by group manager
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

// Removed legacy generateLegacyReport - now using group-based reporting
// Legacy function removed
func (m *Manager) generateLegacyReport_REMOVED(sb *strings.Builder, statusText string) {
	// Summary (show when we have test files, hide for command errors)
	if statusText != "ERRORED" && len(m.state.TestFiles) > 0 {
		sb.WriteString("## Summary\n\n")
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("- Total files: %d\n", m.state.TotalFiles))
		sb.WriteString(fmt.Sprintf("- Files completed: %d\n", m.state.FilesCompleted))
		sb.WriteString(fmt.Sprintf("- Files passed: %d\n", m.state.FilesPassed))
		sb.WriteString(fmt.Sprintf("- Files failed: %d\n", m.state.FilesFailed))
		sb.WriteString(fmt.Sprintf("- Files skipped: %d\n", m.state.FilesSkipped))

		// Calculate pending files
		pendingFiles := m.state.TotalFiles - m.state.FilesCompleted
		sb.WriteString(fmt.Sprintf("- Files pending: %d\n", pendingFiles))
		sb.WriteString(fmt.Sprintf("- Total duration: %.2fs\n\n", m.getTotalDuration()))
	}

	// Test file results section (show for any status with test files)
	if len(m.state.TestFiles) > 0 {
		sb.WriteString("## Test file results\n\n")
		sb.WriteString("| Stat | Test | Duration | Report file |\n")
		sb.WriteString("| ---- | ---- | -------- | ----------- |\n")
		for _, tf := range m.state.TestFiles {
			statusIcon := getTestFileStatusText(tf.Status)
			filename := filepath.Base(tf.File)

			// Calculate total duration for this test file
			var totalDuration float64
			for _, tc := range tf.TestCases {
				if tc.Duration > 0 {
					totalDuration += tc.Duration
				}
			}
			// Convert milliseconds to seconds with 2 decimal places
			durationStr := fmt.Sprintf("%.2fs", totalDuration/1000.0)

			reportPath := fmt.Sprintf("./reports/%s.md", sanitizePathForFilesystem(tf.File))
			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", statusIcon, filename, durationStr, reportPath))
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

// getTotalDuration calculates total duration from all test files
func (m *Manager) getTotalDuration() float64 {
	totalDuration := 0.0
	for _, tf := range m.state.TestFiles {
		for _, tc := range tf.TestCases {
			if tc.Duration > 0 {
				totalDuration += tc.Duration / 1000.0 // Convert ms to seconds
			}
		}
	}
	return totalDuration
}

// getTestFileStatusText returns the status text for test file results section
func getTestFileStatusText(status ipc.TestStatus) string {
	switch status {
	case ipc.TestStatusPass:
		return "PASS"
	case ipc.TestStatusFail:
		return "FAIL"
	case ipc.TestStatusSkip:
		return "SKIP"
	default:
		return "PEND"
	}
}

// Removed legacy updateTestRunReport - now handled by group manager
// Legacy function removed
func (m *Manager) updateTestRunReport_REMOVED() {
	m.state.UpdatedAt = time.Now()
	report := m.generateMarkdownReport()

	reportPath := filepath.Join(m.runDir, "test-run.md")
	if err := os.WriteFile(reportPath, []byte(report), 0644); err != nil {
		m.logger.Error("Failed to write test-run.md: %v", err)
	}
}

// Removed legacy generateIndividualFileReport - now using group-based reporting
// Legacy function removed
func (m *Manager) generateIndividualFileReport_REMOVED(tf ipc.TestFile) string {
	var sb strings.Builder

	// Extract filename from full path for title
	filename := filepath.Base(tf.File)

	// YAML frontmatter
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("test_file: %s\n", tf.File))
	sb.WriteString(fmt.Sprintf("created: %s\n", tf.Created.UTC().Format("2006-01-02T15:04:05.000Z")))
	sb.WriteString(fmt.Sprintf("updated: %s\n", tf.Updated.UTC().Format("2006-01-02T15:04:05.000Z")))

	// Map internal status to spec status
	var statusText string
	switch tf.Status {
	case ipc.TestStatusPending:
		statusText = "PENDING"
	case ipc.TestStatusRunning:
		statusText = "RUNNING"
	case ipc.TestStatusPass, ipc.TestStatusFail, ipc.TestStatusSkip:
		statusText = "COMPLETED"
	default:
		if tf.ExecutionError != "" {
			statusText = "ERRORED"
		} else {
			statusText = "COMPLETED"
		}
	}
	sb.WriteString(fmt.Sprintf("status: %s\n", statusText))
	sb.WriteString("---\n\n")

	// Title
	sb.WriteString(fmt.Sprintf("# Test results for `%s`\n\n", filename))

	// Test case results section
	if len(tf.TestCases) > 0 {
		sb.WriteString("## Test case results\n\n")
		currentSuite := ""
		for _, tc := range tf.TestCases {
			// Group by suite if present
			if tc.Suite != "" && tc.Suite != currentSuite {
				if currentSuite != "" {
					sb.WriteString("\n")
				}
				currentSuite = tc.Suite
			}

			// Test case line as a list item
			icon := getTestCaseIcon(tc.Status)
			sb.WriteString(fmt.Sprintf("- %s %s", icon, tc.Name))

			// Always show duration, even if 0ms
			// Round to nearest millisecond for display
			if tc.Duration >= 0 {
				roundedDuration := math.Round(tc.Duration)
				sb.WriteString(fmt.Sprintf(" (%.0fms)", roundedDuration))
			}
			sb.WriteString("\n")

			// Test failure error (not execution error)
			if tc.Error != "" && tc.Status == ipc.TestStatusFail {
				sb.WriteString("```\n")
				sb.WriteString(tc.Error)
				sb.WriteString("\n```\n")
			}
		}
	}

	// Add spacing after test results before next section
	if len(tf.TestCases) > 0 {
		sb.WriteString("\n")
	}

	// Execution error section (only if execution error occurred)
	if tf.ExecutionError != "" {
		sb.WriteString("## Error\n\n")
		sb.WriteString(tf.ExecutionError)
		sb.WriteString("\n\n")
	}

	// stdout/stderr section (only if content exists)
	stdoutContent := strings.Join(m.stdoutBuffers[tf.File], "")
	stderrContent := strings.Join(m.stderrBuffers[tf.File], "")

	if stdoutContent != "" || stderrContent != "" {
		sb.WriteString("## stdout/stderr\n\n")
		sb.WriteString("```\n")
		if stdoutContent != "" {
			sb.WriteString(stdoutContent)
		}
		if stderrContent != "" {
			sb.WriteString(stderrContent)
		}
		sb.WriteString("\n```\n\n")
	}

	return sb.String()
}

// writeIndividualFileReport writes the structured report for a single test file
func (m *Manager) writeIndividualFileReport(filePath string) error {
	// Legacy file reports removed - now handled by group manager
	return nil
}

// Finalize completes the test run and closes all resources
func (m *Manager) Finalize(exitCode int, errorDetails ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Individual file reports removed - now handled by group manager

	// Close output.log
	if m.outputFile != nil {
		_ = m.outputFile.Close()
	}

	// Update final status
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

// sanitizePathForFilesystem sanitizes a file path to preserve directory structure
// while preventing directory traversal and filesystem issues
func sanitizePathForFilesystem(filePath string) string {
	// Clean the path to normalize it
	cleanPath := filepath.Clean(filePath)

	// Try to make the path relative to the current working directory
	cwd, err := os.Getwd()
	if err == nil {
		if absPath, err := filepath.Abs(cleanPath); err == nil {
			if relPath, err := filepath.Rel(cwd, absPath); err == nil {
				cleanPath = relPath
			}
		}
	}

	// Remove specific extensions while preserving JS/TS extensions
	// This ensures the report path matches what we show in console output
	ext := filepath.Ext(cleanPath)
	switch ext {
	case ".js", ".ts", ".jsx", ".tsx", ".mjs", ".cjs":
		// Keep JavaScript/TypeScript extensions
	case ".go", ".py", ".rb", ".java", ".c", ".cpp", ".rs":
		// Remove other language extensions
		cleanPath = strings.TrimSuffix(cleanPath, ext)
	}

	// Handle paths that start with ".." by replacing with "_UP"
	parts := strings.Split(cleanPath, string(filepath.Separator))

	// Filter out empty parts and sanitize
	var filteredParts []string
	for _, part := range parts {
		if part == "" {
			continue
		}
		if part == ".." {
			filteredParts = append(filteredParts, "_UP")
		} else if part == "." {
			// Skip current directory markers unless it's the only part
			if len(parts) > 1 {
				continue
			}
			filteredParts = append(filteredParts, "_DOT")
		} else {
			// Sanitize other problematic characters in each part
			part = strings.ReplaceAll(part, ":", "_")
			part = strings.ReplaceAll(part, "*", "_")
			part = strings.ReplaceAll(part, "?", "_")
			part = strings.ReplaceAll(part, "\"", "_")
			part = strings.ReplaceAll(part, "<", "_")
			part = strings.ReplaceAll(part, ">", "_")
			part = strings.ReplaceAll(part, "|", "_")
			filteredParts = append(filteredParts, part)
		}
	}

	// If we end up with no parts, use a default
	if len(filteredParts) == 0 {
		return "test"
	}

	// Rejoin the sanitized parts
	return filepath.Join(filteredParts...)
}

// getTestCaseIcon returns an icon for individual test cases
func getTestCaseIcon(status ipc.TestStatus) string {
	switch status {
	case ipc.TestStatusPass:
		return "✓"
	case ipc.TestStatusFail:
		return "✕"
	case ipc.TestStatusSkip:
		return "○"
	default:
		return "~" // Running or unknown
	}
}

// Removed legacy updateIndividualFileReport - now using group-based reporting
// Legacy function removed
func (m *Manager) updateIndividualFileReport_REMOVED(filePath string) {
	// Find the test file in state
	// Legacy file report update removed - handled by group manager
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
