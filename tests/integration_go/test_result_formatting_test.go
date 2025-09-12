package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestTestResultFormattingInLogFiles tests that test results are properly formatted
// in individual log files across all test fixtures
func TestTestResultFormattingInLogFiles(t *testing.T) {
	// Build path to the 3pio binary
	binaryPath, err := filepath.Abs("../../build/3pio")
	if err != nil {
		t.Fatalf("Failed to get absolute path to binary: %v", err)
	}

	// Verify binary exists
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Fatalf("3pio binary not found at %s. Run 'make build' first.", binaryPath)
	}

	// Test cases for different fixtures
	testCases := []struct {
		fixtureName     string
		command         []string
		expectedResults []string // Expected test result patterns in log files
	}{
		{
			fixtureName: "basic-jest",
			command:     []string{"npx", "jest"},
			expectedResults: []string{
				"✓ should concatenate strings",
				"✓ should add numbers correctly", 
				"✕ should fail this test",
				"○ should skip this test",
				"✓ should convert to uppercase",
			},
		},
		{
			fixtureName: "basic-vitest", 
			command:     []string{"npx", "vitest", "run"},
			expectedResults: []string{
				"✓ should concatenate strings",
				"✓ should add numbers correctly",
				"✕ should fail this test", 
				"○ should skip this test",
				"✓ should convert to uppercase",
			},
		},
		{
			fixtureName: "basic-pytest",
			command:     []string{"python", "-m", "pytest", "-v"},
			expectedResults: []string{
				"✓ TestMathOperations::test_add_numbers_correctly",
				"✓ TestStringOperations::test_concatenate_strings",
				"✕ TestStringOperations::test_fail_this_test",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.fixtureName, func(t *testing.T) {
			// Change to fixture directory
			fixtureDir := filepath.Join("../fixtures", tc.fixtureName)
			
			// Clean up any existing .3pio directory
			threepioDir := filepath.Join(fixtureDir, ".3pio")
			if err := os.RemoveAll(threepioDir); err != nil && !os.IsNotExist(err) {
				t.Fatalf("Failed to clean up .3pio directory: %v", err)
			}

			// Prepare command with 3pio
			args := append([]string{binaryPath}, tc.command...)
			cmd := exec.Command(args[0], args[1:]...)
			cmd.Dir = fixtureDir

			// Run the command (may fail due to test failures, which is expected)
			output, err := cmd.CombinedOutput()
			t.Logf("Command output for %s:\n%s", tc.fixtureName, string(output))

			// For pytest, if it's not installed, skip the test
			if tc.fixtureName == "basic-pytest" && strings.Contains(string(output), "No module named pytest") {
				t.Skip("pytest not installed, skipping test")
			}

			// Check that .3pio directory was created
			if _, err := os.Stat(threepioDir); os.IsNotExist(err) {
				t.Fatalf("Expected .3pio directory to be created in %s", fixtureDir)
			}

			// Find the run directory (should be only one)
			runsDir := filepath.Join(threepioDir, "runs")
			runDirs, err := os.ReadDir(runsDir)
			if err != nil {
				t.Fatalf("Failed to read runs directory: %v", err)
			}

			if len(runDirs) != 1 {
				t.Fatalf("Expected exactly one run directory, found %d", len(runDirs))
			}

			runDir := filepath.Join(runsDir, runDirs[0].Name())
			logsDir := filepath.Join(runDir, "logs")

			// Check that logs directory exists
			if _, err := os.Stat(logsDir); os.IsNotExist(err) {
				t.Fatalf("Expected logs directory to exist at %s", logsDir)
			}

			// Read all log files and verify test results are present
			logFiles, err := os.ReadDir(logsDir)
			if err != nil {
				t.Fatalf("Failed to read logs directory: %v", err)
			}

			if len(logFiles) == 0 {
				t.Fatalf("Expected at least one log file, found none")
			}

			// Collect all log content
			var allLogContent strings.Builder
			for _, logFile := range logFiles {
				logPath := filepath.Join(logsDir, logFile.Name())
				content, err := os.ReadFile(logPath)
				if err != nil {
					t.Fatalf("Failed to read log file %s: %v", logPath, err)
				}

				t.Logf("Log file %s content:\n%s", logFile.Name(), string(content))
				allLogContent.Write(content)
			}

			logContent := allLogContent.String()

			// Verify expected test results are present
			for _, expectedResult := range tc.expectedResults {
				if !strings.Contains(logContent, expectedResult) {
					t.Errorf("Expected test result '%s' not found in log files for %s", expectedResult, tc.fixtureName)
				}
			}

			// Verify that test results have proper formatting (icons and durations where applicable)
			// Check for passing tests with checkmarks
			if !strings.Contains(logContent, "✓") {
				t.Errorf("Expected to find passing test indicators (✓) in log files for %s", tc.fixtureName)
			}

			// For Jest and Vitest, we expect failing and skipped tests
			if tc.fixtureName == "basic-jest" || tc.fixtureName == "basic-vitest" {
				if !strings.Contains(logContent, "✕") {
					t.Errorf("Expected to find failing test indicators (✕) in log files for %s", tc.fixtureName)
				}
				if !strings.Contains(logContent, "○") {
					t.Errorf("Expected to find skipped test indicators (○) in log files for %s", tc.fixtureName)
				}
				
				// Check for error details in failing tests (code blocks)
				if !strings.Contains(logContent, "```") {
					t.Errorf("Expected to find error details in code blocks (```) for failing tests in %s", tc.fixtureName)
				}
			}

			// For pytest, check for failing test
			if tc.fixtureName == "basic-pytest" {
				if !strings.Contains(logContent, "✕") {
					t.Errorf("Expected to find failing test indicators (✕) in log files for %s", tc.fixtureName)
				}
			}

			// Check that durations are included where available
			// Look for patterns like "(5ms)" or "(10ms)" 
			durationPattern := "ms)"
			if tc.fixtureName != "basic-pytest" { // pytest might not always have durations
				if !strings.Contains(logContent, durationPattern) {
					t.Errorf("Expected to find test durations in log files for %s", tc.fixtureName)
				}
			}
		})
	}
}