package integration_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// getBinaryPath returns the path to the 3pio binary with appropriate extension for the platform
func getBinaryPath() string {
	binaryName := "3pio"
	if runtime.GOOS == "windows" {
		binaryName = "3pio.exe"
	}

	// First check if THREEPIO_TEST_BINARY env var is set
	if envPath := os.Getenv("THREEPIO_TEST_BINARY"); envPath != "" {
		return envPath
	}

	// Try to find the binary relative to the current working directory
	// This works when tests are run from the project root
	if _, err := os.Stat(filepath.Join("build", binaryName)); err == nil {
		absPath, _ := filepath.Abs(filepath.Join("build", binaryName))
		return absPath
	}

	// Fallback to the path relative to the test file location
	// This works when tests are run from tests/integration_go directory
	testPath := filepath.Join("..", "..", "build", binaryName)
	if _, err := os.Stat(testPath); err == nil {
		absPath, _ := filepath.Abs(testPath)
		return absPath
	}

	// Last resort: return relative path
	return filepath.Join("..", "..", "build", binaryName)
}

// cleanTestDir removes the .3pio directory for a clean test environment
func cleanTestDir(t *testing.T, dir string) {
	t.Helper()

	threepioDir := filepath.Join(dir, ".3pio")
	if err := os.RemoveAll(threepioDir); err != nil && !os.IsNotExist(err) {
		t.Fatalf("Failed to clean test directory: %v", err)
	}
}

// assertFileExists verifies that a file exists
func assertFileExists(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("Expected file to exist: %s", path)
	}
}

// assertNoFile verifies that a file does not exist
func assertNoFile(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("Expected file to not exist: %s", path)
	}
}

// getLatestRunDir returns the path to the most recent run directory
func getLatestRunDir(t *testing.T, baseDir string) string {
	t.Helper()

	runsDir := filepath.Join(baseDir, ".3pio", "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		t.Fatalf("Failed to read runs directory: %v", err)
		return ""
	}

	if len(entries) == 0 {
		t.Fatal("No run directories found")
		return ""
	}

	// Get the last entry (most recent by timestamp)
	latestEntry := entries[len(entries)-1]
	return filepath.Join(runsDir, latestEntry.Name())
}

// assertReportExists verifies that the main test report exists in the run directory
func assertReportExists(t *testing.T, runDir string) {
	t.Helper()

	reportPath := filepath.Join(runDir, "test-run.md")
	assertFileExists(t, reportPath)
}

// assertOutputLogExists verifies that the output log exists in the run directory
func assertOutputLogExists(t *testing.T, runDir string) {
	t.Helper()

	outputPath := filepath.Join(runDir, "output.log")
	assertFileExists(t, outputPath)
}
