package definitions

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/zk/3pio/internal/logger"
)

// TestGoTestIncompleteTestHandling verifies that incomplete tests are properly handled
func TestGoTestIncompleteTestHandling(t *testing.T) {
	// Create a temporary IPC file
	tmpDir := t.TempDir()
	ipcPath := tmpDir + "/test.jsonl"

	// Create logger
	log, _ := logger.NewFileLogger()
	defer func() { _ = log.Close() }()

	// Create GoTestDefinition
	g := NewGoTestDefinition(log)

	// Simulate a test run that crashes mid-execution
	// This is what go test -json outputs when a test crashes with t.Fatalf
	jsonOutput := `{"Time":"2025-09-20T17:10:25.328640-10:00","Action":"run","Package":"github.com/zk/3pio/internal/ipc","Test":"TestManager_HandleUnknownEventTypes"}
{"Time":"2025-09-20T17:10:25.328644-10:00","Action":"output","Package":"github.com/zk/3pio/internal/ipc","Test":"TestManager_HandleUnknownEventTypes","Output":"=== RUN   TestManager_HandleUnknownEventTypes\n"}
{"Time":"2025-09-20T17:10:25.328861-10:00","Action":"output","Package":"github.com/zk/3pio/internal/ipc","Test":"TestManager_HandleUnknownEventTypes","Output":"    unknown_event_test.go:80: The foo didn't bar\n"}
`

	// Process the output
	reader := bytes.NewReader([]byte(jsonOutput))
	err := g.ProcessOutput(reader, ipcPath)
	if err != nil {
		t.Fatalf("ProcessOutput failed: %v", err)
	}

	// Read the IPC file to verify incomplete test was marked as ERROR
	data, err := os.ReadFile(ipcPath)
	if err != nil {
		t.Fatalf("Failed to read IPC file: %v", err)
	}

	// Parse all IPC events
	var foundErrorEvent bool
	var errorMessage string
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		var event map[string]interface{}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}

		if event["eventType"] == "testCase" {
			payload := event["payload"].(map[string]interface{})
			if payload["testName"] == "TestManager_HandleUnknownEventTypes" {
				status := payload["status"].(string)
				if status == "ERROR" {
					foundErrorEvent = true
					// Error is nested as an object with "message" field
					if errObj, ok := payload["error"].(map[string]interface{}); ok {
						if msg, ok := errObj["message"].(string); ok {
							errorMessage = msg
						}
					}
				}
			}
		}
	}

	if !foundErrorEvent {
		t.Error("Expected ERROR event for incomplete test but didn't find one")
	}
	t.Logf("IPC events:\n%s", data)

	if !strings.Contains(errorMessage, "The foo didn't bar") && !strings.Contains(errorMessage, "unknown_event_test.go:80: The foo didn't bar") {
		t.Errorf("Expected error message to contain 'The foo didn't bar', got: %s", errorMessage)
	}
}

// TestGoTestIncompleteTestNoOutput verifies handling when test crashes with no output
func TestGoTestIncompleteTestNoOutput(t *testing.T) {
	// Create a temporary IPC file
	tmpDir := t.TempDir()
	ipcPath := tmpDir + "/test.jsonl"

	// Create logger
	log, _ := logger.NewFileLogger()
	defer func() { _ = log.Close() }()

	// Create GoTestDefinition
	g := NewGoTestDefinition(log)

	// Simulate a test run that starts but never completes and has no output
	jsonOutput := `{"Time":"2025-09-20T17:10:25.328640-10:00","Action":"start","Package":"github.com/example/pkg"}
{"Time":"2025-09-20T17:10:25.328644-10:00","Action":"run","Package":"github.com/example/pkg","Test":"TestCrashWithNoOutput"}
{"Time":"2025-09-20T17:10:25.328861-10:00","Action":"output","Package":"github.com/example/pkg","Test":"TestCrashWithNoOutput","Output":"=== RUN   TestCrashWithNoOutput\n"}
`

	// Process the output
	reader := bytes.NewReader([]byte(jsonOutput))
	err := g.ProcessOutput(reader, ipcPath)
	if err != nil {
		t.Fatalf("ProcessOutput failed: %v", err)
	}

	// Read the IPC file
	data, err := os.ReadFile(ipcPath)
	if err != nil {
		t.Fatalf("Failed to read IPC file: %v", err)
	}

	// Parse all IPC events
	var foundErrorEvent bool
	var errorMessage string
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		var event map[string]interface{}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}

		if event["eventType"] == "testCase" {
			payload := event["payload"].(map[string]interface{})
			if payload["testName"] == "TestCrashWithNoOutput" {
				status := payload["status"].(string)
				if status == "ERROR" {
					foundErrorEvent = true
					// Error is nested as an object with "message" field
					if errObj, ok := payload["error"].(map[string]interface{}); ok {
						if msg, ok := errObj["message"].(string); ok {
							errorMessage = msg
						}
					}
				}
			}
		}
	}

	if !foundErrorEvent {
		t.Error("Expected ERROR event for incomplete test but didn't find one")
		t.Logf("IPC events:\n%s", data)
	}

	// The error message should be the RUN output or generic message if truly empty
	if errorMessage == "" || (!strings.Contains(errorMessage, "RUN") && !strings.Contains(errorMessage, "did not complete")) {
		t.Errorf("Expected error message for incomplete test, got: %s", errorMessage)
	}
}
