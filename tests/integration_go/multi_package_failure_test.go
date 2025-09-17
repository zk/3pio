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
		// The "Test failures!" summary section should exist alongside inline display
		if !strings.Contains(output, "Test failures!") {
			t.Errorf("Expected 'Test failures!' summary section to exist")
		}
	})

	// Verify that failures are shown inline after FAIL message
	t.Run("inline_failures_displayed", func(t *testing.T) {
		// Check that when pkg_zebra fails, failures are shown inline
		lines := strings.Split(output, "\n")
		foundInlineFailures := false
		for i, line := range lines {
			if strings.Contains(line, "FAIL") && strings.Contains(line, "pkg_zebra") {
				// Check the next few lines for inline failure display
				for j := i + 1; j < len(lines) && j < i+10; j++ {
					if strings.Contains(lines[j], "x TestZebraFail") {
						foundInlineFailures = true
						// Also verify report path is shown inline
						for k := j + 1; k < len(lines) && k < j+5; k++ {
							if strings.Contains(lines[k], "See .3pio") && strings.Contains(lines[k], "pkg_zebra") {
								// Found inline report path pointing to correct package
								break
							} else if strings.Contains(lines[k], "See .3pio") && !strings.Contains(lines[k], "pkg_zebra") {
								t.Errorf("Inline report path should point to pkg_zebra. Got: %s", lines[k])
							}
						}
						break
					}
				}
			}
		}

		if !foundInlineFailures {
			t.Errorf("Expected to see failures displayed inline after FAIL message for pkg_zebra")
		}
	})

	// Verify the failed tests are shown inline (not in summary)
	t.Run("shows_zebra_failures_inline", func(t *testing.T) {
		// These should appear inline after FAIL message, not in a summary
		lines := strings.Split(output, "\n")
		foundZebraSection := false
		for i, line := range lines {
			if strings.Contains(line, "FAIL") && strings.Contains(line, "pkg_zebra") {
				foundZebraSection = true
				// Check next 10 lines for the failures
				failuresFound := 0
				for j := i + 1; j < len(lines) && j < i+15; j++ {
					if strings.Contains(lines[j], "x TestZebraFail1") {
						failuresFound++
					}
					if strings.Contains(lines[j], "x TestZebraFail2") {
						failuresFound++
					}
					if strings.Contains(lines[j], "x TestZebraFail3") {
						failuresFound++
					}
				}
				if failuresFound < 3 {
					t.Errorf("Expected to see all 3 TestZebraFail tests inline after FAIL message, found %d", failuresFound)
				}
				break
			}
		}
		if !foundZebraSection {
			t.Errorf("Did not find FAIL line for pkg_zebra")
		}
	})

	// Verify pkg_alpha passes (shown in results summary)
	t.Run("shows_alpha_passes", func(t *testing.T) {
		// With our change to only show failure lines, passing packages won't have a dedicated line
		// Check the results summary to verify 1 package passed
		if !strings.Contains(output, "1 passed") {
			t.Errorf("Expected results to show 1 package passed (pkg_alpha)")
		}
	})
}
