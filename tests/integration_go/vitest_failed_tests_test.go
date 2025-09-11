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

	// Should show failed test file
	if !strings.Contains(output, "FAIL") {
		t.Error("Should show failed test file")
	}

	// Should show individual failed test names (this is what we're testing)
	if !strings.Contains(output, "✕ should fail this test") {
		t.Error("Should show individual failed test names")
	}

	// Verify the console output shows the specific test failure
	if !strings.Contains(output, "should fail this test") {
		t.Errorf("Console output should show specific failed test name. Output:\n%s", output)
	}
}