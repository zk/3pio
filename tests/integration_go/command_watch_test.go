package integration_test

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestJestWatchModeRejection verifies that jest --watch is rejected
func TestJestWatchModeRejection(t *testing.T) {
	testDir := filepath.Join(fixturesDir, "basic-jest")
	cleanTestDir(t, testDir)

	// Try to run jest with --watch
	cmd := exec.Command(getBinaryPath(), "npx", "jest", "--watch")
	cmd.Dir = testDir

	output, err := cmd.CombinedOutput()

	// Should fail with non-zero exit code
	if err == nil {
		t.Error("Expected jest --watch to be rejected, but it succeeded")
	}

	// Check for clear error message
	outputStr := string(output)
	if !strings.Contains(outputStr, "watch") || !strings.Contains(outputStr, "not supported") {
		t.Errorf("Expected clear error message about watch mode not being supported, got: %s", outputStr)
	}

	// Verify exit code is non-zero
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() == 0 {
			t.Error("Expected non-zero exit code for watch mode rejection")
		}
	}

	// Verify no reports were created
	assertNoFile(t, filepath.Join(testDir, ".3pio", "runs"))
}

// TestJestWatchAllModeRejection verifies that jest --watchAll is rejected
func TestJestWatchAllModeRejection(t *testing.T) {
	testDir := filepath.Join(fixturesDir, "basic-jest")
	cleanTestDir(t, testDir)

	// Try to run jest with --watchAll
	cmd := exec.Command(getBinaryPath(), "npx", "jest", "--watchAll")
	cmd.Dir = testDir

	output, err := cmd.CombinedOutput()

	// Should fail with non-zero exit code
	if err == nil {
		t.Error("Expected jest --watchAll to be rejected, but it succeeded")
	}

	// Check for clear error message
	outputStr := string(output)
	if !strings.Contains(outputStr, "watch") || !strings.Contains(outputStr, "not supported") {
		t.Errorf("Expected clear error message about watch mode not being supported, got: %s", outputStr)
	}
}

// TestVitestWatchModeRejection verifies that vitest --watch is rejected
func TestVitestWatchModeRejection(t *testing.T) {
	testDir := filepath.Join(fixturesDir, "basic-vitest")
	cleanTestDir(t, testDir)

	// Try to run vitest with --watch
	cmd := exec.Command(getBinaryPath(), "npx", "vitest", "--watch")
	cmd.Dir = testDir

	output, err := cmd.CombinedOutput()

	// Should fail with non-zero exit code
	if err == nil {
		t.Error("Expected vitest --watch to be rejected, but it succeeded")
	}

	// Check for clear error message
	outputStr := string(output)
	if !strings.Contains(outputStr, "watch") || !strings.Contains(outputStr, "not supported") {
		t.Errorf("Expected clear error message about watch mode not being supported, got: %s", outputStr)
	}
}

// TestVitestDefaultWatchModeRejection verifies that plain 'vitest' (which defaults to watch) is rejected
func TestVitestDefaultWatchModeRejection(t *testing.T) {
	testDir := filepath.Join(fixturesDir, "basic-vitest")
	cleanTestDir(t, testDir)

	// Try to run vitest without 'run' (defaults to watch mode)
	cmd := exec.Command(getBinaryPath(), "npx", "vitest")
	cmd.Dir = testDir

	output, err := cmd.CombinedOutput()

	// Should fail with non-zero exit code
	if err == nil {
		t.Error("Expected plain vitest (watch mode default) to be rejected, but it succeeded")
	}

	// Check for clear error message
	outputStr := string(output)
	if !strings.Contains(outputStr, "watch") || !strings.Contains(outputStr, "not supported") {
		t.Errorf("Expected clear error message about watch mode not being supported, got: %s", outputStr)
	}
}

// TestPytestWatchModeRejection verifies that pytest-watch is rejected
func TestPytestWatchModeRejection(t *testing.T) {
	testDir := filepath.Join(fixturesDir, "basic-pytest")
	cleanTestDir(t, testDir)

	// Try to run pytest-watch
	cmd := exec.Command(getBinaryPath(), "pytest-watch")
	cmd.Dir = testDir

	output, err := cmd.CombinedOutput()

	// Should fail with non-zero exit code
	if err == nil {
		t.Error("Expected pytest-watch to be rejected, but it succeeded")
	}

	// Check for clear error message
	outputStr := string(output)
	if !strings.Contains(outputStr, "watch") || !strings.Contains(outputStr, "not supported") {
		t.Errorf("Expected clear error message about watch mode not being supported, got: %s", outputStr)
	}
}

// TestPtestWatchModeRejection verifies that ptw (pytest-watch alias) is rejected
func TestPtestWatchAliasRejection(t *testing.T) {
	testDir := filepath.Join(fixturesDir, "basic-pytest")
	cleanTestDir(t, testDir)

	// Try to run ptw (pytest-watch alias)
	cmd := exec.Command(getBinaryPath(), "ptw")
	cmd.Dir = testDir

	output, err := cmd.CombinedOutput()

	// Should fail with non-zero exit code
	if err == nil {
		t.Error("Expected ptw to be rejected, but it succeeded")
	}

	// Check for clear error message
	outputStr := string(output)
	if !strings.Contains(outputStr, "watch") || !strings.Contains(outputStr, "not supported") {
		t.Errorf("Expected clear error message about watch mode not being supported, got: %s", outputStr)
	}
}

// TestNpmTestWatchModeRejection verifies that npm test -- --watch is rejected
func TestNpmTestWatchModeRejection(t *testing.T) {
	testDir := filepath.Join(fixturesDir, "basic-jest")
	cleanTestDir(t, testDir)

	// Try to run npm test with watch flag
	cmd := exec.Command(getBinaryPath(), "npm", "test", "--", "--watch")
	cmd.Dir = testDir

	output, err := cmd.CombinedOutput()

	// Should fail with non-zero exit code
	if err == nil {
		t.Error("Expected npm test -- --watch to be rejected, but it succeeded")
	}

	// Check for clear error message
	outputStr := string(output)
	if !strings.Contains(outputStr, "watch") || !strings.Contains(outputStr, "not supported") {
		t.Errorf("Expected clear error message about watch mode not being supported, got: %s", outputStr)
	}
}

// TestYarnTestWatchModeRejection verifies that yarn test --watch is rejected
func TestYarnTestWatchModeRejection(t *testing.T) {
	testDir := filepath.Join(fixturesDir, "basic-jest")
	cleanTestDir(t, testDir)

	// Try to run yarn test with watch flag
	cmd := exec.Command(getBinaryPath(), "yarn", "test", "--watch")
	cmd.Dir = testDir

	output, err := cmd.CombinedOutput()

	// Should fail with non-zero exit code
	if err == nil {
		t.Error("Expected yarn test --watch to be rejected, but it succeeded")
	}

	// Check for clear error message
	outputStr := string(output)
	if !strings.Contains(outputStr, "watch") || !strings.Contains(outputStr, "not supported") {
		t.Errorf("Expected clear error message about watch mode not being supported, got: %s", outputStr)
	}
}

// TestWatchModeErrorMessage verifies the error message is clear and helpful
func TestWatchModeErrorMessage(t *testing.T) {
	testDir := filepath.Join(fixturesDir, "basic-jest")
	cleanTestDir(t, testDir)

	// Try to run jest with --watch
	cmd := exec.Command(getBinaryPath(), "npx", "jest", "--watch")
	cmd.Dir = testDir

	output, err := cmd.CombinedOutput()

	if err == nil {
		t.Fatal("Expected watch mode to be rejected")
	}

	outputStr := string(output)

	// Check for helpful error message components
	requiredPhrases := []string{
		"watch",
		"not supported",
	}

	for _, phrase := range requiredPhrases {
		if !strings.Contains(strings.ToLower(outputStr), phrase) {
			t.Errorf("Error message missing required phrase '%s'. Got: %s", phrase, outputStr)
		}
	}

	// Verify it suggests using run mode
	if strings.Contains(outputStr, "vitest") && !strings.Contains(outputStr, "vitest run") {
		t.Error("Error message for vitest should suggest using 'vitest run'")
	}
}