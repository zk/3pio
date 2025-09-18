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

    t.Run("go_test_shows_minimal_summary", func(t *testing.T) {
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

        // New format prints a minimal summary with counts and a report path
        if !strings.Contains(output, "FAIL(") {
            t.Errorf("Expected to see FAIL count in output")
        }
        if !strings.Contains(output, "/reports/") {
            t.Errorf("Expected to see report path in output")
        }

		// The "Test failures!" summary section can still exist alongside inline display
	})

    t.Run("jest_shows_minimal_summary", func(t *testing.T) {
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

        // Minimal summary with counts and report path is expected
        if !strings.Contains(output, "FAIL(") || !strings.Contains(output, "/reports/") {
            t.Errorf("Expected minimal summary with report path for Jest")
        }
    })

    t.Run("vitest_shows_minimal_summary", func(t *testing.T) {
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

        if !strings.Contains(output, "FAIL(") || !strings.Contains(output, "/reports/") {
            t.Errorf("Expected minimal summary with report path for Vitest")
        }
    })

    t.Run("pytest_shows_minimal_summary", func(t *testing.T) {
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

        if !strings.Contains(output, "FAIL(") || !strings.Contains(output, "/reports/") {
            t.Errorf("Expected minimal summary with report path for pytest")
        }
    })
}
