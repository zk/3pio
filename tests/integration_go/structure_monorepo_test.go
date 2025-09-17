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

	// Verify that both packages' test files were processed in hierarchical structure
	reportsDir := filepath.Join(runDir, "reports")
	if !fileExists(reportsDir) {
		t.Error("Expected reports directory does not exist")
	}

	// Check that report files exist for both packages in the hierarchical structure
	var foundPackageAReports, foundPackageBReports bool
	err = filepath.Walk(reportsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), "index.md") {
			// Check if this report is for package-a or package-b
			// Note: paths are sanitized with underscores replacing dots and slashes
			if strings.Contains(path, "math_test") {
				foundPackageAReports = true
			}
			if strings.Contains(path, "string_test") {
				foundPackageBReports = true
			}
		}
		return nil
	})
	if err != nil {
		t.Errorf("Failed to walk reports directory: %v", err)
	}

	if !foundPackageAReports {
		t.Error("Expected to find report files for package-a (math.test.js)")
	}
	if !foundPackageBReports {
		t.Error("Expected to find report files for package-b (string.test.js)")
	}

	// Verify the test-run.md contains both packages
	testRunContent := readFile(t, filepath.Join(runDir, "test-run.md"))

	// Check that both test files appear in the inline format (not table)
	expectedPackages := []string{
		"math.test.js", // File names should appear in content
		"string.test.js",
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

	// Check IPC file exists in the run directory and contains events from both packages
	ipcPath := filepath.Join(runDir, "ipc.jsonl")
	if _, err := os.Stat(ipcPath); os.IsNotExist(err) {
		t.Fatalf("IPC file does not exist: %s", ipcPath)
	}

	// Read the IPC file and verify it contains events from both packages
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

	// Check for group events (new format)
	if !strings.Contains(content, `"eventType":"testGroupDiscovered"`) {
		t.Error("IPC file does not contain testGroupDiscovered events")
	}

	// Note: testGroupResult events may not be present if groups complete implicitly
	// Just check that we have group discovery and test case events
	t.Logf("IPC file contains group discovery and test case events")

	// Verify summary shows tests from both packages
	testRunContent := readFile(t, filepath.Join(runDir, "test-run.md"))

	// Individual test names are now inline in the main report
	// Check that both test files appear in the content
	expectedFiles := []string{
		"math.test.js",
		"string.test.js",
	}

	for _, expectedFile := range expectedFiles {
		if !strings.Contains(testRunContent, expectedFile) {
			t.Errorf("test-run.md should contain '%s'", expectedFile)
		}
	}

	// Check that we have test results displayed
	if !strings.Contains(testRunContent, "PASS") {
		t.Logf("test-run.md content:\n%s", testRunContent)
		t.Error("Expected to see PASS status indicators in main report")
	}
}
