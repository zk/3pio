package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Helper function to run the 3pio binary
func runBinary(t *testing.T, projectDir string, args ...string) (string, string, int) {
	binaryPath, err := filepath.Abs("../../build/3pio")
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

// TestErrorReportingToConsole verifies that errors are properly displayed to the user
func TestErrorReportingToConsole(t *testing.T) {
	t.Parallel()

	// Create a temporary directory with a broken Jest setup
	tempDir := t.TempDir()
	
	// Create a package.json with Jest but missing configuration
	packageJSON := `{
		"name": "broken-jest",
		"scripts": {
			"test": "jest"
		},
		"devDependencies": {
			"jest": "^29.0.0"
		}
	}`
	if err := os.WriteFile(filepath.Join(tempDir, "package.json"), []byte(packageJSON), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a broken Jest config that will cause an error
	jestConfig := `module.exports = {
		preset: 'non-existent-preset'
	};`
	if err := os.WriteFile(filepath.Join(tempDir, "jest.config.js"), []byte(jestConfig), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a simple test file
	testFile := `describe('test', () => {
		it('should work', () => {
			expect(true).toBe(true);
		});
	});`
	if err := os.WriteFile(filepath.Join(tempDir, "test.spec.js"), []byte(testFile), 0644); err != nil {
		t.Fatal(err)
	}

	// Run 3pio and capture output
	output, _, exitCode := runBinary(t, tempDir, "npx", "jest")

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

	// Create a temporary directory with a broken Jest setup
	tempDir := t.TempDir()
	
	// Create a package.json with Jest but missing configuration
	packageJSON := `{
		"name": "broken-jest",
		"scripts": {
			"test": "jest"
		},
		"devDependencies": {
			"jest": "^29.0.0"
		}
	}`
	if err := os.WriteFile(filepath.Join(tempDir, "package.json"), []byte(packageJSON), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a Jest config that references a non-existent preset
	jestConfig := `{
		"preset": "non-existent-preset"
	}`
	if err := os.WriteFile(filepath.Join(tempDir, "jest.config.json"), []byte(jestConfig), 0644); err != nil {
		t.Fatal(err)
	}

	// Run 3pio
	_, _, exitCode := runBinary(t, tempDir, "npx", "jest")

	// Should fail
	if exitCode == 0 {
		t.Error("Expected non-zero exit code for broken configuration")
	}

	// Check the test-run.md file
	runDir := findLatestRunDir(t, tempDir)
	reportPath := filepath.Join(runDir, "test-run.md")
	
	reportContent, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("Failed to read test-run.md: %v", err)
	}

	report := string(reportContent)

	// Check that status is ERROR
	if !strings.Contains(report, "**Status:** ERROR") {
		t.Error("Report should show ERROR status")
	}

	// Check that error details are included
	if !strings.Contains(report, "## Error Details") && !strings.Contains(report, "## Error") {
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

	// Create a temporary directory with TypeScript Jest config
	tempDir := t.TempDir()
	
	// Create a package.json with Jest
	packageJSON := `{
		"name": "jest-ts-config",
		"devDependencies": {
			"jest": "^29.0.0"
		}
	}`
	if err := os.WriteFile(filepath.Join(tempDir, "package.json"), []byte(packageJSON), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a TypeScript Jest config (without ts-node installed)
	jestConfig := `export default {
		testEnvironment: 'node'
	};`
	if err := os.WriteFile(filepath.Join(tempDir, "jest.config.ts"), []byte(jestConfig), 0644); err != nil {
		t.Fatal(err)
	}

	// Run 3pio
	output, _, exitCode := runBinary(t, tempDir, "npx", "jest")

	// Should fail
	if exitCode == 0 {
		t.Error("Expected non-zero exit code for missing ts-node")
	}

	// Check that the ts-node error is displayed
	if !strings.Contains(output, "ts-node") || !strings.Contains(output, "required") {
		t.Errorf("ts-node error not displayed to console. Output:\n%s", output)
	}

	// Check the report includes the error
	runDir := findLatestRunDir(t, tempDir)
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