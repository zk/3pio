package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestMonorepoIPCPathInjection(t *testing.T) {
	// Skip if vitest is not available
	if _, err := exec.LookPath("npx"); err != nil {
		t.Skip("npx not found in PATH, skipping monorepo tests")
	}

	projectDir := filepath.Join(fixturesDir, "monorepo-vitest")

	// Check if the fixture exists
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		t.Skipf("Monorepo fixture not found at %s", projectDir)
	}

	// Clean output directory
	if err := cleanProjectOutput(projectDir); err != nil {
		t.Fatalf("Failed to clean project output: %v", err)
	}

	// Get absolute path to binary
	binaryPath, err := filepath.Abs(threePioBinary)
	if err != nil {
		t.Fatalf("Failed to get absolute path to binary: %v", err)
	}

	// Run tests in the monorepo
	cmd := exec.Command(binaryPath, "npx", "vitest", "run")
	cmd.Dir = projectDir

	// Run the command
	output, _ := cmd.CombinedOutput()
	outputStr := string(output)

	// Check that we got output
	if len(outputStr) == 0 {
		t.Error("Expected some output from test run")
	}

	// Find the latest run directory
	runDir := getLatestRunDir(t, projectDir)

	// Verify that both packages' test files were processed
	// With directory preservation, files are now in package subdirectories
	expectedLogFiles := []string{
		"reports/packages/package-a/math.test.js.md",
		"reports/packages/package-b/string.test.js.md",
	}

	for _, expectedFile := range expectedLogFiles {
		filePath := filepath.Join(runDir, expectedFile)
		if !fileExists(filePath) {
			t.Errorf("Expected log file %s does not exist", expectedFile)
		}
	}

	// Verify the test-run.md contains both packages
	testRunContent := readFile(t, filepath.Join(runDir, "test-run.md"))

	expectedPackages := []string{
		"packages/package-a/math.test.js",
		"packages/package-b/string.test.js",
		"Package A Math Operations",
		"Package B String Operations",
	}

	for _, expected := range expectedPackages {
		if !strings.Contains(testRunContent, expected) {
			t.Errorf("test-run.md should contain '%s'", expected)
		}
	}

	// Check that adapters were created with unique IPC paths
	adaptersDir := filepath.Join(projectDir, ".3pio", "adapters")

	// List all adapter directories
	entries, err := os.ReadDir(adaptersDir)
	if err != nil {
		t.Logf("Warning: Could not read adapters directory: %v", err)
	} else {
		// Should have exactly one adapter directory for this run
		if len(entries) != 1 {
			t.Errorf("Expected 1 adapter directory, found %d", len(entries))
		}

		// Verify the adapter contains the injected IPC path
		for _, entry := range entries {
			if entry.IsDir() {
				adapterPath := filepath.Join(adaptersDir, entry.Name(), "vitest.js")
				if fileExists(adapterPath) {
					content := readFile(t, adapterPath)

					// Check that the adapter does NOT contain template markers
					if strings.Contains(content, "/*__IPC_PATH__*/") {
						t.Error("Adapter still contains template markers")
					}

					// Check that the adapter does NOT contain WILL_BE_REPLACED
					if strings.Contains(content, "WILL_BE_REPLACED") {
						t.Error("Adapter still contains WILL_BE_REPLACED placeholder")
					}

					// Check that the adapter contains a valid IPC path
					// On Windows, paths use backslashes which are escaped in JSON strings
					hasForwardSlash := strings.Contains(content, ".3pio/ipc/")
					hasBackslash := strings.Contains(content, ".3pio\\\\ipc\\\\") // Escaped backslashes in JSON
					hasJsonl := strings.Contains(content, ".jsonl")

					if !hasJsonl || (!hasForwardSlash && !hasBackslash) {
						t.Logf("Adapter path check failed. Has forward slash: %v, Has escaped backslash: %v, Has .jsonl: %v",
							hasForwardSlash, hasBackslash, hasJsonl)
						t.Logf("Sample of adapter content: %s", content[:min(500, len(content))])
						t.Error("Adapter does not contain a valid injected IPC path")
					}
				}
			}
		}
	}
}

func TestMonorepoMultiplePackagesParallel(t *testing.T) {
	// This test verifies that when running tests in a monorepo,
	// each package's tests use the same adapter with the same IPC path
	// and all events are written to the same IPC file

	projectDir := filepath.Join(fixturesDir, "monorepo-vitest")

	// Check if the fixture exists
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		t.Skipf("Monorepo fixture not found at %s", projectDir)
	}

	// Clean output directory
	if err := cleanProjectOutput(projectDir); err != nil {
		t.Fatalf("Failed to clean project output: %v", err)
	}

	// Get absolute path to binary
	binaryPath, err := filepath.Abs(threePioBinary)
	if err != nil {
		t.Fatalf("Failed to get absolute path to binary: %v", err)
	}

	// Run tests
	cmd := exec.Command(binaryPath, "npx", "vitest", "run")
	cmd.Dir = projectDir

	// Run the command
	output, exitErr := cmd.CombinedOutput()

	// Check exit code (should be 0 if all tests pass)
	if exitErr != nil {
		if exitError, ok := exitErr.(*exec.ExitError); ok {
			if exitError.ExitCode() != 0 {
				t.Logf("Command exited with non-zero code: %d", exitError.ExitCode())
				t.Logf("Output: %s", string(output))
			}
		}
	}

	// Find the latest run directory
	runDir := getLatestRunDir(t, projectDir)

	// Check IPC file exists and contains events from both packages
	ipcDir := filepath.Join(projectDir, ".3pio", "ipc")
	entries, err := os.ReadDir(ipcDir)
	if err != nil {
		t.Fatalf("Failed to read IPC directory: %v", err)
	}

	if len(entries) == 0 {
		t.Error("No IPC files found")
	}

	// Read the IPC file and verify it contains events from both packages
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".jsonl") {
			ipcPath := filepath.Join(ipcDir, entry.Name())
			content := readFile(t, ipcPath)

			// Check for events from both packages
			if !strings.Contains(content, "packages/package-a/math.test.js") {
				t.Error("IPC file does not contain events from package-a")
			}

			if !strings.Contains(content, "packages/package-b/string.test.js") {
				t.Error("IPC file does not contain events from package-b")
			}

			// Check for test case events
			if !strings.Contains(content, `"eventType":"testCase"`) {
				t.Error("IPC file does not contain testCase events")
			}

			// Check for test file events
			if !strings.Contains(content, `"eventType":"testFileStart"`) {
				t.Error("IPC file does not contain testFileStart events")
			}

			if !strings.Contains(content, `"eventType":"testFileResult"`) {
				t.Error("IPC file does not contain testFileResult events")
			}
		}
	}

	// Verify summary shows tests from both packages
	testRunContent := readFile(t, filepath.Join(runDir, "test-run.md"))

	// Verify that all 6 tests are present (3 from each package)
	expectedTests := []string{
		"should add numbers correctly",
		"should subtract numbers correctly",
		"should multiply numbers correctly",
		"should concatenate strings",
		"should convert to uppercase",
		"should check string length",
	}

	for _, test := range expectedTests {
		if !strings.Contains(testRunContent, test) {
			t.Errorf("Expected test '%s' not found in test-run.md", test)
		}
	}

	// Verify both files passed
	if !strings.Contains(testRunContent, "Files Passed: 2") {
		t.Logf("test-run.md content:\n%s", testRunContent)
		t.Error("Expected 2 files passed")
	}
}
