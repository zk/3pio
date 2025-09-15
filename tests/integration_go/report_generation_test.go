package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var (
	// Path to the Go binary we're testing
	threePioBinary = getBinaryPath()
	// Fixtures directory
	fixturesDir = "../fixtures"
)

// Helper function to clean .3pio directory
func cleanProjectOutput(projectPath string) error {
	threePioDir := filepath.Join(projectPath, ".3pio")
	return os.RemoveAll(threePioDir)
}

// Helper function to check if file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// Helper function to read file contents
func readFile(t *testing.T, path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", path, err)
	}
	return string(content)
}

func TestReportFileGeneration(t *testing.T) {
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

	// Run 3pio with vitest
	cmd := exec.Command(binaryPath, "npx", "vitest", "run", "math.test.js", "string.test.js")
	cmd.Dir = projectDir
	// Ignore output and exit code - test failures are expected
	// Read output to prevent pipe deadlock
	_, _ = cmd.CombinedOutput()

	// Find the latest run directory
	runDir := getLatestRunDir(t, projectDir)

	// Check for test-run.md
	testRunPath := filepath.Join(runDir, "test-run.md")
	if !fileExists(testRunPath) {
		t.Error("test-run.md file should exist")
	}

	// Check for output.log
	outputLogPath := filepath.Join(runDir, "output.log")
	if !fileExists(outputLogPath) {
		t.Error("output.log file should exist")
	}

	// Check for reports directory
	reportsDir := filepath.Join(runDir, "reports")
	if !fileExists(reportsDir) {
		t.Error("reports directory should exist")
	}

	// Check for hierarchical report files (index.md files in group directories)
	var foundMathReport, foundStringReport bool
	err = filepath.Walk(reportsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), "index.md") {
			// Note: paths are sanitized with underscores replacing dots and slashes
			if strings.Contains(path, "math_test") {
				foundMathReport = true
			}
			if strings.Contains(path, "string_test") {
				foundStringReport = true
			}
		}
		return nil
	})
	if err != nil {
		t.Errorf("Failed to walk reports directory: %v", err)
	}

	if !foundMathReport {
		t.Error("Expected to find report for math.test.js in hierarchical structure")
	}
	if !foundStringReport {
		t.Error("Expected to find report for string.test.js in hierarchical structure")
	}
}

func TestTestRunMdContent(t *testing.T) {
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

	// Run 3pio with vitest
	cmd := exec.Command(binaryPath, "npx", "vitest", "run", "math.test.js", "string.test.js")
	cmd.Dir = projectDir
	// Ignore exit code - test failures are expected
	// Read output to prevent pipe deadlock
	_, _ = cmd.CombinedOutput()

	// Find and read test-run.md
	runDir := getLatestRunDir(t, projectDir)
	testRunPath := filepath.Join(runDir, "test-run.md")
	content := readFile(t, testRunPath)

	// Check main sections
	if !strings.Contains(content, "# 3pio Test Run") {
		t.Error("test-run.md should contain '# 3pio Test Run'")
	}

	// Check that test files are referenced in inline format (not table)
	if !strings.Contains(content, "math.test.js") {
		t.Error("test-run.md should contain math.test.js")
	}

	if !strings.Contains(content, "string.test.js") {
		t.Error("test-run.md should contain string.test.js")
	}

	// Check for status indicators in inline format
	if !strings.Contains(content, "PASS") {
		t.Error("test-run.md should contain PASS status")
	}

	if !strings.Contains(content, "FAIL") {
		t.Error("test-run.md should contain FAIL status")
	}

	// Check that output.log is referenced in header
	if !strings.Contains(content, "./output.log") {
		t.Error("test-run.md should reference output.log")
	}
}

func TestOutputLogContent(t *testing.T) {
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

	// Run 3pio with vitest
	cmd := exec.Command(binaryPath, "npx", "vitest", "run", "math.test.js", "string.test.js")
	cmd.Dir = projectDir
	// Ignore exit code - test failures are expected
	// Read output to prevent pipe deadlock
	_, _ = cmd.CombinedOutput()

	// Find and read output.log
	runDir := getLatestRunDir(t, projectDir)
	outputLogPath := filepath.Join(runDir, "output.log")
	content := readFile(t, outputLogPath)

	// Check that output.log has content (no header anymore - direct output capture)
	// Just verify the file exists and has some test output
	if len(content) == 0 {
		t.Error("output.log should contain test output")
	}

	// Verify it contains some test-related output
	if !strings.Contains(content, "test") && !strings.Contains(content, "Test") && !strings.Contains(content, "PASS") {
		t.Error("output.log should contain test execution output")
	}
}

func TestJestIntegration(t *testing.T) {
	projectDir := filepath.Join(fixturesDir, "basic-jest")

	// Clean output directory
	if err := cleanProjectOutput(projectDir); err != nil {
		t.Fatalf("Failed to clean project output: %v", err)
	}

	// Get absolute path to binary
	binaryPath, err := filepath.Abs(threePioBinary)
	if err != nil {
		t.Fatalf("Failed to get absolute path to binary: %v", err)
	}

	// Run 3pio with jest
	cmd := exec.Command(binaryPath, "npx", "jest", "math.test.js", "string.test.js")
	cmd.Dir = projectDir
	// Ignore exit code - test failures are expected
	// Read output to prevent pipe deadlock
	_, _ = cmd.CombinedOutput()

	// Find the latest run directory
	runDir := getLatestRunDir(t, projectDir)

	// Check for basic file structure
	testRunPath := filepath.Join(runDir, "test-run.md")
	if !fileExists(testRunPath) {
		t.Error("Jest integration should create test-run.md")
	}

	outputLogPath := filepath.Join(runDir, "output.log")
	if !fileExists(outputLogPath) {
		t.Error("Jest integration should create output.log")
	}

	// Check content basics
	content := readFile(t, testRunPath)
	if !strings.Contains(content, "# 3pio Test Run") {
		t.Error("Jest integration should create proper report structure")
	}
}

func TestPytestIntegration(t *testing.T) {
	projectDir := filepath.Join(fixturesDir, "basic-pytest")

	// Check if fixture exists
	if !fileExists(projectDir) {
		t.Skip("basic-pytest fixture not found, skipping pytest integration test")
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

	// Run 3pio with pytest
	cmd := exec.Command(binaryPath, "pytest", "-v")
	cmd.Dir = projectDir
	// Ignore exit code - test failures are expected
	// Read output to prevent pipe deadlock
	_, _ = cmd.CombinedOutput()

	// Find the latest run directory
	runDir := getLatestRunDir(t, projectDir)

	// Check for basic file structure
	testRunPath := filepath.Join(runDir, "test-run.md")
	if !fileExists(testRunPath) {
		t.Error("Pytest integration should create test-run.md")
	}

	outputLogPath := filepath.Join(runDir, "output.log")
	if !fileExists(outputLogPath) {
		t.Error("Pytest integration should create output.log")
	}

	// Check content basics
	content := readFile(t, testRunPath)
	if !strings.Contains(content, "# 3pio Test Run") {
		t.Error("Pytest integration should create proper report structure")
	}
}
