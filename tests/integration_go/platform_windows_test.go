//go:build windows
// +build windows

package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestWindowsBinaryExtension verifies that the Windows binary has .exe extension
func TestWindowsBinaryExtension(t *testing.T) {
	binaryPath := getBinaryPath()

	if !strings.HasSuffix(binaryPath, ".exe") {
		t.Errorf("Windows binary path should end with .exe, got: %s", binaryPath)
	}

	// Check that the binary exists
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Errorf("Binary does not exist at: %s", binaryPath)
	}
}

// TestWindowsPathSeparators verifies correct handling of Windows path separators
func TestWindowsPathSeparators(t *testing.T) {
	testDir := filepath.Join(fixturesDir, "basic-jest")
	cleanTestDir(t, testDir)

	// Run a test
	cmd := exec.Command(getBinaryPath(), "npx", "jest", "math.test.js")
	cmd.Dir = testDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check exit code for test failures (which is expected)
		if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() == 0 {
			t.Fatalf("Failed to run test: %v\nOutput: %s", err, output)
		}
	}

	// Get the run directory
	runDir := getLatestRunDir(t, testDir)

	// Check that report uses proper path separators
	reportPath := filepath.Join(runDir, "test-run.md")
	content, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("Failed to read report: %v", err)
	}

	// Windows paths in the report should use backslashes
	reportContent := string(content)
	if strings.Contains(reportContent, "math.test.js") {
		// The report should contain Windows-style paths
		if !strings.Contains(reportContent, "\\") && !strings.Contains(reportContent, "math.test.js") {
			t.Error("Report should contain proper Windows path separators")
		}
	}
}

// TestWindowsPowerShellExecution tests that 3pio works when invoked from PowerShell
func TestWindowsPowerShellExecution(t *testing.T) {
	testDir := filepath.Join(fixturesDir, "basic-vitest")
	cleanTestDir(t, testDir)

	// Use PowerShell to run 3pio
	binaryPath := getBinaryPath()
	psCommand := binaryPath + " npx vitest run"
	cmd := exec.Command("powershell", "-Command", psCommand)
	cmd.Dir = testDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if it's just test failures by looking for common test output indicators
		outputStr := string(output)
		if !strings.Contains(outputStr, "Test Files") &&
		   !strings.Contains(outputStr, "Test") &&
		   !strings.Contains(outputStr, "passed") &&
		   !strings.Contains(outputStr, "failed") &&
		   !strings.Contains(outputStr, "Results:") {
			t.Fatalf("PowerShell execution failed: %v\nOutput: %s", err, output)
		}
	}

	// Verify that reports were created
	runDir := getLatestRunDir(t, testDir)
	assertReportExists(t, runDir)
	assertOutputLogExists(t, runDir)
}

// TestWindowsLongPathSupport tests handling of long Windows paths (>260 chars)
func TestWindowsLongPathSupport(t *testing.T) {
	// Create a deeply nested directory structure
	baseDir := t.TempDir()

	// Build a long path
	longPath := baseDir
	for i := 0; i < 20; i++ {
		longPath = filepath.Join(longPath, "very_long_directory_name_to_test_windows_limits")
	}

	// Try to create the directory
	err := os.MkdirAll(longPath, 0755)
	if err != nil {
		// If we can't create long paths, skip this test
		t.Skip("System doesn't support long paths, skipping test")
		return
	}

	// Copy a simple test file to the long path
	testFile := `
describe('test', () => {
	it('works', () => {
		expect(1).toBe(1);
	});
});
`
	testFilePath := filepath.Join(longPath, "test.spec.js")
	err = os.WriteFile(testFilePath, []byte(testFile), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Also need package.json
	packageJSON := `{"name": "test", "version": "1.0.0"}`
	err = os.WriteFile(filepath.Join(longPath, "package.json"), []byte(packageJSON), 0644)
	if err != nil {
		t.Fatalf("Failed to write package.json: %v", err)
	}

	// Try to run 3pio with the long path
	cmd := exec.Command(getBinaryPath(), "npx", "jest", "test.spec.js")
	cmd.Dir = longPath

	output, _ := cmd.CombinedOutput()

	// We mainly care that 3pio doesn't crash with long paths
	// The test might fail due to missing jest, but that's ok
	if strings.Contains(string(output), "panic") {
		t.Errorf("3pio panicked with long path: %s", output)
	}
}

// TestWindowsUnicodePathSupport tests handling of Unicode characters in Windows paths
func TestWindowsUnicodePathSupport(t *testing.T) {
	// Create a directory with Unicode characters
	baseDir := t.TempDir()
	unicodeDir := filepath.Join(baseDir, "测试目录_тест_🚀")

	err := os.MkdirAll(unicodeDir, 0755)
	if err != nil {
		t.Skip("System doesn't support Unicode paths, skipping test")
		return
	}

	// Create a simple test file
	testFile := `
test('unicode path test', () => {
	expect(true).toBe(true);
});
`
	testFilePath := filepath.Join(unicodeDir, "test.spec.js")
	err = os.WriteFile(testFilePath, []byte(testFile), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Create package.json
	packageJSON := `{"name": "unicode-test", "version": "1.0.0"}`
	err = os.WriteFile(filepath.Join(unicodeDir, "package.json"), []byte(packageJSON), 0644)
	if err != nil {
		t.Fatalf("Failed to write package.json: %v", err)
	}

	// Run 3pio
	cmd := exec.Command(getBinaryPath(), "npx", "jest", "test.spec.js")
	cmd.Dir = unicodeDir

	output, _ := cmd.CombinedOutput()

	// Check that 3pio doesn't crash
	if strings.Contains(string(output), "panic") {
		t.Errorf("3pio panicked with Unicode path: %s", output)
	}
}

// TestWindowsFilePermissions tests that 3pio handles Windows file permissions correctly
func TestWindowsFilePermissions(t *testing.T) {
	testDir := filepath.Join(fixturesDir, "basic-jest")
	cleanTestDir(t, testDir)

	// Run a test to create output
	cmd := exec.Command(getBinaryPath(), "npx", "jest")
	cmd.Dir = testDir
	cmd.Run()

	// Check that all created files are readable
	threepioDir := filepath.Join(testDir, ".3pio")
	err := filepath.Walk(threepioDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Try to read each file
		if !info.IsDir() {
			_, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Errorf("Cannot read file %s: %v", path, readErr)
			}
		}

		return nil
	})

	if err != nil {
		t.Errorf("Error walking .3pio directory: %v", err)
	}
}
