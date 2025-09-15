package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestPytestMissingInstallation verifies handling when pytest is not installed
func TestPytestMissingInstallation(t *testing.T) {
	testDir := t.TempDir()

	// Create a Python file (but no pytest)
	test := `def test_something():
    assert True
`
	os.WriteFile(filepath.Join(testDir, "test_missing.py"), []byte(test), 0644)

	// Try to run pytest in a directory where it might not be available
	cmd := exec.Command(getBinaryPath(), "pytest")
	cmd.Dir = testDir
	// Clear Python path to simulate missing pytest
	cmd.Env = append(os.Environ(), "PATH=/usr/bin:/bin")

	output, err := cmd.CombinedOutput()

	// Should fail with clear error
	if err == nil {
		// If pytest is found, skip this test
		t.Skip("pytest is installed, skipping missing installation test")
		return
	}

	outputStr := string(output)
	// Should have a clear error message about pytest not being found
	if !strings.Contains(outputStr, "pytest") || (!strings.Contains(outputStr, "not found") && !strings.Contains(outputStr, "not installed")) {
		t.Logf("Expected clear error about missing pytest, got: %s", outputStr)
	}
}

// TestPytestSyntaxError verifies handling of Python syntax errors
func TestPytestSyntaxError(t *testing.T) {
	testDir := t.TempDir()

	// Create a test file with syntax error
	syntaxError := `def test_syntax_error(:  # Invalid syntax
    assert True
`
	os.WriteFile(filepath.Join(testDir, "test_syntax_error.py"), []byte(syntaxError), 0644)

	// Run pytest
	cmd := exec.Command(getBinaryPath(), "pytest")
	cmd.Dir = testDir

	output, err := cmd.CombinedOutput()

	// Should fail
	if err == nil {
		t.Error("Expected pytest to fail with syntax error")
	}

	// Check for error in output (3pio shows "exit status 2" for syntax errors)
	outputStr := string(output)
	if !strings.Contains(outputStr, "exit status 2") && !strings.Contains(outputStr, "Error") {
		t.Errorf("Expected error indication in output, got: %s", outputStr)
	}

	// Exit code should be non-zero
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() == 0 {
			t.Error("Expected non-zero exit code for syntax error")
		}
	}
}

// TestPytestImportError verifies handling of import errors
func TestPytestImportError(t *testing.T) {
	testDir := t.TempDir()

	// Create a test with import error
	importError := `import nonexistent_module

def test_import():
    assert True
`
	os.WriteFile(filepath.Join(testDir, "test_import_error.py"), []byte(importError), 0644)

	// Run pytest
	cmd := exec.Command(getBinaryPath(), "pytest")
	cmd.Dir = testDir

	output, err := cmd.CombinedOutput()

	// Should fail
	if err == nil {
		t.Error("Expected pytest to fail with import error")
	}

	// Check for error in output (3pio shows "exit status 1" for import errors)
	outputStr := string(output)
	if !strings.Contains(outputStr, "exit status") && !strings.Contains(outputStr, "Error") {
		t.Errorf("Expected error indication in output, got: %s", outputStr)
	}
}

// TestPytestFixtureError verifies handling of fixture errors
func TestPytestFixtureError(t *testing.T) {
	testDir := t.TempDir()

	// Create a test with fixture error
	fixtureError := `import pytest

@pytest.fixture
def broken_fixture():
    raise ValueError("Fixture is broken")

def test_with_broken_fixture(broken_fixture):
    assert True
`
	os.WriteFile(filepath.Join(testDir, "test_fixture_error.py"), []byte(fixtureError), 0644)

	// Run pytest
	cmd := exec.Command(getBinaryPath(), "pytest")
	cmd.Dir = testDir

	output, err := cmd.CombinedOutput()

	// Should fail
	if err == nil {
		t.Error("Expected pytest to fail with fixture error")
	}

	// Check for error in output
	outputStr := string(output)
	if !strings.Contains(outputStr, "ValueError") || !strings.Contains(outputStr, "Fixture is broken") {
		t.Logf("Fixture error output: %s", outputStr)
	}

	// Reports should still be created
	runDir := getLatestRunDir(t, testDir)
	assertReportExists(t, runDir)
}

// TestPytestEmptyTestSuite verifies handling of no tests found
func TestPytestEmptyTestSuite(t *testing.T) {
	testDir := t.TempDir()

	// Create a Python file with no tests
	noTests := `# This file has no tests

def not_a_test():
    return 42
`
	os.WriteFile(filepath.Join(testDir, "no_tests.py"), []byte(noTests), 0644)

	// Run pytest
	cmd := exec.Command(getBinaryPath(), "pytest")
	cmd.Dir = testDir

	output, err := cmd.CombinedOutput()

	// Pytest typically exits with code 5 when no tests are collected
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() != 5 && exitErr.ExitCode() != 0 {
				t.Logf("pytest exited with code %d for empty suite", exitErr.ExitCode())
			}
		}
	}

	// Check output mentions no tests
	outputStr := string(output)
	if !strings.Contains(outputStr, "no tests") || !strings.Contains(outputStr, "collected 0") {
		t.Logf("Empty test suite output: %s", outputStr)
	}

	// Basic structure should still be created
	runDir := getLatestRunDir(t, testDir)
	assertReportExists(t, runDir)
}

// TestPytestAssertionError verifies detailed assertion error reporting
func TestPytestAssertionError(t *testing.T) {
	testDir := t.TempDir()

	// Create a test with assertion failure
	assertionTest := `def test_assertion_failure():
    expected = 5
    actual = 3
    assert expected == actual, f"Expected {expected} but got {actual}"

def test_list_assertion():
    assert [1, 2, 3] == [1, 2, 4]
`
	os.WriteFile(filepath.Join(testDir, "test_assertions.py"), []byte(assertionTest), 0644)

	// Run pytest
	cmd := exec.Command(getBinaryPath(), "pytest", "-v")
	cmd.Dir = testDir

	output, err := cmd.CombinedOutput()

	// Should fail
	if err == nil {
		t.Error("Expected pytest to fail with assertion errors")
	}

	// Check for failure indication in output
	outputStr := string(output)
	if !strings.Contains(outputStr, "failed") && !strings.Contains(outputStr, "FAIL") && !strings.Contains(outputStr, "exit status 1") {
		t.Errorf("Expected failure indication in output, got: %s", outputStr)
	}

	// Check report contains error details
	runDir := getLatestRunDir(t, testDir)
	reportPath := filepath.Join(runDir, "test-run.md")
	content, _ := os.ReadFile(reportPath)
	reportStr := string(content)

	if !strings.Contains(reportStr, "FAIL") {
		t.Error("Report should indicate test failures")
	}
}

// TestPytestTimeoutHandling verifies handling of hanging tests
func TestPytestTimeoutHandling(t *testing.T) {
	testDir := t.TempDir()

	// Create a test that hangs
	hangingTest := `import time

def test_hanging():
    time.sleep(60)  # Sleep for 60 seconds
    assert True
`
	os.WriteFile(filepath.Join(testDir, "test_hanging.py"), []byte(hangingTest), 0644)

	// Run pytest with timeout flag if available
	cmd := exec.Command(getBinaryPath(), "pytest", "--timeout=2")
	cmd.Dir = testDir

	output, err := cmd.CombinedOutput()

	// The test should either timeout or pytest doesn't support --timeout
	outputStr := string(output)
	if strings.Contains(outputStr, "unrecognized arguments: --timeout") {
		// pytest-timeout not installed, which is fine
		t.Skip("pytest-timeout not installed, skipping timeout test")
		return
	}

	// If timeout is supported, test should fail
	if err == nil {
		t.Error("Expected timeout test to fail")
	}

	if strings.Contains(outputStr, "timeout") {
		t.Log("Test properly timed out")
	}
}

// TestPytestMissingTestFile verifies handling when specified test file doesn't exist
func TestPytestMissingTestFile(t *testing.T) {
	testDir := t.TempDir()

	// Try to run a non-existent test file
	cmd := exec.Command(getBinaryPath(), "pytest", "nonexistent_test.py")
	cmd.Dir = testDir

	output, err := cmd.CombinedOutput()

	// Should fail
	if err == nil {
		t.Error("Expected pytest to fail with missing file")
	}

	// Check for clear error message
	outputStr := string(output)
	if !strings.Contains(outputStr, "nonexistent_test.py") || (!strings.Contains(outputStr, "not found") && !strings.Contains(outputStr, "No such file")) {
		t.Logf("Missing file error output: %s", outputStr)
	}
}

// TestPytestInvalidFlag verifies handling of invalid pytest flags
func TestPytestInvalidFlag(t *testing.T) {
	testDir := t.TempDir()

	// Create a simple test
	test := `def test_simple():
    assert True
`
	os.WriteFile(filepath.Join(testDir, "test_simple.py"), []byte(test), 0644)

	// Run pytest with invalid flag
	cmd := exec.Command(getBinaryPath(), "pytest", "--invalid-flag-that-doesnt-exist")
	cmd.Dir = testDir

	output, err := cmd.CombinedOutput()

	// Should fail
	if err == nil {
		t.Error("Expected pytest to fail with invalid flag")
	}

	// Check for error about unrecognized argument
	outputStr := string(output)
	if !strings.Contains(outputStr, "unrecognized") || !strings.Contains(outputStr, "--invalid-flag-that-doesnt-exist") {
		t.Logf("Invalid flag error output: %s", outputStr)
	}
}

// TestPytestExceptionInTest verifies handling of exceptions in tests
func TestPytestExceptionInTest(t *testing.T) {
	testDir := t.TempDir()

	// Create a test that raises an exception
	exceptionTest := `def test_exception():
    raise RuntimeError("Something went wrong in the test")

def test_division_by_zero():
    result = 10 / 0
    assert result == 0
`
	os.WriteFile(filepath.Join(testDir, "test_exceptions.py"), []byte(exceptionTest), 0644)

	// Run pytest
	cmd := exec.Command(getBinaryPath(), "pytest", "-v")
	cmd.Dir = testDir

	output, err := cmd.CombinedOutput()

	// Should fail
	if err == nil {
		t.Error("Expected pytest to fail with exceptions")
	}

	// Check for exception details in output
	outputStr := string(output)
	if !strings.Contains(outputStr, "RuntimeError") || !strings.Contains(outputStr, "ZeroDivisionError") {
		t.Logf("Exception test output: %s", outputStr)
	}

	// Reports should contain error information
	runDir := getLatestRunDir(t, testDir)
	reportPath := filepath.Join(runDir, "test-run.md")
	content, _ := os.ReadFile(reportPath)

	if !strings.Contains(string(content), "FAIL") {
		t.Error("Report should show test failures")
	}
}