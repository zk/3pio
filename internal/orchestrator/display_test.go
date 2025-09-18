package orchestrator

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/zk/3pio/internal/logger"
	"github.com/zk/3pio/internal/report"
)

// TestDisplayGroupWithFailures tests the new display format for groups with failures
func TestDisplayGroupWithFailures(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Create a test orchestrator with logger
	testLogger, _ := logger.NewFileLogger()
	o := &Orchestrator{
		runID:        "20250917T120000-test-run",
		startTime:    time.Now(),
		logger:       testLogger,
		noTestGroups: make(map[string]bool),
	}

	// Create a test group with failures
	group := &report.TestGroup{
		Name: "../../../private/tmp/3pio-open-source/jest/packages/jest-validate/src/__tests__/validate.test.ts",
		Stats: report.TestGroupStats{
			TotalTests:   12,
			PassedTests:  11,
			FailedTests:  12,
			SkippedTests: 0,
		},
		ParentNames: []string{}, // Top-level group
		TestCases: []report.TestCase{
			{Name: "pretty prints valid config for Boolean", Status: report.TestStatusFail},
			{Name: "pretty prints valid config for Array", Status: report.TestStatusFail},
			{Name: "pretty prints valid config for String", Status: report.TestStatusFail},
			{Name: "test 4", Status: report.TestStatusFail},
			{Name: "test 5", Status: report.TestStatusFail},
			{Name: "test 6", Status: report.TestStatusFail},
			{Name: "test 7", Status: report.TestStatusFail},
			{Name: "test 8", Status: report.TestStatusFail},
			{Name: "test 9", Status: report.TestStatusFail},
			{Name: "test 10", Status: report.TestStatusFail},
			{Name: "test 11", Status: report.TestStatusFail},
			{Name: "test 12", Status: report.TestStatusFail},
		},
	}

	// Call the display function
	o.displayGroupHierarchy(group, 0, 970.0) // 0.97s duration

	// Restore stdout and read captured output
	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	// Verify the output format
	// Should contain: [T+ Xs] FAIL(12) PASS(11) <path> (0.97s) report: $base_dir/reports/...
	expectedParts := []string{
		"[T+",                        // Time prefix
		"FAIL(12) PASS(11)",          // Status with counts
		"validate.test.ts",           // Part of the path
		"(0.97s)",                    // Duration
		"report: $base_dir/reports/", // Report path with placeholder
	}

	for _, part := range expectedParts {
		if !strings.Contains(output, part) {
			t.Errorf("Output missing expected part '%s'. Got: %s", part, output)
		}
	}

	// Verify that individual test names are NOT shown
	unexpectedParts := []string{
		"  x pretty prints valid config for Boolean",
		"  x pretty prints valid config for Array",
		"  +9 more",
		"  See .3pio",
	}

	for _, part := range unexpectedParts {
		if strings.Contains(output, part) {
			t.Errorf("Output contains unexpected part '%s'. Got: %s", part, output)
		}
	}

	// Verify it's on a single line
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 1 {
		t.Errorf("Expected single line output, got %d lines: %s", len(lines), output)
	}
}

// TestDisplayGroupWithoutFailures tests that groups without failures are not displayed with failure details
func TestDisplayGroupWithoutFailures(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Create a test orchestrator with logger
	testLogger, _ := logger.NewFileLogger()
	o := &Orchestrator{
		runID:        "20250917T120000-test-run",
		startTime:    time.Now(),
		logger:       testLogger,
		noTestGroups: make(map[string]bool),
	}

	// Create a test group without failures
	group := &report.TestGroup{
		Name: "test.js",
		Stats: report.TestGroupStats{
			TotalTests:   5,
			PassedTests:  5,
			FailedTests:  0,
			SkippedTests: 0,
		},
		ParentNames: []string{}, // Top-level group
		TestCases: []report.TestCase{
			{Name: "test 1", Status: report.TestStatusPass},
			{Name: "test 2", Status: report.TestStatusPass},
			{Name: "test 3", Status: report.TestStatusPass},
			{Name: "test 4", Status: report.TestStatusPass},
			{Name: "test 5", Status: report.TestStatusPass},
		},
	}

	// Call the display function
	o.displayGroupHierarchy(group, 0, 500.0) // 0.5s duration

	// Restore stdout and read captured output
	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	// For passing groups, the failure block should not be shown
	if strings.Contains(output, "FAIL(") {
		t.Errorf("Passing group should not show FAIL output. Got: %s", output)
	}
	if strings.Contains(output, "report:") {
		t.Errorf("Passing group should not show report path. Got: %s", output)
	}
}

// TestFormatElapsedTime tests the elapsed time formatting
func TestFormatElapsedTime(t *testing.T) {
	tests := []struct {
		name     string
		elapsed  int // seconds
		expected string
	}{
		{"under_minute", 45, "[T+ 45s]"},
		{"exactly_minute", 60, "[T+ 1m0s]"},
		{"minute_and_seconds", 75, "[T+ 1m15s]"},
		{"exactly_hour", 3600, "[T+ 1h0m0s]"},
		{"hour_minute_second", 3665, "[T+ 1h1m5s]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Orchestrator{
				startTime: time.Now().Add(-time.Duration(tt.elapsed) * time.Second),
			}
			result := o.formatElapsedTime()
			if result != tt.expected {
				t.Errorf("formatElapsedTime() = %s, want %s", result, tt.expected)
			}
		})
	}
}
