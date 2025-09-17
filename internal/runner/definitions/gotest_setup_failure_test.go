package definitions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zk/3pio/internal/ipc"
)

// TestGoTestDefinition_SetupFailure tests the new setup failure handling
func TestGoTestDefinition_SetupFailure(t *testing.T) {
	tests := []struct {
		name              string
		events            []GoTestEvent
		expectedGroupError bool
		expectedErrorMessage string
		expectedTotals    map[string]interface{}
	}{
		{
			name: "Setup failure with error message",
			events: []GoTestEvent{
				{Action: "start", Package: "example.com/pkg"},
				{Action: "output", Package: "example.com/pkg", Output: "No test mode selected, please selected either e2e mode with \"--tags e2e\" or integration mode with \"--tags integration\"\n"},
				{Action: "output", Package: "example.com/pkg", Output: "FAIL\texample.com/pkg\t0.854s\n"},
				{Action: "fail", Package: "example.com/pkg", Elapsed: 0.856},
			},
			expectedGroupError: true,
			expectedErrorMessage: "No test mode selected, please selected either e2e mode with \"--tags e2e\" or integration mode with \"--tags integration\"",
			expectedTotals: map[string]interface{}{
				"total": 0, "passed": 0, "failed": 0, "skipped": 0, "setupFailed": true,
			},
		},
		{
			name: "Setup failure with compilation error",
			events: []GoTestEvent{
				{Action: "start", Package: "example.com/pkg"},
				{Action: "output", Package: "example.com/pkg", Output: "# example.com/pkg\n"},
				{Action: "output", Package: "example.com/pkg", Output: "./main.go:5:2: undefined: nonExistentFunction\n"},
				{Action: "fail", Package: "example.com/pkg", Elapsed: 0.123},
			},
			expectedGroupError: true,
			expectedErrorMessage: "./main.go:5:2: undefined: nonExistentFunction",
			expectedTotals: map[string]interface{}{
				"total": 0, "passed": 0, "failed": 0, "skipped": 0, "setupFailed": true,
			},
		},
		{
			name: "Normal package failure with tests",
			events: []GoTestEvent{
				{Action: "start", Package: "example.com/pkg"},
				{Action: "run", Package: "example.com/pkg", Test: "TestSomething"},
				{Action: "fail", Package: "example.com/pkg", Test: "TestSomething", Elapsed: 0.1},
				{Action: "fail", Package: "example.com/pkg", Elapsed: 0.2},
			},
			expectedGroupError: false,
			expectedTotals: map[string]interface{}{
				"total": 1, "passed": 0, "failed": 1, "skipped": 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test logger
			testLogger := createTestLogger(t)

			// Create Go test definition
			def := NewGoTestDefinition(testLogger)

			// Create mock IPC file to capture events
			tempDir := t.TempDir()
			ipcPath := filepath.Join(tempDir, "events.jsonl")

			// Initialize IPC writer directly since we can't call ProcessOutput
			var err error
			def.ipcWriter, err = NewIPCWriter(ipcPath)
			if err != nil {
				t.Fatalf("Failed to create IPC writer: %v", err)
			}
			defer func() {
				if err := def.ipcWriter.Close(); err != nil {
					t.Logf("Failed to close IPC writer: %v", err)
				}
			}()

			// Process all events
			for _, event := range tt.events {
				err := def.processEvent(&event)
				if err != nil {
					t.Fatalf("Failed to process event: %v", err)
				}
			}

			// Read events from IPC file
			var sentEvents []ipc.Event
			if data, err := os.ReadFile(ipcPath); err == nil {
				lines := strings.Split(strings.TrimSpace(string(data)), "\n")
				for _, line := range lines {
					if line == "" {
						continue
					}
					var genericEvent map[string]interface{}
					if err := json.Unmarshal([]byte(line), &genericEvent); err == nil {
						eventType, ok := genericEvent["eventType"].(string)
						if ok && eventType == "testGroupError" {
							var groupErrorEvent ipc.GroupErrorEvent
							if err := json.Unmarshal([]byte(line), &groupErrorEvent); err == nil {
								sentEvents = append(sentEvents, groupErrorEvent)
							}
						} else if ok && eventType == "testGroupResult" {
							var groupResultEvent ipc.GroupResultEvent
							if err := json.Unmarshal([]byte(line), &groupResultEvent); err == nil {
								sentEvents = append(sentEvents, groupResultEvent)
							}
						}
					}
				}
			}

			// Check if testGroupError was sent
			var groupErrorFound bool
			var groupErrorEvent *ipc.GroupErrorEvent
			var groupResultEvent *ipc.GroupResultEvent

			for _, event := range sentEvents {
				switch e := event.(type) {
				case ipc.GroupErrorEvent:
					groupErrorFound = true
					groupErrorEvent = &e
				case ipc.GroupResultEvent:
					groupResultEvent = &e
				}
			}

			// Verify error event expectation
			if tt.expectedGroupError && !groupErrorFound {
				t.Errorf("Expected testGroupError event but none was sent")
			}
			if !tt.expectedGroupError && groupErrorFound {
				t.Errorf("Unexpected testGroupError event was sent")
			}

			// Verify error message content
			if tt.expectedGroupError && groupErrorEvent != nil {
				if groupErrorEvent.Payload.Error == nil {
					t.Errorf("Expected error info but got nil")
				} else if !strings.Contains(groupErrorEvent.Payload.Error.Message, tt.expectedErrorMessage) {
					t.Errorf("Expected error message to contain %q, got %q",
						tt.expectedErrorMessage, groupErrorEvent.Payload.Error.Message)
				}
				if groupErrorEvent.Payload.ErrorType != "SETUP_FAILURE" {
					t.Errorf("Expected error type SETUP_FAILURE, got %q", groupErrorEvent.Payload.ErrorType)
				}
			}

			// Verify group result totals
			if groupResultEvent != nil {
				totals := groupResultEvent.Payload.Totals
				for key, expected := range tt.expectedTotals {
					var actual interface{}
					switch key {
					case "total":
						actual = totals.Total
					case "passed":
						actual = totals.Passed
					case "failed":
						actual = totals.Failed
					case "skipped":
						actual = totals.Skipped
					case "setupFailed":
						actual = totals.SetupFailed
					}
					if actual != expected {
						t.Errorf("Expected %s=%v, got %v", key, expected, actual)
					}
				}
			}
		})
	}
}

// TestGoTestDefinition_ErrorMessageConstruction tests error message filtering
func TestGoTestDefinition_ErrorMessageConstruction(t *testing.T) {
	tests := []struct {
		name           string
		outputLines    []string
		expectedResult string
	}{
		{
			name: "Clean error message",
			outputLines: []string{
				"No test mode selected, please selected either e2e mode with \"--tags e2e\" or integration mode with \"--tags integration\"",
				"FAIL\texample.com/pkg\t0.854s",
			},
			expectedResult: "No test mode selected, please selected either e2e mode with \"--tags e2e\" or integration mode with \"--tags integration\"",
		},
		{
			name: "Compilation error",
			outputLines: []string{
				"# example.com/pkg",
				"./main.go:5:2: undefined: nonExistentFunction",
				"FAIL\texample.com/pkg\t0.123s",
			},
			expectedResult: "# example.com/pkg\n./main.go:5:2: undefined: nonExistentFunction",
		},
		{
			name: "Multiple error lines",
			outputLines: []string{
				"./main.go:5:2: undefined: functionA",
				"./main.go:10:3: undefined: functionB",
				"FAIL\texample.com/pkg\t0.123s",
			},
			expectedResult: "./main.go:5:2: undefined: functionA\n./main.go:10:3: undefined: functionB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def := &GoTestDefinition{
				packageErrors: make(map[string][]string),
			}

			// Simulate adding error lines
			packageName := "example.com/pkg"
			for _, line := range tt.outputLines {
				if def.isErrorOutput(strings.TrimSpace(line)) {
					def.packageErrors[packageName] = append(def.packageErrors[packageName], strings.TrimSpace(line))
				}
			}

			// Construct error message
			result := def.constructErrorMessage(packageName)
			if result != tt.expectedResult {
				t.Errorf("Expected %q, got %q", tt.expectedResult, result)
			}
		})
	}
}

// TestGoTestDefinition_IsErrorOutput tests error line filtering
func TestGoTestDefinition_IsErrorOutput(t *testing.T) {
	def := &GoTestDefinition{}

	tests := []struct {
		line     string
		expected bool
	}{
		{"No test mode selected", true},
		{"./main.go:5:2: undefined: function", true},
		{"syntax error: unexpected token", true},
		{"FAIL\texample.com/pkg\t0.854s", false},
		{"ok  \texample.com/pkg\t0.123s", false},
		{"?   \texample.com/pkg\t[no test files]", false},
		{"coverage: 85.2% of statements", false},
		{"=== RUN TestSomething", false},
		{"--- PASS: TestSomething", false},
		{"--- FAIL: TestSomething", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			result := def.isErrorOutput(tt.line)
			if result != tt.expected {
				t.Errorf("isErrorOutput(%q) = %v, expected %v", tt.line, result, tt.expected)
			}
		})
	}
}