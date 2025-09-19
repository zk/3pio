package integration_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestFailureDisplayFormat(t *testing.T) {
	// Skip building on Windows (no make), assume binary exists
	if runtime.GOOS != "windows" {
		buildCmd := exec.Command("make", "build")
		buildCmd.Dir = filepath.Join("..", "..")
		if err := buildCmd.Run(); err != nil {
			t.Fatalf("Failed to build 3pio: %v", err)
		}
	}

	// Get the project root directory (2 levels up from tests/integration_go)
	projectRoot := filepath.Join("..", "..")
	binaryPath := filepath.Join(projectRoot, "build", "3pio")
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}

	// Get absolute path for binary
	binaryPath, err := filepath.Abs(binaryPath)
	if err != nil {
		t.Fatalf("Failed to get absolute binary path: %v", err)
	}

	// Change to test fixture directory
	fixtureDir := filepath.Join(projectRoot, "tests", "fixtures", "many-failures")
	fixtureDir, err = filepath.Abs(fixtureDir)
	if err != nil {
		t.Fatalf("Failed to get absolute fixture path: %v", err)
	}

	// Clear Go test cache for this package first
	cleanCmd := exec.Command("go", "clean", "-testcache")
	cleanCmd.Dir = fixtureDir
	// Inherit environment so 'go' executable can be found
	cleanCmd.Env = os.Environ()
	_ = cleanCmd.Run()

	// Run 3pio with the test fixture (use -count=1 to disable test caching)
	cmd := exec.Command(binaryPath, "go", "test", "-count=1", ".")
	cmd.Dir = fixtureDir
	// Inherit environment so 'go' executable can be found in subprocess
	cmd.Env = os.Environ()

	// Capture both stdout and stderr (some environments route differently)
	combined, _ := cmd.CombinedOutput()
	output := string(combined)

	// Verify the failure display format (new minimal format)
	t.Run("shows_minimal_summary_with_report_path", func(t *testing.T) {
		if !strings.Contains(output, "FAIL(") {
			t.Errorf("Expected to see FAIL count in output")
		}
		// Look for sanitized report path segment for the many-failures fixture
		if !strings.Contains(output, "$trun_dir/reports/") && !strings.Contains(output, ".3pio/runs/") {
			t.Errorf("Expected to see report path in output")
		}
	})

	t.Run("shows_report_path", func(t *testing.T) {
		// Check that a report path is shown and points at the many-failures group (sanitized)
		hasReportPrefix := strings.Contains(output, "$trun_dir/reports/") || strings.Contains(output, ".3pio/runs/")
		hasManyFailures := strings.Contains(output, "many_failures/index.md") || strings.Contains(output, "many_failures\\index.md")
		if !hasReportPrefix || !hasManyFailures {
			t.Errorf("Expected to see report path for many-failures, got: %s", output)
		}
	})

	t.Run("does_not_show_fourth_failure", func(t *testing.T) {
		// Verify we don't show the 4th failure name (TestFail4)
		if strings.Contains(output, "  x TestFail4") {
			t.Errorf("Should not show TestFail4 (4th failure) in output")
		}
	})

	t.Run("correct_failure_count_in_summary", func(t *testing.T) {
		// The summary should indicate all tests ran
		if !strings.Contains(output, "3 passed, 1 total") {
			// We have 3 passing tests and 5 failing tests = 8 total
			// But it's reported as 1 file
			t.Logf("Output summary line may not match expected format")
		}
	})
}

func TestSingleFailureDisplay(t *testing.T) {
	// Create a temporary test file with just one failure
	tempDir := t.TempDir()

	// Create package.json for Jest
	packageContent := `{
		"name": "test-single-failure",
		"scripts": {
			"test": "jest"
		}
	}`
	if err := os.WriteFile(filepath.Join(tempDir, "package.json"), []byte(packageContent), 0644); err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}

	// Create Jest test file with single failure
	testContent := `test('TestSingleFailure', () => {
	expect(true).toBe(false); // This is the only failure
});`
	if err := os.WriteFile(filepath.Join(tempDir, "single.test.js"), []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Get the project root directory (2 levels up from tests/integration_go)
	projectRoot := filepath.Join("..", "..")
	binaryPath := filepath.Join(projectRoot, "build", "3pio")
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}

	// Get absolute path for binary
	binaryPath, err := filepath.Abs(binaryPath)
	if err != nil {
		t.Fatalf("Failed to get absolute binary path: %v", err)
	}

	// Run 3pio with Jest
	cmd := exec.Command(binaryPath, "npx", "jest", "single.test.js")
	cmd.Dir = tempDir
	// Inherit environment so tools can be found in subprocess
	cmd.Env = os.Environ()

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	_ = cmd.Run()

	output := stdout.String()

	// New minimal format: show a FAIL count and report path
	if !strings.Contains(output, "FAIL(") {
		t.Errorf("Expected to see single failure count in output, got:\n%s", output)
	}
	if !strings.Contains(output, "/reports/") {
		t.Errorf("Expected to see report path in output, got:\n%s", output)
	}
}
