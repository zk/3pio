package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
	mathLogPath := filepath.Join(reportsDir, "math.test.js.log")
	if !fileExists(mathLogPath) {
		t.Error("math.test.js.log file should exist")
	}

	stringLogPath := filepath.Join(reportsDir, "string.test.js.log")
	if !fileExists(stringLogPath) {
		t.Error("string.test.js.log file should exist")
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

	if !strings.Contains(content, "- Total Files: 2") {
		t.Error("test-run.md should show 'Total Files: 2'")
	}

	if !strings.Contains(content, "- Files Completed: 2") {
		t.Error("test-run.md should show 'Files Completed: 2'")
	}

	if !strings.Contains(content, "- Files Passed: 1") {
		t.Error("test-run.md should show 'Files Passed: 1'")
	}

	if !strings.Contains(content, "- Files Failed: 1") {
		t.Error("test-run.md should show 'Files Failed: 1'")
	}

	// Check math.test.js section
	if !strings.Contains(content, "## math.test.js") {
		t.Error("test-run.md should contain math.test.js section")
	}

	if !strings.Contains(content, "Status: **PASS**") {
		t.Error("math.test.js should have PASS status")
	}

	if !strings.Contains(content, "### Math operations") {
		t.Error("test-run.md should contain Math operations suite")
	}

	if !strings.Contains(content, "✓ should add numbers correctly") {
		t.Error("test-run.md should show passing add test")
	}

	if !strings.Contains(content, "✓ should multiply numbers correctly") {
		t.Error("test-run.md should show passing multiply test")
	}

	if !strings.Contains(content, "✓ should handle division") {
		t.Error("test-run.md should show passing division test")
	}

	// Check string.test.js section
	if !strings.Contains(content, "## string.test.js") {
		t.Error("test-run.md should contain string.test.js section")
	}

	if !strings.Contains(content, "Status: **FAIL**") {
		t.Error("string.test.js should have FAIL status")
	}

	if !strings.Contains(content, "### String operations") {
		t.Error("test-run.md should contain String operations suite")
	}

	if !strings.Contains(content, "✓ should concatenate strings") {
		t.Error("test-run.md should show passing concatenate test")
	}

	if !strings.Contains(content, "✕ should fail this test") {
		t.Error("test-run.md should show failing test")
	}

	if !strings.Contains(content, "○ should skip this test") {
		t.Error("test-run.md should show skipped test")
	}

	if !strings.Contains(content, "✓ should convert to uppercase") {
		t.Error("test-run.md should show passing uppercase test")
	}

	// Check for error message
	if !regexp.MustCompile(`expected 'foo' to be 'bar'`).MatchString(content) {
		t.Error("test-run.md should contain expected error message")
	}

	// Check for duration format (ms or s)
	durationRegex := regexp.MustCompile(`\(\d+(\.\d+)?\s*(ms|s)\)`)
	if !durationRegex.MatchString(content) {
		t.Error("test-run.md should contain duration information")
	}

	// Check for log file links
	if !strings.Contains(content, "[Log](./reports/math.test.js.log)") {
		t.Error("test-run.md should link to math.test.js.log")
	}

	if !strings.Contains(content, "[Log](./reports/string.test.js.log)") {
		t.Error("test-run.md should link to string.test.js.log")
	}

	if !strings.Contains(content, "[output.log](./output.log)") {
		t.Error("test-run.md should link to output.log")
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
