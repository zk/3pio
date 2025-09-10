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
	defer manager.Finalize(0)

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
	defer manager.Finalize(0)

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
	defer manager.Finalize(0)

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
	defer manager.Finalize(0)

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
	defer manager.Finalize(0)

	testFile := "math.test.js"
	if err := manager.Initialize([]string{testFile}, "npx jest"); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Send test case event
	event := ipc.TestCaseEvent{
		EventType: ipc.EventTypeTestCase,
		Payload: struct {
			FilePath  string          `json:"filePath"`
			TestName  string          `json:"testName"`
			SuiteName string          `json:"suiteName,omitempty"`
			Status    ipc.TestStatus  `json:"status"`
			Duration  int             `json:"duration,omitempty"`
			Error     string          `json:"error,omitempty"`
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
	manager.Finalize(0)

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
	defer manager.Finalize(0)

	testFile := "math.test.js"
	if err := manager.Initialize([]string{testFile}, "npx jest"); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Send test file result event
	event := ipc.TestFileResultEvent{
		EventType: ipc.EventTypeTestFileResult,
		Payload: struct {
			FilePath     string          `json:"filePath"`
			Status       ipc.TestStatus  `json:"status"`
			FailedTests  []struct {
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
	manager.Finalize(0)

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
	defer manager.Finalize(0)
	
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
			Status:   ipc.TestStatusPass,
			Duration: 10,
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
			Status:   ipc.TestStatusFail,
			Duration: 5,
			Error:    "Error: Expected true to be false\n    at line 10",
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
			Status:   ipc.TestStatusPass,
			Duration: 8,
		},
	}
	
	// Handle all events
	manager.HandleEvent(event1)
	manager.HandleEvent(event2)
	manager.HandleEvent(event3)
	
	// Wait for state updates
	time.Sleep(100 * time.Millisecond)
	
	// Finalize and check report
	manager.Finalize(0)
	
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
	defer manager.Finalize(0)

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
	defer manager.Finalize(0)

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
	defer manager.Finalize(0)

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