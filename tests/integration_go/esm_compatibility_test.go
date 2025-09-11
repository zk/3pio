package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestESModuleJestCompatibility verifies that 3pio works with ES module Jest projects
func TestESModuleJestCompatibility(t *testing.T) {
	t.Parallel()

	// Use the jest-esm fixture with absolute path
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	fixtureDir := filepath.Join(filepath.Dir(filepath.Dir(cwd)), "tests", "fixtures", "jest-esm")

	// Clean up any previous runs
	_ = os.RemoveAll(filepath.Join(fixtureDir, ".3pio"))

	// Run 3pio with Jest on ES module project
	output, _, exitCode := runBinary(t, fixtureDir, "npx", "jest")

	// Should succeed (not fail with module loading errors)
	if exitCode != 0 {
		t.Errorf("Expected success but got exit code %d. Output:\n%s", exitCode, output)
	}

	// Should not contain ES module compatibility errors
	if strings.Contains(output, "module is not defined") {
		t.Error("Should not have ES module compatibility errors")
	}

	// Should have proper test execution
	if !strings.Contains(output, "PASS") {
		t.Error("Should show passing tests")
	}

	// Check the generated report
	runDir := findLatestRunDir(t, fixtureDir)
	reportPath := filepath.Join(runDir, "test-run.md")

	reportContent, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("Failed to read test-run.md: %v", err)
	}

	report := string(reportContent)

	// Should have COMPLETE status (not ERROR)
	if !strings.Contains(report, "**Status:** COMPLETE") {
		t.Error("Report should show COMPLETE status for successful ES module tests")
	}

	// Should include Summary section
	if !strings.Contains(report, "## Summary") {
		t.Error("Report should include Summary section for successful tests")
	}

	// Should not have Error Details section
	if strings.Contains(report, "## Error Details") {
		t.Error("Report should not include Error Details section for successful tests")
	}
}
