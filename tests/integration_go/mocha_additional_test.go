package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zk/3pio/tests/testutil"
)

// Pattern matching across all mocha specs in fixture
func TestMochaPatternMatching(t *testing.T) {
	if _, err := testutil.LookPath("npm"); err != nil {
		t.Skip("npm not found in PATH")
	}
	if err := testutil.CommandAvailable("npx", "mocha", "--version"); err != nil {
		t.Skipf("mocha command failed: %v", err)
	}

	fixtureDir := filepath.Join("..", "fixtures", "basic-mocha")
	if _, err := os.Stat(fixtureDir); os.IsNotExist(err) {
		t.Skip("fixture not found")
	}

	result := testutil.RunThreepio(t, fixtureDir, []string{"npx", "mocha", "**/*.spec.js"}...)
	if result.ExitCode != 0 {
		t.Fatalf("expected success running pattern, got %d", result.ExitCode)
	}

	runDir := filepath.Join(fixtureDir, ".3pio", "runs", result.RunID)
	reports := filepath.Join(runDir, "reports")
	entries, err := os.ReadDir(reports)
	if err != nil {
		t.Fatalf("failed to read reports dir: %v", err)
	}
	if len(entries) < 1 {
		t.Errorf("expected at least 1 report dir for pattern run, got %d", len(entries))
	}
}

// Missing file handling
func TestMochaMissingFileHandling(t *testing.T) {
	if _, err := testutil.LookPath("npm"); err != nil {
		t.Skip("npm not found in PATH")
	}
	if err := testutil.CommandAvailable("npx", "mocha", "--version"); err != nil {
		t.Skipf("mocha command failed: %v", err)
	}

	fixtureDir := filepath.Join("..", "fixtures", "basic-mocha")
	if _, err := os.Stat(fixtureDir); os.IsNotExist(err) {
		t.Skip("fixture not found")
	}

	result := testutil.RunThreepio(t, fixtureDir, []string{"npx", "mocha", "does_not_exist.spec.js"}...)
	if result.ExitCode == 0 {
		t.Fatalf("expected non-zero exit for missing file")
	}
	runsDir := filepath.Join(fixtureDir, ".3pio", "runs")
	if _, err := os.Stat(runsDir); os.IsNotExist(err) {
		t.Error("expected .3pio/runs to exist even on error")
	}
}
