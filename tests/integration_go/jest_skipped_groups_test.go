package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestJestSkippedGroupsDiscovery tests that describe blocks containing only skipped tests
// are properly discovered before their results are sent
func TestJestSkippedGroupsDiscovery(t *testing.T) {
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

	// Create a test file with a describe block containing only skipped tests
	testFile := `
describe('Group with only skipped tests', () => {
  test.skip('skipped test 1', () => {
    expect(true).toBe(true);
  });

  test.skip('skipped test 2', () => {
    expect(true).toBe(true);
  });
});

describe('Group with mixed tests', () => {
  test('passing test', () => {
    expect(true).toBe(true);
  });

  test.skip('skipped test', () => {
    expect(true).toBe(true);
  });
});
`
	testPath := filepath.Join(tempDir, "skipped.test.js")
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

	// Jest should succeed with skipped tests
	if err != nil && !strings.Contains(outputStr, "1 passed") {
		t.Logf("Command output: %s", outputStr)
		// Don't fail if the command exits non-zero due to test failures
		// We're testing the adapter behavior, not the test results
	}

	// Check that there are no "group not found" errors in the debug log
	debugLogPath := filepath.Join(tempDir, ".3pio", "debug.log")
	debugLog, err := os.ReadFile(debugLogPath)
	if err == nil {
		debugContent := string(debugLog)
		// Check for the specific error that occurs when groups aren't discovered
		if strings.Contains(debugContent, "group not found") {
			t.Errorf("Debug log contains 'group not found' errors. This indicates groups are not being discovered before their results are sent.")
			// Log a sample of the errors for debugging
			lines := strings.Split(debugContent, "\n")
			for _, line := range lines {
				if strings.Contains(line, "group not found") {
					t.Logf("Error line: %s", line)
					break
				}
			}
		}
	}

	// Check the IPC events to verify proper discovery
	ipcDir := filepath.Join(tempDir, ".3pio", "ipc")
	entries, err := os.ReadDir(ipcDir)
	if err == nil && len(entries) > 0 {
		ipcFile := filepath.Join(ipcDir, entries[0].Name())
		ipcContent, err := os.ReadFile(ipcFile)
		if err != nil {
			t.Fatalf("Failed to read IPC file: %v", err)
		}

		lines := strings.Split(string(ipcContent), "\n")

		// Track discovered groups and groups with results
		discoveredGroups := make(map[string]bool)
		resultGroups := make(map[string]bool)

		for _, line := range lines {
			if strings.Contains(line, `"eventType":"testGroupDiscovered"`) {
				if strings.Contains(line, "Group with only skipped tests") {
					discoveredGroups["Group with only skipped tests"] = true
				}
				if strings.Contains(line, "Group with mixed tests") {
					discoveredGroups["Group with mixed tests"] = true
				}
			}
			if strings.Contains(line, `"eventType":"testGroupResult"`) {
				if strings.Contains(line, "Group with only skipped tests") {
					resultGroups["Group with only skipped tests"] = true
				}
				if strings.Contains(line, "Group with mixed tests") {
					resultGroups["Group with mixed tests"] = true
				}
			}
		}

		// Both groups should be discovered
		if !discoveredGroups["Group with only skipped tests"] {
			t.Error("Group with only skipped tests should be discovered")
		}
		if !discoveredGroups["Group with mixed tests"] {
			t.Error("Group with mixed tests should be discovered")
		}

		// Both groups should have results
		if !resultGroups["Group with only skipped tests"] {
			t.Error("Group with only skipped tests should have results")
		}
		if !resultGroups["Group with mixed tests"] {
			t.Error("Group with mixed tests should have results")
		}

		// Verify discovery happens before results
		var skippedGroupDiscoveryIndex = -1
		var skippedGroupResultIndex = -1

		for i, line := range lines {
			if strings.Contains(line, `"eventType":"testGroupDiscovered"`) &&
				strings.Contains(line, "Group with only skipped tests") {
				skippedGroupDiscoveryIndex = i
			}
			if strings.Contains(line, `"eventType":"testGroupResult"`) &&
				strings.Contains(line, "Group with only skipped tests") {
				skippedGroupResultIndex = i
			}
		}

		if skippedGroupDiscoveryIndex != -1 && skippedGroupResultIndex != -1 {
			if skippedGroupDiscoveryIndex >= skippedGroupResultIndex {
				t.Error("Group discovery should happen before group result")
			}
		}
	}
}
