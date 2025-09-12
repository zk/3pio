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
	"github.com/zk/3pio/internal/runner"
)

// Manager handles report generation and file I/O
type Manager struct {
	runDir       string
	state        *ipc.TestRunState
	outputParser runner.OutputParser
	logger       Logger

	// File handles for incremental writing
	fileHandles     map[string]*os.File
	fileBuffers     map[string][]string
	debouncers      map[string]*time.Timer
	outputFile      *os.File
	
	// Separate stdout/stderr buffers for structured reports
	stdoutBuffers   map[string][]string
	stderrBuffers   map[string][]string

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
func NewManager(runDir string, parser runner.OutputParser, logger Logger) (*Manager, error) {
	if logger == nil {
		logger = &noopLogger{}
	}

	// Create run directory
	if err := os.MkdirAll(runDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create run directory: %w", err)
	}

	// Create reports subdirectory
	reportsDir := filepath.Join(runDir, "reports")
	if err := os.MkdirAll(reportsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create reports directory: %w", err)
	}

	// Open output.log file
	outputPath := filepath.Join(runDir, "output.log")
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create output.log: %w", err)
	}

	return &Manager{
		runDir:        runDir,
		outputParser:  parser,
		logger:        logger,
		fileHandles:   make(map[string]*os.File),
		fileBuffers:   make(map[string][]string),
		debouncers:    make(map[string]*time.Timer),
		outputFile:    outputFile,
		stdoutBuffers: make(map[string][]string),
		stderrBuffers: make(map[string][]string),
		debounceTime:  100 * time.Millisecond,
		maxWaitTime:   500 * time.Millisecond,
	}, nil
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
	case ipc.TestFileStartEvent:
		return m.handleTestFileStart(e)

	case ipc.TestCaseEvent:
		return m.handleTestCase(e)

	case ipc.TestFileResultEvent:
		return m.handleTestFileResult(e)

	case ipc.StdoutChunkEvent:
		return m.handleStdoutChunk(e)

	case ipc.StderrChunkEvent:
		return m.handleStderrChunk(e)

	case ipc.CollectionErrorEvent:
		return m.handleCollectionError(e)

	default:
		m.logger.Debug("Unknown event type: %T", event)
	}

	return nil
}

// handleTestFileStart handles test file start events
func (m *Manager) handleTestFileStart(event ipc.TestFileStartEvent) error {
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

	return m.scheduleWrite()
}

// handleTestCase handles individual test case events
func (m *Manager) handleTestCase(event ipc.TestCaseEvent) error {
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

				// Write test case boundary to log file
				// For Jest: Write on RUNNING status (test starting)
				// For Vitest: Write on first event (which is the completion event)
				// This prevents duplicate boundaries while supporting both test runners
				m.appendToFileBuffer(filePath, fmt.Sprintf("\n--- Test: %s ---\n", event.Payload.TestName))

				// If test is already complete (not RUNNING), write the result
				if event.Payload.Status != "RUNNING" {
					m.writeTestResultToLog(filePath, event.Payload)
				}
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
				
				if event.Payload.Status != "RUNNING" {
					// Test case was updated with final status, write the result
					m.writeTestResultToLog(filePath, event.Payload)
				}
			}
			break
		}
	}

	return m.scheduleWrite()
}

// handleTestFileResult handles test file completion events
func (m *Manager) handleTestFileResult(event ipc.TestFileResultEvent) error {
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

	// Flush buffer for this file immediately
	m.flushFileBuffer(filePath)

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

	// Append to file buffer (for legacy format)
	m.appendToFileBuffer(event.Payload.FilePath, event.Payload.Chunk)

	// Also append to stdout buffer for structured reports
	if buffer, ok := m.stdoutBuffers[event.Payload.FilePath]; ok {
		m.stdoutBuffers[event.Payload.FilePath] = append(buffer, event.Payload.Chunk)
	}

	return nil
}

// handleStderrChunk handles stderr output chunks
func (m *Manager) handleStderrChunk(event ipc.StderrChunkEvent) error {
	// Write to output.log
	if _, err := m.outputFile.WriteString(event.Payload.Chunk); err != nil {
		m.logger.Error("Failed to write stderr to output.log: %v", err)
	}

	// Append to file buffer (for legacy format)
	m.appendToFileBuffer(event.Payload.FilePath, event.Payload.Chunk)

	// Also append to stderr buffer for structured reports
	if buffer, ok := m.stderrBuffers[event.Payload.FilePath]; ok {
		m.stderrBuffers[event.Payload.FilePath] = append(buffer, event.Payload.Chunk)
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
	// Create log file with preserved directory structure
	logFileName := sanitizePathForFilesystem(filePath) + ".md"
	logPath := filepath.Join(m.runDir, "reports", logFileName)
	
	// Create parent directories if needed
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		m.logger.Error("Failed to create log directory for %s: %v", filePath, err)
	}

	// Add to state first (so we have timestamps available)
	now := time.Now()

	// Open file handle
	file, err := os.Create(logPath)
	if err != nil {
		m.logger.Error("Failed to create log file for %s: %v", filePath, err)
	} else {
		m.fileHandles[filePath] = file
		m.fileBuffers[filePath] = make([]string, 0)
		m.stdoutBuffers[filePath] = make([]string, 0)
		m.stderrBuffers[filePath] = make([]string, 0)

		// Write structured YAML header to the log file
		filename := filepath.Base(filePath)
		header := fmt.Sprintf(`---
test_file: %s
created: %s
updated: %s
status: RUNNING
---

# Test results for `+"`%s`"+`

## Test case results

`, filePath, now.UTC().Format("2006-01-02T15:04:05.000Z"), now.UTC().Format("2006-01-02T15:04:05.000Z"), filename)
		if _, err := file.WriteString(header); err != nil {
			m.logger.Error("Failed to write header to log file %s: %v", filePath, err)
		}
		_ = file.Sync()
	}
	m.state.TestFiles = append(m.state.TestFiles, ipc.TestFile{
		Status:    ipc.TestStatusPending,
		File:      filePath,
		LogFile:   logFileName,
		TestCases: make([]ipc.TestCase, 0),
		Created:   now,
		Updated:   now,
	})
}

// appendToFileBuffer appends output to a file's buffer
func (m *Manager) appendToFileBuffer(filePath string, content string) {
	if buffer, ok := m.fileBuffers[filePath]; ok {
		m.fileBuffers[filePath] = append(buffer, content)
		m.scheduleFileWrite(filePath)
	}
}

// scheduleFileWrite schedules a debounced write for a specific file
func (m *Manager) scheduleFileWrite(filePath string) {
	// Cancel existing timer if any
	if timer, ok := m.debouncers[filePath]; ok {
		timer.Stop()
	}

	// Create new debounced timer
	m.debouncers[filePath] = time.AfterFunc(m.debounceTime, func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		m.flushFileBuffer(filePath)
	})
}

// flushFileBuffer writes buffered content to file
func (m *Manager) flushFileBuffer(filePath string) {
	buffer, ok := m.fileBuffers[filePath]
	if !ok || len(buffer) == 0 {
		return
	}

	file, ok := m.fileHandles[filePath]
	if !ok {
		return
	}

	// Write all buffered content
	for _, content := range buffer {
		if _, err := file.WriteString(content); err != nil {
			m.logger.Error("Failed to write to log file %s: %v", filePath, err)
		}
	}

	// Clear buffer
	m.fileBuffers[filePath] = make([]string, 0)

	// Sync to disk
	_ = file.Sync()
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

	// Header
	sb.WriteString("# 3pio Test Run\n\n")
	sb.WriteString(fmt.Sprintf("**Started:** %s\n", m.state.Timestamp.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("**Status:** %s\n", m.state.Status))
	sb.WriteString(fmt.Sprintf("**Arguments:** `%s`\n\n", m.state.Arguments))

	// Summary (only show for successful runs, hide for command errors)
	if m.state.Status == "COMPLETE" {
		sb.WriteString("## Summary\n\n")
		sb.WriteString(fmt.Sprintf("- Total Files: %d\n", m.state.TotalFiles))
		sb.WriteString(fmt.Sprintf("- Files Completed: %d\n", m.state.FilesCompleted))
		sb.WriteString(fmt.Sprintf("- Files Passed: %d\n", m.state.FilesPassed))
		sb.WriteString(fmt.Sprintf("- Files Failed: %d\n", m.state.FilesFailed))
		sb.WriteString(fmt.Sprintf("- Files Skipped: %d\n\n", m.state.FilesSkipped))
	}

	// Error details if status is ERROR
	if m.state.Status == "ERROR" && m.state.ErrorDetails != "" {
		sb.WriteString("## Error Details\n\n")
		sb.WriteString("```\n")
		sb.WriteString(m.state.ErrorDetails)
		sb.WriteString("\n```\n\n")
	}

	// Test Files
	for _, tf := range m.state.TestFiles {
		sb.WriteString(fmt.Sprintf("## %s\n\n", tf.File))

		// Status
		statusText := strings.ToUpper(string(tf.Status))
		sb.WriteString(fmt.Sprintf("Status: **%s**\n\n", statusText))

		// Log file link
		if tf.LogFile != "" {
			sb.WriteString(fmt.Sprintf("[Log](./reports/%s)\n\n", tf.LogFile))
		}

		// Test cases
		if len(tf.TestCases) > 0 {
			currentSuite := ""
			for _, tc := range tf.TestCases {
				// Only print suite header when it changes
				if tc.Suite != "" && tc.Suite != currentSuite {
					if currentSuite != "" {
						// Add extra space between different suites
						sb.WriteString("\n")
					}
					sb.WriteString(fmt.Sprintf("### %s\n\n", tc.Suite))
					currentSuite = tc.Suite
				}

				tcIcon := getTestCaseIcon(tc.Status)
				sb.WriteString(fmt.Sprintf("%s %s", tcIcon, tc.Name))

				if tc.Duration > 0 {
					sb.WriteString(fmt.Sprintf(" (%.0fms)", tc.Duration))
				}
				sb.WriteString("\n")

				if tc.Error != "" {
					sb.WriteString(fmt.Sprintf("```\n%s\n```\n", tc.Error))
				}

				// Add newline after each test case for proper spacing
				sb.WriteString("\n")
			}
		}
	}

	// Output log link
	sb.WriteString("[output.log](./output.log)\n\n")

	// Footer
	sb.WriteString(fmt.Sprintf("---\n*Updated: %s*\n", m.state.UpdatedAt.Format(time.RFC3339)))

	return sb.String()
}

// generateIndividualFileReport generates a structured report for a single test file
func (m *Manager) generateIndividualFileReport(tf ipc.TestFile) string {
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
		currentSuite := ""
		for _, tc := range tf.TestCases {
			// Group by suite if present
			if tc.Suite != "" && tc.Suite != currentSuite {
				if currentSuite != "" {
					sb.WriteString("\n")
				}
				currentSuite = tc.Suite
			}

			// Test case line
			icon := getTestCaseIcon(tc.Status)
			sb.WriteString(fmt.Sprintf("%s %s", icon, tc.Name))

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
	// Find the test file in state
	normalizedPath := m.normalizePath(filePath)
	var targetFile *ipc.TestFile
	for i := range m.state.TestFiles {
		if m.normalizePath(m.state.TestFiles[i].File) == normalizedPath {
			targetFile = &m.state.TestFiles[i]
			break
		}
	}

	if targetFile == nil {
		return fmt.Errorf("test file not found in state: %s", filePath)
	}

	// Generate report content
	report := m.generateIndividualFileReport(*targetFile)

	// Write to file
	logFileName := sanitizePathForFilesystem(filePath) + ".md"
	logPath := filepath.Join(m.runDir, "reports", logFileName)
	
	// Create parent directories if needed
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory for %s: %w", filePath, err)
	}

	return os.WriteFile(logPath, []byte(report), 0644)
}

// Finalize completes the test run and closes all resources
func (m *Manager) Finalize(exitCode int, errorDetails ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Flush all buffers
	for filePath := range m.fileBuffers {
		m.flushFileBuffer(filePath)
	}

	// Close all file handles
	for _, file := range m.fileHandles {
		_ = file.Close()
	}

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

// writeTestResultToLog writes the test result to the log file
func (m *Manager) writeTestResultToLog(filePath string, payload struct {
	FilePath  string         `json:"filePath"`
	TestName  string         `json:"testName"`
	SuiteName string         `json:"suiteName,omitempty"`
	Status    ipc.TestStatus `json:"status"`
	Duration  float64        `json:"duration,omitempty"`
	Error     string         `json:"error,omitempty"`
}) {
	icon := getTestCaseIcon(payload.Status)

	// Format the test result line
	var result string
	if payload.Duration > 0 {
		result = fmt.Sprintf("%s %s (%dms)\n", icon, payload.TestName, int(payload.Duration))
	} else {
		result = fmt.Sprintf("%s %s\n", icon, payload.TestName)
	}

	m.appendToFileBuffer(filePath, result)

	// If there's an error, write it with proper indentation
	if payload.Error != "" && payload.Status == ipc.TestStatusFail {
		// Add the error with indentation
		errorLines := strings.Split(payload.Error, "\n")
		var formattedError strings.Builder
		formattedError.WriteString("```\n")
		for _, line := range errorLines {
			formattedError.WriteString(line)
			formattedError.WriteString("\n")
		}
		formattedError.WriteString("```\n")
		m.appendToFileBuffer(filePath, formattedError.String())
	}
}

// noopLogger is a default logger that does nothing
type noopLogger struct{}

func (n *noopLogger) Debug(format string, args ...interface{}) {}
func (n *noopLogger) Error(format string, args ...interface{}) {}
func (n *noopLogger) Info(format string, args ...interface{})  {}
