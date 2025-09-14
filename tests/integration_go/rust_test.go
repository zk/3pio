package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zk/3pio/tests/testutil"
)

func TestCargoTestBasicProject(t *testing.T) {
	// Skip if cargo is not available
	if _, err := testutil.LookPath("cargo"); err != nil {
		t.Skip("cargo not found in PATH")
	}

	testCases := []struct {
		name        string
		fixture     string
		expectPass  bool
		minTests    int
		checkOutput []string
	}{
		{
			name:       "rust-basic all tests",
			fixture:    "rust-basic",
			expectPass: true,
			minTests:   7,
			checkOutput: []string{
				"tests::test_add",
				"tests::test_subtract",
				"integration_tests::test_combined_operations",
			},
		},
		{
			name:       "rust-workspace multiple crates",
			fixture:    "rust-workspace",
			expectPass: true,
			minTests:   15,
			checkOutput: []string{
				"app",
				"core",
				"utils",
			},
		},
		{
			name:       "rust-edge-cases with failures",
			fixture:    "rust-edge-cases",
			expectPass: false,
			minTests:   10,
			checkOutput: []string{
				"test_assertion_failure",
				"test_unexpected_panic",
				"FAIL",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fixtureDir := filepath.Join("..", "fixtures", tc.fixture)
			if _, err := os.Stat(fixtureDir); os.IsNotExist(err) {
				t.Skipf("fixture %s not found", tc.fixture)
			}

			result := testutil.RunThreepio(t, fixtureDir, "cargo", "test")

			// Check exit code
			if tc.expectPass && result.ExitCode != 0 {
				t.Errorf("Expected success but got exit code %d", result.ExitCode)
			} else if !tc.expectPass && result.ExitCode == 0 {
				t.Error("Expected failure but got success")
			}

			// Check report exists
			reportPath := filepath.Join(fixtureDir, ".3pio", "runs", result.RunID, "test-run.md")
			if _, err := os.Stat(reportPath); os.IsNotExist(err) {
				t.Errorf("Report file not found: %s", reportPath)
			}

			// Read report
			reportContent, err := os.ReadFile(reportPath)
			if err != nil {
				t.Fatalf("Failed to read report: %v", err)
			}
			report := string(reportContent)

			// Check for expected output
			for _, expected := range tc.checkOutput {
				if !strings.Contains(report, expected) {
					t.Errorf("Report missing expected content: %s", expected)
				}
			}

			// Check minimum test count
			if tc.minTests > 0 {
				testCountStr := testutil.ExtractTestCount(report)
				if testCountStr < tc.minTests {
					t.Errorf("Expected at least %d tests, found %d", tc.minTests, testCountStr)
				}
			}
		})
	}
}

func TestCargoTestWithFlags(t *testing.T) {
	if _, err := testutil.LookPath("cargo"); err != nil {
		t.Skip("cargo not found in PATH")
	}

	fixtureDir := filepath.Join("..", "fixtures", "rust-comprehensive")
	if _, err := os.Stat(fixtureDir); os.IsNotExist(err) {
		t.Skip("rust-comprehensive fixture not found")
	}

	testCases := []struct {
		name     string
		args     []string
		contains []string
	}{
		{
			name:     "lib tests only",
			args:     []string{"cargo", "test", "--lib"},
			contains: []string{"unit_tests"},
		},
		{
			name:     "doc tests only",
			args:     []string{"cargo", "test", "--doc"},
			contains: []string{"src/lib.rs"},
		},
		{
			name:     "test filtering",
			args:     []string{"cargo", "test", "test_addition"},
			contains: []string{"test_addition"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := testutil.RunThreepio(t, fixtureDir, tc.args...)

			// Read report
			reportPath := filepath.Join(fixtureDir, ".3pio", "runs", result.RunID, "test-run.md")
			reportContent, err := os.ReadFile(reportPath)
			if err != nil {
				t.Fatalf("Failed to read report: %v", err)
			}
			report := string(reportContent)

			// Check expected content
			for _, expected := range tc.contains {
				if !strings.Contains(report, expected) {
					t.Errorf("Report missing expected content: %s", expected)
				}
			}
		})
	}
}

func TestCargoNextest(t *testing.T) {
	// Check if cargo-nextest is installed
	if _, err := testutil.LookPath("cargo"); err != nil {
		t.Skip("cargo not found in PATH")
	}

	// Try to run nextest version to check if installed
	if err := testutil.CommandAvailable("cargo", "nextest", "--version"); err != nil {
		t.Skip("cargo-nextest not installed")
	}

	testCases := []struct {
		name        string
		fixture     string
		args        []string
		expectPass  bool
		checkOutput []string
	}{
		{
			name:       "nextest basic project",
			fixture:    "rust-basic",
			args:       []string{"cargo", "nextest", "run"},
			expectPass: true,
			checkOutput: []string{
				"rust-basic",
				"PASS",
			},
		},
		{
			name:       "nextest workspace",
			fixture:    "rust-workspace",
			args:       []string{"cargo", "nextest", "run", "--workspace"},
			expectPass: true,
			checkOutput: []string{
				"app",
				"core",
				"utils",
			},
		},
		{
			name:       "nextest with partition",
			fixture:    "rust-basic",
			args:       []string{"cargo", "nextest", "run", "--partition", "count:1/2"},
			expectPass: true,
			checkOutput: []string{
				"rust-basic",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fixtureDir := filepath.Join("..", "fixtures", tc.fixture)
			if _, err := os.Stat(fixtureDir); os.IsNotExist(err) {
				t.Skipf("fixture %s not found", tc.fixture)
			}

			result := testutil.RunThreepio(t, fixtureDir, tc.args...)

			// Check exit code
			if tc.expectPass && result.ExitCode != 0 {
				t.Errorf("Expected success but got exit code %d", result.ExitCode)
			} else if !tc.expectPass && result.ExitCode == 0 {
				t.Error("Expected failure but got success")
			}

			// Read report
			reportPath := filepath.Join(fixtureDir, ".3pio", "runs", result.RunID, "test-run.md")
			reportContent, err := os.ReadFile(reportPath)
			if err != nil {
				t.Fatalf("Failed to read report: %v", err)
			}
			report := string(reportContent)

			// Check expected content
			for _, expected := range tc.checkOutput {
				if !strings.Contains(report, expected) {
					t.Errorf("Report missing expected content: %s", expected)
				}
			}
		})
	}
}

func TestRustToolchainSupport(t *testing.T) {
	if _, err := testutil.LookPath("cargo"); err != nil {
		t.Skip("cargo not found in PATH")
	}

	// Check if nightly toolchain is installed
	if err := testutil.CommandAvailable("cargo", "+nightly", "--version"); err != nil {
		t.Skip("nightly toolchain not installed")
	}

	fixtureDir := filepath.Join("..", "fixtures", "rust-basic")
	if _, err := os.Stat(fixtureDir); os.IsNotExist(err) {
		t.Skip("rust-basic fixture not found")
	}

	result := testutil.RunThreepio(t, fixtureDir, "cargo", "+nightly", "test")

	if result.ExitCode != 0 {
		t.Errorf("cargo +nightly test failed with exit code %d", result.ExitCode)
	}

	// Check that report was generated
	reportPath := filepath.Join(fixtureDir, ".3pio", "runs", result.RunID, "test-run.md")
	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		t.Errorf("Report file not found: %s", reportPath)
	}
}