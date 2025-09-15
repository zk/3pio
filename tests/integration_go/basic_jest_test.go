package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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
	projectDir := filepath.Join(fixturesDir, fixtureDir)

	// Clean output directory
	if err := cleanProjectOutput(projectDir); err != nil {
		t.Fatalf("Failed to clean project output: %v", err)
	}

	// Get absolute path to binary
	binaryPath, err := filepath.Abs(threePioBinary)
	if err != nil {
		t.Fatalf("Failed to get absolute path to binary: %v", err)
	}

	// Prepare full command
	fullCmd := append([]string{binaryPath}, command...)
	cmd := exec.Command(fullCmd[0], fullCmd[1:]...)
	cmd.Dir = projectDir

	// Run the command (may have non-zero exit due to test failures)
	output, _ := cmd.CombinedOutput()

	// Basic output verification
	outputStr := string(output)
	if len(outputStr) == 0 {
		t.Error("Expected some output from test run")
	}

	// Find the latest run directory
	runDir := getLatestRunDir(t, projectDir)

	// Verify all expected files exist - new hierarchical structure
	expectedFiles := []string{
		"test-run.md",
		"output.log",
	}

	for _, expectedFile := range expectedFiles {
		filePath := filepath.Join(runDir, expectedFile)
		if !fileExists(filePath) {
			t.Errorf("Expected file %s does not exist", expectedFile)
		}
	}

	// Verify reports directory exists with hierarchical structure
	reportsDir := filepath.Join(runDir, "reports")
	if !fileExists(reportsDir) {
		t.Error("Expected reports directory does not exist")
	}

	// Check that some report files exist in the hierarchical structure
	// Note: The exact paths depend on group names and will be sanitized
	var foundReports []string
	err = filepath.Walk(reportsDir, func(path string, info os.FileInfo, err error) error {
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
	testRunContent := readFile(t, filepath.Join(runDir, "test-run.md"))

	expectedSections := []string{
		"# 3pio Test Run",
		"math.test.js",
		"string.test.js",
	}

	for _, section := range expectedSections {
		if !strings.Contains(testRunContent, section) {
			t.Errorf("test-run.md should contain '%s'", section)
		}
	}

	// Verify output.log has content (no header anymore - direct output capture)
	outputLogContent := readFile(t, filepath.Join(runDir, "output.log"))

	// Just verify the file exists and has content
	if len(outputLogContent) == 0 {
		t.Error("output.log should contain test output")
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
	projectDir := filepath.Join(fixturesDir, fixtureDir)

	// Check if fixture exists
	if !fileExists(projectDir) {
		t.Skipf("Fixture %s not found, skipping test", fixtureDir)
	}

	// Clean output directory
	if err := cleanProjectOutput(projectDir); err != nil {
		t.Fatalf("Failed to clean project output: %v", err)
	}

	// Get absolute path to binary
	binaryPath, err := filepath.Abs(threePioBinary)
	if err != nil {
		t.Fatalf("Failed to get absolute path to binary: %v", err)
	}

	// Prepare full command
	fullCmd := append([]string{binaryPath}, command...)
	cmd := exec.Command(fullCmd[0], fullCmd[1:]...)
	cmd.Dir = projectDir

	// Run the command (read output to prevent pipe deadlock)
	_, _ = cmd.CombinedOutput()

	// Check if basic structure was created even for empty test suite
	threePioDir := filepath.Join(projectDir, ".3pio")
	if !fileExists(threePioDir) {
		t.Error("Empty test suite should still create .3pio directory")
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
	projectDir := filepath.Join(fixturesDir, fixtureDir)

	// Check if fixture exists
	if !fileExists(projectDir) {
		t.Skipf("Fixture %s not found, skipping test", fixtureDir)
	}

	// Clean output directory
	if err := cleanProjectOutput(projectDir); err != nil {
		t.Fatalf("Failed to clean project output: %v", err)
	}

	// Get absolute path to binary
	binaryPath, err := filepath.Abs(threePioBinary)
	if err != nil {
		t.Fatalf("Failed to get absolute path to binary: %v", err)
	}

	// Prepare full command
	fullCmd := append([]string{binaryPath}, command...)
	cmd := exec.Command(fullCmd[0], fullCmd[1:]...)
	cmd.Dir = projectDir

	// Run the command (read output to prevent pipe deadlock)
	_, _ = cmd.CombinedOutput()

	// Find the latest run directory
	runDir := getLatestRunDir(t, projectDir)

	// Check that files were created
	testRunPath := filepath.Join(runDir, "test-run.md")
	if !fileExists(testRunPath) {
		t.Error("Long names test should create test-run.md")
	}

	// Check that content was generated (basic smoke test)
	content := readFile(t, testRunPath)
	if !strings.Contains(content, "# 3pio Test Run") {
		t.Error("Long names test should create proper report structure")
	}
}

func TestErrorRecovery(t *testing.T) {
	// Test that the system handles various error conditions gracefully

	t.Run("Invalid Command", func(t *testing.T) {
		projectDir := filepath.Join(fixturesDir, "basic-vitest")

		// Clean output directory
		if err := cleanProjectOutput(projectDir); err != nil {
			t.Fatalf("Failed to clean project output: %v", err)
		}

		// Get absolute path to binary
		binaryPath, err := filepath.Abs(threePioBinary)
		if err != nil {
			t.Fatalf("Failed to get absolute path to binary: %v", err)
		}

		// Try to run with invalid command
		cmd := exec.Command(binaryPath, "invalid-command")
		cmd.Dir = projectDir

		_, err = cmd.CombinedOutput()
		if err == nil {
			t.Error("Expected error for invalid command")
		}
	})

	t.Run("Nonexistent Test Files", func(t *testing.T) {
		projectDir := filepath.Join(fixturesDir, "basic-vitest")

		// Clean output directory
		if err := cleanProjectOutput(projectDir); err != nil {
			t.Fatalf("Failed to clean project output: %v", err)
		}

		// Get absolute path to binary
		binaryPath, err := filepath.Abs(threePioBinary)
		if err != nil {
			t.Fatalf("Failed to get absolute path to binary: %v", err)
		}

		// Try to run with nonexistent test file
		cmd := exec.Command(binaryPath, "npx", "vitest", "run", "nonexistent.test.js")
		cmd.Dir = projectDir

		// This should not crash, even though the test file doesn't exist
		// Read output to prevent pipe deadlock
		_, _ = cmd.CombinedOutput()

		// The system should still create basic structure
		threePioDir := filepath.Join(projectDir, ".3pio")
		if fileExists(threePioDir) {
			t.Log("System handled nonexistent test file gracefully")
		}
	})
}
