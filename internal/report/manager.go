package report

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/zk/3pio/internal/ipc"
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

	// File handles for incremental writing
	fileHandles map[string]*os.File
	fileBuffers map[string][]string
	debouncers  map[string]*time.Timer
	outputFile  *os.File

	// Separate stdout/stderr buffers for structured reports
	stdoutBuffers map[string][]string
	stderrBuffers map[string][]string

	// Debouncing for main report writes
	writeTimer   *time.Timer
	writeMutex   sync.Mutex
	pendingWrite bool

	mu           sync.RWMutex
	debounceTime time.Duration
	maxWaitTime  time.Duration

	// Track test run start time for wall-clock duration
	startTime time.Time
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
	groupManager := NewGroupManager(runDir, "", lg)

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
		pendingWrite:    false,
		debounceTime:    200 * time.Millisecond,
		maxWaitTime:     500 * time.Millisecond,
		startTime:       time.Now(),
	}, nil
}

// UpdateModifiedCommand updates the modified command after adapter extraction
func (m *Manager) UpdateModifiedCommand(command string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.modifiedCommand = command
}

// Initialize sets up the initial test run state
func (m *Manager) Initialize(args string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	m.startTime = now // Record exact start time for wall-clock duration
	m.state = &ipc.TestRunState{
		Timestamp: now,
		Status:    "RUNNING",
		UpdatedAt: now,
		Arguments: args,
		// File-based tracking removed - using group-based model
		TestFiles: make([]ipc.TestFile, 0),
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

	// Group events - forward to GroupManager and trigger report updates
	case ipc.GroupDiscoveredEvent:
		if m.groupManager != nil {
			err := m.groupManager.ProcessGroupDiscovered(e)
			if err != nil {
				return err
			}
			// Schedule test-run.md update when new groups are discovered
			return m.scheduleWrite()
		}

	case ipc.GroupStartEvent:
		if m.groupManager != nil {
			err := m.groupManager.ProcessGroupStart(e)
			if err != nil {
				return err
			}
			// Update test-run.md to show group as RUNNING
			return m.scheduleWrite()
		}

	case ipc.GroupResultEvent:
		if m.groupManager != nil {
			err := m.groupManager.ProcessGroupResult(e)
			if err != nil {
				return err
			}
			// Update test-run.md with final group status
			return m.scheduleWrite()
		}

	case ipc.GroupErrorEvent:
		if m.groupManager != nil {
			err := m.groupManager.ProcessGroupError(e)
			if err != nil {
				return err
			}
			// Update test-run.md with error information
			return m.scheduleWrite()
		}

	case ipc.GroupTestCaseEvent:
		if m.groupManager != nil {
			err := m.groupManager.ProcessTestCase(e)
			if err != nil {
				return err
			}
			// Update test-run.md with incremental test counts
			return m.scheduleWrite()
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

// Legacy file registration methods removed - using group-based model

// scheduleWrite schedules a debounced state write
func (m *Manager) scheduleWrite() error {
	m.writeMutex.Lock()
	defer m.writeMutex.Unlock()

	m.pendingWrite = true

	// Cancel existing timer
	if m.writeTimer != nil {
		m.writeTimer.Stop()
	}

	// Schedule write after debounceTime of inactivity
	m.writeTimer = time.AfterFunc(m.debounceTime, func() {
		m.flushWrite()
	})

	return nil
}

// flushWrite executes pending write to disk
func (m *Manager) flushWrite() {
	m.writeMutex.Lock()
	if !m.pendingWrite {
		m.writeMutex.Unlock()
		return
	}
	m.pendingWrite = false
	m.writeMutex.Unlock()

	if err := m.writeState(); err != nil {
		m.logger.Error("Failed to write state: %v", err)
	}
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
	sb := &strings.Builder{}

	// Extract run ID from runDir
	runID := filepath.Base(m.runDir)

	// Map internal status to spec status
	var statusText string
	// Use group-based tracking for completion status
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

	// YAML frontmatter
	sb.WriteString("---\n")
	fmt.Fprintf(sb, "run_id: %s\n", runID)
	fmt.Fprintf(sb, "run_path: %s\n", m.runDir)
	fmt.Fprintf(sb, "detected_runner: %s\n", m.detectedRunner)
	fmt.Fprintf(sb, "modified_command: `%s`\n", m.modifiedCommand)
	fmt.Fprintf(sb, "created: %s\n", m.state.Timestamp.UTC().Format("2006-01-02T15:04:05.000Z"))
	fmt.Fprintf(sb, "updated: %s\n", m.state.UpdatedAt.UTC().Format("2006-01-02T15:04:05.000Z"))
	fmt.Fprintf(sb, "status: %s\n", statusText)
	sb.WriteString("---\n\n")

	// Header
	sb.WriteString("# 3pio Test Run\n\n")
	fmt.Fprintf(sb, "- Test command: `%s`\n", m.state.Arguments)
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
		m.generateGroupBasedReport(sb, statusText)
	} else {
		// No test results to report
		sb.WriteString("## Test Results\n\n")
		sb.WriteString("No test results available.\n")
	}

	return sb.String()
}

// generateGroupBasedReport generates summary and results using hierarchical group data
func (m *Manager) generateGroupBasedReport(sb *strings.Builder, statusText string) {
	// Summary section with test case statistics
	if statusText != "ERRORED" {
		sb.WriteString("## Summary\n\n")

		// Calculate test case statistics from group data
		rootGroups := m.groupManager.GetRootGroups()
		totalTestCases := 0
		completedTestCases := 0
		passedTestCases := 0
		failedTestCases := 0
		skippedTestCases := 0
		runningTestCases := 0

		// Calculate wall-clock duration from start time
		totalDuration := time.Since(m.startTime).Seconds()

		for _, group := range rootGroups {
			// Count all test cases in the group and its subgroups
			totalTestCases += countTotalTestCases(group)
			completedTestCases += countCompletedTestCases(group)
			passedTestCases += countPassedTestCases(group)
			failedTestCases += countFailedTestCases(group)
			skippedTestCases += countSkippedTestCases(group)
			runningTestCases += countRunningTestCases(group)
		}

		fmt.Fprintf(sb, "- Total test cases: %d\n", totalTestCases)
		fmt.Fprintf(sb, "- Test cases completed: %d\n", completedTestCases)
		if runningTestCases > 0 {
			fmt.Fprintf(sb, "- Test cases running: %d\n", runningTestCases)
		}
		fmt.Fprintf(sb, "- Test cases passed: %d\n", passedTestCases)
		fmt.Fprintf(sb, "- Test cases failed: %d\n", failedTestCases)
		fmt.Fprintf(sb, "- Test cases skipped: %d\n", skippedTestCases)
		fmt.Fprintf(sb, "- Total duration: %.2fs\n\n", totalDuration)
	}

	// Test group results section with table format
	if len(m.groupManager.GetRootGroups()) > 0 {
		sb.WriteString("## Test group results\n\n")
		sb.WriteString("| Status | Name | Tests | Duration | Report |\n")
		sb.WriteString("|--------|------|-------|----------|--------|\n")

		for _, group := range m.groupManager.GetRootGroups() {
			statusStr := strings.ToUpper(string(group.Status))
			if statusStr == "" {
				statusStr = "PENDING"
			}
			filename := filepath.Base(group.Name)

			// Tests column - show breakdown of test results including running tests
			var testsStr string
			// Calculate recursive counts on-the-fly for accurate display
			runningCount := countRunningTestCases(group)
			totalCount := countTotalTestCases(group)
			passedCount := countPassedTestCases(group)
			failedCount := countFailedTestCases(group)
			skippedCount := countSkippedTestCases(group)

			if statusStr == "RUNNING" || runningCount > 0 {
				// Show running progress
				parts := []string{}
				if passedCount > 0 {
					parts = append(parts, fmt.Sprintf("%d passed", passedCount))
				}
				if failedCount > 0 {
					parts = append(parts, fmt.Sprintf("%d failed", failedCount))
				}
				if runningCount > 0 {
					parts = append(parts, fmt.Sprintf("%d running", runningCount))
				}
				if skippedCount > 0 {
					parts = append(parts, fmt.Sprintf("%d skipped", skippedCount))
				}
				if totalCount > 0 && len(parts) == 0 {
					// No tests have started yet
					testsStr = fmt.Sprintf("%d pending", totalCount)
				} else if len(parts) > 0 {
					testsStr = strings.Join(parts, ", ")
				} else {
					testsStr = "-"
				}
			} else if totalCount > 0 {
				// Completed group - show final results based on recursive counts
				parts := []string{}
				if passedCount > 0 {
					parts = append(parts, fmt.Sprintf("%d passed", passedCount))
				}
				if failedCount > 0 {
					parts = append(parts, fmt.Sprintf("%d failed", failedCount))
				}
				if skippedCount > 0 {
					parts = append(parts, fmt.Sprintf("%d skipped", skippedCount))
				}
				testsStr = strings.Join(parts, ", ")
			} else if group.Stats.SetupFailed {
				// Setup failure - no tests ran
				testsStr = "setup failed"
			} else {
				testsStr = "0 tests"
			}

			// Duration column - show elapsed time for running groups, final duration for completed
			var durationStr string
			if statusStr == "RUNNING" && !group.StartTime.IsZero() {
				// Show elapsed time for running groups
				elapsed := time.Since(group.StartTime).Seconds()
				durationStr = fmt.Sprintf("%.2fs", elapsed)
			} else if group.Duration > 0 {
				// Show final duration for completed groups
				durationStr = fmt.Sprintf("%.2fs", group.Duration.Seconds())
			} else if statusStr == "PENDING" {
				// Not started yet
				durationStr = "-"
			} else {
				// Fallback
				durationStr = "0.00s"
			}

			// Generate report file path
			reportFile := GetReportFilePath(group, m.runDir)
			// Make it relative to the run directory
			if relPath, err := filepath.Rel(m.runDir, reportFile); err == nil {
				reportFile = "./" + relPath
			}

			fmt.Fprintf(sb, "| %s | %s | %s | %s | %s |\n", statusStr, filename, testsStr, durationStr, reportFile)
		}
	}
}

// Helper functions to count test cases recursively
func countTotalTestCases(group *TestGroup) int {
	count := len(group.TestCases)
	for _, subgroup := range group.Subgroups {
		count += countTotalTestCases(subgroup)
	}
	return count
}

func countCompletedTestCases(group *TestGroup) int {
	count := 0
	for _, test := range group.TestCases {
		if test.Status != TestStatusPending && test.Status != TestStatusRunning {
			count++
		}
	}
	for _, subgroup := range group.Subgroups {
		count += countCompletedTestCases(subgroup)
	}
	return count
}

func countPassedTestCases(group *TestGroup) int {
	count := 0
	for _, test := range group.TestCases {
		if test.Status == TestStatusPass {
			count++
		}
	}
	for _, subgroup := range group.Subgroups {
		count += countPassedTestCases(subgroup)
	}
	return count
}

func countFailedTestCases(group *TestGroup) int {
	count := 0
	for _, test := range group.TestCases {
		if test.Status == TestStatusFail {
			count++
		}
	}
	for _, subgroup := range group.Subgroups {
		count += countFailedTestCases(subgroup)
	}
	return count
}

func countSkippedTestCases(group *TestGroup) int {
	count := 0
	for _, test := range group.TestCases {
		if test.Status == TestStatusSkip {
			count++
		}
	}
	for _, subgroup := range group.Subgroups {
		count += countSkippedTestCases(subgroup)
	}
	return count
}

func countRunningTestCases(group *TestGroup) int {
	count := 0
	for _, test := range group.TestCases {
		if test.Status == TestStatusRunning || test.Status == TestStatusPending {
			count++
		}
	}
	for _, subgroup := range group.Subgroups {
		count += countRunningTestCases(subgroup)
	}
	return count
}

// generateGroupReportSection is kept for potential future use but currently not called
// It generates a hierarchical report section for a group
/* func (m *Manager) generateGroupReportSection(sb *strings.Builder, group *TestGroup, indent int) {
	indentStr := strings.Repeat("  ", indent)

	// File-level groups (root groups)
	if len(group.ParentNames) == 0 {
		statusIcon := getGroupStatusIcon(group.Status)
		filename := filepath.Base(group.Name)
		durationStr := fmt.Sprintf("%.2fs", group.Duration.Seconds())

		fmt.Fprintf(sb, "%s**%s** - %s %s", indentStr, filename, statusIcon, durationStr)

		// Add test counts
		if group.Stats.TotalTests > 0 {
			fmt.Fprintf(sb, " (%d tests: %d passed, %d failed, %d skipped)",
				group.Stats.TotalTests, group.Stats.PassedTests, group.Stats.FailedTests, group.Stats.SkippedTests)
		}
		sb.WriteString("\n")

		// Add report link for failed files
		// Report links are now handled by the group's own report path
		// which is generated in the group-based directory structure
	} else {
		// Suite-level groups
		statusIcon := getGroupStatusIcon(group.Status)
		fmt.Fprintf(sb, "%s**%s** - %s", indentStr, group.Name, statusIcon)

		if group.Stats.TotalTests > 0 {
			fmt.Fprintf(sb, " (%d tests)", group.Stats.TotalTests)
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
		fmt.Fprintf(sb, "%s%s %s%s\n", testIndent, testIcon, testCase.Name, durationStr)

		// Show error for failed tests
		if testCase.Status == TestStatusFail && testCase.Error != nil && testCase.Error.Message != "" {
			errorIndent := strings.Repeat("  ", indent+2)
			fmt.Fprintf(sb, "%s```\n", errorIndent)
			fmt.Fprintf(sb, "%s%s\n", errorIndent, testCase.Error.Message)
			fmt.Fprintf(sb, "%s```\n", errorIndent)
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
} */

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

	// Update final status if we have state and it's not already ERROR
	if m.state != nil && m.state.Status != "ERROR" {
		// Cancel any pending timer to prevent race condition
		if m.writeTimer != nil {
			m.writeTimer.Stop()
			m.writeTimer = nil
		}

		// Set ERROR status for actual command errors, COMPLETE otherwise
		if len(errorDetails) > 0 && errorDetails[0] != "" {
			m.state.Status = "ERROR"
			m.state.ErrorDetails = errorDetails[0]
		} else {
			m.state.Status = "COMPLETE"
		}

		// Write final state immediately (bypass debouncing)
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
