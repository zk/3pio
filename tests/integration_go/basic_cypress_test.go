package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zk/3pio/tests/testutil"
)

// TestCypressIntegration ensures 3pio can run a basic Cypress project when Cypress is available.
func TestCypressIntegration(t *testing.T) {
	t.Parallel()

	// Skip if npm/npx or cypress are not available
	if _, err := testutil.LookPath("npm"); err != nil {
		t.Skip("npm not found in PATH")
	}
	if err := testutil.CommandAvailable("npx", "cypress", "--version"); err != nil {
		t.Skipf("cypress not available: %v", err)
	}

	fixtureDir := filepath.Join("..", "fixtures", "basic-cypress")
	if _, err := os.Stat(fixtureDir); os.IsNotExist(err) {
		t.Skip("cypress fixture not found")
	}

	// Clean .3pio from previous runs
	testutil.CleanupTestRuns(t, fixtureDir)

	// Run Cypress headless on a single spec to keep it quick
	result := testutil.RunThreepio(t, fixtureDir, "npx", "cypress", "run", "--headless", "--spec", "cypress/e2e/sample.cy.js")

	// A passing spec should exit 0
	if result.ExitCode != 0 {
		t.Fatalf("expected success, got exit code %d. Output: %s", result.ExitCode, result.Stdout)
	}

	// Verify report exists
	reportPath := filepath.Join(fixtureDir, ".3pio", "runs", result.RunID, "test-run.md")
	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		t.Fatalf("report not found: %s", reportPath)
	}

	// Verify run marked completed
	content, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("failed to read report: %v", err)
	}
	if !strings.Contains(string(content), "detected_runner: cypress") || !strings.Contains(string(content), "status: COMPLETED") {
		t.Fatalf("report missing expected fields.\n%s", string(content))
	}
}
