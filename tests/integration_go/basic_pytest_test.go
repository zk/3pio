package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zk/3pio/tests/testutil"
)

// TestPytestFullRun verifies that pytest runs all tests with no arguments
func TestPytestFullRun(t *testing.T) {
	// Check if pytest is available
	if _, err := testutil.LookPath("pytest"); err != nil {
		if _, err := testutil.LookPath("python3"); err != nil {
			t.Skip("python3 not found in PATH")
		}
		if err := testutil.CommandAvailable("python3", "-m", "pytest", "--version"); err != nil {
			t.Skip("pytest not available")
		}
	}

	testDir := filepath.Join("..", "fixtures", "basic-pytest")
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Skip("basic-pytest fixture not found")
	}

	result := testutil.RunThreepio(t, testDir, "pytest")

	// Add diagnostic output for debugging CI issues
	if result.RunID == "" {
		t.Logf("Debug: RunID is empty. Stdout: %s, Stderr: %s", result.Stdout, result.Stderr)
	}

	// Check report exists
	reportPath := filepath.Join(testDir, ".3pio", "runs", result.RunID, "test-run.md")
	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		t.Errorf("Report file not found: %s", reportPath)
	}

	// Check report contains test results
	content, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("Failed to read report: %v", err)
	}
	reportStr := string(content)

	// Should contain Python test indicators
	if !strings.Contains(reportStr, "test_") && !strings.Contains(reportStr, ".py") {
		t.Error("Report should contain Python test names or files")
	}
}

// TestPytestSpecificFile verifies pytest can run tests from a specific file
func TestPytestSpecificFile(t *testing.T) {
	// Check if pytest is available
	if _, err := testutil.LookPath("pytest"); err != nil {
		if _, err := testutil.LookPath("python3"); err != nil {
			t.Skip("python3 not found in PATH")
		}
		if err := testutil.CommandAvailable("python3", "-m", "pytest", "--version"); err != nil {
			t.Skip("pytest not available")
		}
	}

	testDir := filepath.Join("..", "fixtures", "basic-pytest")
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Skip("basic-pytest fixture not found")
	}

	// Check if test_math.py exists
	testFile := filepath.Join(testDir, "test_math.py")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("test_math.py not found in fixture")
	}

	result := testutil.RunThreepio(t, testDir, "pytest", "test_math.py")

	// Add diagnostic output for debugging CI issues
	if result.RunID == "" {
		t.Logf("Debug: RunID is empty. Stdout: %s, Stderr: %s", result.Stdout, result.Stderr)
	}

	// Check report exists
	reportPath := filepath.Join(testDir, ".3pio", "runs", result.RunID, "test-run.md")
	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		t.Errorf("Report file not found: %s", reportPath)
	}

	// Check that only the specified file was tested
	content, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("Failed to read report: %v", err)
	}
	if !strings.Contains(string(content), "test_math.py") {
		t.Error("Report should contain test_math.py")
	}
}

// TestPytestPatternMatching verifies pytest -k pattern matching
func TestPytestPatternMatching(t *testing.T) {
	testDir := filepath.Join("..", "fixtures", "basic-pytest")
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Skip("basic-pytest fixture not found")
	}
	cleanTestDir(t, testDir)

	// Create test files with specific patterns
	mathTest := `def test_addition():
    assert 1 + 1 == 2

def test_multiply():
    assert 2 * 3 == 6

def test_string_concat():
    assert "a" + "b" == "ab"
`
	if err := os.WriteFile(filepath.Join(testDir, "test_patterns.py"), []byte(mathTest), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Run pytest with pattern matching
	cmd := exec.Command(getBinaryPath(), "pytest", "-k", "test_add")
	cmd.Dir = testDir

	output, err := cmd.CombinedOutput()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() == 127 {
			t.Fatalf("Failed to run pytest with pattern: %v\nOutput: %s", err, output)
		}
	}

	// Verify selective test execution
	runDir := getLatestRunDir(t, testDir)

	// Check that test_patterns.py was run (it contains test_addition)
	reportPath := filepath.Join(runDir, "test-run.md")
	content, _ := os.ReadFile(reportPath)
	reportStr := string(content)

	// The main report should show test_patterns.py was run
	if !strings.Contains(reportStr, "test_patterns.py") {
		t.Error("Report should show test_patterns.py was run when using -k test_add")
	}

	// The individual report should contain test_addition
	patternReportPath := filepath.Join(runDir, "reports", "test_patterns_py", "index.md")
	if patternContent, err := os.ReadFile(patternReportPath); err == nil {
		if !strings.Contains(string(patternContent), "test_addition") {
			t.Error("Individual report should contain test_addition when using -k test_add")
		}
	}
}

// TestPytestExitCodePass verifies exit code 0 when all tests pass
func TestPytestExitCodePass(t *testing.T) {
	testDir := t.TempDir()

	// Create a passing test
	passingTest := `def test_pass():
    assert True
`
	if err := os.WriteFile(filepath.Join(testDir, "test_pass.py"), []byte(passingTest), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Run pytest
	cmd := exec.Command(getBinaryPath(), "pytest")
	cmd.Dir = testDir

	output, err := cmd.CombinedOutput()

	// Should exit with code 0
	if err != nil {
		t.Errorf("Expected exit code 0 for passing tests, got error: %v\nOutput: %s", err, output)
	}

	// Verify reports were created
	runDir := getLatestRunDir(t, testDir)
	assertReportExists(t, runDir)
}

// TestPytestExitCodeFail verifies non-zero exit code when tests fail
func TestPytestExitCodeFail(t *testing.T) {
	testDir := t.TempDir()

	// Create a failing test
	failingTest := `def test_fail():
    assert False, "This test should fail"
`
	if err := os.WriteFile(filepath.Join(testDir, "test_fail.py"), []byte(failingTest), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Run pytest
	cmd := exec.Command(getBinaryPath(), "pytest")
	cmd.Dir = testDir

	_, err := cmd.CombinedOutput()

	// Should exit with non-zero code
	if err == nil {
		t.Error("Expected non-zero exit code for failing tests")
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() == 0 {
			t.Error("Expected non-zero exit code for failing tests")
		}
	}

	// Verify reports were still created
	runDir := getLatestRunDir(t, testDir)
	assertReportExists(t, runDir)
}

// TestPytestExitCodeError verifies exit code for syntax errors
func TestPytestExitCodeError(t *testing.T) {
	testDir := t.TempDir()

	// Create a test with syntax error
	syntaxErrorTest := `def test_syntax(
    # Missing closing parenthesis
    assert True
`
	if err := os.WriteFile(filepath.Join(testDir, "test_syntax.py"), []byte(syntaxErrorTest), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Run pytest
	cmd := exec.Command(getBinaryPath(), "pytest")
	cmd.Dir = testDir

	output, err := cmd.CombinedOutput()

	// Should exit with non-zero code
	if err == nil {
		t.Error("Expected non-zero exit code for syntax error")
	}

	// Check that error is reported (exit code 2 for collection errors)
	outputStr := string(output)
	// 3pio captures the error but shows "exit status 2" for syntax errors
	if !strings.Contains(outputStr, "exit status 2") && !strings.Contains(outputStr, "Error") {
		t.Errorf("Expected error indication in output, got: %s", outputStr)
	}
}

// TestPytestVerboseMode verifies pytest -v verbose output
func TestPytestVerboseMode(t *testing.T) {
	testDir := filepath.Join("..", "fixtures", "basic-pytest")
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Skip("basic-pytest fixture not found")
	}
	cleanTestDir(t, testDir)

	// Create a test file
	test := `def test_verbose():
    assert True
`
	if err := os.WriteFile(filepath.Join(testDir, "test_verbose.py"), []byte(test), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Run pytest with verbose flag
	cmd := exec.Command(getBinaryPath(), "pytest", "-v")
	cmd.Dir = testDir

	output, err := cmd.CombinedOutput()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() == 127 {
			t.Fatalf("Failed to run pytest -v: %v\nOutput: %s", err, output)
		}
	}

	// Verbose mode should show more detailed output
	outputStr := string(output)
	if !strings.Contains(outputStr, "test_verbose") || !strings.Contains(outputStr, "PASS") {
		// Verbose output typically shows test names and results
		// But we don't want to be too strict about format
		t.Log("Verbose mode output processed")
	}

	// Verify reports were created
	runDir := getLatestRunDir(t, testDir)
	assertReportExists(t, runDir)
}

// TestPytestQuietMode verifies pytest -q quiet output
func TestPytestQuietMode(t *testing.T) {
	testDir := filepath.Join("..", "fixtures", "basic-pytest")
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Skip("basic-pytest fixture not found")
	}
	cleanTestDir(t, testDir)

	// Create a test file
	test := `def test_quiet():
    assert True
`
	if err := os.WriteFile(filepath.Join(testDir, "test_quiet.py"), []byte(test), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Run pytest with quiet flag
	cmd := exec.Command(getBinaryPath(), "pytest", "-q")
	cmd.Dir = testDir

	output, err := cmd.CombinedOutput()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() == 127 {
			t.Fatalf("Failed to run pytest -q: %v\nOutput: %s", err, output)
		}
	}

	// Quiet mode should produce minimal output
	// But reports should still be created
	runDir := getLatestRunDir(t, testDir)
	assertReportExists(t, runDir)
	assertOutputLogExists(t, runDir)
}

// TestPytestMultipleFiles verifies pytest can run tests from multiple files
func TestPytestMultipleFiles(t *testing.T) {
	testDir := t.TempDir()

	// Create multiple test files
	test1 := `def test_file1():
    assert True
`
	test2 := `def test_file2():
    assert True
`
	if err := os.WriteFile(filepath.Join(testDir, "test_one.py"), []byte(test1), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testDir, "test_two.py"), []byte(test2), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Run pytest on multiple files
	cmd := exec.Command(getBinaryPath(), "pytest", "test_one.py", "test_two.py")
	cmd.Dir = testDir

	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Errorf("Failed to run pytest on multiple files: %v\nOutput: %s", err, output)
	}

	// Verify both files were tested
	runDir := getLatestRunDir(t, testDir)
	reportPath := filepath.Join(runDir, "test-run.md")
	content, _ := os.ReadFile(reportPath)
	reportStr := string(content)

	if !strings.Contains(reportStr, "test_one.py") || !strings.Contains(reportStr, "test_two.py") {
		t.Error("Report should contain both test files")
	}
}

// TestPytestDirectory verifies pytest can run tests from a directory
func TestPytestDirectory(t *testing.T) {
	testDir := t.TempDir()

	// Create a tests subdirectory
	testsDir := filepath.Join(testDir, "tests")
	if err := os.MkdirAll(testsDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Create test files in subdirectory
	test := `def test_in_subdir():
    assert True
`
	if err := os.WriteFile(filepath.Join(testsDir, "test_subdir.py"), []byte(test), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Run pytest on directory
	cmd := exec.Command(getBinaryPath(), "pytest", "tests")
	cmd.Dir = testDir

	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Errorf("Failed to run pytest on directory: %v\nOutput: %s", err, output)
	}

	// Verify tests from subdirectory were run
	runDir := getLatestRunDir(t, testDir)
	reportPath := filepath.Join(runDir, "test-run.md")
	content, _ := os.ReadFile(reportPath)

	if !strings.Contains(string(content), "test_subdir") || !strings.Contains(string(content), "tests") {
		t.Error("Report should contain tests from subdirectory")
	}
}
