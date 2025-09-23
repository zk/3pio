package ipc

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/zk/3pio/internal/logger"
)

// TestManager_RaceConditionWithMockedWatcher tests the race condition by
// preventing the file watcher from delivering events after a certain point
func TestManager_RaceConditionWithMockedWatcher(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "ipc-race-mock-test")
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

	// Write initial events to the file BEFORE starting the watcher
	// This simulates events that exist when the manager starts
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

	// Start watching - this will read the initial 5 events
	if err := manager.WatchEvents(); err != nil {
		t.Fatalf("Failed to start watching: %v", err)
	}

	// Collect events in a goroutine
	var receivedEvents []Event
	done := make(chan struct{})
	go func() {
		defer close(done)
		for event := range manager.Events {
			receivedEvents = append(receivedEvents, event)
		}
	}()

	// Give the initial read time to complete
	time.Sleep(50 * time.Millisecond)

	// Now simulate the race: write more events but immediately cleanup
	// WITHOUT giving the file watcher time to notify
	file2, err := os.OpenFile(ipcPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to reopen file: %v", err)
	}

	// Write the late events that would get missed in the bug
	for i := 0; i < 3; i++ {
		event := map[string]interface{}{
			"eventType": "testCase",
			"payload": map[string]interface{}{
				"testName":    "late_test_" + string(rune('X'+i)),
				"parentNames": []string{"test_file.py"},
				"status":      "FAIL", // These failing tests were getting missed!
			},
		}
		data, _ := json.Marshal(event)
		_, _ = file2.Write(data)
		_, _ = file2.Write([]byte("\n"))
	}

	// Close the file but DON'T sync - this delays the OS notification
	_ = file2.Close()

	// IMMEDIATELY call cleanup - this is the race condition
	// Without our fix, the watcher wouldn't have time to see the new writes
	if err := manager.Cleanup(); err != nil {
		t.Errorf("Cleanup failed: %v", err)
	}

	// Wait for event processing to complete
	<-done

	// With the fix, we should have all 8 events
	// Without the fix, we'd only have the initial 5
	if len(receivedEvents) != 8 {
		t.Errorf("Expected 8 events, got %d", len(receivedEvents))
		t.Logf("This demonstrates the race condition where late events are missed!")
		t.Logf("Received events:")
		for i, evt := range receivedEvents {
			if testCase, ok := evt.(GroupTestCaseEvent); ok {
				t.Logf("  %d: %s (status: %s)", i+1, testCase.Payload.TestName, testCase.Payload.Status)
			}
		}
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

// TestManager_RaceConditionStressTest runs the race test many times to catch intermittent failures
func TestManager_RaceConditionStressTest(t *testing.T) {
	// Run the race test multiple times to increase chances of catching the bug
	failures := 0
	iterations := 100

	for i := 0; i < iterations; i++ {
		tmpDir, err := os.MkdirTemp("", "ipc-stress-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}

		ipcPath := filepath.Join(tmpDir, "test.jsonl")
		manager, err := NewManager(ipcPath, logger.NewTestLogger())
		if err != nil {
			_ = os.RemoveAll(tmpDir)
			t.Fatalf("Failed to create manager: %v", err)
		}

		// Write a burst of events
		file, err := os.OpenFile(ipcPath, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			_ = os.RemoveAll(tmpDir)
			t.Fatalf("Failed to open file: %v", err)
		}

		// Write 100 events rapidly
		for j := 0; j < 100; j++ {
			event := map[string]interface{}{
				"eventType": "testCase",
				"payload": map[string]interface{}{
					"testName":    "test_" + string(rune(j%26+'A')),
					"parentNames": []string{"test_file.py"},
					"status":      "PASS",
				},
			}
			data, _ := json.Marshal(event)
			_, _ = file.Write(data)
			_, _ = file.Write([]byte("\n"))
		}
		// Don't sync - let the OS buffer the writes
		_ = file.Close()

		// Start watching
		if err := manager.WatchEvents(); err != nil {
			_ = os.RemoveAll(tmpDir)
			t.Fatalf("Failed to start watching: %v", err)
		}

		// Count events
		eventCount := 0
		done := make(chan struct{})
		go func() {
			defer close(done)
			for range manager.Events {
				eventCount++
			}
		}()

		// Immediately cleanup (race condition)
		_ = manager.Cleanup()
		<-done

		// Check if we got all events
		if eventCount != 100 {
			failures++
			t.Logf("Iteration %d: Expected 100 events, got %d (missed %d)",
				i+1, eventCount, 100-eventCount)
		}

		_ = os.RemoveAll(tmpDir)
	}

	if failures > 0 {
		t.Errorf("Race condition detected in %d/%d iterations", failures, iterations)
		t.Logf("This proves the race condition exists without the fix!")
	} else {
		t.Logf("All %d iterations passed - fix is working!", iterations)
	}
}
