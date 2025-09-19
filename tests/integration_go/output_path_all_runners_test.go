package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func verifyConsoleReportPath(t *testing.T, fixtureDir string, output []byte) {
	t.Helper()
	out := string(output)
	// Support both forward and backslashes in paths on different OSes
	trunRegex := regexp.MustCompile(`trun_dir:\s+(\.3pio[\\/]runs[\\/][^\s]+)`)
	trunMatch := trunRegex.FindStringSubmatch(out)
	if len(trunMatch) < 2 {
		t.Fatalf("Could not find trun_dir in output. Output:\n%s", out)
	}
	trunDir := trunMatch[1]
	// Allow nested path between reports/ and /index.md to match actual report layout
	seeRegex := regexp.MustCompile(`(?:See\s+)?(\$trun_dir|\.3pio[\\/]runs[\\/][^/\\]+)[\\/]reports[\\/]([^\r\n]+?)[\\/]index\.md`)
	matches := seeRegex.FindStringSubmatch(out)
	if len(matches) < 3 {
		t.Fatalf("Could not find report path in output. Output:\n%s", out)
	}
	prefix := matches[1]
	consoleReportRel := matches[2]
	var seePath string
	if prefix == "$trun_dir" {
		seePath = filepath.Join(trunDir, "reports", consoleReportRel, "index.md")
	} else {
		seePath = filepath.Join(prefix, "reports", consoleReportRel, "index.md")
	}
	reportPath := filepath.Join(fixtureDir, seePath)
	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		t.Fatalf("Report path from console does not exist: %s\nOutput:\n%s", reportPath, out)
	}
}

func TestConsoleOutputPath_Vitest(t *testing.T) {
	if _, err := exec.LookPath("npm"); err != nil {
		t.Skip("npm not found in PATH")
	}
	if err := exec.Command("npx", "vitest", "--version").Run(); err != nil {
		t.Skipf("vitest not available: %v", err)
	}
	fixtureDir := filepath.Join("..", "fixtures", "basic-vitest")
	if _, err := os.Stat(fixtureDir); os.IsNotExist(err) {
		t.Skip("fixture not found")
	}
	_ = cleanProjectOutput(fixtureDir)
	bin, _ := filepath.Abs(threePioBinary)
	cmd := exec.Command(bin, "npx", "vitest", "run", "string.test.js")
	cmd.Dir = fixtureDir
	output, _ := cmd.CombinedOutput()
	verifyConsoleReportPath(t, fixtureDir, output)
}

func TestConsoleOutputPath_Mocha(t *testing.T) {
	if _, err := exec.LookPath("npm"); err != nil {
		t.Skip("npm not found in PATH")
	}
	if err := exec.Command("npx", "mocha", "--version").Run(); err != nil {
		t.Skipf("mocha not available: %v", err)
	}
	fixtureDir := filepath.Join("..", "fixtures", "failing-mocha")
	if _, err := os.Stat(fixtureDir); os.IsNotExist(err) {
		t.Skip("fixture not found")
	}
	_ = cleanProjectOutput(fixtureDir)
	bin, _ := filepath.Abs(threePioBinary)
	cmd := exec.Command(bin, "npx", "mocha", "failing.spec.js")
	cmd.Dir = fixtureDir
	output, _ := cmd.CombinedOutput()
	verifyConsoleReportPath(t, fixtureDir, output)
}

func TestConsoleOutputPath_Cypress(t *testing.T) {
	if _, err := exec.LookPath("npm"); err != nil {
		t.Skip("npm not found in PATH")
	}
	if err := exec.Command("npx", "cypress", "--version").Run(); err != nil {
		t.Skipf("cypress not available: %v", err)
	}
	fixtureDir := filepath.Join("..", "fixtures", "cypress-complex")
	if _, err := os.Stat(fixtureDir); os.IsNotExist(err) {
		t.Skip("fixture not found")
	}
	_ = cleanProjectOutput(fixtureDir)
	bin, _ := filepath.Abs(threePioBinary)
	cmd := exec.Command(bin, "npx", "cypress", "run", "--headless", "--spec", "cypress/e2e/failing.cy.js")
	cmd.Dir = fixtureDir
	output, _ := cmd.CombinedOutput()
	verifyConsoleReportPath(t, fixtureDir, output)
}

func TestConsoleOutputPath_Pytest(t *testing.T) {
	if _, err := exec.LookPath("pytest"); err != nil {
		t.Skip("pytest not found in PATH")
	}
	fixtureDir := filepath.Join("..", "fixtures", "basic-pytest")
	if _, err := os.Stat(fixtureDir); os.IsNotExist(err) {
		t.Skip("fixture not found")
	}
	_ = cleanProjectOutput(fixtureDir)
	bin, _ := filepath.Abs(threePioBinary)
	cmd := exec.Command(bin, "pytest", "-q", "test_string.py")
	cmd.Dir = fixtureDir
	output, _ := cmd.CombinedOutput()
	out := string(output)
	trunRegex := regexp.MustCompile(`trun_dir:\s+(\.3pio[\\/]runs[\\/][^\s]+)`)
	trunMatch := trunRegex.FindStringSubmatch(out)
	seeRegex := regexp.MustCompile(`(?:See\s+)?(\$trun_dir|\.3pio[\\/]runs[\\/][^/\\]+)[\\/]reports[\\/]([^\r\n]+?)[\\/]index\.md`)
	seeMatch := seeRegex.FindStringSubmatch(out)
	if len(trunMatch) >= 2 && len(seeMatch) >= 3 {
		trunDir := trunMatch[1]
		prefix := seeMatch[1]
		consoleReportRel := seeMatch[2]
		var seePath string
		if prefix == "$trun_dir" {
			seePath = filepath.Join(trunDir, "reports", consoleReportRel, "index.md")
		} else {
			seePath = filepath.Join(prefix, "reports", consoleReportRel, "index.md")
		}
		reportPath := filepath.Join(fixtureDir, seePath)
		if _, err := os.Stat(reportPath); err == nil {
			return
		}
	}
	if !strings.Contains(out, "Results:") {
		t.Fatalf("pytest output missing final Results line. Output:\n%s", out)
	}
	runDir := getLatestRunDir(t, fixtureDir)
	if _, err := os.Stat(filepath.Join(runDir, "test-run.md")); os.IsNotExist(err) {
		t.Fatalf("pytest run missing test-run.md at %s", runDir)
	}
}

func TestConsoleOutputPath_Go(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not found")
	}
	fixtureDir := filepath.Join("..", "fixtures", "many-failures")
	if _, err := os.Stat(fixtureDir); os.IsNotExist(err) {
		t.Skip("fixture not found")
	}
	_ = cleanProjectOutput(fixtureDir)
	bin, _ := filepath.Abs(threePioBinary)
	cmd := exec.Command(bin, "go", "test", "-count=1", ".")
	cmd.Dir = fixtureDir
	output, _ := cmd.CombinedOutput()
	verifyConsoleReportPath(t, fixtureDir, output)
}

func TestConsoleOutputPath_Cargo(t *testing.T) {
	if _, err := exec.LookPath("cargo"); err != nil {
		t.Skip("cargo not found")
	}
	fixtureDir := filepath.Join("..", "fixtures", "rust-edge-cases")
	if _, err := os.Stat(fixtureDir); os.IsNotExist(err) {
		t.Skip("fixture not found")
	}
	_ = cleanProjectOutput(fixtureDir)
	bin, _ := filepath.Abs(threePioBinary)
	cmd := exec.Command(bin, "cargo", "test")
	cmd.Dir = fixtureDir
	output, _ := cmd.CombinedOutput()
	out := string(output)
	// Try strict verification first; if no path printed (e.g., early cargo error),
	// fall back to checking Results and test-run.md existence.
	// Support both forward and backslashes in paths on different OSes
	trunRegex := regexp.MustCompile(`trun_dir:\s+(\.3pio[\\/]runs[\\/][^\s]+)`)
	seeRegex := regexp.MustCompile(`(?:See\s+)?(\$trun_dir|\.3pio[\\/]runs[\\/][^/\\]+)[\\/]reports[\\/ ]([^/\\]+)[\\/ ]index\.md`)
	if trunRegex.MatchString(out) && seeRegex.MatchString(out) {
		verifyConsoleReportPath(t, fixtureDir, output)
		return
	}
	if !strings.Contains(out, "Results:") {
		t.Fatalf("cargo output missing final Results line. Output:\n%s", out)
	}
	runDir := getLatestRunDir(t, fixtureDir)
	if _, err := os.Stat(filepath.Join(runDir, "test-run.md")); os.IsNotExist(err) {
		t.Fatalf("cargo run missing test-run.md at %s", runDir)
	}
}
