package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestVitestFailedTestsReporting verifies that Vitest adapter reports individual failed tests
func TestVitestFailedTestsReporting(t *testing.T) {
	t.Parallel()

	// Use the basic-vitest fixture which has a failing test
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	fixtureDir := filepath.Join(filepath.Dir(filepath.Dir(cwd)), "tests", "fixtures", "basic-vitest")

	// Clean up any previous runs
	_ = os.RemoveAll(filepath.Join(fixtureDir, ".3pio"))

	// Run 3pio with Vitest on fixture with failing tests
	output, _, exitCode := runBinary(t, fixtureDir, "npx", "vitest", "run")

	// Should fail due to failing tests
	if exitCode == 0 {
		t.Error("Expected non-zero exit code for failing tests")
	}

    // Console output now shows a concise summary line per failing file.
    if !strings.Contains(output, "FAIL(") {
        t.Error("Should show a FAIL summary line")
    }
    if !strings.Contains(output, "$trun_dir/reports/") {
        t.Error("Should include report path for the failing file")
    }
}
