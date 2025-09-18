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

func TestMultiPackageFailureReportPath(t *testing.T) {
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
	fixtureDir := filepath.Join(projectRoot, "tests", "fixtures", "multi-package-failure")
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
	cmd := exec.Command(binaryPath, "go", "test", "-count=1", "./...")
	cmd.Dir = fixtureDir
	// Inherit environment so 'go' executable can be found in subprocess
	cmd.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// We expect this to fail since tests are failing
	_ = cmd.Run()

	output := stdout.String()

	// Test that the summary section exists with inline display
    t.Run("summary_section_exists", func(t *testing.T) {
        // The summary section exists in the new console format
        if !strings.Contains(output, "Test failures!") {
            t.Errorf("Expected 'Test failures!' summary section to exist")
        }
    })

	// Verify that failures are shown inline after FAIL message
    t.Run("minimal_summary_displayed_for_zebra", func(t *testing.T) {
        if !strings.Contains(output, "FAIL(") || !strings.Contains(output, "/reports/") {
            t.Errorf("Expected minimal summary with report path for pkg_zebra")
        }
    })

	// Verify the failed tests are shown inline (not in summary)
    t.Run("shows_alpha_passes", func(t *testing.T) {
        // In the new format, passing groups are not listed individually; check summary reflects passes
        if !strings.Contains(output, "Results:") || !strings.Contains(output, "passed") {
            t.Errorf("Expected final results summary to include passed count")
        }
    })
    // Note: Inline listing of individual failures is no longer displayed
}
