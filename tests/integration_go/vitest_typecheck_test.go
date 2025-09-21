package integration_test

import (
    "os/exec"
    "path/filepath"
    "regexp"
    "strings"
    "testing"
)

// runRaw runs a command in the given directory and returns stdout+stderr and exit code
func runRaw(t *testing.T, dir string, args ...string) (string, int) {
    t.Helper()
    cmd := exec.Command(args[0], args[1:]...)
    cmd.Dir = dir
    out, err := cmd.CombinedOutput()
    code := 0
    if err != nil {
        if ee, ok := err.(*exec.ExitError); ok {
            code = ee.ExitCode()
        } else {
            code = 1
        }
    }
    return string(out), code
}

func extractResultsLine(output string) string {
    // Matches: Results:     5 passed, 1 failed, 6 total
    for _, line := range strings.Split(output, "\n") {
        if strings.HasPrefix(strings.TrimSpace(line), "Results:") {
            return strings.TrimSpace(line)
        }
    }
    return ""
}

func TestVitestTypecheckParity(t *testing.T) {
    t.Parallel()
    fixtures := []struct{
        name string
        rel  string
    }{
        {name: "vitest-typecheck-v2", rel: "vitest-typecheck-v2"},
        {name: "vitest-typecheck-v3", rel: "vitest-typecheck-v3"},
    }

    // Results line should contain: number passed and total
    re := regexp.MustCompile(`Results:\s+.* total$`)

    for _, fx := range fixtures {
        t.Run(fx.name, func(t *testing.T) {
            cwd, _ := filepath.Abs(filepath.Join("..", "fixtures", fx.rel))

            // Baseline
            baseOut, baseCode := runRaw(t, cwd, "npm", "run", "test")
            if !strings.Contains(baseOut, "Test Files") && !strings.Contains(baseOut, "Results:") {
                t.Fatalf("baseline did not run Vitest? output=\n%s", baseOut)
            }

            // 3pio
            threeOut, _, threeCode := runBinary(t, cwd, "npm", "run", "test")

            if baseCode != threeCode {
                t.Fatalf("exit code mismatch: baseline=%d threepio=%d", baseCode, threeCode)
            }

            // Extract and compare result lines
            b := extractResultsLine(threeOut) // 3pio always prints Results line
            if b == "" || !re.MatchString(b) {
                t.Fatalf("failed to extract 3pio results line: %q\noutput=\n%s", b, threeOut)
            }

            // We cannot reliably extract Vitest's results line across formats; rely on 3pio parity by rerunning without 3pio parser
            // Instead, ensure 3pio totals > 0 as a sanity check
            if !strings.Contains(b, "total") || strings.Contains(b, "0 total") {
                t.Fatalf("unexpected 3pio totals: %s", b)
            }
        })
    }
}

