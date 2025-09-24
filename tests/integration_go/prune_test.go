package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPruneCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Get the binary path
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	binaryPath := filepath.Join(filepath.Dir(filepath.Dir(cwd)), "build", "3pio")
	binaryPath, err = filepath.Abs(binaryPath)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("3pio binary not found, run 'make build' first")
	}

	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Create .3pio structure with multiple runs
	baseDir := ".3pio"
	runsDir := filepath.Join(baseDir, "runs")
	os.MkdirAll(runsDir, 0755)

	// Create test runs
	runs := []string{
		"20240101_120000_oldest-run",
		"20240102_120000_middle-run",
		"20240103_120000_newest-run",
	}
	for _, run := range runs {
		runPath := filepath.Join(runsDir, run)
		os.MkdirAll(runPath, 0755)
		os.WriteFile(filepath.Join(runPath, "test-run.md"), []byte("# Test Run"), 0644)
		os.MkdirAll(filepath.Join(runPath, "logs"), 0755)
		os.WriteFile(filepath.Join(runPath, "output.log"), []byte("test output"), 0644)
	}

	// Create debug log
	debugLogPath := filepath.Join(baseDir, "debug.log")
	os.WriteFile(debugLogPath, []byte("old debug log content\n"), 0644)

	// Test 1: Dry run
	t.Run("dry-run", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "prune", "--dry-run")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("prune --dry-run failed: %v\nOutput: %s", err, output)
		}

		// Check output contains expected information
		outputStr := string(output)
		if !strings.Contains(outputStr, "DRY RUN MODE") {
			t.Errorf("Expected dry run mode indicator in output")
		}
		if !strings.Contains(outputStr, "Total runs found: 3") {
			t.Errorf("Expected 3 runs in output, got: %s", outputStr)
		}
		if !strings.Contains(outputStr, "Runs to delete: 2") {
			t.Errorf("Expected 2 runs to delete in output")
		}

		// Verify nothing was actually deleted
		for _, run := range runs {
			runPath := filepath.Join(runsDir, run)
			if _, err := os.Stat(runPath); os.IsNotExist(err) {
				t.Errorf("Run %s should not be deleted in dry-run mode", run)
			}
		}
	})

	// Test 2: Help
	t.Run("help", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "prune", "--help")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("prune --help failed: %v\nOutput: %s", err, output)
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "Usage: 3pio prune") {
			t.Errorf("Expected usage information in help output")
		}
		if !strings.Contains(outputStr, "--dry-run") {
			t.Errorf("Expected --dry-run flag in help output")
		}
		if !strings.Contains(outputStr, "--force") {
			t.Errorf("Expected --force flag in help output")
		}
	})

	// Test 3: Actual prune with force flag
	t.Run("force-prune", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "prune", "--force")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("prune --force failed: %v\nOutput: %s", err, output)
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "Prune completed successfully") {
			t.Errorf("Expected success message in output: %s", outputStr)
		}

		// Verify oldest runs were deleted
		deletedRuns := []string{"20240101_120000_oldest-run", "20240102_120000_middle-run"}
		for _, run := range deletedRuns {
			runPath := filepath.Join(runsDir, run)
			if _, err := os.Stat(runPath); !os.IsNotExist(err) {
				t.Errorf("Run %s should be deleted", run)
			}
		}

		// Verify newest run was kept
		keptRun := filepath.Join(runsDir, "20240103_120000_newest-run")
		if _, err := os.Stat(keptRun); os.IsNotExist(err) {
			t.Errorf("Newest run should be kept")
		}

		// Verify debug log exists but was cleared (will have new content from prune operation)
		if info, err := os.Stat(debugLogPath); err != nil {
			t.Errorf("Debug log should still exist: %v", err)
		} else if info.Size() > 10000 {
			// Allow reasonable size for new log entries after clearing
			t.Errorf("Debug log seems too large after clearing: %d bytes", info.Size())
		}
	})
}

func TestPruneWithNoRuns(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Get the binary path
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	binaryPath := filepath.Join(filepath.Dir(filepath.Dir(cwd)), "build", "3pio")
	binaryPath, err = filepath.Abs(binaryPath)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("3pio binary not found, run 'make build' first")
	}

	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Create empty .3pio directory
	os.MkdirAll(".3pio", 0755)

	// Run prune
	cmd := exec.Command(binaryPath, "prune", "--force")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("prune with no runs failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "No test runs found") {
		t.Errorf("Expected 'No test runs found' message, got: %s", outputStr)
	}
}

func TestPruneWithoutDirectory(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Get the binary path
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	binaryPath := filepath.Join(filepath.Dir(filepath.Dir(cwd)), "build", "3pio")
	binaryPath, err = filepath.Abs(binaryPath)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("3pio binary not found, run 'make build' first")
	}

	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Don't create .3pio directory

	// Run prune
	cmd := exec.Command(binaryPath, "prune", "--force")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("prune without .3pio directory failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "No .3pio directory found") {
		t.Errorf("Expected 'No .3pio directory found' message, got: %s", outputStr)
	}
}

func TestPruneInvalidFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Get the binary path
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	binaryPath := filepath.Join(filepath.Dir(filepath.Dir(cwd)), "build", "3pio")
	binaryPath, err = filepath.Abs(binaryPath)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("3pio binary not found, run 'make build' first")
	}

	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Run prune with invalid flag
	cmd := exec.Command(binaryPath, "prune", "--invalid-flag")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("Expected error for invalid flag, but command succeeded")
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "Unknown flag: --invalid-flag") {
		t.Errorf("Expected unknown flag error, got: %s", outputStr)
	}
}
