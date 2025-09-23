package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestVitestV2FailedTestsReporting verifies that Vitest v2 adapter path reports individual failed tests
func TestVitestV2FailedTestsReporting(t *testing.T) {
	t.Parallel()

	// Use the basic-vitest-v2 fixture which has a failing test
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	fixtureDir := filepath.Join(filepath.Dir(filepath.Dir(cwd)), "tests", "fixtures", "basic-vitest-v2")

	// Clean up any previous runs
	_ = os.RemoveAll(filepath.Join(fixtureDir, ".3pio"))

	// Run 3pio with Vitest on fixture with failing tests
	output, _, exitCode := runBinary(t, fixtureDir, "npx", "vitest", "run")

	// Should fail due to failing tests
	if exitCode == 0 {
		t.Error("Expected non-zero exit code for failing tests (v2)")
	}

	// Console output should show a concise summary line per failing file
	if !strings.Contains(output, "FAIL ") {
		t.Error("Should show a FAIL summary line (v2)")
	}
	if !strings.Contains(output, "$trun_dir/reports/") {
		t.Error("Should include report path for the failing file (v2)")
	}
}
