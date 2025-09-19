package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zk/3pio/tests/testutil"
)

func TestEmptyMochaSuite(t *testing.T) {
	if _, err := testutil.LookPath("npm"); err != nil {
		t.Skip("npm not found in PATH")
	}
	if err := testutil.CommandAvailable("npx", "mocha", "--version"); err != nil {
		t.Skipf("mocha command failed: %v", err)
	}

	fixtureDir := filepath.Join("..", "fixtures", "empty-mocha")
	if _, err := os.Stat(fixtureDir); os.IsNotExist(err) {
		t.Skipf("fixture %s not found", fixtureDir)
	}

	// Run a single empty spec
	_ = testutil.RunThreepio(t, fixtureDir, []string{"npx", "mocha", "empty.spec.js"}...)

	runsDir := filepath.Join(fixtureDir, ".3pio", "runs")
	if _, err := os.Stat(runsDir); os.IsNotExist(err) {
		t.Error("Empty mocha suite should create .3pio/runs directory")
	}
}
