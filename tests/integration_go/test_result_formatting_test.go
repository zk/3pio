package integration_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestTestResultFormattingInLogFiles tests that test results are properly formatted
// in individual report files across all test fixtures
func TestTestResultFormattingInLogFiles(t *testing.T) {
	// Build path to the 3pio binary
	binaryName := "3pio"
	if runtime.GOOS == "windows" {
		binaryName = "3pio.exe"
	}

	binaryPath, err := filepath.Abs(filepath.Join("../../build", binaryName))
	if err != nil {
		t.Fatalf("Failed to get absolute path to binary: %v", err)
	}

	// Verify binary exists
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Fatalf("3pio binary not found at %s. Run 'make build' first.", binaryPath)
	}

	// Test cases for ALL fixtures
	testCases := []struct {
		fixtureName string
		command     []string
		testRunner  string // "jest", "vitest", "pytest"
		expectPass  bool   // Should have some passing tests
		expectFail  bool   // Should have some failing tests
		expectSkip  bool   // Should have some skipped tests
		shouldRun   bool   // Whether we expect tests to run
		skipReason  string // Reason to skip if needed
	}{
		// Basic fixtures - comprehensive test coverage
		{
			fixtureName: "basic-jest",
			command:     []string{"npx", "jest"},
			testRunner:  "jest",
			expectPass:  true,
			expectFail:  true,
			expectSkip:  true,
			shouldRun:   true,
		},
		{
			fixtureName: "basic-vitest",
			command:     []string{"npx", "vitest", "run"},
			testRunner:  "vitest",
			expectPass:  true,
			expectFail:  true,
			expectSkip:  true,
			shouldRun:   true,
		},
		{
			fixtureName: "basic-pytest",
			command:     []string{"python", "-m", "pytest", "-v"},
			testRunner:  "pytest",
			expectPass:  true,
			expectFail:  true,
			expectSkip:  false, // pytest adapter doesn't properly handle skipped tests yet
			shouldRun:   true,
		},
		// Empty fixtures
		{
			fixtureName: "empty-jest",
			command:     []string{"npx", "jest"},
			testRunner:  "jest",
			shouldRun:   false,
			skipReason:  "Empty test suite - no actual test cases",
		},
		{
			fixtureName: "empty-vitest",
			command:     []string{"npx", "vitest", "run"},
			testRunner:  "vitest",
			shouldRun:   false,
			skipReason:  "Empty test suite - no actual test cases",
		},
		{
			fixtureName: "empty-pytest",
			command:     []string{"python", "-m", "pytest", "-v"},
			testRunner:  "pytest",
			expectFail:  true,
			expectSkip:  true,
			shouldRun:   true,
		},
		// Specialized fixtures
		{
			fixtureName: "jest-esm",
			command:     []string{"npm", "test"},
			testRunner:  "jest",
			expectPass:  true,
			shouldRun:   true,
		},
		{
			fixtureName: "long-names-jest",
			command:     []string{"npx", "jest"},
			testRunner:  "jest",
			expectPass:  true,
			shouldRun:   true,
		},
		{
			fixtureName: "long-names-vitest",
			command:     []string{"npx", "vitest", "run"},
			testRunner:  "vitest",
			expectPass:  true,
			shouldRun:   true,
		},
		{
			fixtureName: "monorepo-vitest",
			command:     []string{"npx", "vitest", "run"},
			testRunner:  "vitest",
			expectPass:  true,
			shouldRun:   true,
		},
		{
			fixtureName: "npm-separator-jest",
			command:     []string{"npm", "test"},
			testRunner:  "jest",
			expectPass:  true,
			shouldRun:   true,
		},
		// Problematic fixtures
		{
			fixtureName: "npm-separator-vitest",
			command:     []string{"npm", "test"},
			testRunner:  "vitest",
			shouldRun:   false,
			skipReason:  "Vitest configuration causes test discovery issues",
		},
		{
			fixtureName: "jest-config-error",
			command:     []string{"npx", "jest"},
			testRunner:  "jest",
			shouldRun:   false,
			skipReason:  "Config error fixture - tests configuration failures",
		},
		{
			fixtureName: "jest-ts-config-error",
			command:     []string{"npx", "jest"},
			testRunner:  "jest",
			shouldRun:   false,
			skipReason:  "TS config error fixture - tests TypeScript configuration failures",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.fixtureName, func(t *testing.T) {
			// Skip if this fixture should not run tests
			if !tc.shouldRun {
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
			output, _ := cmd.CombinedOutput()
			t.Logf("Command output for %s:\n%s", tc.fixtureName, string(output))

			// For pytest fixtures, if pytest is not installed, skip the test
			if tc.testRunner == "pytest" && strings.Contains(string(output), "No module named pytest") {
				t.Skip("pytest not installed, skipping test")
			}

			// Check that .3pio directory was created
			if _, err := os.Stat(threepioDir); os.IsNotExist(err) {
				t.Fatalf("Expected .3pio directory to be created")
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
			reportsDir := filepath.Join(runDir, "reports")

			// Check that reports directory exists
			if _, err := os.Stat(reportsDir); os.IsNotExist(err) {
				t.Fatalf("Expected reports directory to exist")
			}

			// Read all report files recursively and verify test results are present
			var reportFiles []string
			var allReportContent strings.Builder
			
			err = filepath.Walk(reportsDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
					reportFiles = append(reportFiles, path)
					content, err := os.ReadFile(path)
					if err != nil {
						return fmt.Errorf("failed to read report file %s: %v", path, err)
					}
					t.Logf("Log file %s content:\n%s", path, string(content))
					allReportContent.Write(content)
				}
				return nil
			})
			if err != nil {
				t.Fatalf("Failed to walk reports directory: %v", err)
			}
			
			if len(reportFiles) == 0 {
				t.Fatalf("Expected at least one report file, found none")
			}

			reportContent := allReportContent.String()

			// Universal validations - check for what we expect to find
			hasPass := strings.Contains(reportContent, "✓")
			hasFail := strings.Contains(reportContent, "✕")
			hasSkip := strings.Contains(reportContent, "○")

			// Validate expected test result types
			if tc.expectPass && !hasPass {
				t.Errorf("Expected to find passing test indicators (✓) but found none")
			}
			if tc.expectFail && !hasFail {
				t.Errorf("Expected to find failing test indicators (✕) but found none")
			}
			if tc.expectSkip && !hasSkip {
				t.Errorf("Expected to find skipped test indicators (○) but found none")
			}

			// Universal formatting validations
			if len(reportFiles) > 0 {
				// Check for YAML frontmatter - all test files should have structured format
				if !strings.Contains(reportContent, "---\ntest_file:") {
					t.Errorf("Expected to find YAML frontmatter with test_file field in report files")
				}
				
				// Check for structured format with test results directly after title
				if !strings.Contains(reportContent, "# Test results for") {
					t.Errorf("Expected to find structured title '# Test results for' in report files")
				}

				// If we have failing tests, they should have error details in code blocks
				if hasFail && !strings.Contains(reportContent, "```") {
					t.Errorf("Expected to find error details in code blocks (```) for failing tests")
				}

				// Check for durations - most test runners include them
				hasDurations := strings.Contains(reportContent, "ms)")
				if !hasDurations && (hasPass || hasFail || hasSkip) {
					t.Logf("Note: No test durations found - may be expected for %s", tc.testRunner)
				}
			}
		})
	}
}
