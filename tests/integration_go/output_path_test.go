package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func TestConsoleOutputMatchesActualDirectoryStructure(t *testing.T) {
	// This test ensures that the console "See ..." output exactly matches
	// the actual directory structure created by the report system

	fixtureDir := filepath.Join("..", "fixtures", "basic-jest")
	if _, err := os.Stat(fixtureDir); err != nil {
		t.Skipf("Fixture directory not found: %s", fixtureDir)
	}

	// Build path to 3pio binary
	cwd, _ := os.Getwd()
	threePioBinary := filepath.Join(filepath.Dir(filepath.Dir(cwd)), "build", "3pio")
	if runtime.GOOS == "windows" {
		threePioBinary += ".exe"
	}
	if _, err := os.Stat(threePioBinary); err != nil {
		t.Fatalf("3pio binary not found at %s. Run 'make build' first.", threePioBinary)
	}

	// Clean up any existing runs
	threePioDir := filepath.Join(fixtureDir, ".3pio")
	_ = os.RemoveAll(threePioDir)

	// Run a test that we know will fail (string.test.js has a failing test)
	cmd := exec.Command(threePioBinary, "npx", "jest", "string.test.js")
	cmd.Dir = fixtureDir

	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("Expected string.test.js to fail, but it passed")
	}

	outputStr := string(output)

	// Extract the "See" path from console output
	seeRegex := regexp.MustCompile(`See (\.3pio/runs/[^/]+/reports/([^/]+)/index\.md)`)
	matches := seeRegex.FindStringSubmatch(outputStr)
	if len(matches) < 3 {
		t.Fatalf("Could not find 'See' path in output. Output:\n%s", outputStr)
	}

	seePath := matches[1]
	consoleReportDir := matches[2] // The directory name from console output

	t.Logf("Console output shows path: %s", seePath)
	t.Logf("Console report directory: %s", consoleReportDir)

	// Verify the actual directory exists and matches
	reportPath := filepath.Join(fixtureDir, seePath)
	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		t.Fatalf("Report directory does not exist: %s", reportPath)
	}

	// Find the actual reports directory to verify the directory name
	reportsPath := filepath.Join(fixtureDir, ".3pio/runs")
	runDirs, err := os.ReadDir(reportsPath)
	if err != nil {
		t.Fatalf("Failed to read runs directory: %v", err)
	}

	if len(runDirs) != 1 {
		t.Fatalf("Expected exactly 1 run directory, found %d", len(runDirs))
	}

	reportsDirPath := filepath.Join(reportsPath, runDirs[0].Name(), "reports")
	reportEntries, err := os.ReadDir(reportsDirPath)
	if err != nil {
		t.Fatalf("Failed to read reports directory: %v", err)
	}

	if len(reportEntries) != 1 {
		t.Fatalf("Expected exactly 1 report directory, found %d", len(reportEntries))
	}

	actualDirName := reportEntries[0].Name()
	t.Logf("Actual directory name: %s", actualDirName)

	// Most importantly: verify console output matches actual directory
	if consoleReportDir != actualDirName {
		t.Errorf("Console output directory (%q) does not match actual directory (%q)", consoleReportDir, actualDirName)
	}

	// The directory name should end with string_test_js
	// In CI environments, the full path might be sanitized (e.g., d_a_3pio_3pio_tests_fixtures_basic_jest_string_test_js)
	// But it should always end with string_test_js
	if !strings.HasSuffix(actualDirName, "string_test_js") {
		t.Errorf("Directory name should end with 'string_test_js', got %q", actualDirName)
	}
	if !strings.HasSuffix(consoleReportDir, "string_test_js") {
		t.Errorf("Console output should end with 'string_test_js', got %q", consoleReportDir)
	}
}
