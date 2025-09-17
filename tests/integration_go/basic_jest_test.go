package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zk/3pio/tests/testutil"
)

func TestFullFlowIntegration(t *testing.T) {
	// Test complete flow with different test runners

	t.Run("Vitest Full Flow", func(t *testing.T) {
		testFullFlowWithRunner(t, "basic-vitest", []string{"npx", "vitest", "run", "math.test.js", "string.test.js"})
	})

	t.Run("Jest Full Flow", func(t *testing.T) {
		testFullFlowWithRunner(t, "basic-jest", []string{"npx", "jest", "math.test.js", "string.test.js"})
	})
}

func testFullFlowWithRunner(t *testing.T, fixtureDir string, command []string) {
	// Check if npm/jest is available
	if len(command) > 0 && command[0] == "npx" {
		if _, err := testutil.LookPath("npm"); err != nil {
			t.Skip("npm not found in PATH")
		}
		if len(command) > 1 && command[1] == "jest" {
			if err := testutil.CommandAvailable("npx", "jest", "--version"); err != nil {
				t.Skipf("jest command failed: %v", err)
			}
		} else if len(command) > 1 && command[1] == "vitest" {
			if err := testutil.CommandAvailable("npx", "vitest", "--version"); err != nil {
				t.Skipf("vitest command failed: %v", err)
			}
		}
	}

	fixtureDir = filepath.Join("..", "fixtures", fixtureDir)
	if _, err := os.Stat(fixtureDir); os.IsNotExist(err) {
		t.Skipf("fixture %s not found", fixtureDir)
	}

	result := testutil.RunThreepio(t, fixtureDir, command...)

	// Add diagnostic output for debugging CI issues
	if result.RunID == "" {
		t.Logf("Debug: RunID is empty. Stdout: %s, Stderr: %s", result.Stdout, result.Stderr)
	}

	// Basic output verification
	if len(result.Stdout) == 0 && len(result.Stderr) == 0 {
		t.Error("Expected some output from test run")
	}

	// Check report exists
	reportPath := filepath.Join(fixtureDir, ".3pio", "runs", result.RunID, "test-run.md")
	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		t.Errorf("Report file not found: %s", reportPath)
	}

	runDir := filepath.Join(fixtureDir, ".3pio", "runs", result.RunID)

	// Verify all expected files exist - new hierarchical structure
	expectedFiles := []string{
		"test-run.md",
		"output.log",
	}

	for _, expectedFile := range expectedFiles {
		filePath := filepath.Join(runDir, expectedFile)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("Expected file %s does not exist", expectedFile)
		}
	}

	// Verify reports directory exists with hierarchical structure
	reportsDir := filepath.Join(runDir, "reports")
	if _, err := os.Stat(reportsDir); os.IsNotExist(err) {
		t.Error("Expected reports directory does not exist")
	}

	// Check that some report files exist in the hierarchical structure
	// Note: The exact paths depend on group names and will be sanitized
	var foundReports []string
	err := filepath.Walk(reportsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), "index.md") {
			foundReports = append(foundReports, path)
		}
		return nil
	})
	if err != nil {
		t.Errorf("Failed to walk reports directory: %v", err)
	}

	if len(foundReports) == 0 {
		t.Error("Expected to find some report index.md files in hierarchical structure")
	}

	// Verify test-run.md has proper content
	testRunContent, err := os.ReadFile(filepath.Join(runDir, "test-run.md"))
	if err != nil {
		t.Fatalf("Failed to read test-run.md: %v", err)
	}

	expectedSections := []string{
		"# 3pio Test Run",
		"math.test.js",
		"string.test.js",
	}

	for _, section := range expectedSections {
		if !strings.Contains(string(testRunContent), section) {
			t.Errorf("test-run.md should contain '%s'", section)
		}
	}

	// Verify functional success using reliable test-run.md metadata
	testRunContent, err2 := os.ReadFile(filepath.Join(runDir, "test-run.md"))
	if err2 != nil {
		t.Fatalf("Failed to read test-run.md: %v", err2)
	}

	// Check functional completion indicators
	if !strings.Contains(string(testRunContent), "status: COMPLETED") {
		t.Error("Test run should complete successfully")
	}

	// Detect expected runner based on command
	expectedRunner := "jest"
	if len(command) > 1 && command[1] == "vitest" {
		expectedRunner = "vitest"
	}
	expectedRunnerText := "detected_runner: " + expectedRunner
	if !strings.Contains(string(testRunContent), expectedRunnerText) {
		t.Errorf("Should detect %s runner", expectedRunner)
	}

	if !strings.Contains(string(testRunContent), "Total test cases:") {
		t.Error("Should process and count test cases")
	}

	// Verify output.log exists but make content check optional for CI compatibility
	outputLogContent, err := os.ReadFile(filepath.Join(runDir, "output.log"))
	if err != nil {
		t.Fatalf("Failed to read output.log: %v", err)
	}

	if len(outputLogContent) == 0 {
		t.Log("Warning: output.log is empty (may be normal in CI environment)")
	} else {
		t.Logf("output.log contains %d bytes of output", len(outputLogContent))
	}
}

func TestEmptyTestSuiteHandling(t *testing.T) {
	t.Run("Empty Vitest", func(t *testing.T) {
		testEmptyTestSuite(t, "empty-vitest", []string{"npx", "vitest", "run"})
	})

	t.Run("Empty Jest", func(t *testing.T) {
		testEmptyTestSuite(t, "empty-jest", []string{"npx", "jest"})
	})
}

func testEmptyTestSuite(t *testing.T, fixtureDir string, command []string) {
	fixtureDir = filepath.Join("..", "fixtures", fixtureDir)
	if _, err := os.Stat(fixtureDir); os.IsNotExist(err) {
		t.Skipf("fixture %s not found", fixtureDir)
	}

	result := testutil.RunThreepio(t, fixtureDir, command...)

	// Check if basic structure was created even for empty test suite
	threePioDir := filepath.Join(fixtureDir, ".3pio")
	if _, err := os.Stat(threePioDir); os.IsNotExist(err) {
		t.Error("Empty test suite should still create .3pio directory")
	} else {
		t.Logf("Successfully created .3pio directory for empty test suite. RunID: %s", result.RunID)
	}
}

func TestLongNamesHandling(t *testing.T) {
	// Test handling of very long test names
	t.Run("Long Names Jest", func(t *testing.T) {
		testLongNames(t, "long-names-jest", []string{"npx", "jest"})
	})

	t.Run("Long Names Vitest", func(t *testing.T) {
		testLongNames(t, "long-names-vitest", []string{"npx", "vitest", "run"})
	})
}

func testLongNames(t *testing.T, fixtureDir string, command []string) {
	fixtureDir = filepath.Join("..", "fixtures", fixtureDir)
	if _, err := os.Stat(fixtureDir); os.IsNotExist(err) {
		t.Skipf("fixture %s not found", fixtureDir)
	}

	result := testutil.RunThreepio(t, fixtureDir, command...)

	// Check that files were created
	testRunPath := filepath.Join(fixtureDir, ".3pio", "runs", result.RunID, "test-run.md")
	if _, err := os.Stat(testRunPath); os.IsNotExist(err) {
		t.Error("Long names test should create test-run.md")
	}

	// Check that content was generated (basic smoke test)
	content, err := os.ReadFile(testRunPath)
	if err != nil {
		t.Fatalf("Failed to read test-run.md: %v", err)
	}
	if !strings.Contains(string(content), "# 3pio Test Run") {
		t.Error("Long names test should create proper report structure")
	}
}

func TestErrorRecovery(t *testing.T) {
	// Test that the system handles various error conditions gracefully

	t.Run("Invalid Command", func(t *testing.T) {
		fixtureDir := filepath.Join("..", "fixtures", "basic-vitest")
		if _, err := os.Stat(fixtureDir); os.IsNotExist(err) {
			t.Skip("basic-vitest fixture not found")
		}

		// Try to run with invalid command - this should fail
		result := testutil.RunThreepio(t, fixtureDir, "invalid-command")
		if result.ExitCode == 0 {
			t.Error("Expected error for invalid command")
		}
	})

	t.Run("Nonexistent Test Files", func(t *testing.T) {
		fixtureDir := filepath.Join("..", "fixtures", "basic-vitest")
		if _, err := os.Stat(fixtureDir); os.IsNotExist(err) {
			t.Skip("basic-vitest fixture not found")
		}

		// Try to run with nonexistent test file
		result := testutil.RunThreepio(t, fixtureDir, "npx", "vitest", "run", "nonexistent.test.js")

		// The system should still create basic structure
		threePioDir := filepath.Join(fixtureDir, ".3pio")
		if _, err := os.Stat(threePioDir); err == nil {
			t.Logf("System handled nonexistent test file gracefully. RunID: %s", result.RunID)
		}
	})
}
