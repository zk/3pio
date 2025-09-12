package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

var (
	// Path to the Go binary we're testing
	threePioBinary = getBinaryPath()
	// Fixtures directory
	fixturesDir = "../fixtures"
)

func getBinaryPath() string {
	if runtime.GOOS == "windows" {
		return "../../build/3pio.exe"
	}
	return "../../build/3pio"
}

// Helper function to get the latest run directory
func getLatestRunDir(t *testing.T, projectPath string) string {
	runsDir := filepath.Join(projectPath, ".3pio", "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		t.Fatalf("Failed to read runs directory: %v", err)
	}

	var runDirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			runDirs = append(runDirs, entry.Name())
		}
	}

	if len(runDirs) == 0 {
		t.Fatal("No run directories found")
	}

	sort.Strings(runDirs)
	latestRun := runDirs[len(runDirs)-1]
	return filepath.Join(runsDir, latestRun)
}

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

	// Check for individual log files
	mathLogPath := filepath.Join(reportsDir, "math.test.js.md")
	if !fileExists(mathLogPath) {
		t.Error("math.test.js.md file should exist")
	}

	stringLogPath := filepath.Join(reportsDir, "string.test.js.md")
	if !fileExists(stringLogPath) {
		t.Error("string.test.js.md file should exist")
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

	if !strings.Contains(content, "## Summary") {
		t.Error("test-run.md should contain '## Summary'")
	}

	if !strings.Contains(content, "- Total files: 2") {
		t.Error("test-run.md should show '- Total files: 2'")
	}

	if !strings.Contains(content, "- Files completed: 2") {
		t.Error("test-run.md should show '- Files completed: 2'")
	}

	if !strings.Contains(content, "- Files passed: 1") {
		t.Error("test-run.md should show '- Files passed: 1'")
	}

	if !strings.Contains(content, "- Files failed: 1") {
		t.Error("test-run.md should show '- Files failed: 1'")
	}

	// Check math.test.js in test file results table
	if !strings.Contains(content, "| PASS | math.test.js |") {
		t.Error("test-run.md should contain math.test.js with PASS status in table")
	}

	// Check string.test.js in test file results table with FAIL status
	if !strings.Contains(content, "| FAIL | string.test.js |") {
		t.Error("test-run.md should contain string.test.js with FAIL status in table")
	}

	// Individual test details are now in separate report files
	// Main report only shows file-level summary in table format

	// Check that report files are referenced in the table
	if !strings.Contains(content, "./reports/math.test.js.md") {
		t.Error("test-run.md should reference math.test.js.md in table")
	}

	if !strings.Contains(content, "./reports/string.test.js.md") {
		t.Error("test-run.md should reference string.test.js.md in table")
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

	// Check header
	if !strings.Contains(content, "# 3pio Test Output Log") {
		t.Error("output.log should contain header")
	}

	if !strings.Contains(content, "# Timestamp:") {
		t.Error("output.log should contain timestamp")
	}

	if !strings.Contains(content, "# Command: npx vitest run math.test.js string.test.js") {
		t.Error("output.log should contain executed command")
	}

	if !strings.Contains(content, "# This file contains all stdout/stderr output from the test run.") {
		t.Error("output.log should contain description")
	}

	if !strings.Contains(content, "# ---") {
		t.Error("output.log should contain separator")
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
