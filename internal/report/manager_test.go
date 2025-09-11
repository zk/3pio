package report

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zk/3pio/internal/ipc"
	"github.com/zk/3pio/internal/runner"
)

// mockLogger for testing
type mockLogger struct {
	debugMessages []string
	errorMessages []string
	infoMessages  []string
}

func (l *mockLogger) Debug(format string, args ...interface{}) {
	l.debugMessages = append(l.debugMessages, strings.TrimSpace(fmt.Sprintf(format, args...)))
}

func (l *mockLogger) Error(format string, args ...interface{}) {
	l.errorMessages = append(l.errorMessages, strings.TrimSpace(fmt.Sprintf(format, args...)))
}

func (l *mockLogger) Info(format string, args ...interface{}) {
	l.infoMessages = append(l.infoMessages, strings.TrimSpace(fmt.Sprintf(format, args...)))
}

func TestManager_Initialize(t *testing.T) {
	tempDir := t.TempDir()
	logger := &mockLogger{}
	parser := runner.NewJestOutputParser()

	manager, err := NewManager(tempDir, parser, logger)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer func() { _ = manager.Finalize(0) }()

	// Test with empty test files list (dynamic discovery)
	testFiles := []string{}
	args := "npm test"

	if err := manager.Initialize(testFiles, args); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Check that test-run.md was created
	reportPath := filepath.Join(tempDir, "test-run.md")
	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		t.Error("test-run.md was not created")
	}

	// Check that logs directory was created
	logsDir := filepath.Join(tempDir, "logs")
	if _, err := os.Stat(logsDir); os.IsNotExist(err) {
		t.Error("logs directory was not created")
	}

	// Check that output.log was created
	outputLogPath := filepath.Join(tempDir, "output.log")
	if _, err := os.Stat(outputLogPath); os.IsNotExist(err) {
		t.Error("output.log was not created")
	}
}

func TestManager_InitializeWithStaticFiles(t *testing.T) {
	tempDir := t.TempDir()
	logger := &mockLogger{}
	parser := runner.NewJestOutputParser()

	manager, err := NewManager(tempDir, parser, logger)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer func() { _ = manager.Finalize(0) }()

	// Test with static test files
	testFiles := []string{"math.test.js", "string.test.js"}
	args := "npx jest"

	if err := manager.Initialize(testFiles, args); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Check that individual log files were created
	for _, file := range testFiles {
		logPath := filepath.Join(tempDir, "logs", file+".log")
		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			t.Errorf("Log file for %s was not created at %s", file, logPath)
		}
	}
}

func TestManager_HandleTestFileStartEvent(t *testing.T) {
	tempDir := t.TempDir()
	logger := &mockLogger{}
	parser := runner.NewJestOutputParser()

	manager, err := NewManager(tempDir, parser, logger)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer func() { _ = manager.Finalize(0) }()

	// Initialize with empty files (dynamic discovery)
	if err := manager.Initialize([]string{}, "npm test"); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Send a testFileStart event (dynamic registration)
	testFile := "/absolute/path/to/new.test.js"
	event := ipc.TestFileStartEvent{
		EventType: ipc.EventTypeTestFileStart,
		Payload: struct {
			FilePath string `json:"filePath"`
		}{
			FilePath: testFile,
		},
	}

	if err := manager.HandleEvent(event); err != nil {
		t.Fatalf("HandleEvent failed: %v", err)
	}

	// Check that log file was created for the dynamically registered test
	// The Go implementation sanitizes file names by using only the base name
	sanitizedName := filepath.Base(testFile) + ".log"
	logPath := filepath.Join(tempDir, "logs", sanitizedName)
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Errorf("Log file for dynamically registered file was not created at %s", logPath)
	}
}

func TestManager_HandleStdoutChunkEvent(t *testing.T) {
	tempDir := t.TempDir()
	logger := &mockLogger{}
	parser := runner.NewJestOutputParser()

	manager, err := NewManager(tempDir, parser, logger)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer func() { _ = manager.Finalize(0) }()

	testFile := "math.test.js"
	if err := manager.Initialize([]string{testFile}, "npx jest"); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Send stdout chunk event
	testOutput := "✓ should add numbers correctly\n"
	event := ipc.StdoutChunkEvent{
		EventType: ipc.EventTypeStdoutChunk,
		Payload: struct {
			FilePath string `json:"filePath"`
			Chunk    string `json:"chunk"`
		}{
			FilePath: testFile,
			Chunk:    testOutput,
		},
	}

	if err := manager.HandleEvent(event); err != nil {
		t.Fatalf("HandleEvent failed: %v", err)
	}

	// Wait for debounced write
	time.Sleep(200 * time.Millisecond)

	// Check that content was written to test log
	logPath := filepath.Join(tempDir, "logs", testFile+".log")
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read test log: %v", err)
	}

	if !strings.Contains(string(content), testOutput) {
		t.Errorf("Expected test log to contain output, got: %s", string(content))
	}
}

func TestManager_HandleTestCaseEvent(t *testing.T) {
	tempDir := t.TempDir()
	logger := &mockLogger{}
	parser := runner.NewJestOutputParser()

	manager, err := NewManager(tempDir, parser, logger)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer func() { _ = manager.Finalize(0) }()

	testFile := "math.test.js"
	if err := manager.Initialize([]string{testFile}, "npx jest"); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Send test case event
	event := ipc.TestCaseEvent{
		EventType: ipc.EventTypeTestCase,
		Payload: struct {
			FilePath  string         `json:"filePath"`
			TestName  string         `json:"testName"`
			SuiteName string         `json:"suiteName,omitempty"`
			Status    ipc.TestStatus `json:"status"`
			Duration  int            `json:"duration,omitempty"`
			Error     string         `json:"error,omitempty"`
		}{
			FilePath:  testFile,
			TestName:  "should add numbers correctly",
			SuiteName: "Math operations",
			Status:    ipc.TestStatusPass,
			Duration:  10,
		},
	}

	if err := manager.HandleEvent(event); err != nil {
		t.Fatalf("HandleEvent failed: %v", err)
	}

	// Wait for state update
	time.Sleep(100 * time.Millisecond)

	// Check that test case was recorded in the final report
	// We can't directly access internal state, but we can check the final report generation
	// by finalizing the manager and reading the markdown file
	_ = manager.Finalize(0)

	reportPath := filepath.Join(tempDir, "test-run.md")
	content, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("Failed to read test report: %v", err)
	}

	reportContent := string(content)
	if !strings.Contains(reportContent, "should add numbers correctly") {
		t.Errorf("Expected report to contain test case name, got: %s", reportContent)
	}

	if !strings.Contains(reportContent, "Math operations") {
		t.Errorf("Expected report to contain suite name, got: %s", reportContent)
	}
}

func TestManager_HandleTestFileResultEvent(t *testing.T) {
	tempDir := t.TempDir()
	logger := &mockLogger{}
	parser := runner.NewJestOutputParser()

	manager, err := NewManager(tempDir, parser, logger)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer func() { _ = manager.Finalize(0) }()

	testFile := "math.test.js"
	if err := manager.Initialize([]string{testFile}, "npx jest"); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Send test file result event
	event := ipc.TestFileResultEvent{
		EventType: ipc.EventTypeTestFileResult,
		Payload: struct {
			FilePath    string         `json:"filePath"`
			Status      ipc.TestStatus `json:"status"`
			FailedTests []struct {
				Name     string `json:"name"`
				Duration int    `json:"duration,omitempty"`
			} `json:"failedTests,omitempty"`
		}{
			FilePath: testFile,
			Status:   ipc.TestStatusPass,
		},
	}

	if err := manager.HandleEvent(event); err != nil {
		t.Fatalf("HandleEvent failed: %v", err)
	}

	// Wait for state update
	time.Sleep(100 * time.Millisecond)

	// Finalize to generate final report
	_ = manager.Finalize(0)

	reportPath := filepath.Join(tempDir, "test-run.md")
	content, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("Failed to read test report: %v", err)
	}

	reportContent := string(content)
	if !strings.Contains(reportContent, "Status: **PASS**") {
		t.Errorf("Expected report to show PASS status, got: %s", reportContent)
	}
}

func TestManager_TestCaseFormatting(t *testing.T) {
	tempDir := t.TempDir()
	logger := &mockLogger{}
	parser := runner.NewJestOutputParser()

	manager, err := NewManager(tempDir, parser, logger)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer func() { _ = manager.Finalize(0) }()

	testFile := "test.js"
	if err := manager.Initialize([]string{testFile}, "jest"); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Add multiple test cases with different statuses
	// First test case - passing
	event1 := ipc.TestCaseEvent{
		EventType: ipc.EventTypeTestCase,
		Payload: struct {
			FilePath  string         `json:"filePath"`
			TestName  string         `json:"testName"`
			SuiteName string         `json:"suiteName,omitempty"`
			Status    ipc.TestStatus `json:"status"`
			Duration  int            `json:"duration,omitempty"`
			Error     string         `json:"error,omitempty"`
		}{
			FilePath:  testFile,
			TestName:  "should pass",
			SuiteName: "Test Suite",
			Status:    ipc.TestStatusPass,
			Duration:  10,
		},
	}

	// Second test case - failing with error
	event2 := ipc.TestCaseEvent{
		EventType: ipc.EventTypeTestCase,
		Payload: struct {
			FilePath  string         `json:"filePath"`
			TestName  string         `json:"testName"`
			SuiteName string         `json:"suiteName,omitempty"`
			Status    ipc.TestStatus `json:"status"`
			Duration  int            `json:"duration,omitempty"`
			Error     string         `json:"error,omitempty"`
		}{
			FilePath:  testFile,
			TestName:  "should fail",
			SuiteName: "Test Suite",
			Status:    ipc.TestStatusFail,
			Duration:  5,
			Error:     "Error: Expected true to be false\n    at line 10",
		},
	}

	// Third test case - another passing test
	event3 := ipc.TestCaseEvent{
		EventType: ipc.EventTypeTestCase,
		Payload: struct {
			FilePath  string         `json:"filePath"`
			TestName  string         `json:"testName"`
			SuiteName string         `json:"suiteName,omitempty"`
			Status    ipc.TestStatus `json:"status"`
			Duration  int            `json:"duration,omitempty"`
			Error     string         `json:"error,omitempty"`
		}{
			FilePath:  testFile,
			TestName:  "should also pass",
			SuiteName: "Test Suite",
			Status:    ipc.TestStatusPass,
			Duration:  8,
		},
	}

	// Handle all events
	_ = manager.HandleEvent(event1)
	_ = manager.HandleEvent(event2)
	_ = manager.HandleEvent(event3)

	// Wait for state updates
	time.Sleep(100 * time.Millisecond)

	// Finalize and check report
	_ = manager.Finalize(0)

	reportPath := filepath.Join(tempDir, "test-run.md")
	content, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("Failed to read test report: %v", err)
	}

	reportContent := string(content)

	// Check that there's proper spacing after the error block
	// The pattern should be: error block ending with ``` followed by TWO newlines before the next test
	if !strings.Contains(reportContent, "```\n\n### Test Suite") {
		// Check for the specific formatting pattern
		lines := strings.Split(reportContent, "\n")
		foundErrorBlock := false
		properSpacing := false

		for i, line := range lines {
			if strings.Contains(line, "```") && foundErrorBlock {
				// This is the closing of an error block
				// Check if there's an empty line after it before the next test case
				if i+1 < len(lines) && lines[i+1] == "" {
					if i+2 < len(lines) && (strings.HasPrefix(lines[i+2], "✓") ||
						strings.HasPrefix(lines[i+2], "✕") ||
						strings.HasPrefix(lines[i+2], "###")) {
						properSpacing = true
						break
					}
				}
			}
			if strings.Contains(line, "Error:") {
				foundErrorBlock = true
			}
		}

		if !properSpacing {
			t.Errorf("Expected proper spacing after error blocks in report.\nGot:\n%s", reportContent)
		}
	}

	// Verify all test cases are present
	if !strings.Contains(reportContent, "should pass") {
		t.Errorf("Missing 'should pass' test case in report")
	}
	if !strings.Contains(reportContent, "should fail") {
		t.Errorf("Missing 'should fail' test case in report")
	}
	if !strings.Contains(reportContent, "should also pass") {
		t.Errorf("Missing 'should also pass' test case in report")
	}

	// Verify error is included
	if !strings.Contains(reportContent, "Error: Expected true to be false") {
		t.Errorf("Missing error message in report")
	}
}

func TestManager_HandleRunCompleteEvent(t *testing.T) {
	tempDir := t.TempDir()
	logger := &mockLogger{}
	parser := runner.NewJestOutputParser()

	manager, err := NewManager(tempDir, parser, logger)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer func() { _ = manager.Finalize(0) }()

	if err := manager.Initialize([]string{}, "npm test"); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Send runComplete event - should not cause error
	event := ipc.RunCompleteEvent{
		EventType: ipc.EventTypeRunComplete,
		Payload:   struct{}{},
	}

	if err := manager.HandleEvent(event); err != nil {
		t.Errorf("HandleEvent should handle runComplete gracefully, got error: %v", err)
	}

	// Should not generate any error logs
	for _, msg := range logger.errorMessages {
		if strings.Contains(msg, "runComplete") {
			t.Errorf("Unexpected error message about runComplete: %s", msg)
		}
	}
}

func TestManager_Debouncing(t *testing.T) {
	tempDir := t.TempDir()
	logger := &mockLogger{}
	parser := runner.NewJestOutputParser()

	manager, err := NewManager(tempDir, parser, logger)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer func() { _ = manager.Finalize(0) }()

	testFile := "math.test.js"
	if err := manager.Initialize([]string{testFile}, "npx jest"); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Send multiple stdout chunks quickly (should be debounced)
	for i := 0; i < 5; i++ {
		event := ipc.StdoutChunkEvent{
			EventType: ipc.EventTypeStdoutChunk,
			Payload: struct {
				FilePath string `json:"filePath"`
				Chunk    string `json:"chunk"`
			}{
				FilePath: testFile,
				Chunk:    fmt.Sprintf("Line %d\n", i),
			},
		}

		if err := manager.HandleEvent(event); err != nil {
			t.Fatalf("HandleEvent failed: %v", err)
		}

		time.Sleep(10 * time.Millisecond) // Much less than debounce delay
	}

	// Wait for debounce to trigger
	time.Sleep(200 * time.Millisecond)

	// Check that all content was written (debouncing should collect all writes)
	logPath := filepath.Join(tempDir, "logs", testFile+".log")
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read test log: %v", err)
	}

	for i := 0; i < 5; i++ {
		expected := fmt.Sprintf("Line %d", i)
		if !strings.Contains(string(content), expected) {
			t.Errorf("Expected test log to contain %s, got: %s", expected, string(content))
		}
	}
}

func TestManager_NoDuplicateTestBoundaries(t *testing.T) {
	tempDir := t.TempDir()
	logger := &mockLogger{}
	parser := runner.NewJestOutputParser()

	manager, err := NewManager(tempDir, parser, logger)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer func() { _ = manager.Finalize(0) }()

	// Initialize with a test file
	testFile := "test.spec.js"
	if err := manager.Initialize([]string{testFile}, "npm test"); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Send multiple events for the same test case (simulating RUNNING then PASS)
	testName := "should work correctly"
	suiteName := "Test Suite"

	// First event: RUNNING status
	runningEvent := ipc.TestCaseEvent{
		EventType: ipc.EventTypeTestCase,
	}
	runningEvent.Payload.FilePath = testFile
	runningEvent.Payload.TestName = testName
	runningEvent.Payload.SuiteName = suiteName
	runningEvent.Payload.Status = ipc.TestStatusRunning

	if err := manager.HandleEvent(runningEvent); err != nil {
		t.Fatalf("HandleEvent failed for RUNNING: %v", err)
	}

	// Second event: PASS status (duplicate)
	passEvent1 := ipc.TestCaseEvent{
		EventType: ipc.EventTypeTestCase,
	}
	passEvent1.Payload.FilePath = testFile
	passEvent1.Payload.TestName = testName
	passEvent1.Payload.SuiteName = suiteName
	passEvent1.Payload.Status = ipc.TestStatusPass
	passEvent1.Payload.Duration = 10

	if err := manager.HandleEvent(passEvent1); err != nil {
		t.Fatalf("HandleEvent failed for first PASS: %v", err)
	}

	// Third event: Another PASS status (another duplicate)
	passEvent2 := ipc.TestCaseEvent{
		EventType: ipc.EventTypeTestCase,
	}
	passEvent2.Payload.FilePath = testFile
	passEvent2.Payload.TestName = testName
	passEvent2.Payload.SuiteName = suiteName
	passEvent2.Payload.Status = ipc.TestStatusPass
	passEvent2.Payload.Duration = 10

	if err := manager.HandleEvent(passEvent2); err != nil {
		t.Fatalf("HandleEvent failed for second PASS: %v", err)
	}

	// Add some stdout output after the test
	stdoutEvent := ipc.StdoutChunkEvent{
		EventType: ipc.EventTypeStdoutChunk,
	}
	stdoutEvent.Payload.FilePath = testFile
	stdoutEvent.Payload.Chunk = "Test output here\n"

	if err := manager.HandleEvent(stdoutEvent); err != nil {
		t.Fatalf("HandleEvent failed for stdout: %v", err)
	}

	// Wait for debounce to complete
	time.Sleep(200 * time.Millisecond)

	// Read the log file and verify only one test boundary was written
	logPath := filepath.Join(tempDir, "logs", testFile+".log")
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read test log: %v", err)
	}

	// Count occurrences of the test boundary
	boundary := fmt.Sprintf("--- Test: %s ---", testName)
	occurrences := strings.Count(string(content), boundary)

	if occurrences != 1 {
		t.Errorf("Expected exactly 1 test boundary, but found %d occurrences\nLog content:\n%s", occurrences, string(content))
	}

	// Verify the test output is still present
	if !strings.Contains(string(content), "Test output here") {
		t.Errorf("Expected stdout output to be present in log")
	}
}

func TestManager_TestCaseOutputAssociation(t *testing.T) {
	// Create temp directory for test output
	tempDir, err := os.MkdirTemp("", "3pio-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create manager
	manager, err := NewManager(tempDir, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer func() { _ = manager.Finalize(0) }()

	testFile := "example.test.js"

	// Initialize with the test file
	_ = manager.Initialize([]string{testFile}, "test command")

	// Test 1: Start test "foo" (RUNNING status like Jest does)
	test1StartEvent := ipc.TestCaseEvent{
		EventType: ipc.EventTypeTestCase,
	}
	test1StartEvent.Payload.FilePath = testFile
	test1StartEvent.Payload.TestName = "should foo correctly"
	test1StartEvent.Payload.Status = "RUNNING" // Jest sends RUNNING first

	if err := manager.HandleEvent(test1StartEvent); err != nil {
		t.Fatalf("HandleEvent failed for test 1 start: %v", err)
	}

	// Output from test 1
	stdout1Event := ipc.StdoutChunkEvent{
		EventType: ipc.EventTypeStdoutChunk,
	}
	stdout1Event.Payload.FilePath = testFile
	stdout1Event.Payload.Chunk = "Output from foo test\n"

	if err := manager.HandleEvent(stdout1Event); err != nil {
		t.Fatalf("HandleEvent failed for stdout 1: %v", err)
	}

	// Test 1: Complete (PASS status)
	test1CompleteEvent := ipc.TestCaseEvent{
		EventType: ipc.EventTypeTestCase,
	}
	test1CompleteEvent.Payload.FilePath = testFile
	test1CompleteEvent.Payload.TestName = "should foo correctly"
	test1CompleteEvent.Payload.Status = ipc.TestStatusPass
	test1CompleteEvent.Payload.Duration = 10

	if err := manager.HandleEvent(test1CompleteEvent); err != nil {
		t.Fatalf("HandleEvent failed for test 1 complete: %v", err)
	}

	// Test 2: Start test "bar" (RUNNING)
	test2StartEvent := ipc.TestCaseEvent{
		EventType: ipc.EventTypeTestCase,
	}
	test2StartEvent.Payload.FilePath = testFile
	test2StartEvent.Payload.TestName = "should bar correctly"
	test2StartEvent.Payload.Status = "RUNNING"

	if err := manager.HandleEvent(test2StartEvent); err != nil {
		t.Fatalf("HandleEvent failed for test 2 start: %v", err)
	}

	// Output from test 2
	stdout2Event := ipc.StdoutChunkEvent{
		EventType: ipc.EventTypeStdoutChunk,
	}
	stdout2Event.Payload.FilePath = testFile
	stdout2Event.Payload.Chunk = "Output from bar test\n"

	if err := manager.HandleEvent(stdout2Event); err != nil {
		t.Fatalf("HandleEvent failed for stdout 2: %v", err)
	}

	// Test 2: Complete (PASS)
	test2CompleteEvent := ipc.TestCaseEvent{
		EventType: ipc.EventTypeTestCase,
	}
	test2CompleteEvent.Payload.FilePath = testFile
	test2CompleteEvent.Payload.TestName = "should bar correctly"
	test2CompleteEvent.Payload.Status = ipc.TestStatusPass
	test2CompleteEvent.Payload.Duration = 15

	if err := manager.HandleEvent(test2CompleteEvent); err != nil {
		t.Fatalf("HandleEvent failed for test 2 complete: %v", err)
	}

	// Test 3: Start test "baz" (RUNNING)
	test3StartEvent := ipc.TestCaseEvent{
		EventType: ipc.EventTypeTestCase,
	}
	test3StartEvent.Payload.FilePath = testFile
	test3StartEvent.Payload.TestName = "should baz correctly"
	test3StartEvent.Payload.Status = "RUNNING"

	if err := manager.HandleEvent(test3StartEvent); err != nil {
		t.Fatalf("HandleEvent failed for test 3 start: %v", err)
	}

	// Output from test 3 (stderr)
	stderr3Event := ipc.StderrChunkEvent{
		EventType: ipc.EventTypeStderrChunk,
	}
	stderr3Event.Payload.FilePath = testFile
	stderr3Event.Payload.Chunk = "Error output from baz test\n"

	if err := manager.HandleEvent(stderr3Event); err != nil {
		t.Fatalf("HandleEvent failed for stderr 3: %v", err)
	}

	// Test 3: Complete (FAIL)
	test3CompleteEvent := ipc.TestCaseEvent{
		EventType: ipc.EventTypeTestCase,
	}
	test3CompleteEvent.Payload.FilePath = testFile
	test3CompleteEvent.Payload.TestName = "should baz correctly"
	test3CompleteEvent.Payload.Status = ipc.TestStatusFail
	test3CompleteEvent.Payload.Duration = 5
	test3CompleteEvent.Payload.Error = "Expected true to be false"

	if err := manager.HandleEvent(test3CompleteEvent); err != nil {
		t.Fatalf("HandleEvent failed for test 3 complete: %v", err)
	}

	// Jest sends duplicate events at the end (simulate this behavior)
	// These should NOT create duplicate test boundaries
	for _, dupEvent := range []ipc.TestCaseEvent{
		test1CompleteEvent,
		test2CompleteEvent,
		test3CompleteEvent,
	} {
		if err := manager.HandleEvent(dupEvent); err != nil {
			t.Fatalf("HandleEvent failed for duplicate event: %v", err)
		}
	}

	// Wait for debounce to complete
	time.Sleep(200 * time.Millisecond)

	// Read the log file
	logPath := filepath.Join(tempDir, "logs", testFile+".log")
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read test log: %v", err)
	}

	logContent := string(content)

	// Debug: Print actual content
	t.Logf("Actual log content:\n%s", logContent)

	// Expected format:
	// --- Test: should foo correctly ---
	// Output from foo test
	//
	// --- Test: should bar correctly ---
	// Output from bar test
	//
	// --- Test: should baz correctly ---
	// Error output from baz test

	// Check that each test boundary appears exactly once
	boundaries := []string{
		"--- Test: should foo correctly ---",
		"--- Test: should bar correctly ---",
		"--- Test: should baz correctly ---",
	}

	for _, boundary := range boundaries {
		count := strings.Count(logContent, boundary)
		if count != 1 {
			t.Errorf("Expected boundary %q to appear exactly once, got %d occurrences", boundary, count)
		}
	}

	// Check that output is associated with the correct test
	// Find the position of each boundary
	foo_pos := strings.Index(logContent, boundaries[0])
	bar_pos := strings.Index(logContent, boundaries[1])
	baz_pos := strings.Index(logContent, boundaries[2])

	if foo_pos == -1 || bar_pos == -1 || baz_pos == -1 {
		t.Fatal("Not all test boundaries found in log")
	}

	// Check order
	if foo_pos >= bar_pos || bar_pos >= baz_pos {
		t.Error("Test boundaries are not in the expected order")
	}

	// Check that foo's output comes after foo's boundary but before bar's boundary
	foo_output_pos := strings.Index(logContent, "Output from foo test")
	if foo_output_pos == -1 {
		t.Error("Foo test output not found")
	} else if foo_output_pos <= foo_pos || foo_output_pos >= bar_pos {
		t.Error("Foo test output is not properly associated with foo test")
	}

	// Check that bar's output comes after bar's boundary but before baz's boundary
	bar_output_pos := strings.Index(logContent, "Output from bar test")
	if bar_output_pos == -1 {
		t.Error("Bar test output not found")
	} else if bar_output_pos <= bar_pos || bar_output_pos >= baz_pos {
		t.Error("Bar test output is not properly associated with bar test")
	}

	// Check that baz's error output comes after baz's boundary
	baz_output_pos := strings.Index(logContent, "Error output from baz test")
	if baz_output_pos == -1 {
		t.Error("Baz test error output not found")
	} else if baz_output_pos <= baz_pos {
		t.Error("Baz test error output is not properly associated with baz test")
	}
}
