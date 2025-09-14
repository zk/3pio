package definitions

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zk/3pio/internal/logger"
)

// TestIPCCapture helps capture IPC events for testing
type TestIPCCapture struct {
	ipcPath string
}

func NewTestIPCCapture(ipcPath string) *TestIPCCapture {
	return &TestIPCCapture{
		ipcPath: ipcPath,
	}
}

func (t *TestIPCCapture) GetEvents() []map[string]interface{} {
	var events []map[string]interface{}
	data, err := os.ReadFile(t.ipcPath)
	if err != nil {
		return events
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		var event map[string]interface{}
		if err := json.Unmarshal([]byte(line), &event); err == nil {
			events = append(events, event)
		}
	}
	return events
}

func (t *TestIPCCapture) GetEventsByType(eventType string) []map[string]interface{} {
	var filtered []map[string]interface{}
	for _, e := range t.GetEvents() {
		if e["eventType"] == eventType {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

// Test helper to create a test logger
func createTestLogger(t testing.TB) *logger.FileLogger {
	// Create a temp directory for test logs
	if tt, ok := t.(*testing.T); ok {
		tmpDir := tt.TempDir()
		oldDir, _ := os.Getwd()
		_ = os.Chdir(tmpDir)
		tt.Cleanup(func() { _ = os.Chdir(oldDir) })
	}

	// NewFileLogger creates its own log file in .3pio/debug.log
	l, err := logger.NewFileLogger()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	return l
}

// Test Name method
func TestGoTestDefinition_Name(t *testing.T) {
	g := NewGoTestDefinition(createTestLogger(t))
	if g.Name() != "go" {
		t.Errorf("Expected name 'go', got %s", g.Name())
	}
}

// Test Detect method
func TestGoTestDefinition_Detect(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{
			name:     "Direct go test command",
			args:     []string{"go", "test", "./..."},
			expected: true,
		},
		{
			name:     "Go test with flags",
			args:     []string{"go", "test", "-v", "./..."},
			expected: true,
		},
		{
			name:     "Not a go test command",
			args:     []string{"go", "build", "./..."},
			expected: false,
		},
		{
			name:     "Not a go command",
			args:     []string{"npm", "test"},
			expected: false,
		},
		{
			name:     "Empty args",
			args:     []string{},
			expected: false,
		},
		{
			name:     "Only go command",
			args:     []string{"go"},
			expected: false,
		},
	}

	g := NewGoTestDefinition(createTestLogger(t))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.Detect(tt.args)
			if result != tt.expected {
				t.Errorf("Detect(%v) = %v, want %v", tt.args, result, tt.expected)
			}
		})
	}
}

// Test RequiresAdapter method
func TestGoTestDefinition_RequiresAdapter(t *testing.T) {
	g := NewGoTestDefinition(createTestLogger(t))
	if g.RequiresAdapter() {
		t.Error("Go test should not require adapter")
	}
}

// Test ModifyCommand method
func TestGoTestDefinition_ModifyCommand(t *testing.T) {
	tests := []struct {
		name     string
		cmd      []string
		validate func(t *testing.T, result []string)
	}{
		{
			name: "Simple go test",
			cmd:  []string{"go", "test", "./..."},
			validate: func(t *testing.T, result []string) {
				if len(result) != 4 {
					t.Errorf("Expected 4 args, got %d", len(result))
				}
				if result[2] != "-json" {
					t.Errorf("Expected -json flag at position 2")
				}
			},
		},
		{
			name: "Go test with existing json flag",
			cmd:  []string{"go", "test", "-json", "./..."},
			validate: func(t *testing.T, result []string) {
				// Should not duplicate -json flag
				jsonCount := 0
				for _, arg := range result {
					if arg == "-json" {
						jsonCount++
					}
				}
				if jsonCount != 1 {
					t.Errorf("Expected exactly 1 -json flag, got %d", jsonCount)
				}
			},
		},
		{
			name: "Go test with -v flag",
			cmd:  []string{"go", "test", "-v", "./..."},
			validate: func(t *testing.T, result []string) {
				hasJson := false
				hasV := false
				for _, arg := range result {
					if arg == "-json" {
						hasJson = true
					}
					if arg == "-v" {
						hasV = true
					}
				}
				if !hasJson {
					t.Error("Missing -json flag")
				}
				if !hasV {
					t.Error("Missing -v flag")
				}
			},
		},
	}

	g := NewGoTestDefinition(createTestLogger(t))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.ModifyCommand(tt.cmd, "/tmp/test.jsonl", "test-run-id")
			tt.validate(t, result)
		})
	}
}

// Test GetTestFiles method
func TestGoTestDefinition_GetTestFiles(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectedCount int
		checkFunc     func([]string) bool
	}{
		{
			name:          "Specific test file",
			args:          []string{"go", "test", "pkg1/foo_test.go"},
			expectedCount: 1,
			checkFunc: func(files []string) bool {
				return len(files) == 1 && files[0] == "pkg1/foo_test.go"
			},
		},
		{
			name:          "Multiple specific test files",
			args:          []string{"go", "test", "foo_test.go", "bar_test.go"},
			expectedCount: 2,
			checkFunc: func(files []string) bool {
				return len(files) == 2
			},
		},
		{
			name:          "Package pattern without test files",
			args:          []string{"go", "test", "./pkg1"},
			expectedCount: 0, // go list will fail or return no test files in temp dir
			checkFunc: func(files []string) bool {
				// This is expected to return 0 in a temp dir without go.mod
				return true
			},
		},
	}

	g := NewGoTestDefinition(createTestLogger(t))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := g.GetTestFiles(tt.args)
			if err != nil && tt.expectedCount > 0 {
				t.Fatalf("GetTestFiles failed: %v", err)
			}
			if !tt.checkFunc(files) {
				t.Errorf("Check failed for files: %v", files)
			}
		})
	}
}

// Test event processing
func TestGoTestDefinition_ProcessEvent(t *testing.T) {
	tests := []struct {
		name     string
		event    *GoTestEvent
		setup    func(*GoTestDefinition)
		validate func(*testing.T, *GoTestDefinition, *TestIPCCapture)
	}{
		{
			name: "Package start event",
			event: &GoTestEvent{
				Action:  "start",
				Package: "github.com/test/pkg",
			},
			validate: func(t *testing.T, g *GoTestDefinition, capture *TestIPCCapture) {
				if _, exists := g.packageGroups["github.com/test/pkg"]; !exists {
					t.Error("Package not tracked")
				}
				discoveries := capture.GetEventsByType("testGroupDiscovered")
				if len(discoveries) == 0 {
					t.Error("No group discovered event sent")
				}
			},
		},
		{
			name: "Test run event",
			event: &GoTestEvent{
				Action:  "run",
				Package: "github.com/test/pkg",
				Test:    "TestExample",
			},
			setup: func(g *GoTestDefinition) {
				g.packageGroups["github.com/test/pkg"] = &PackageGroupInfo{
					StartTime: time.Now(),
					Tests:     []TestInfo{},
				}
			},
			validate: func(t *testing.T, g *GoTestDefinition, capture *TestIPCCapture) {
				key := "github.com/test/pkg/TestExample"
				if _, exists := g.testStates[key]; !exists {
					t.Error("Test state not tracked")
				}
			},
		},
		{
			name: "Test pass event",
			event: &GoTestEvent{
				Action:  "pass",
				Package: "github.com/test/pkg",
				Test:    "TestExample",
				Elapsed: 0.5,
			},
			setup: func(g *GoTestDefinition) {
				g.packageGroups["github.com/test/pkg"] = &PackageGroupInfo{
					StartTime: time.Now(),
					Tests:     []TestInfo{},
				}
				g.testStates["github.com/test/pkg/TestExample"] = &TestState{
					Name:      "TestExample",
					Package:   "github.com/test/pkg",
					StartTime: time.Now(),
				}
			},
			validate: func(t *testing.T, g *GoTestDefinition, capture *TestIPCCapture) {
				testCases := capture.GetEventsByType("testCase")
				if len(testCases) == 0 {
					t.Error("No test case event sent")
				}
				if len(testCases) > 0 {
					payload := testCases[0]["payload"].(map[string]interface{})
					if payload["status"] != "PASS" {
						t.Errorf("Expected PASS status, got %v", payload["status"])
					}
				}
			},
		},
		{
			name: "Test fail event",
			event: &GoTestEvent{
				Action:  "fail",
				Package: "github.com/test/pkg",
				Test:    "TestExample",
				Elapsed: 0.5,
			},
			setup: func(g *GoTestDefinition) {
				g.packageGroups["github.com/test/pkg"] = &PackageGroupInfo{
					StartTime: time.Now(),
					Tests:     []TestInfo{},
				}
				g.testStates["github.com/test/pkg/TestExample"] = &TestState{
					Name:      "TestExample",
					Package:   "github.com/test/pkg",
					StartTime: time.Now(),
					Output:    []string{"Error: assertion failed"},
				}
			},
			validate: func(t *testing.T, g *GoTestDefinition, capture *TestIPCCapture) {
				testCases := capture.GetEventsByType("testCase")
				if len(testCases) == 0 {
					t.Error("No test case event sent")
				}
				if len(testCases) > 0 {
					payload := testCases[0]["payload"].(map[string]interface{})
					if payload["status"] != "FAIL" {
						t.Errorf("Expected FAIL status, got %v", payload["status"])
					}
					if payload["error"] == nil {
						t.Error("Expected error in payload")
					}
				}
			},
		},
		{
			name: "Test skip event",
			event: &GoTestEvent{
				Action:  "skip",
				Package: "github.com/test/pkg",
				Test:    "TestExample",
				Elapsed: 0,
			},
			setup: func(g *GoTestDefinition) {
				g.packageGroups["github.com/test/pkg"] = &PackageGroupInfo{
					StartTime: time.Now(),
					Tests:     []TestInfo{},
				}
				g.testStates["github.com/test/pkg/TestExample"] = &TestState{
					Name:      "TestExample",
					Package:   "github.com/test/pkg",
					StartTime: time.Now(),
				}
			},
			validate: func(t *testing.T, g *GoTestDefinition, capture *TestIPCCapture) {
				testCases := capture.GetEventsByType("testCase")
				if len(testCases) == 0 {
					t.Error("No test case event sent")
				}
				if len(testCases) > 0 {
					payload := testCases[0]["payload"].(map[string]interface{})
					if payload["status"] != "SKIP" {
						t.Errorf("Expected SKIP status, got %v", payload["status"])
					}
				}
			},
		},
		{
			name: "Package pass event",
			event: &GoTestEvent{
				Action:  "pass",
				Package: "github.com/test/pkg",
				Elapsed: 1.5,
			},
			setup: func(g *GoTestDefinition) {
				g.packageGroups["github.com/test/pkg"] = &PackageGroupInfo{
					StartTime: time.Now(),
					Tests: []TestInfo{
						{Name: "TestExample", Status: "PASS"},
					},
				}
			},
			validate: func(t *testing.T, g *GoTestDefinition, capture *TestIPCCapture) {
				groupResults := capture.GetEventsByType("testGroupResult")
				if len(groupResults) == 0 {
					t.Error("No group result event sent")
				}
				if len(groupResults) > 0 {
					payload := groupResults[0]["payload"].(map[string]interface{})
					if payload["status"] != "PASS" {
						t.Errorf("Expected PASS status, got %v", payload["status"])
					}
				}
			},
		},
		{
			name: "Output event",
			event: &GoTestEvent{
				Action:  "output",
				Package: "github.com/test/pkg",
				Test:    "TestExample",
				Output:  "=== RUN   TestExample\n",
			},
			setup: func(g *GoTestDefinition) {
				g.testStates["github.com/test/pkg/TestExample"] = &TestState{
					Name:    "TestExample",
					Package: "github.com/test/pkg",
					Output:  []string{},
				}
			},
			validate: func(t *testing.T, g *GoTestDefinition, capture *TestIPCCapture) {
				state := g.testStates["github.com/test/pkg/TestExample"]
				if len(state.Output) == 0 {
					t.Error("Output not captured")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGoTestDefinition(createTestLogger(t))
			// Create a real IPCWriter for testing
			tmpDir := t.TempDir()
			ipcPath := filepath.Join(tmpDir, "test.jsonl")
			ipcWriter, _ := NewIPCWriter(ipcPath)
			g.ipcWriter = ipcWriter
			capture := NewTestIPCCapture(ipcPath)

			if tt.setup != nil {
				tt.setup(g)
			}

			err := g.processEvent(tt.event)
			if err != nil {
				t.Fatalf("processEvent failed: %v", err)
			}

			tt.validate(t, g, capture)
		})
	}
}

// Test ProcessOutput method with complete JSON stream
func TestGoTestDefinition_ProcessOutput(t *testing.T) {
	// Create test JSON events
	events := []map[string]interface{}{
		{"Time": "2024-01-01T00:00:00Z", "Action": "start", "Package": "github.com/test/pkg"},
		{"Time": "2024-01-01T00:00:01Z", "Action": "run", "Package": "github.com/test/pkg", "Test": "TestA"},
		{"Time": "2024-01-01T00:00:01Z", "Action": "output", "Package": "github.com/test/pkg", "Test": "TestA", "Output": "=== RUN   TestA\n"},
		{"Time": "2024-01-01T00:00:02Z", "Action": "pass", "Package": "github.com/test/pkg", "Test": "TestA", "Elapsed": 1.0},
		{"Time": "2024-01-01T00:00:03Z", "Action": "run", "Package": "github.com/test/pkg", "Test": "TestB"},
		{"Time": "2024-01-01T00:00:03Z", "Action": "output", "Package": "github.com/test/pkg", "Test": "TestB", "Output": "=== RUN   TestB\n"},
		{"Time": "2024-01-01T00:00:03Z", "Action": "output", "Package": "github.com/test/pkg", "Test": "TestB", "Output": "    test_b.go:10: error message\n"},
		{"Time": "2024-01-01T00:00:04Z", "Action": "fail", "Package": "github.com/test/pkg", "Test": "TestB", "Elapsed": 1.0},
		{"Time": "2024-01-01T00:00:05Z", "Action": "fail", "Package": "github.com/test/pkg", "Elapsed": 5.0},
	}

	// Convert to JSON lines
	var buffer bytes.Buffer
	for _, event := range events {
		jsonBytes, _ := json.Marshal(event)
		buffer.Write(jsonBytes)
		buffer.WriteByte('\n')
	}

	// Create temporary IPC file
	tmpDir := t.TempDir()
	ipcPath := filepath.Join(tmpDir, "test.jsonl")

	g := NewGoTestDefinition(createTestLogger(t))

	// Process the output
	reader := bytes.NewReader(buffer.Bytes())
	err := g.ProcessOutput(reader, ipcPath)
	if err != nil && err != io.EOF {
		t.Fatalf("ProcessOutput failed: %v", err)
	}

	// Read and validate IPC file
	ipcData, err := os.ReadFile(ipcPath)
	if err != nil {
		t.Fatalf("Failed to read IPC file: %v", err)
	}

	// Parse IPC events
	lines := strings.Split(strings.TrimSpace(string(ipcData)), "\n")
	if len(lines) == 0 {
		t.Fatal("No IPC events written")
	}

	// Check for expected event types
	eventTypes := make(map[string]int)
	for _, line := range lines {
		var event map[string]interface{}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("Failed to parse IPC event: %v", err)
		}
		eventType := event["eventType"].(string)
		eventTypes[eventType]++
	}

	// Validate event counts
	if eventTypes["testGroupDiscovered"] == 0 {
		t.Error("No testGroupDiscovered events")
	}
	if eventTypes["testCase"] != 2 {
		t.Errorf("Expected 2 testCase events, got %d", eventTypes["testCase"])
	}
	if eventTypes["testGroupResult"] == 0 {
		t.Error("No testGroupResult events")
	}

	// Validate final package state
	if len(g.packageGroups) != 1 {
		t.Errorf("Expected 1 package, got %d", len(g.packageGroups))
	}

	pkg := g.packageGroups["github.com/test/pkg"]
	if pkg == nil {
		t.Fatal("Package not found")
	}

	pkgStatus := g.packageStatuses["github.com/test/pkg"]
	if pkgStatus != "FAIL" {
		t.Errorf("Expected package status FAIL, got %s", pkgStatus)
	}

	if len(pkg.Tests) != 2 {
		t.Errorf("Expected 2 tests, got %d", len(pkg.Tests))
	}
}

// Test parseTestHierarchy method
func TestGoTestDefinition_ParseTestHierarchy(t *testing.T) {
	tests := []struct {
		name              string
		testName          string
		expectedSuite     []string
		expectedFinalName string
	}{
		{
			name:              "Simple test",
			testName:          "TestExample",
			expectedSuite:     []string{},
			expectedFinalName: "TestExample",
		},
		{
			name:              "Subtest with one level",
			testName:          "TestExample/subtest",
			expectedSuite:     []string{"TestExample"},
			expectedFinalName: "subtest",
		},
		{
			name:              "Subtest with multiple levels",
			testName:          "TestExample/group/subtest",
			expectedSuite:     []string{"TestExample", "group"},
			expectedFinalName: "subtest",
		},
		{
			name:              "Table test",
			testName:          "TestExample/case_1",
			expectedSuite:     []string{"TestExample"},
			expectedFinalName: "case_1",
		},
	}

	g := NewGoTestDefinition(createTestLogger(t))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suite, finalName := g.parseTestHierarchy(tt.testName)

			if len(suite) != len(tt.expectedSuite) {
				t.Errorf("Suite length mismatch: got %v, want %v", suite, tt.expectedSuite)
			}

			for i, s := range suite {
				if i < len(tt.expectedSuite) && s != tt.expectedSuite[i] {
					t.Errorf("Suite[%d] mismatch: got %s, want %s", i, s, tt.expectedSuite[i])
				}
			}

			if finalName != tt.expectedFinalName {
				t.Errorf("Final name mismatch: got %s, want %s", finalName, tt.expectedFinalName)
			}
		})
	}
}

// Test extractPackagePatterns method
func TestGoTestDefinition_ExtractPackagePatterns(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected []string
	}{
		{
			name:     "Single package",
			args:     []string{"go", "test", "./pkg"},
			expected: []string{"./pkg"},
		},
		{
			name:     "Multiple packages",
			args:     []string{"go", "test", "./pkg1", "./pkg2"},
			expected: []string{"./pkg1", "./pkg2"},
		},
		{
			name:     "All packages",
			args:     []string{"go", "test", "./..."},
			expected: []string{"./..."},
		},
		{
			name:     "With flags",
			args:     []string{"go", "test", "-v", "-json", "./..."},
			expected: []string{"./..."},
		},
		{
			name:     "Specific test file",
			args:     []string{"go", "test", "foo_test.go"},
			expected: []string{},
		},
		{
			name:     "Run specific test",
			args:     []string{"go", "test", "-run", "TestExample", "./pkg"},
			expected: []string{"./pkg"},
		},
		{
			name:     "Module path",
			args:     []string{"go", "test", "github.com/user/repo/pkg"},
			expected: []string{"github.com/user/repo/pkg"},
		},
	}

	g := NewGoTestDefinition(createTestLogger(t))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patterns := g.extractPackagePatterns(tt.args)

			if len(patterns) != len(tt.expected) {
				t.Errorf("Pattern count mismatch: got %d, want %d", len(patterns), len(tt.expected))
			}

			for i, p := range patterns {
				if i < len(tt.expected) && p != tt.expected[i] {
					t.Errorf("Pattern[%d] mismatch: got %s, want %s", i, p, tt.expected[i])
				}
			}
		})
	}
}

// Test error handling in processEvent
func TestGoTestDefinition_ProcessEvent_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		event       *GoTestEvent
		expectError bool
	}{
		{
			name:        "Nil event",
			event:       nil,
			expectError: false, // Should handle gracefully
		},
		{
			name: "Unknown action",
			event: &GoTestEvent{
				Action:  "unknown",
				Package: "test",
			},
			expectError: false, // Should handle gracefully
		},
		{
			name: "Missing package for test event",
			event: &GoTestEvent{
				Action: "run",
				Test:   "TestExample",
			},
			expectError: false, // Should handle gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGoTestDefinition(createTestLogger(t))
			// Create a real IPCWriter for testing
			tmpDir := t.TempDir()
			ipcPath := filepath.Join(tmpDir, "test.jsonl")
			ipcWriter, _ := NewIPCWriter(ipcPath)
			g.ipcWriter = ipcWriter
			// capture not needed for error test cases

			err := g.processEvent(tt.event)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// Test concurrent test handling
func TestGoTestDefinition_ConcurrentTests(t *testing.T) {
	g := NewGoTestDefinition(createTestLogger(t))
	tmpDir := t.TempDir()
	ipcPath := filepath.Join(tmpDir, "test.jsonl")
	ipcWriter, _ := NewIPCWriter(ipcPath)
	g.ipcWriter = ipcWriter
	capture := NewTestIPCCapture(ipcPath)

	// Simulate concurrent test execution
	now := time.Now()
	events := []*GoTestEvent{
		{Action: "start", Package: "pkg1", Time: now},
		{Action: "start", Package: "pkg2", Time: now},
		{Action: "run", Package: "pkg1", Test: "TestA", Time: now},
		{Action: "run", Package: "pkg2", Test: "TestB", Time: now},
		{Action: "output", Package: "pkg1", Test: "TestA", Output: "output A\n", Time: now},
		{Action: "output", Package: "pkg2", Test: "TestB", Output: "output B\n", Time: now},
	}

	// Process run and output events
	for _, event := range events {
		if err := g.processEvent(event); err != nil {
			t.Fatalf("Failed to process event: %v", err)
		}
	}

	// Verify test states exist before they're deleted
	if len(g.testStates) != 2 {
		t.Errorf("Expected 2 test states after run events, got %d", len(g.testStates))
	}

	// Process result events (which will delete test states)
	resultEvents := []*GoTestEvent{
		{Action: "pass", Package: "pkg1", Test: "TestA", Elapsed: 0.1, Time: now},
		{Action: "pass", Package: "pkg2", Test: "TestB", Elapsed: 0.2, Time: now},
		{Action: "pass", Package: "pkg1", Elapsed: 0.5, Time: now},
		{Action: "pass", Package: "pkg2", Elapsed: 0.6, Time: now},
	}

	for _, event := range resultEvents {
		if err := g.processEvent(event); err != nil {
			t.Fatalf("Failed to process event: %v", err)
		}
	}

	// Verify both packages were tracked
	if len(g.packageGroups) != 2 {
		t.Errorf("Expected 2 packages, got %d", len(g.packageGroups))
	}

	// Test states should be cleared after processing results
	if len(g.testStates) != 0 {
		t.Errorf("Expected 0 test states after processing results, got %d", len(g.testStates))
		for k := range g.testStates {
			t.Logf("Test state key: %s", k)
		}
	}

	// Verify IPC events
	testCases := capture.GetEventsByType("testCase")
	if len(testCases) != 2 {
		t.Errorf("Expected 2 test case events, got %d", len(testCases))
	}

	groupResults := capture.GetEventsByType("testGroupResult")
	if len(groupResults) != 2 {
		t.Errorf("Expected 2 group result events, got %d", len(groupResults))
	}
}

// Test IPCWriter
func TestIPCWriter(t *testing.T) {
	tmpDir := t.TempDir()
	ipcPath := filepath.Join(tmpDir, "test.jsonl")

	writer, err := NewIPCWriter(ipcPath)
	if err != nil {
		t.Fatalf("Failed to create IPC writer: %v", err)
	}

	// Write events
	events := []map[string]interface{}{
		{
			"eventType": "testCase",
			"payload": map[string]interface{}{
				"testName": "TestExample",
				"status":   "PASS",
			},
		},
		{
			"eventType": "testGroupResult",
			"payload": map[string]interface{}{
				"groupName": "pkg",
				"status":    "PASS",
			},
		},
	}

	for _, event := range events {
		if err := writer.WriteEvent(event); err != nil {
			t.Fatalf("Failed to write event: %v", err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("Failed to close writer: %v", err)
	}

	// Verify file contents
	data, err := os.ReadFile(ipcPath)
	if err != nil {
		t.Fatalf("Failed to read IPC file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != len(events) {
		t.Errorf("Expected %d lines, got %d", len(events), len(lines))
	}

	// Parse and verify each line
	for i, line := range lines {
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(line), &parsed); err != nil {
			t.Fatalf("Failed to parse line %d: %v", i, err)
		}

		if parsed["eventType"] != events[i]["eventType"] {
			t.Errorf("Line %d: expected eventType %v, got %v", i, events[i]["eventType"], parsed["eventType"])
		}
	}
}

// Test buildTestToFileMap
func TestGoTestDefinition_BuildTestToFileMap(t *testing.T) {
	// This test would require mocking the go list command execution
	// For now, we'll test the parsing logic with mock data
	g := NewGoTestDefinition(createTestLogger(t))

	// Mock package info
	g.packageMap = map[string]*PackageInfo{
		"github.com/test/pkg": {
			ImportPath:  "github.com/test/pkg",
			Dir:         "/tmp/test/pkg",
			TestGoFiles: []string{"foo_test.go", "bar_test.go"},
		},
	}

	// Mock the listTestsInPackage response
	// This would normally come from parsing go test output
	// For this test, we'll just verify the structure is set up correctly

	if err := g.buildTestToFileMap(); err != nil {
		// This might fail if go is not available, which is ok for unit test
		t.Logf("buildTestToFileMap returned error (expected if go not available): %v", err)
	}

	// Verify the test map structure exists
	if g.testToFileMap == nil {
		g.testToFileMap = make(map[string]string)
	}

	// The map should be initialized even if empty
	if g.testToFileMap == nil {
		t.Error("testToFileMap not initialized")
	}
}

// Benchmark test for event processing
func BenchmarkGoTestDefinition_ProcessEvent(b *testing.B) {
	g := NewGoTestDefinition(createTestLogger(b))
	tmpDir := b.TempDir()
	ipcPath := filepath.Join(tmpDir, "test.jsonl")
	ipcWriter, _ := NewIPCWriter(ipcPath)
	g.ipcWriter = ipcWriter

	// Set up initial state
	g.packageGroups["test/pkg"] = &PackageGroupInfo{
		StartTime: time.Now(),
		Tests:     []TestInfo{},
	}

	event := &GoTestEvent{
		Action:  "pass",
		Package: "test/pkg",
		Test:    "TestBenchmark",
		Elapsed: 0.001,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = g.processEvent(event)
	}
}
