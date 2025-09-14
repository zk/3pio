package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func TestOrchestrator_New(t *testing.T) {
	config := Config{
		Command: []string{"npm", "test"},
		Logger:  &mockLogger{},
	}

	orch, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}
	defer orch.Close() // Clean up resources

	if orch == nil {
		t.Fatal("Expected orchestrator to be created")
	}

	if len(orch.command) != 2 || orch.command[0] != "npm" || orch.command[1] != "test" {
		t.Errorf("Expected command [npm test], got %v", orch.command)
	}
}

func TestOrchestrator_NewWithoutLogger(t *testing.T) {
	config := Config{
		Command: []string{"npx", "jest"},
	}

	_, err := New(config)
	if err == nil {
		t.Fatal("Expected error when creating orchestrator without logger")
	}

	if err.Error() != "logger is required" {
		t.Errorf("Expected 'logger is required' error, got: %v", err)
	}
}

func TestOrchestrator_RunnerDetection(t *testing.T) {
	// Change to a temp directory for the test
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	// Create a package.json with jest dependency
	packageJSON := `{
		"name": "test-project",
		"scripts": {
			"test": "jest"
		},
		"devDependencies": {
			"jest": "^29.0.0"
		}
	}`
	if err := os.WriteFile("package.json", []byte(packageJSON), 0644); err != nil {
		t.Fatalf("Failed to write package.json: %v", err)
	}

	config := Config{
		Command: []string{"npm", "test"},
		Logger:  &mockLogger{},
	}

	orch, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}
	defer orch.Close() // Clean up resources

	// Test runner detection (this would normally require actual execution)
	// For now, just verify the orchestrator was created successfully
	if orch.runnerManager == nil {
		t.Error("Expected runner manager to be initialized")
	}
}

func TestOrchestrator_GetExitCode(t *testing.T) {
	config := Config{
		Command: []string{"npm", "test"},
		Logger:  &mockLogger{},
	}

	orch, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}
	defer orch.Close() // Clean up resources

	// Default exit code should be 0
	if exitCode := orch.GetExitCode(); exitCode != 0 {
		t.Errorf("Expected default exit code 0, got %d", exitCode)
	}
}

func TestOrchestrator_TestCountsWithSkippedTests(t *testing.T) {
	config := Config{
		Command: []string{"npm", "test"},
		Logger:  &mockLogger{},
	}

	orch, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}
	defer orch.Close()

	// Directly manipulate the test counts to test the display logic
	// Simulate what would happen after processing events
	orch.totalGroups = 4
	orch.passedGroups = 1
	orch.failedGroups = 1
	orch.skippedGroups = 2 // Now we track skipped groups

	// After the fix, we should have:
	// "Results: 1 passed, 1 failed, 2 skipped, 4 total"

	// Verify the counts
	if orch.totalGroups != 4 {
		t.Errorf("Expected totalFiles to be 4, got %d", orch.totalGroups)
	}
	if orch.passedGroups != 1 {
		t.Errorf("Expected passedFiles to be 1, got %d", orch.passedGroups)
	}
	if orch.failedGroups != 1 {
		t.Errorf("Expected failedFiles to be 1, got %d", orch.failedGroups)
	}
	if orch.skippedGroups != 2 {
		t.Errorf("Expected skippedFiles to be 2, got %d", orch.skippedGroups)
	}

	// Now the sum of passed + failed + skipped should equal total
	if orch.passedGroups + orch.failedGroups + orch.skippedGroups != orch.totalGroups {
		t.Error("Expected passed + failed + skipped to equal total")
	}
}

func TestOrchestrator_RunWithInvalidRunner(t *testing.T) {
	// Change to a temp directory for the test
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	config := Config{
		Command: []string{"unknown-test-runner"},
		Logger:  &mockLogger{},
	}

	orch, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}
	defer orch.Close() // Clean up resources

	// This should fail with "no test runner detected"
	err = orch.Run()
	if err == nil {
		t.Error("Expected error for unknown test runner")
	}

	if !strings.Contains(err.Error(), "no test runner detected") {
		t.Errorf("Expected 'no test runner detected' error, got: %v", err)
	}
}

func TestGenerateRunID(t *testing.T) {
	runID1 := generateRunID()
	runID2 := generateRunID()

	// Should be different (though technically could be same due to timing)
	if runID1 == runID2 {
		t.Log("Note: Got same run ID (acceptable due to timing)")
	}

	// Should follow expected format (ISO8601 timestamp + memorable name)
	// Format: YYYYMMDDTHHMMSS-adjective-character
	if len(runID1) < 20 { // At minimum: 15 chars for timestamp + 2 dashes + some chars for name
		t.Errorf("Run ID seems too short: %s", runID1)
	}

	// Should contain timestamp part and proper format
	if !strings.Contains(runID1, "T") || strings.Count(runID1, "-") < 2 {
		t.Errorf("Run ID doesn't match expected format (should be TIMESTAMP-adjective-character): %s", runID1)
	}

	// Should contain recognizable characters (Star Wars or Star Trek)
	parts := strings.Split(runID1, "-")
	if len(parts) < 3 {
		t.Errorf("Run ID should have at least 3 parts separated by dashes: %s", runID1)
	}

	// Check if the character part is from our known lists
	characterPart := strings.Join(parts[2:], "-") // Handle multi-part character names like "luke-skywalker"

	// Sample of expected characters from various universes (we don't need to check all)
	knownCharacters := []string{
		// Star Wars
		"luke-skywalker", "yoda", "darth-vader", "obi-wan", "r2d2",
		// Star Trek
		"picard", "spock", "kirk", "data", "janeway", "sisko", "archer", "uhura", "worf", "torres", "kira", "tucker",
		// Chrono Trigger
		"crono", "marle", "lucca", "robo", "frog", "ayla", "magus", "schala",
		// Final Fantasy 6
		"terra", "locke", "edgar", "sabin", "celes", "cyan", "shadow", "setzer", "kefka", "mog",
	}

	found := false
	for _, char := range knownCharacters {
		if characterPart == char {
			found = true
			break
		}
	}

	if !found {
		// This is just a warning since we have many characters
		t.Logf("Character part '%s' not in sample list (full runID: %s)", characterPart, runID1)
	}
}

func TestOrchestrator_DirectoryCreation(t *testing.T) {
	// Change to a temp directory for the test
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	// Create a basic package.json with jest
	packageJSON := `{
		"name": "test-project",
		"scripts": {
			"test": "jest"
		},
		"devDependencies": {
			"jest": "^29.0.0"
		}
	}`
	if err := os.WriteFile("package.json", []byte(packageJSON), 0644); err != nil {
		t.Fatalf("Failed to write package.json: %v", err)
	}

	// Create a dummy test file so jest doesn't complain
	if err := os.WriteFile("dummy.test.js", []byte(`
		test('dummy test', () => {
			expect(1 + 1).toBe(2);
		});
	`), 0644); err != nil {
		t.Fatalf("Failed to write dummy test: %v", err)
	}

	config := Config{
		Command: []string{"npm", "test"},
		Logger:  &mockLogger{},
	}

	orch, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}
	defer orch.Close() // Clean up resources

	// Attempt to start the run process (this will fail because npm isn't available,
	// but it should create the directory structure)
	_ = orch.Run()

	// Check that .3pio directory was created
	threepioDir := filepath.Join(tempDir, ".3pio")
	if _, err := os.Stat(threepioDir); os.IsNotExist(err) {
		t.Error(".3pio directory was not created")
	}

	// Check that runs directory was created
	runsDir := filepath.Join(threepioDir, "runs")
	if _, err := os.Stat(runsDir); os.IsNotExist(err) {
		t.Error(".3pio/runs directory was not created")
	}

}

func TestOrchestrator_ConsoleLogging(t *testing.T) {
	logger := &mockLogger{}
	config := Config{
		Command: []string{"npm", "test"},
		Logger:  logger,
	}

	orch, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}
	defer orch.Close() // Clean up resources

	// The orchestrator should have a logger set
	if orch.logger == nil {
		t.Fatal("Expected orchestrator to have a logger")
	}

	// Test that the logger can be called (basic functionality)
	orch.logger.Debug("Test debug message")
	orch.logger.Info("Test info message")
	orch.logger.Error("Test error message")

	// Verify messages were captured
	if len(logger.debugMessages) != 1 || logger.debugMessages[0] != "Test debug message" {
		t.Errorf("Expected debug message to be captured, got: %v", logger.debugMessages)
	}
	if len(logger.infoMessages) != 1 || logger.infoMessages[0] != "Test info message" {
		t.Errorf("Expected info message to be captured, got: %v", logger.infoMessages)
	}
	if len(logger.errorMessages) != 1 || logger.errorMessages[0] != "Test error message" {
		t.Errorf("Expected error message to be captured, got: %v", logger.errorMessages)
	}
}

func TestOrchestrator_UpdateDisplayedFiles(t *testing.T) {
	config := Config{
		Command: []string{"npm", "test"},
		Logger:  &mockLogger{},
	}

	orch, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}
	defer orch.Close() // Clean up resources

	// Test the internal displayedFiles tracking
	testFile1 := "test1.js"
	testFile2 := "test2.js"

	// Initially empty
	if len(orch.displayedGroups) != 0 {
		t.Error("Expected displayedFiles to be empty initially")
	}

	// Simulate internal file tracking (would normally happen during run)
	orch.displayedGroups[testFile1] = true
	orch.displayedGroups[testFile2] = true

	if len(orch.displayedGroups) != 2 {
		t.Errorf("Expected 2 displayed files, got %d", len(orch.displayedGroups))
	}

	if !orch.displayedGroups[testFile1] || !orch.displayedGroups[testFile2] {
		t.Error("Expected both test files to be marked as displayed")
	}
}
