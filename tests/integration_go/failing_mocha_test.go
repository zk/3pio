package integration_test

import (
    "os"
    "path/filepath"
    "strings"
    "testing"

    "github.com/zk/3pio/tests/testutil"
)

func TestMochaFailureCase(t *testing.T) {
    if _, err := testutil.LookPath("npm"); err != nil {
        t.Skip("npm not found in PATH")
    }
    if err := testutil.CommandAvailable("npx", "mocha", "--version"); err != nil {
        t.Skipf("mocha command failed: %v", err)
    }

    fixtureDir := filepath.Join("..", "fixtures", "failing-mocha")
    if _, err := os.Stat(fixtureDir); os.IsNotExist(err) {
        t.Skipf("fixture %s not found", fixtureDir)
    }

    result := testutil.RunThreepio(t, fixtureDir, []string{"npx", "mocha", "failing.spec.js"}...)
    runDir := filepath.Join(fixtureDir, ".3pio", "runs", result.RunID)
    report := filepath.Join(runDir, "test-run.md")
    content, err := os.ReadFile(report)
    if err != nil {
        t.Fatalf("failed to read test-run.md: %v", err)
    }
    if !strings.Contains(string(content), "detected_runner: mocha") {
        t.Error("expected detected_runner: mocha in report")
    }
    if !strings.Contains(string(content), "fail") && !strings.Contains(strings.ToLower(string(content)), "fail") {
        t.Error("expected failure mentioned in report")
    }
}

