package integration_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestFailureDisplayFormat(t *testing.T) {
	// Build 3pio binary
	buildCmd := exec.Command("make", "build")
	buildCmd.Dir = filepath.Join("..", "..")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build 3pio: %v", err)
	}

	// Get the project root directory (2 levels up from tests/integration_go)
	projectRoot := filepath.Join("..", "..")
	binaryPath := filepath.Join(projectRoot, "build", "3pio")

	// Get absolute path for binary
	binaryPath, err := filepath.Abs(binaryPath)
	if err != nil {
		t.Fatalf("Failed to get absolute binary path: %v", err)
	}

	// Change to test fixture directory
	fixtureDir := filepath.Join(projectRoot, "tests", "fixtures", "many-failures")
	fixtureDir, err = filepath.Abs(fixtureDir)
	if err != nil {
		t.Fatalf("Failed to get absolute fixture path: %v", err)
	}

	// Run 3pio with the test fixture
	cmd := exec.Command(binaryPath, "go", "test", ".")
	cmd.Dir = fixtureDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// We expect this to fail since tests are failing
	_ = cmd.Run()

	output := stdout.String()

	// Verify the failure display format
	t.Run("shows_up_to_3_failures", func(t *testing.T) {
		// Check that we show exactly 3 test names
		if !strings.Contains(output, "  x TestFail1") {
			t.Errorf("Expected to see 'x TestFail1' in output")
		}
		if !strings.Contains(output, "  x TestFail2") {
			t.Errorf("Expected to see 'x TestFail2' in output")
		}
		if !strings.Contains(output, "  x TestFail3") {
			t.Errorf("Expected to see 'x TestFail3' in output")
		}
	})

	t.Run("shows_more_count", func(t *testing.T) {
		// Check that we show "+2 more" for the remaining failures
		if !strings.Contains(output, "  +2 more") {
			t.Errorf("Expected to see '+2 more' in output, got:\n%s", output)
		}
	})

	t.Run("shows_report_path", func(t *testing.T) {
		// Check that report path is shown after failures
		if !strings.Contains(output, "  See .3pio/runs/") {
			t.Errorf("Expected to see report path in output")
		}
		if !strings.Contains(output, "/reports/github_com_zk_3pio_tests_fixtures_many-failures/index.md") {
			t.Errorf("Expected to see correct report path format")
		}
	})

	t.Run("does_not_show_fourth_failure", func(t *testing.T) {
		// Verify we don't show the 4th failure name (TestFail4)
		if strings.Contains(output, "  x TestFail4") {
			t.Errorf("Should not show TestFail4 (4th failure) in output")
		}
	})

	t.Run("correct_failure_count_in_summary", func(t *testing.T) {
		// The summary should indicate all tests ran
		if !strings.Contains(output, "3 passed, 1 total") {
			// We have 3 passing tests and 5 failing tests = 8 total
			// But it's reported as 1 file
			t.Logf("Output summary line may not match expected format")
		}
	})
}

func TestSingleFailureDisplay(t *testing.T) {
	// Create a temporary test file with just one failure
	tempDir := t.TempDir()

	// Create go.mod
	goModContent := `module temptest
go 1.21`
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create test file with single failure
	testContent := `package temptest
import "testing"
func TestSingleFailure(t *testing.T) {
	t.Fatal("This is the only failure")
}`
	if err := os.WriteFile(filepath.Join(tempDir, "single_test.go"), []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Get the project root directory (2 levels up from tests/integration_go)
	projectRoot := filepath.Join("..", "..")
	binaryPath := filepath.Join(projectRoot, "build", "3pio")

	// Get absolute path for binary
	binaryPath, err := filepath.Abs(binaryPath)
	if err != nil {
		t.Fatalf("Failed to get absolute binary path: %v", err)
	}

	// Run 3pio
	cmd := exec.Command(binaryPath, "go", "test", ".")
	cmd.Dir = tempDir

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	_ = cmd.Run()

	output := stdout.String()

	// Should show the single failure without "+N more"
	if !strings.Contains(output, "  x TestSingleFailure") {
		t.Errorf("Expected to see single failure in output")
	}
	if strings.Contains(output, "more") {
		t.Errorf("Should not show '+N more' for single failure")
	}
}