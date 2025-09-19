package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zk/3pio/tests/testutil"
)

func TestMochaFullFlow(t *testing.T) {
	// Ensure npm/npx and mocha are available
	if _, err := testutil.LookPath("npm"); err != nil {
		t.Skip("npm not found in PATH")
	}
	if err := testutil.CommandAvailable("npx", "mocha", "--version"); err != nil {
		t.Skipf("mocha command failed: %v", err)
	}

	fixtureDir := filepath.Join("..", "fixtures", "basic-mocha")
	if _, err := os.Stat(fixtureDir); os.IsNotExist(err) {
		t.Skipf("fixture %s not found", fixtureDir)
	}

	// Run 3pio with npx mocha against both spec files
	result := testutil.RunThreepio(t, fixtureDir, []string{"npx", "mocha", "math.spec.js", "string.spec.js"}...)

	if len(result.Stdout) == 0 && len(result.Stderr) == 0 {
		t.Error("Expected some output from test run")
	}

	runDir := filepath.Join(fixtureDir, ".3pio", "runs", result.RunID)

	// Verify required files
	for _, f := range []string{"test-run.md", "output.log"} {
		if _, err := os.Stat(filepath.Join(runDir, f)); os.IsNotExist(err) {
			t.Fatalf("expected %s to exist", f)
		}
	}

	// Verify reports directory exists and contains at least one index.md
	reportsDir := filepath.Join(runDir, "reports")
	if _, err := os.Stat(reportsDir); os.IsNotExist(err) {
		t.Fatal("Expected reports directory to exist")
	}

	// Verify test-run content
	testRunContent, err := os.ReadFile(filepath.Join(runDir, "test-run.md"))
	if err != nil {
		t.Fatalf("Failed to read test-run.md: %v", err)
	}

	// Should include both spec names
	for _, s := range []string{"math.spec.js", "string.spec.js"} {
		if !strings.Contains(string(testRunContent), s) {
			t.Errorf("test-run.md should contain '%s'", s)
		}
	}

	// Should mark as completed and detect mocha
	if !strings.Contains(string(testRunContent), "status: COMPLETED") {
		t.Error("Test run should complete successfully")
	}
	if !strings.Contains(string(testRunContent), "detected_runner: mocha") {
		t.Error("Should detect mocha runner")
	}
}
