package integration_test

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Helper function to run the 3pio binary
func runBinary(t *testing.T, projectDir string, args ...string) (string, string, int) {
	binaryName := "3pio"
	if runtime.GOOS == "windows" {
		binaryName = "3pio.exe"
	}

	binaryPath, err := filepath.Abs(filepath.Join("../../build", binaryName))
	if err != nil {
		t.Fatalf("Failed to get absolute path to binary: %v", err)
	}

	// Prepare full command
	fullCmd := append([]string{binaryPath}, args...)
	cmd := exec.Command(fullCmd[0], fullCmd[1:]...)
	cmd.Dir = projectDir

	// Run the command
	output, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = 1
		}
	}

	return string(output), "", exitCode
}

// Helper function to find the latest run directory
func findLatestRunDir(t *testing.T, projectDir string) string {
	runsDir := filepath.Join(projectDir, ".3pio", "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		t.Fatalf("Failed to read runs directory: %v", err)
	}

	var latestDir string
	for _, entry := range entries {
		if entry.IsDir() {
			if latestDir == "" || entry.Name() > latestDir {
				latestDir = entry.Name()
			}
		}
	}

	if latestDir == "" {
		t.Fatal("No run directories found")
	}

	return filepath.Join(runsDir, latestDir)
}

// Helper function to recursively copy a directory
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get the relative path from src
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		// Create the destination path
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			// Create directory
			return os.MkdirAll(dstPath, info.Mode())
		} else {
			// Copy file
			srcFile, err := os.Open(path)
			if err != nil {
				return err
			}
			defer srcFile.Close()

			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
				return err
			}

			dstFile, err := os.Create(dstPath)
			if err != nil {
				return err
			}
			defer dstFile.Close()

			// Copy the file content
			if _, err = io.Copy(dstFile, srcFile); err != nil {
				return err
			}

			// Preserve file permissions
			return os.Chmod(dstPath, info.Mode())
		}
	})
}

// TestErrorReportingToConsole verifies that errors are properly displayed to the user
func TestErrorReportingToConsole(t *testing.T) {
	t.Parallel()

	// Use the jest-config-error fixture by copying it to a temp directory
	sourceFixtureDir := filepath.Join("..", "fixtures", "jest-config-error")
	if _, err := os.Stat(sourceFixtureDir); err != nil {
		t.Skipf("Fixture directory not found: %s", sourceFixtureDir)
	}

	// Create a temporary directory for this test
	tempDir := t.TempDir()
	fixtureDir := filepath.Join(tempDir, "jest-config-error")

	// Copy the fixture to the temp directory
	err := copyDir(sourceFixtureDir, fixtureDir)
	if err != nil {
		t.Fatalf("Failed to copy fixture: %v", err)
	}

	// Run 3pio and capture output
	output, _, exitCode := runBinary(t, fixtureDir, "npx", "jest")

	// The test should fail due to missing preset
	if exitCode == 0 {
		t.Error("Expected non-zero exit code for broken configuration")
	}

	// Check that the error is displayed to the console
	if !strings.Contains(output, "Error:") || !strings.Contains(output, "preset") || !strings.Contains(output, "non-existent-preset") {
		t.Errorf("Error message not displayed to console. Output:\n%s", output)
	}

	// Verify that we still get the standard 3pio output
	if !strings.Contains(output, "Full report:") {
		t.Error("Missing standard 3pio output")
	}
}

// TestErrorDetailsInReport verifies that error details are included in test-run.md
func TestErrorDetailsInReport(t *testing.T) {
	t.Parallel()

	// Use the jest-config-error fixture by copying it to a temp directory
	sourceFixtureDir := filepath.Join("..", "fixtures", "jest-config-error")
	if _, err := os.Stat(sourceFixtureDir); err != nil {
		t.Skipf("Fixture directory not found: %s", sourceFixtureDir)
	}

	// Create a temporary directory for this test
	tempDir := t.TempDir()
	fixtureDir := filepath.Join(tempDir, "jest-config-error")

	// Copy the fixture to the temp directory
	err := copyDir(sourceFixtureDir, fixtureDir)
	if err != nil {
		t.Fatalf("Failed to copy fixture: %v", err)
	}

	// Run 3pio
	_, _, exitCode := runBinary(t, fixtureDir, "npx", "jest")

	// Should fail
	if exitCode == 0 {
		t.Error("Expected non-zero exit code for broken configuration")
	}

	// Check the test-run.md file
	runDir := findLatestRunDir(t, fixtureDir)
	reportPath := filepath.Join(runDir, "test-run.md")

	reportContent, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("Failed to read test-run.md: %v", err)
	}

	report := string(reportContent)

	// Check that status is ERRORED in YAML frontmatter
	if !strings.Contains(report, "status: ERRORED") {
		t.Error("Report should show ERRORED status in YAML frontmatter")
	}

	// Check that error details are included
	if !strings.Contains(report, "## Error") {
		t.Error("Report should include an Error section")
	}

	// Check that the actual error message is included
	if !strings.Contains(report, "preset") || !strings.Contains(report, "non-existent-preset") {
		t.Errorf("Report should include the actual error message. Report:\n%s", report)
	}

	// Check that test summary is NOT included when there's an error
	if strings.Contains(report, "## Summary") {
		t.Error("Report should not include Summary section when there's an error")
	}
}

// TestJestConfigError verifies TypeScript config errors are reported properly
func TestJestConfigError(t *testing.T) {
	t.Parallel()

	// Use the jest-ts-config-error fixture by copying it to a temp directory
	sourceFixtureDir := filepath.Join("..", "fixtures", "jest-ts-config-error")
	if _, err := os.Stat(sourceFixtureDir); err != nil {
		t.Skipf("Fixture directory not found: %s", sourceFixtureDir)
	}

	// Create a temporary directory for this test
	tempDir := t.TempDir()
	fixtureDir := filepath.Join(tempDir, "jest-ts-config-error")

	// Copy the fixture to the temp directory
	err := copyDir(sourceFixtureDir, fixtureDir)
	if err != nil {
		t.Fatalf("Failed to copy fixture: %v", err)
	}

	// Run 3pio
	output, _, exitCode := runBinary(t, fixtureDir, "npx", "jest")

	// Should fail
	if exitCode == 0 {
		t.Error("Expected non-zero exit code for missing ts-node")
	}

	// Check that the ts-node error is displayed
	if !strings.Contains(output, "ts-node") || !strings.Contains(output, "required") {
		t.Errorf("ts-node error not displayed to console. Output:\n%s", output)
	}

	// Check the report includes the error
	runDir := findLatestRunDir(t, fixtureDir)
	reportPath := filepath.Join(runDir, "test-run.md")

	reportContent, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("Failed to read test-run.md: %v", err)
	}

	report := string(reportContent)
	if !strings.Contains(report, "ts-node") {
		t.Error("Report should include ts-node error details")
	}
}
