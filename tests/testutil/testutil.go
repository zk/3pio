package testutil

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestResult contains the results of running 3pio
type TestResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
	RunID    string
}

// RunThreepio executes 3pio with the given arguments in the specified directory
func RunThreepio(t *testing.T, dir string, args ...string) TestResult {
	t.Helper()

	// Build path to 3pio binary
	// When tests run, working directory varies (package dir locally, project root in CI)
	cwd, _ := os.Getwd()

	// Determine binary name based on OS
	binaryName := "3pio"
	if runtime.GOOS == "windows" {
		binaryName = "3pio.exe"
	}

	// Try relative path from package directory (local development)
	threepioPath := filepath.Join(cwd, "..", "..", "build", binaryName)
	if _, err := os.Stat(threepioPath); os.IsNotExist(err) {
		// Try from project root (CI environment)
		threepioPath = filepath.Join(cwd, "build", binaryName)
		if _, err := os.Stat(threepioPath); os.IsNotExist(err) {
			// Try finding 3pio in PATH
			if pathBinary, err := exec.LookPath(binaryName); err == nil {
				threepioPath = pathBinary
			} else {
				t.Fatalf("3pio binary not found. Tried: %s, %s, and PATH. Run 'make build' first",
					filepath.Join(cwd, "..", "..", "build", binaryName),
					filepath.Join(cwd, "build", binaryName))
			}
		}
	}

	// Create command
	cmd := exec.Command(threepioPath, args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run command
	err := cmd.Run()
	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("Failed to run 3pio: %v", err)
	}

	// Extract run ID from output
	runID := extractRunID(stdout.String())
	if runID == "" {
		// Try to find the most recent run directory
		runsDir := filepath.Join(dir, ".3pio", "runs")
		if entries, err := os.ReadDir(runsDir); err == nil && len(entries) > 0 {
			// Get the last entry (most recent)
			runID = entries[len(entries)-1].Name()
		}
	}

	return TestResult{
		ExitCode: exitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		RunID:    runID,
	}
}

// LookPath searches for an executable in PATH
func LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

// CommandAvailable checks if a command can be executed successfully
func CommandAvailable(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

// ExtractTestCount extracts the total test count from a report
func ExtractTestCount(report string) int {
	// Look for patterns like "Total test cases: 15"
	re := regexp.MustCompile(`Total test cases:\s*(\d+)`)
	matches := re.FindStringSubmatch(report)
	if len(matches) > 1 {
		count, _ := strconv.Atoi(matches[1])
		return count
	}
	return 0
}

// extractRunID extracts the run ID from 3pio output
func extractRunID(output string) string {
	// Look for pattern like "trun_dir: .3pio/runs/20240101T120000-happy-name"
	re := regexp.MustCompile(`trun_dir:\s+\.3pio/runs/([^\s]+)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) > 1 {
		return matches[1]
	}

	// Fallback to old base_dir format for compatibility
	re = regexp.MustCompile(`base_dir:\s+\.3pio/runs/([^\s]+)`)
	matches = re.FindStringSubmatch(output)
	if len(matches) > 1 {
		return matches[1]
	}

	// Fallback to old trailing path format
	re = regexp.MustCompile(`\.3pio/runs/([^/]+)/test-run\.md`)
	matches = re.FindStringSubmatch(output)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// CleanupTestRuns removes .3pio directory from a test fixture
func CleanupTestRuns(t *testing.T, dir string) {
	t.Helper()
	threepioDir := filepath.Join(dir, ".3pio")
	if err := os.RemoveAll(threepioDir); err != nil && !os.IsNotExist(err) {
		t.Logf("Warning: failed to cleanup .3pio directory: %v", err)
	}
}

// WaitForFile waits for a file to exist with a timeout
func WaitForFile(path string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for file: %s", path)
}

// AssertFileContains checks if a file contains expected content
func AssertFileContains(t *testing.T, path string, expected ...string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", path, err)
	}

	text := string(content)
	for _, exp := range expected {
		if !strings.Contains(text, exp) {
			t.Errorf("File %s does not contain expected string: %s", path, exp)
		}
	}
}

// AssertFileNotContains checks if a file does not contain certain content
func AssertFileNotContains(t *testing.T, path string, unexpected ...string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", path, err)
	}

	text := string(content)
	for _, unexp := range unexpected {
		if strings.Contains(text, unexp) {
			t.Errorf("File %s contains unexpected string: %s", path, unexp)
		}
	}
}
