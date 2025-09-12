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

	// Test cases for ALL fixtures
	testCases := []struct {
		fixtureName     string
		command         []string
		expectedResults []string // Expected test result patterns in log files
		shouldHaveTests bool     // Whether we expect tests to run (some fixtures test error handling)
		skipReason      string   // Reason to skip if needed
	}{
		// Basic fixtures with actual tests
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
			shouldHaveTests: true,
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
			shouldHaveTests: true,
		},
		{
			fixtureName: "basic-pytest",
			command:     []string{"python", "-m", "pytest", "-v"},
			expectedResults: []string{
				"✓ TestMathOperations::test_add_numbers_correctly",
				"✓ TestStringOperations::test_concatenate_strings",
				"✕ TestStringOperations::test_fail_this_test",
			},
			shouldHaveTests: true,
		},
		// Empty fixtures (minimal or no test presence)
		{
			fixtureName:     "empty-jest",
			command:         []string{"npx", "jest"},
			expectedResults: []string{}, // No actual tests in empty.test.js
			shouldHaveTests:  false, // Empty suite with no actual tests
			skipReason:       "Empty test suite - no actual test cases",
		},
		{
			fixtureName:     "empty-vitest", 
			command:         []string{"npx", "vitest", "run"},
			expectedResults: []string{}, // No actual tests in empty.test.js
			shouldHaveTests:  false, // Empty suite with no actual tests
			skipReason:       "Empty test suite - no actual test cases",
		},
		{
			fixtureName: "empty-pytest",
			command:     []string{"python", "-m", "pytest", "-v"},
			expectedResults: []string{
				"✕ test_should_fail", // test_failures.py has actual failing tests
				"✕ test_another_failure",
			},
			shouldHaveTests: true,
		},
		// ESM fixture
		{
			fixtureName: "jest-esm",
			command:     []string{"npm", "test"}, // Uses NODE_OPTIONS
			expectedResults: []string{
				"✓ should add", // math.test.js
			},
			shouldHaveTests: true,
		},
		// Long names fixtures
		{
			fixtureName: "long-names-jest",
			command:     []string{"npx", "jest"},
			expectedResults: []string{
				"✓", // longname.test.js has tests
			},
			shouldHaveTests: true,
		},
		{
			fixtureName: "long-names-vitest",
			command:     []string{"npx", "vitest", "run"},
			expectedResults: []string{
				"✓", // longname.test.js has tests
			},
			shouldHaveTests: true,
		},
		// Monorepo fixture
		{
			fixtureName: "monorepo-vitest",
			command:     []string{"npx", "vitest", "run"},
			expectedResults: []string{
				"✓", // Has tests in packages
			},
			shouldHaveTests: true,
		},
		// NPM separator fixtures
		{
			fixtureName: "npm-separator-jest",
			command:     []string{"npm", "test"},
			expectedResults: []string{
				"✓", // example.test.js
			},
			shouldHaveTests: true,
		},
		{
			fixtureName:     "npm-separator-vitest",
			command:         []string{"npm", "test"},
			expectedResults: []string{},
			shouldHaveTests:  false, // Vitest config issues prevent test discovery
			skipReason:       "Vitest configuration causes test discovery issues",
		},
		// Config error fixtures (may not have successful tests)
		{
			fixtureName:     "jest-config-error",
			command:         []string{"npx", "jest"},
			expectedResults: []string{},
			shouldHaveTests:  false, // This tests config errors
			skipReason:       "Config error fixture - tests configuration failures",
		},
		{
			fixtureName:     "jest-ts-config-error", 
			command:         []string{"npx", "jest"},
			expectedResults: []string{},
			shouldHaveTests:  false, // This tests TS config errors
			skipReason:       "TS config error fixture - tests TypeScript configuration failures",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.fixtureName, func(t *testing.T) {
			// Skip if this fixture is designed for error testing
			if !tc.shouldHaveTests {
				t.Skipf("Skipping %s: %s", tc.fixtureName, tc.skipReason)
			}

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

			// For pytest fixtures, if pytest is not installed, skip the test
			if strings.Contains(tc.fixtureName, "pytest") && strings.Contains(string(output), "No module named pytest") {
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
			if len(tc.expectedResults) > 0 {
				for _, expectedResult := range tc.expectedResults {
					if !strings.Contains(logContent, expectedResult) {
						t.Errorf("Expected test result '%s' not found in log files for %s", expectedResult, tc.fixtureName)
					}
				}
			} else {
				// If no specific results expected, at least verify we have some test indicators
				hasTestIndicators := strings.Contains(logContent, "✓") || strings.Contains(logContent, "✕") || strings.Contains(logContent, "○")
				if !hasTestIndicators && len(logFiles) > 0 {
					t.Logf("Note: No test result indicators found for %s - this may be expected for error/empty fixtures", tc.fixtureName)
				}
			}

			// Basic validation: if we have log files, they should contain test indicators for most fixtures
			if len(logFiles) > 0 {
				// Check for passing tests with checkmarks (most fixtures should have some passing tests)
				// Exception: empty-pytest only has failing and skipped tests
				if (strings.Contains(tc.fixtureName, "basic-") || strings.Contains(tc.fixtureName, "long-") || 
				   strings.Contains(tc.fixtureName, "npm-") || strings.Contains(tc.fixtureName, "monorepo-") || 
				   strings.Contains(tc.fixtureName, "jest-esm")) && tc.fixtureName != "empty-pytest" {
					if !strings.Contains(logContent, "✓") {
						t.Errorf("Expected to find passing test indicators (✓) in log files for %s", tc.fixtureName)
					}
				}

				// Specific fixture validations
				switch tc.fixtureName {
				case "basic-jest", "basic-vitest":
					// These should have failing and skipped tests
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
				case "basic-pytest", "empty-pytest":
					// pytest fixtures should have failure indicators 
					if !strings.Contains(logContent, "✕") {
						t.Errorf("Expected to find failing test indicators (✕) in log files for %s", tc.fixtureName)
					}
				}

				// Check that durations are included where available (most fixtures should have some durations)
				// Look for patterns like "(5ms)" or "(10ms)" 
				if !strings.Contains(tc.fixtureName, "config-error") && !strings.Contains(tc.fixtureName, "empty-pytest") {
					durationPattern := "ms)"
					if !strings.Contains(logContent, durationPattern) && !strings.Contains(tc.fixtureName, "pytest") {
						t.Logf("Note: No test durations found in log files for %s - this may be expected", tc.fixtureName)
					}
				}
			}
		})
	}
}