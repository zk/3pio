package ipc

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/zk/3pio/internal/logger"
)

// TestManager_RaceConditionLateEvents tests that all events are read even when
// written just before cleanup is called (reproduces the missing test bug)
func TestManager_RaceConditionLateEvents(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "ipc-race-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	ipcPath := filepath.Join(tmpDir, "test.jsonl")

	// Create IPC manager
	manager, err := NewManager(ipcPath, logger.NewTestLogger())
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Start watching
	if err := manager.WatchEvents(); err != nil {
		t.Fatalf("Failed to start watching: %v", err)
	}

	// Collect events
	var receivedEvents []Event
	done := make(chan struct{})
	go func() {
		defer close(done)
		for event := range manager.Events {
			receivedEvents = append(receivedEvents, event)
		}
	}()

	// Write initial events
	file, err := os.OpenFile(ipcPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}

	// Write some early events
	for i := 0; i < 5; i++ {
		event := map[string]interface{}{
			"eventType": "testCase",
			"payload": map[string]interface{}{
				"testName":    "early_test_" + string(rune('A'+i)),
				"parentNames": []string{"test_file.py"},
				"status":      "PASS",
			},
		}
		data, _ := json.Marshal(event)
		_, _ = file.Write(data)
		_, _ = file.Write([]byte("\n"))
	}
	_ = file.Sync()
	_ = file.Close()

	// Give watcher time to process early events
	time.Sleep(50 * time.Millisecond)

	// Now simulate the race condition: write late events just before cleanup
	// Open file again to write more events (simulating another process)
	file2, err := os.OpenFile(ipcPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to reopen file: %v", err)
	}

	for i := 0; i < 3; i++ {
		event := map[string]interface{}{
			"eventType": "testCase",
			"payload": map[string]interface{}{
				"testName":    "late_test_" + string(rune('X'+i)),
				"parentNames": []string{"test_file.py"},
				"status":      "FAIL", // These are the failing tests that get missed!
			},
		}
		data, _ := json.Marshal(event)
		_, _ = file2.Write(data)
		_, _ = file2.Write([]byte("\n"))
	}
	// Don't sync or close - simulate abrupt process end
	// file2.Sync() // INTENTIONALLY OMITTED
	// file2.Close() // INTENTIONALLY OMITTED

	// Immediately call cleanup (simulating process exit)
	// The bug: cleanup is called but doesn't read remaining unprocessed events
	if err := manager.Cleanup(); err != nil {
		t.Errorf("Cleanup failed: %v", err)
	}

	// Wait for event processing to complete
	<-done

	// Check results - we should have all 8 events
	if len(receivedEvents) != 8 {
		t.Errorf("Expected 8 events, got %d", len(receivedEvents))
		t.Logf("Received events:")
		for i, evt := range receivedEvents {
			if testCase, ok := evt.(GroupTestCaseEvent); ok {
				t.Logf("  %d: %s (status: %s)", i+1, testCase.Payload.TestName, testCase.Payload.Status)
			}
		}
		t.Logf("\nMissing the late events demonstrates the race condition!")
	}

	// Check specifically for the late failing tests
	failCount := 0
	for _, evt := range receivedEvents {
		if testCase, ok := evt.(GroupTestCaseEvent); ok {
			if testCase.Payload.Status == "FAIL" {
				failCount++
			}
		}
	}

	if failCount != 3 {
		t.Errorf("Expected 3 failing tests, got %d - late events were not read before cleanup!", failCount)
	}
}
