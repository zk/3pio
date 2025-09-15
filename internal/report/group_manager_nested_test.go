package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zk/3pio/internal/logger"
)

func TestGenerateGroupReport_NestedTestAggregation(t *testing.T) {
	// This test verifies that subgroup tables show aggregated test counts
	// for all nested tests, not just direct children
	tmpDir := t.TempDir()
	log, _ := logger.NewFileLogger()
	t.Cleanup(func() { _ = log.Close() })
	gm := NewGroupManager(tmpDir, "", log)

	// Create a deeply nested structure like Rust tests:
	// actix-http (root)
	//   └── body (module with no direct tests)
	//       └── body_stream (submodule with no direct tests)
	//           └── tests (submodule with actual tests)
	//               ├── test1 (PASS)
	//               └── test2 (PASS)

	// Create root group
	rootID := GenerateGroupID("actix-http", []string{})
	root := &TestGroup{
		ID:          rootID,
		Name:        "actix-http",
		ParentNames: []string{},
		Status:      TestStatusPass,
		StartTime:   time.Now(),
		Subgroups:   make(map[string]*TestGroup),
	}
	gm.groups[rootID] = root
	gm.rootGroups = append(gm.rootGroups, root)

	// Create body module (no direct tests)
	bodyID := GenerateGroupID("body", []string{"actix-http"})
	body := &TestGroup{
		ID:          bodyID,
		Name:        "body",
		ParentID:    rootID,
		ParentNames: []string{"actix-http"},
		Status:      TestStatusPass,
		Subgroups:   make(map[string]*TestGroup),
		Depth:       1,
	}
	gm.groups[bodyID] = body
	root.Subgroups[bodyID] = body

	// Create body_stream submodule (no direct tests)
	streamID := GenerateGroupID("body_stream", []string{"actix-http", "body"})
	stream := &TestGroup{
		ID:          streamID,
		Name:        "body_stream",
		ParentID:    bodyID,
		ParentNames: []string{"actix-http", "body"},
		Status:      TestStatusPass,
		Subgroups:   make(map[string]*TestGroup),
		Depth:       2,
	}
	gm.groups[streamID] = stream
	body.Subgroups[streamID] = stream

	// Create tests submodule with actual tests
	testsID := GenerateGroupID("tests", []string{"actix-http", "body", "body_stream"})
	tests := &TestGroup{
		ID:          testsID,
		Name:        "tests",
		ParentID:    streamID,
		ParentNames: []string{"actix-http", "body", "body_stream"},
		Status:      TestStatusPass,
		Subgroups:   make(map[string]*TestGroup),
		Depth:       3,
		TestCases: []TestCase{
			{
				Name:      "stream_string_error",
				Status:    TestStatusPass,
				Duration:  500 * time.Microsecond,
				StartTime: time.Now(),
				EndTime:   time.Now(),
			},
			{
				Name:      "stream_immediate_error",
				Status:    TestStatusPass,
				Duration:  2 * time.Millisecond,
				StartTime: time.Now(),
				EndTime:   time.Now(),
			},
		},
	}
	tests.Stats.TotalTests = 2
	tests.Stats.PassedTests = 2
	gm.groups[testsID] = tests
	stream.Subgroups[testsID] = tests

	// Update recursive stats from bottom up
	tests.UpdateStats()
	stream.UpdateStats()
	body.UpdateStats()
	root.UpdateStats()

	// Generate report to file first
	err := gm.generateGroupReport(root)
	if err != nil {
		t.Fatalf("Failed to generate report: %v", err)
	}

	// Read the generated report content
	reportPath := filepath.Join(tmpDir, "reports", "actix_http", "index.md")
	reportBytes, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("Failed to read report: %v", err)
	}
	content := string(reportBytes)

	// The bug: subgroups table shows "0 tests" for body module
	// even though it contains 2 tests in nested submodules

	// Check that the table shows the correct aggregated test count
	if !strings.Contains(content, "| PASS | body |") {
		t.Error("Should have body module in subgroups table")
	}

	// This should show "2 passed" not "0 tests"
	// The table should use Stats.TotalTestsRecursive instead of Stats.TotalTests
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.Contains(line, "| PASS | body |") {
			if strings.Contains(line, "0 tests") {
				t.Error("Body module should show '2 passed' not '0 tests' - should aggregate nested test counts")
			}
			if !strings.Contains(line, "2 passed") {
				t.Error("Body module should show '2 passed' for aggregated nested tests")
			}
			break
		}
	}

	// Also verify the module has the correct recursive stats
	if body.Stats.TotalTestsRecursive != 2 {
		t.Errorf("Body module recursive test count = %d, want 2", body.Stats.TotalTestsRecursive)
	}
	if body.Stats.PassedTestsRecursive != 2 {
		t.Errorf("Body module recursive passed count = %d, want 2", body.Stats.PassedTestsRecursive)
	}
}