package integration_test

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestJestCoverageRejection verifies that jest --coverage is rejected
func TestJestCoverageRejection(t *testing.T) {
	testDir := filepath.Join(fixturesDir, "basic-jest")
	cleanTestDir(t, testDir)

	// Try to run jest with --coverage
	cmd := exec.Command(getBinaryPath(), "npx", "jest", "--coverage")
	cmd.Dir = testDir

	output, err := cmd.CombinedOutput()

	// Should fail with non-zero exit code
	if err == nil {
		t.Error("Expected jest --coverage to be rejected, but it succeeded")
	}

	// Check for clear error message
	outputStr := string(output)
	if !strings.Contains(outputStr, "coverage") || !strings.Contains(outputStr, "not supported") {
		t.Errorf("Expected clear error message about coverage not being supported, got: %s", outputStr)
	}

	// Verify exit code is non-zero
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() == 0 {
			t.Error("Expected non-zero exit code for coverage mode rejection")
		}
	}

	// Verify no reports were created
	assertNoFile(t, filepath.Join(testDir, ".3pio", "runs"))
}

// TestJestCollectCoverageRejection verifies that jest --collectCoverage is rejected
func TestJestCollectCoverageRejection(t *testing.T) {
	testDir := filepath.Join(fixturesDir, "basic-jest")
	cleanTestDir(t, testDir)

	// Try to run jest with --collectCoverage
	cmd := exec.Command(getBinaryPath(), "npx", "jest", "--collectCoverage")
	cmd.Dir = testDir

	output, err := cmd.CombinedOutput()

	// Should fail with non-zero exit code
	if err == nil {
		t.Error("Expected jest --collectCoverage to be rejected, but it succeeded")
	}

	// Check for clear error message
	outputStr := string(output)
	if !strings.Contains(outputStr, "coverage") || !strings.Contains(outputStr, "not supported") {
		t.Errorf("Expected clear error message about coverage not being supported, got: %s", outputStr)
	}
}

// TestVitestCoverageRejection verifies that vitest --coverage is rejected
func TestVitestCoverageRejection(t *testing.T) {
	testDir := filepath.Join(fixturesDir, "basic-vitest")
	cleanTestDir(t, testDir)

	// Try to run vitest with --coverage
	cmd := exec.Command(getBinaryPath(), "npx", "vitest", "run", "--coverage")
	cmd.Dir = testDir

	output, err := cmd.CombinedOutput()

	// Should fail with non-zero exit code
	if err == nil {
		t.Error("Expected vitest --coverage to be rejected, but it succeeded")
	}

	// Check for clear error message
	outputStr := string(output)
	if !strings.Contains(outputStr, "coverage") || !strings.Contains(outputStr, "not supported") {
		t.Errorf("Expected clear error message about coverage not being supported, got: %s", outputStr)
	}
}

// TestPytestCovRejection verifies that pytest --cov is rejected
func TestPytestCovRejection(t *testing.T) {
	testDir := filepath.Join(fixturesDir, "basic-pytest")
	cleanTestDir(t, testDir)

	// Try to run pytest with --cov
	cmd := exec.Command(getBinaryPath(), "pytest", "--cov")
	cmd.Dir = testDir

	output, err := cmd.CombinedOutput()

	// Should fail with non-zero exit code
	if err == nil {
		t.Error("Expected pytest --cov to be rejected, but it succeeded")
	}

	// Check for clear error message
	outputStr := string(output)
	if !strings.Contains(outputStr, "coverage") || !strings.Contains(outputStr, "not supported") {
		t.Errorf("Expected clear error message about coverage not being supported, got: %s", outputStr)
	}
}

// TestPytestCovReportRejection verifies that pytest --cov-report is rejected
func TestPytestCovReportRejection(t *testing.T) {
	testDir := filepath.Join(fixturesDir, "basic-pytest")
	cleanTestDir(t, testDir)

	// Try to run pytest with --cov-report
	cmd := exec.Command(getBinaryPath(), "pytest", "--cov-report=html")
	cmd.Dir = testDir

	output, err := cmd.CombinedOutput()

	// Should fail with non-zero exit code
	if err == nil {
		t.Error("Expected pytest --cov-report to be rejected, but it succeeded")
	}

	// Check for clear error message
	outputStr := string(output)
	if !strings.Contains(outputStr, "coverage") || !strings.Contains(outputStr, "not supported") {
		t.Errorf("Expected clear error message about coverage not being supported, got: %s", outputStr)
	}
}

// TestCargoTarpaulinRejection verifies that cargo tarpaulin is rejected
func TestCargoTarpaulinRejection(t *testing.T) {
	testDir := filepath.Join(fixturesDir, "rust-basic")
	cleanTestDir(t, testDir)

	// Try to run cargo tarpaulin
	cmd := exec.Command(getBinaryPath(), "cargo", "tarpaulin")
	cmd.Dir = testDir

	output, err := cmd.CombinedOutput()

	// Should fail with non-zero exit code
	if err == nil {
		t.Error("Expected cargo tarpaulin to be rejected, but it succeeded")
	}

	// Check for clear error message
	outputStr := string(output)
	if !strings.Contains(outputStr, "coverage") || !strings.Contains(outputStr, "not supported") {
		t.Errorf("Expected clear error message about coverage not being supported, got: %s", outputStr)
	}
}

// TestCargoLlvmCovRejection verifies that cargo llvm-cov is rejected
func TestCargoLlvmCovRejection(t *testing.T) {
	testDir := filepath.Join(fixturesDir, "rust-basic")
	cleanTestDir(t, testDir)

	// Try to run cargo llvm-cov
	cmd := exec.Command(getBinaryPath(), "cargo", "llvm-cov")
	cmd.Dir = testDir

	output, err := cmd.CombinedOutput()

	// Should fail with non-zero exit code
	if err == nil {
		t.Error("Expected cargo llvm-cov to be rejected, but it succeeded")
	}

	// Check for clear error message
	outputStr := string(output)
	if !strings.Contains(outputStr, "coverage") || !strings.Contains(outputStr, "not supported") {
		t.Errorf("Expected clear error message about coverage not being supported, got: %s", outputStr)
	}
}

// TestNpmTestCoverageRejection verifies that npm test -- --coverage is rejected
func TestNpmTestCoverageRejection(t *testing.T) {
	testDir := filepath.Join(fixturesDir, "basic-jest")
	cleanTestDir(t, testDir)

	// Try to run npm test with coverage flag
	cmd := exec.Command(getBinaryPath(), "npm", "test", "--", "--coverage")
	cmd.Dir = testDir

	output, err := cmd.CombinedOutput()

	// Should fail with non-zero exit code
	if err == nil {
		t.Error("Expected npm test -- --coverage to be rejected, but it succeeded")
	}

	// Check for clear error message
	outputStr := string(output)
	if !strings.Contains(outputStr, "coverage") || !strings.Contains(outputStr, "not supported") {
		t.Errorf("Expected clear error message about coverage not being supported, got: %s", outputStr)
	}
}

// TestYarnTestCoverageRejection verifies that yarn test --coverage is rejected
func TestYarnTestCoverageRejection(t *testing.T) {
	testDir := filepath.Join(fixturesDir, "basic-jest")
	cleanTestDir(t, testDir)

	// Try to run yarn test with coverage flag
	cmd := exec.Command(getBinaryPath(), "yarn", "test", "--coverage")
	cmd.Dir = testDir

	output, err := cmd.CombinedOutput()

	// Should fail with non-zero exit code
	if err == nil {
		t.Error("Expected yarn test --coverage to be rejected, but it succeeded")
	}

	// Check for clear error message
	outputStr := string(output)
	if !strings.Contains(outputStr, "coverage") || !strings.Contains(outputStr, "not supported") {
		t.Errorf("Expected clear error message about coverage not being supported, got: %s", outputStr)
	}
}

// TestCoverageErrorMessage verifies the error message is clear and helpful
func TestCoverageErrorMessage(t *testing.T) {
	testDir := filepath.Join(fixturesDir, "basic-jest")
	cleanTestDir(t, testDir)

	// Try to run jest with --coverage
	cmd := exec.Command(getBinaryPath(), "npx", "jest", "--coverage")
	cmd.Dir = testDir

	output, err := cmd.CombinedOutput()

	if err == nil {
		t.Fatal("Expected coverage mode to be rejected")
	}

	outputStr := string(output)

	// Check for helpful error message components
	requiredPhrases := []string{
		"coverage",
		"not supported",
	}

	for _, phrase := range requiredPhrases {
		if !strings.Contains(strings.ToLower(outputStr), phrase) {
			t.Errorf("Error message missing required phrase '%s'. Got: %s", phrase, outputStr)
		}
	}

	// Verify it suggests running without coverage
	if !strings.Contains(outputStr, "without") || !strings.Contains(outputStr, "coverage") {
		t.Error("Error message should suggest running tests without coverage")
	}
}

// TestNycCoverageRejection verifies that nyc (coverage tool) is rejected
func TestNycCoverageRejection(t *testing.T) {
	testDir := filepath.Join(fixturesDir, "basic-jest")
	cleanTestDir(t, testDir)

	// Try to run tests with nyc
	cmd := exec.Command(getBinaryPath(), "nyc", "jest")
	cmd.Dir = testDir

	output, err := cmd.CombinedOutput()

	// Should fail with non-zero exit code
	if err == nil {
		t.Error("Expected nyc jest to be rejected, but it succeeded")
	}

	// Check for clear error message
	outputStr := string(output)
	if !strings.Contains(outputStr, "coverage") || !strings.Contains(outputStr, "not supported") {
		t.Errorf("Expected clear error message about coverage not being supported, got: %s", outputStr)
	}
}

// TestC8CoverageRejection verifies that c8 (V8 coverage) is rejected
func TestC8CoverageRejection(t *testing.T) {
	testDir := filepath.Join(fixturesDir, "basic-vitest")
	cleanTestDir(t, testDir)

	// Try to run tests with c8
	cmd := exec.Command(getBinaryPath(), "c8", "vitest", "run")
	cmd.Dir = testDir

	output, err := cmd.CombinedOutput()

	// Should fail with non-zero exit code
	if err == nil {
		t.Error("Expected c8 vitest to be rejected, but it succeeded")
	}

	// Check for clear error message
	outputStr := string(output)
	if !strings.Contains(outputStr, "coverage") || !strings.Contains(outputStr, "not supported") {
		t.Errorf("Expected clear error message about coverage not being supported, got: %s", outputStr)
	}
}