package integration_test

import (
    "os"
    "path/filepath"
    "strings"
    "testing"

    "github.com/zk/3pio/tests/testutil"
)

// Full run on complex fixture: expect a failure (failing.cy.js) and multiple reports
func TestCypressFullRun_AllSpecs_Fails(t *testing.T) {
    if _, err := testutil.LookPath("npm"); err != nil { t.Skip("npm not found in PATH") }
    if err := testutil.CommandAvailable("npx", "cypress", "--version"); err != nil { t.Skipf("cypress not available: %v", err) }

    fixtureDir := filepath.Join("..", "fixtures", "cypress-complex")
    if _, err := os.Stat(fixtureDir); os.IsNotExist(err) { t.Skip("cypress fixture not found") }
    testutil.CleanupTestRuns(t, fixtureDir)

    // Run all specs (no --spec)
    result := testutil.RunThreepio(t, fixtureDir, "npx", "cypress", "run", "--headless")

    if result.ExitCode == 0 {
        t.Fatalf("expected non-zero exit due to failing spec, got %d", result.ExitCode)
    }

    runDir := filepath.Join(fixtureDir, ".3pio", "runs", result.RunID)
    // Verify test-run.md
    tr := filepath.Join(runDir, "test-run.md")
    data, err := os.ReadFile(tr)
    if err != nil { t.Fatalf("failed to read test-run.md: %v", err) }
    if !strings.Contains(string(data), "detected_runner: cypress") { t.Error("expected detected_runner: cypress") }
    if !strings.Contains(string(data), "FAIL") { t.Error("expected failure recorded in report") }

    // Verify multiple report directories exist
    reports := filepath.Join(runDir, "reports")
    entries, err := os.ReadDir(reports)
    if err != nil { t.Fatalf("failed to read reports dir: %v", err) }
    if len(entries) < 2 { t.Errorf("expected at least 2 report dirs, got %d", len(entries)) }
}

// Pattern matching with --spec glob
func TestCypressPatternMatching(t *testing.T) {
    if _, err := testutil.LookPath("npm"); err != nil { t.Skip("npm not found in PATH") }
    if err := testutil.CommandAvailable("npx", "cypress", "--version"); err != nil { t.Skipf("cypress not available: %v", err) }

    fixtureDir := filepath.Join("..", "fixtures", "cypress-complex")
    if _, err := os.Stat(fixtureDir); os.IsNotExist(err) { t.Skip("cypress fixture not found") }
    testutil.CleanupTestRuns(t, fixtureDir)

    // Run only nested strings specs
    result := testutil.RunThreepio(t, fixtureDir, "npx", "cypress", "run", "--headless", "--spec", "cypress/e2e/strings/*.cy.js")
    if result.ExitCode != 0 {
        t.Fatalf("expected success for pattern-only run, got exit %d\nStdout: %s", result.ExitCode, result.Stdout)
    }

    runDir := filepath.Join(fixtureDir, ".3pio", "runs", result.RunID)
    // Verify nested_string report exists
    // We don't know sanitized path entirely, but expect at least one report dir
    reports := filepath.Join(runDir, "reports")
    entries, err := os.ReadDir(reports)
    if err != nil { t.Fatalf("failed to read reports dir: %v", err) }
    if len(entries) == 0 { t.Fatal("expected at least one report dir for pattern run") }
}

// Missing spec handling: ensure non-zero exit and .3pio is still created
func TestCypressMissingSpecHandling(t *testing.T) {
    if _, err := testutil.LookPath("npm"); err != nil { t.Skip("npm not found in PATH") }
    if err := testutil.CommandAvailable("npx", "cypress", "--version"); err != nil { t.Skipf("cypress not available: %v", err) }

    fixtureDir := filepath.Join("..", "fixtures", "basic-cypress")
    if _, err := os.Stat(fixtureDir); os.IsNotExist(err) { t.Skip("cypress fixture not found") }
    testutil.CleanupTestRuns(t, fixtureDir)

    // Run with non-existent spec
    result := testutil.RunThreepio(t, fixtureDir, "npx", "cypress", "run", "--headless", "--spec", "cypress/e2e/does-not-exist.cy.js")
    if result.ExitCode == 0 { t.Fatalf("expected non-zero exit for missing spec") }

    runsDir := filepath.Join(fixtureDir, ".3pio", "runs")
    if _, err := os.Stat(runsDir); os.IsNotExist(err) {
        t.Error("expected .3pio/runs to exist even on error")
    }
}

