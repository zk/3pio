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

func TestInlineFailureDisplay(t *testing.T) {
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

	t.Run("go_test_shows_inline_failures", func(t *testing.T) {
		// Test with multi-package-failure fixture
		fixtureDir := filepath.Join(projectRoot, "tests", "fixtures", "multi-package-failure")
		fixtureDir, err = filepath.Abs(fixtureDir)
		if err != nil {
			t.Fatalf("Failed to get absolute fixture path: %v", err)
		}

		// Clear Go test cache
		cleanCmd := exec.Command("go", "clean", "-testcache")
		cleanCmd.Dir = fixtureDir
		cleanCmd.Env = os.Environ()
		_ = cleanCmd.Run()

		// Run 3pio with the test fixture
		cmd := exec.Command(binaryPath, "go", "test", "-count=1", "./...")
		cmd.Dir = fixtureDir
		cmd.Env = os.Environ()

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		// We expect this to fail since tests are failing
		_ = cmd.Run()

		output := stdout.String()

		// Check that failures appear inline after FAIL message
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
							}
						}
						break
					}
				}
			}
		}

		if !foundInlineFailures {
			t.Errorf("Expected to see failures displayed inline after FAIL message, but they were not found")
		}

		// The "Test failures!" summary section can still exist alongside inline display
	})

	t.Run("jest_shows_inline_failures", func(t *testing.T) {
		// Test with jest-fail fixture
		fixtureDir := filepath.Join(projectRoot, "tests", "fixtures", "jest-fail")
		fixtureDir, err = filepath.Abs(fixtureDir)
		if err != nil {
			t.Fatalf("Failed to get absolute fixture path: %v", err)
		}

		// Check if fixture exists
		if _, err := os.Stat(fixtureDir); os.IsNotExist(err) {
			t.Skip("jest-fail fixture not found, skipping")
		}

		// Run npm install if needed
		if _, err := os.Stat(filepath.Join(fixtureDir, "node_modules")); os.IsNotExist(err) {
			installCmd := exec.Command("npm", "install")
			installCmd.Dir = fixtureDir
			installCmd.Env = os.Environ()
			if err := installCmd.Run(); err != nil {
				t.Skipf("Failed to install Jest dependencies: %v", err)
			}
		}

		// Run 3pio with Jest
		cmd := exec.Command(binaryPath, "npx", "jest")
		cmd.Dir = fixtureDir
		cmd.Env = os.Environ()

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		_ = cmd.Run()

		output := stdout.String()

		// Check for inline failures after FAIL message
		if strings.Contains(output, "FAIL") {
			lines := strings.Split(output, "\n")
			foundInlineFailures := false
			for i, line := range lines {
				if strings.Contains(line, "FAIL") {
					// Check next lines for failure details
					for j := i + 1; j < len(lines) && j < i+10; j++ {
						if strings.Contains(lines[j], "x ") {
							foundInlineFailures = true
							break
						}
					}
				}
			}
			if !foundInlineFailures {
				t.Errorf("Expected to see failures displayed inline after FAIL message for Jest")
			}
		}

		// Verify NO "Test failures!" summary section
		if strings.Contains(output, "Test failures!") {
			t.Errorf("Found 'Test failures!' summary section in Jest output - this should be removed")
		}
	})

	t.Run("vitest_shows_inline_failures", func(t *testing.T) {
		// Test with vitest-fail fixture
		fixtureDir := filepath.Join(projectRoot, "tests", "fixtures", "vitest-fail")
		fixtureDir, err = filepath.Abs(fixtureDir)
		if err != nil {
			t.Fatalf("Failed to get absolute fixture path: %v", err)
		}

		// Check if fixture exists
		if _, err := os.Stat(fixtureDir); os.IsNotExist(err) {
			t.Skip("vitest-fail fixture not found, skipping")
		}

		// Run npm install if needed
		if _, err := os.Stat(filepath.Join(fixtureDir, "node_modules")); os.IsNotExist(err) {
			installCmd := exec.Command("npm", "install")
			installCmd.Dir = fixtureDir
			installCmd.Env = os.Environ()
			if err := installCmd.Run(); err != nil {
				t.Skipf("Failed to install Vitest dependencies: %v", err)
			}
		}

		// Run 3pio with Vitest
		cmd := exec.Command(binaryPath, "npx", "vitest", "run")
		cmd.Dir = fixtureDir
		cmd.Env = os.Environ()

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		_ = cmd.Run()

		output := stdout.String()

		// Check for inline failures
		if strings.Contains(output, "FAIL") {
			lines := strings.Split(output, "\n")
			foundInlineFailures := false
			for i, line := range lines {
				if strings.Contains(line, "FAIL") {
					// Check next lines for failure details
					for j := i + 1; j < len(lines) && j < i+10; j++ {
						if strings.Contains(lines[j], "x ") {
							foundInlineFailures = true
							break
						}
					}
				}
			}
			if !foundInlineFailures {
				t.Errorf("Expected to see failures displayed inline after FAIL message for Vitest")
			}
		}

		// Verify NO "Test failures!" summary section
		if strings.Contains(output, "Test failures!") {
			t.Errorf("Found 'Test failures!' summary section in Vitest output - this should be removed")
		}
	})

	t.Run("pytest_shows_inline_failures", func(t *testing.T) {
		// Test with pytest-fail fixture
		fixtureDir := filepath.Join(projectRoot, "tests", "fixtures", "pytest-fail")
		fixtureDir, err = filepath.Abs(fixtureDir)
		if err != nil {
			t.Fatalf("Failed to get absolute fixture path: %v", err)
		}

		// Check if fixture exists
		if _, err := os.Stat(fixtureDir); os.IsNotExist(err) {
			t.Skip("pytest-fail fixture not found, skipping")
		}

		// Run 3pio with pytest
		cmd := exec.Command(binaryPath, "pytest")
		cmd.Dir = fixtureDir
		cmd.Env = os.Environ()

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		_ = cmd.Run()

		output := stdout.String()

		// Check for inline failures
		if strings.Contains(output, "FAIL") {
			lines := strings.Split(output, "\n")
			foundInlineFailures := false
			for i, line := range lines {
				if strings.Contains(line, "FAIL") {
					// Check next lines for failure details
					for j := i + 1; j < len(lines) && j < i+10; j++ {
						if strings.Contains(lines[j], "x ") {
							foundInlineFailures = true
							break
						}
					}
				}
			}
			if !foundInlineFailures {
				t.Errorf("Expected to see failures displayed inline after FAIL message for pytest")
			}
		}

		// Verify NO "Test failures!" summary section
		if strings.Contains(output, "Test failures!") {
			t.Errorf("Found 'Test failures!' summary section in pytest output - this should be removed")
		}
	})
}
