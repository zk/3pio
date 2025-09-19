package integration_test

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func TestNpmSeparatorHandling_Mocha(t *testing.T) {
	projectDir := filepath.Join(fixturesDir, "npm-separator-mocha")

	// Clean output directory
	if err := cleanProjectOutput(projectDir); err != nil {
		t.Fatalf("Failed to clean project output: %v", err)
	}

	// Get absolute path to binary
	binaryPath, err := filepath.Abs(threePioBinary)
	if err != nil {
		t.Fatalf("Failed to get absolute path to binary: %v", err)
	}

	// Run 3pio with npm test -- format (do not assert exit code)
	cmd := exec.Command(binaryPath, "npm", "test", "--", "example.spec.js")
	cmd.Dir = projectDir
	_, _ = cmd.CombinedOutput()

	// Verify that files were created
	runDir := getLatestRunDir(t, projectDir)
	if runDir == "" {
		t.Fatal("expected a run directory to be created for mocha npm test")
	}
}
