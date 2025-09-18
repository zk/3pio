package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestJestSkippedTestsConsoleOutput verifies that skipped tests appear in console output
func TestJestSkippedTestsConsoleOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Skip building on Windows (no make), assume binary exists
	if runtime.GOOS != "windows" {
		buildCmd := exec.Command("make", "build")
		buildCmd.Dir = filepath.Join("..", "..")
		if err := buildCmd.Run(); err != nil {
			t.Fatalf("Failed to build 3pio: %v", err)
		}
	}

	// Get the binary path
	projectRoot := filepath.Join("..", "..")
	binaryPath := filepath.Join(projectRoot, "build", "3pio")
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}

	binaryPath, err := filepath.Abs(binaryPath)
	if err != nil {
		t.Fatalf("Failed to get absolute binary path: %v", err)
	}

	// Create a temporary test directory
	tempDir := t.TempDir()

	// Create a test file with skipped tests
	testFile := `
describe('Test Suite', () => {
  test('passing test 1', () => {
    expect(true).toBe(true);
  });

  test.skip('skipped test 1', () => {
    expect(true).toBe(false);
  });

  test('passing test 2', () => {
    expect(2 + 2).toBe(4);
  });

  test.skip('skipped test 2', () => {
    expect(true).toBe(false);
  });

  test.skip('skipped test 3', () => {
    expect(true).toBe(false);
  });
});
`
	testPath := filepath.Join(tempDir, "console.test.js")
	if err := os.WriteFile(testPath, []byte(testFile), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Create a package.json
	packageJSON := `{
  "name": "test-project",
  "version": "1.0.0",
  "scripts": {
    "test": "jest"
  },
  "devDependencies": {
    "jest": "^29.0.0"
  }
}
`
	if err := os.WriteFile(filepath.Join(tempDir, "package.json"), []byte(packageJSON), 0644); err != nil {
		t.Fatalf("Failed to write package.json: %v", err)
	}

	// Run npm install first to get Jest
	installCmd := exec.Command("npm", "install")
	installCmd.Dir = tempDir
	installCmd.Env = os.Environ()
	if output, err := installCmd.CombinedOutput(); err != nil {
		t.Fatalf("npm install failed: %v\nOutput: %s", err, output)
	}

	// Run 3pio with Jest
	cmd := exec.Command(binaryPath, "npx", "jest", "--no-coverage")
	cmd.Dir = tempDir
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// Jest should succeed since we have passing tests
	if err != nil && !strings.Contains(outputStr, "2 passed") {
		t.Logf("Command output: %s", outputStr)
	}

	// Check that the console output shows skipped tests
	// We expect: "Results: 2 passed, 3 skipped, 5 total" or similar
	// The key is that it should show "skipped" in the results line
	if !strings.Contains(outputStr, "skipped") {
		t.Errorf("Console output should show skipped test count")
		t.Logf("Output was: %s", outputStr)
	}

	// More specific check - should show "3 skipped" since we have 3 skipped tests
	if !strings.Contains(outputStr, "3 skipped") {
		t.Errorf("Console output should show '3 skipped' but got: %s", outputStr)
	}

	// Check the test run report to verify totals
	runDirs, err := os.ReadDir(filepath.Join(tempDir, ".3pio", "runs"))
	if err != nil || len(runDirs) == 0 {
		t.Fatalf("Failed to find run directory")
	}

	reportPath := filepath.Join(tempDir, ".3pio", "runs", runDirs[0].Name(), "test-run.md")
	reportContent, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("Failed to read report: %v", err)
	}

	report := string(reportContent)

	// The report should show the correct counts
	if !strings.Contains(report, "Test cases passed: 2") {
		t.Errorf("Report should show 2 passed tests")
	}

	if !strings.Contains(report, "Test cases skipped: 3") {
		t.Errorf("Report should show 3 skipped tests")
	}

	// Check IPC events to ensure testCase events are sent for skipped tests
	ipcPath := filepath.Join(tempDir, ".3pio", "runs", runDirs[0].Name(), "ipc.jsonl")
	ipcContent, err := os.ReadFile(ipcPath)
	if err != nil {
		t.Fatalf("Failed to read IPC file: %v", err)
	}

	ipcLines := strings.Split(string(ipcContent), "\n")
	skippedTestCount := 0
	for _, line := range ipcLines {
		if strings.Contains(line, `"eventType":"testCase"`) && strings.Contains(line, `"status":"SKIP"`) {
			skippedTestCount++
		}
	}

	if skippedTestCount != 3 {
		t.Errorf("Expected 3 testCase events with SKIP status, got %d", skippedTestCount)
	}
}
