package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zk/3pio/internal/ipc"
	"github.com/zk/3pio/internal/logger"
)

func TestGroupManager_ProcessGroupDiscovered(t *testing.T) {
	tmpDir := t.TempDir()
	log, _ := logger.NewFileLogger()
	t.Cleanup(func() { _ = log.Close() })
	gm := NewGroupManager(tmpDir, "", log)

	// Test discovering a root group
	event := ipc.GroupDiscoveredEvent{
		EventType: string(ipc.EventTypeGroupDiscovered),
		Payload: ipc.GroupDiscoveredPayload{
			GroupName:   "test.js",
			ParentNames: nil,
		},
	}

	err := gm.ProcessGroupDiscovered(event)
	if err != nil {
		t.Fatalf("ProcessGroupDiscovered failed: %v", err)
	}

	// Verify group was created
	if len(gm.rootGroups) != 1 {
		t.Errorf("Expected 1 root group, got %d", len(gm.rootGroups))
	}

	groupID := GenerateGroupID("test.js", nil)
	group, exists := gm.GetGroup(groupID)
	if !exists {
		t.Fatal("Group not found after discovery")
	}

	if group.Name != "test.js" {
		t.Errorf("Group name = %s, want test.js", group.Name)
	}

	// Test discovering a nested group
	nestedEvent := ipc.GroupDiscoveredEvent{
		EventType: string(ipc.EventTypeGroupDiscovered),
		Payload: ipc.GroupDiscoveredPayload{
			GroupName:   "Calculator",
			ParentNames: []string{"test.js"},
		},
	}

	err = gm.ProcessGroupDiscovered(nestedEvent)
	if err != nil {
		t.Fatalf("ProcessGroupDiscovered nested failed: %v", err)
	}

	// Verify nested group and parent relationship
	nestedID := GenerateGroupID("Calculator", []string{"test.js"})
	nestedGroup, exists := gm.GetGroup(nestedID)
	if !exists {
		t.Fatal("Nested group not found")
	}

	if nestedGroup.Depth != 1 {
		t.Errorf("Nested group depth = %d, want 1", nestedGroup.Depth)
	}

	// Check parent has subgroup
	if len(group.Subgroups) != 1 {
		t.Errorf("Parent subgroups = %d, want 1", len(group.Subgroups))
	}

	// Test idempotency
	err = gm.ProcessGroupDiscovered(event)
	if err != nil {
		t.Errorf("Repeated discovery should be idempotent: %v", err)
	}

	if len(gm.rootGroups) != 1 {
		t.Errorf("Repeated discovery created duplicate: %d root groups", len(gm.rootGroups))
	}
}

func TestGroupManager_ProcessGroupStart(t *testing.T) {
	tmpDir := t.TempDir()
	log, _ := logger.NewFileLogger()
	t.Cleanup(func() { _ = log.Close() })
	gm := NewGroupManager(tmpDir, "", log)

	// Start a group that hasn't been discovered (should auto-discover)
	event := ipc.GroupStartEvent{
		EventType: string(ipc.EventTypeGroupStart),
		Payload: ipc.GroupStartPayload{
			GroupName:   "test.js",
			ParentNames: nil,
		},
	}

	err := gm.ProcessGroupStart(event)
	if err != nil {
		t.Fatalf("ProcessGroupStart failed: %v", err)
	}

	groupID := GenerateGroupID("test.js", nil)
	group, exists := gm.GetGroup(groupID)
	if !exists {
		t.Fatal("Group not found after start")
	}

	if group.Status != TestStatusRunning {
		t.Errorf("Group status = %v, want RUNNING", group.Status)
	}

	if group.StartTime.IsZero() {
		t.Error("Group StartTime not set")
	}
}

func TestGroupManager_ProcessGroupResult(t *testing.T) {
	tmpDir := t.TempDir()
	log, _ := logger.NewFileLogger()
	t.Cleanup(func() { _ = log.Close() })
	gm := NewGroupManager(tmpDir, "", log)

	// First discover and start a group
	_ = gm.ProcessGroupDiscovered(ipc.GroupDiscoveredEvent{
		EventType: string(ipc.EventTypeGroupDiscovered),
		Payload: ipc.GroupDiscoveredPayload{
			GroupName:   "test.js",
			ParentNames: nil,
		},
	})

	_ = gm.ProcessGroupStart(ipc.GroupStartEvent{
		EventType: string(ipc.EventTypeGroupStart),
		Payload: ipc.GroupStartPayload{
			GroupName:   "test.js",
			ParentNames: nil,
		},
	})

	// Complete the group
	event := ipc.GroupResultEvent{
		EventType: string(ipc.EventTypeGroupResult),
		Payload: ipc.GroupResultPayload{
			GroupName:   "test.js",
			ParentNames: nil,
			Status:      "PASS",
			Duration:    1500,
			Totals: ipc.GroupTotals{
				Passed:  5,
				Failed:  1,
				Skipped: 2,
				Total:   8,
			},
		},
	}

	err := gm.ProcessGroupResult(event)
	if err != nil {
		t.Fatalf("ProcessGroupResult failed: %v", err)
	}

	groupID := GenerateGroupID("test.js", nil)
	group, _ := gm.GetGroup(groupID)

	if group.Status != TestStatusPass {
		t.Errorf("Group status = %v, want PASS", group.Status)
	}

	if group.Duration != 1500*time.Millisecond {
		t.Errorf("Group duration = %v, want 1500ms", group.Duration)
	}

	if group.Stats.PassedTests != 5 {
		t.Errorf("Passed tests = %d, want 5", group.Stats.PassedTests)
	}

	if group.Stats.FailedTests != 1 {
		t.Errorf("Failed tests = %d, want 1", group.Stats.FailedTests)
	}
}

func TestGroupManager_ProcessTestCase(t *testing.T) {
	tmpDir := t.TempDir()
	log, _ := logger.NewFileLogger()
	t.Cleanup(func() { _ = log.Close() })
	gm := NewGroupManager(tmpDir, "", log)

	// Process a test case (should auto-create parent hierarchy)
	event := ipc.GroupTestCaseEvent{
		EventType: string(ipc.EventTypeTestCase),
		Payload: ipc.TestCasePayload{
			TestName:    "should add numbers",
			ParentNames: []string{"math.test.js", "Calculator"},
			Status:      "PASS",
			Duration:    50,
		},
	}

	err := gm.ProcessTestCase(event)
	if err != nil {
		t.Fatalf("ProcessTestCase failed: %v", err)
	}

	// Verify parent group was created
	parentID := GenerateGroupIDFromPath([]string{"math.test.js", "Calculator"})
	parent, exists := gm.GetGroup(parentID)
	if !exists {
		t.Fatal("Parent group not created")
	}

	if len(parent.TestCases) != 1 {
		t.Errorf("Parent test cases = %d, want 1", len(parent.TestCases))
	}

	tc := parent.TestCases[0]
	if tc.Name != "should add numbers" {
		t.Errorf("Test name = %s, want 'should add numbers'", tc.Name)
	}

	if tc.Status != TestStatusPass {
		t.Errorf("Test status = %v, want PASS", tc.Status)
	}

	// Test with error
	errorEvent := ipc.GroupTestCaseEvent{
		EventType: string(ipc.EventTypeTestCase),
		Payload: ipc.TestCasePayload{
			TestName:    "should fail",
			ParentNames: []string{"math.test.js", "Calculator"},
			Status:      "FAIL",
			Duration:    25,
			Error: &ipc.TestError{
				Message: "Expected 2 to equal 3",
				Stack:   "at test.js:10",
			},
		},
	}

	err = gm.ProcessTestCase(errorEvent)
	if err != nil {
		t.Fatalf("ProcessTestCase with error failed: %v", err)
	}

	parent, _ = gm.GetGroup(parentID)
	if len(parent.TestCases) != 2 {
		t.Errorf("Parent test cases = %d, want 2", len(parent.TestCases))
	}

	failedTest := parent.TestCases[1]
	if failedTest.Error == nil {
		t.Error("Failed test should have error")
	}

	if failedTest.Error.Message != "Expected 2 to equal 3" {
		t.Errorf("Error message = %s", failedTest.Error.Message)
	}
}

func TestGroupManager_HierarchyBuilding(t *testing.T) {
	tmpDir := t.TempDir()
	log, _ := logger.NewFileLogger()
	t.Cleanup(func() { _ = log.Close() })
	gm := NewGroupManager(tmpDir, "", log)

	// Create a deep hierarchy via test case
	event := ipc.GroupTestCaseEvent{
		EventType: string(ipc.EventTypeTestCase),
		Payload: ipc.TestCasePayload{
			TestName:    "should work",
			ParentNames: []string{"src/math.test.js", "Calculator", "addition", "positive numbers"},
			Status:      "PASS",
		},
	}

	err := gm.ProcessTestCase(event)
	if err != nil {
		t.Fatalf("ProcessTestCase failed: %v", err)
	}

    // Verify full hierarchy was created (normalize like GroupManager does)
    rawPaths := [][]string{
        {"src/math.test.js"},
        {"src/math.test.js", "Calculator"},
        {"src/math.test.js", "Calculator", "addition"},
        {"src/math.test.js", "Calculator", "addition", "positive numbers"},
    }

    for _, path := range rawPaths {
        normalized := make([]string, len(path))
        for i, p := range path {
            normalized[i] = gm.normalizeToAbsolutePath(p)
        }
        groupID := GenerateGroupIDFromPath(normalized)
        _, exists := gm.GetGroup(groupID)
        if !exists {
            t.Errorf("Group not found for path: %v", path)
        }
    }

    // Verify parent-child relationships
    rootID := GenerateGroupID(gm.normalizeToAbsolutePath("src/math.test.js"), nil)
	root, _ := gm.GetGroup(rootID)
	if len(root.Subgroups) != 1 {
		t.Errorf("Root should have 1 subgroup, got %d", len(root.Subgroups))
	}
}

func TestGroupManager_OutputCapture(t *testing.T) {
	tmpDir := t.TempDir()
	log, _ := logger.NewFileLogger()
	t.Cleanup(func() { _ = log.Close() })
	gm := NewGroupManager(tmpDir, "", log)

	// Create a group
	_ = gm.ProcessGroupDiscovered(ipc.GroupDiscoveredEvent{
		EventType: string(ipc.EventTypeGroupDiscovered),
		Payload: ipc.GroupDiscoveredPayload{
			GroupName:   "test.js",
			ParentNames: nil,
		},
	})

	// Add stdout
	err := gm.ProcessStdoutChunk("test.js", nil, "Test output line 1\n")
	if err != nil {
		t.Fatalf("ProcessStdoutChunk failed: %v", err)
	}

	err = gm.ProcessStdoutChunk("test.js", nil, "Test output line 2\n")
	if err != nil {
		t.Fatalf("ProcessStdoutChunk failed: %v", err)
	}

	// Add stderr
	err = gm.ProcessStderrChunk("test.js", nil, "Error output\n")
	if err != nil {
		t.Fatalf("ProcessStderrChunk failed: %v", err)
	}

	groupID := GenerateGroupID("test.js", nil)
	group, _ := gm.GetGroup(groupID)

	expectedStdout := "Test output line 1\nTest output line 2\n"
	if group.Stdout != expectedStdout {
		t.Errorf("Stdout = %q, want %q", group.Stdout, expectedStdout)
	}

	expectedStderr := "Error output\n"
	if group.Stderr != expectedStderr {
		t.Errorf("Stderr = %q, want %q", group.Stderr, expectedStderr)
	}
}

func TestGroupManager_ReportGeneration(t *testing.T) {
	tmpDir := t.TempDir()
	log, _ := logger.NewFileLogger()
	t.Cleanup(func() { _ = log.Close() })
	gm := NewGroupManager(tmpDir, "", log)

	// Create a hierarchy with tests
	_ = gm.ProcessGroupDiscovered(ipc.GroupDiscoveredEvent{
		EventType: string(ipc.EventTypeGroupDiscovered),
		Payload: ipc.GroupDiscoveredPayload{
			GroupName:   "math.test.js",
			ParentNames: nil,
		},
	})

	_ = gm.ProcessTestCase(ipc.GroupTestCaseEvent{
		EventType: string(ipc.EventTypeTestCase),
		Payload: ipc.TestCasePayload{
			TestName:    "should add",
			ParentNames: []string{"math.test.js"},
			Status:      "PASS",
			Duration:    10,
		},
	})

	_ = gm.ProcessTestCase(ipc.GroupTestCaseEvent{
		EventType: string(ipc.EventTypeTestCase),
		Payload: ipc.TestCasePayload{
			TestName:    "should subtract",
			ParentNames: []string{"math.test.js"},
			Status:      "FAIL",
			Duration:    15,
			Error: &ipc.TestError{
				Message: "Expected 5 to equal 6",
			},
		},
	})

	// Complete the group
	_ = gm.ProcessGroupResult(ipc.GroupResultEvent{
		EventType: string(ipc.EventTypeGroupResult),
		Payload: ipc.GroupResultPayload{
			GroupName:   "math.test.js",
			ParentNames: nil,
			Status:      "FAIL",
		},
	})

	// Wait for debounced update
	time.Sleep(200 * time.Millisecond)

	// Check report was generated
	groupID := GenerateGroupID("math.test.js", nil)
	group, _ := gm.GetGroup(groupID)
	reportPath := GetReportFilePath(group, tmpDir)

	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		t.Error("Report file was not generated")
	}

	// Read and verify report content
	content, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("Failed to read report: %v", err)
	}

	reportStr := string(content)

	// Check for expected content
	if !strings.Contains(reportStr, "math.test.js") {
		t.Error("Report should contain group name")
	}

	if !strings.Contains(reportStr, "should add") {
		t.Error("Report should contain passing test")
	}

	if !strings.Contains(reportStr, "should subtract") {
		t.Error("Report should contain failing test")
	}

	if !strings.Contains(reportStr, "Expected 5 to equal 6") {
		t.Error("Report should contain error message")
	}
}

func TestGroupManager_StatusPropagation(t *testing.T) {
	tmpDir := t.TempDir()
	log, _ := logger.NewFileLogger()
	t.Cleanup(func() { _ = log.Close() })
	gm := NewGroupManager(tmpDir, "", log)

	// Create parent and child groups
	_ = gm.ProcessGroupDiscovered(ipc.GroupDiscoveredEvent{
		EventType: string(ipc.EventTypeGroupDiscovered),
		Payload: ipc.GroupDiscoveredPayload{
			GroupName:   "test.js",
			ParentNames: nil,
		},
	})

	_ = gm.ProcessGroupDiscovered(ipc.GroupDiscoveredEvent{
		EventType: string(ipc.EventTypeGroupDiscovered),
		Payload: ipc.GroupDiscoveredPayload{
			GroupName:   "Suite1",
			ParentNames: []string{"test.js"},
		},
	})

	_ = gm.ProcessGroupDiscovered(ipc.GroupDiscoveredEvent{
		EventType: string(ipc.EventTypeGroupDiscovered),
		Payload: ipc.GroupDiscoveredPayload{
			GroupName:   "Suite2",
			ParentNames: []string{"test.js"},
		},
	})

	// Complete child groups
	_ = gm.ProcessGroupResult(ipc.GroupResultEvent{
		EventType: string(ipc.EventTypeGroupResult),
		Payload: ipc.GroupResultPayload{
			GroupName:   "Suite1",
			ParentNames: []string{"test.js"},
			Status:      "PASS",
		},
	})

	_ = gm.ProcessGroupResult(ipc.GroupResultEvent{
		EventType: string(ipc.EventTypeGroupResult),
		Payload: ipc.GroupResultPayload{
			GroupName:   "Suite2",
			ParentNames: []string{"test.js"},
			Status:      "FAIL",
		},
	})

	// Check parent status was updated
	parentID := GenerateGroupID("test.js", nil)
	parent, _ := gm.GetGroup(parentID)

	// Parent should be FAIL because one child failed
	if parent.Status != TestStatusFail {
		t.Errorf("Parent status = %v, want FAIL", parent.Status)
	}
}

func TestGroupManager_FinalReport(t *testing.T) {
	tmpDir := t.TempDir()
	log, _ := logger.NewFileLogger()
	t.Cleanup(func() { _ = log.Close() })
	gm := NewGroupManager(tmpDir, "", log)

	// Create multiple root groups
	_ = gm.ProcessGroupDiscovered(ipc.GroupDiscoveredEvent{
		EventType: string(ipc.EventTypeGroupDiscovered),
		Payload: ipc.GroupDiscoveredPayload{
			GroupName:   "test1.js",
			ParentNames: nil,
		},
	})

	_ = gm.ProcessTestCase(ipc.GroupTestCaseEvent{
		EventType: string(ipc.EventTypeTestCase),
		Payload: ipc.TestCasePayload{
			TestName:    "test 1",
			ParentNames: []string{"test1.js"},
			Status:      "PASS",
		},
	})

	_ = gm.ProcessGroupDiscovered(ipc.GroupDiscoveredEvent{
		EventType: string(ipc.EventTypeGroupDiscovered),
		Payload: ipc.GroupDiscoveredPayload{
			GroupName:   "test2.js",
			ParentNames: nil,
		},
	})

	_ = gm.ProcessTestCase(ipc.GroupTestCaseEvent{
		EventType: string(ipc.EventTypeTestCase),
		Payload: ipc.TestCasePayload{
			TestName:    "test 2",
			ParentNames: []string{"test2.js"},
			Status:      "FAIL",
		},
	})

	// Generate final report
	err := gm.GenerateFinalReport()
	if err != nil {
		t.Fatalf("GenerateFinalReport failed: %v", err)
	}

	// Check summary file exists
	summaryPath := filepath.Join(tmpDir, "test-run.md")
	if _, err := os.Stat(summaryPath); os.IsNotExist(err) {
		t.Error("Summary report was not generated")
	}

	// Read and verify summary content
	content, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("Failed to read summary: %v", err)
	}

	summaryStr := string(content)

	if !strings.Contains(summaryStr, "Test Run Summary") {
		t.Error("Summary should have title")
	}

	if !strings.Contains(summaryStr, "test1.js") {
		t.Error("Summary should list first test file")
	}

	if !strings.Contains(summaryStr, "test2.js") {
		t.Error("Summary should list second test file")
	}

	if !strings.Contains(summaryStr, "Overall Statistics") {
		t.Error("Summary should have statistics")
	}
}

func TestGroupManager_Cleanup(t *testing.T) {
	tmpDir := t.TempDir()
	log, _ := logger.NewFileLogger()
	t.Cleanup(func() { _ = log.Close() })
	gm := NewGroupManager(tmpDir, "", log)

	// Create a group and add pending updates
	_ = gm.ProcessGroupDiscovered(ipc.GroupDiscoveredEvent{
		EventType: string(ipc.EventTypeGroupDiscovered),
		Payload: ipc.GroupDiscoveredPayload{
			GroupName:   "test.js",
			ParentNames: nil,
		},
	})

	groupID := GenerateGroupID("test.js", nil)
	gm.scheduleReportUpdate(groupID)

	// Cleanup should flush pending updates
	gm.Cleanup()

	// Verify timer was stopped
	gm.updateMutex.Lock()
	if gm.updateTimer != nil {
		t.Error("Update timer should be nil after cleanup")
	}
	gm.updateMutex.Unlock()
}

func TestFormatGroupReport_OnlyShowCountsGreaterThanZero(t *testing.T) {
	tmpDir := t.TempDir()
	log, _ := logger.NewFileLogger()
	t.Cleanup(func() { _ = log.Close() })
	gm := NewGroupManager(tmpDir, "", log)

	// Test group with some passed tests, no failed tests, some skipped tests
	group := &TestGroup{
		ID:          "test-group",
		Name:        "Test Group",
		ParentNames: []string{},
		Status:      TestStatusPass,
		Duration:    100 * time.Millisecond,
		Created:     time.Now(),
		Updated:     time.Now(),
		TestCases: []TestCase{
			{Name: "test1", Status: TestStatusPass, Duration: 10 * time.Millisecond},
			{Name: "test2", Status: TestStatusPass, Duration: 20 * time.Millisecond},
			{Name: "test3", Status: TestStatusSkip, Duration: 0},
		},
		Stats: TestGroupStats{
			TotalTests:   3,
			PassedTests:  2,
			FailedTests:  0, // This should not appear in summary
			SkippedTests: 1,
		},
		Subgroups: make(map[string]*TestGroup),
	}

	content := gm.formatGroupReport(group)

	// Should include these lines
	if !strings.Contains(content, "- Group tests: 3") {
		t.Error("Should show group tests")
	}
	if !strings.Contains(content, "- Group tests passed: 2") {
		t.Error("Should show group tests passed")
	}
	if !strings.Contains(content, "- Group tests skipped: 1") {
		t.Error("Should show group tests skipped")
	}

	// Should NOT include this line (failed tests is 0)
	if strings.Contains(content, "- Group tests failed: 0") {
		t.Error("Should not show 'Group tests failed: 0'")
	}
	if strings.Contains(content, "Group tests failed:") {
		t.Error("Should not show failed tests line when count is 0")
	}
}
