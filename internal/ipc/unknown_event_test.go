package ipc

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// mockLogger captures log messages for testing
type mockLogger struct {
	debugMessages []string
	errorMessages []string
}

func (l *mockLogger) Debug(format string, args ...interface{}) {
	l.debugMessages = append(l.debugMessages, strings.TrimSpace(fmt.Sprintf(format, args...)))
}

func (l *mockLogger) Error(format string, args ...interface{}) {
	l.errorMessages = append(l.errorMessages, strings.TrimSpace(fmt.Sprintf(format, args...)))
}

func TestManager_HandleUnknownEventTypes(t *testing.T) {
	// Create temporary IPC file
	tempDir := t.TempDir()
	ipcPath := filepath.Join(tempDir, "test.jsonl")
	
	logger := &mockLogger{}
	manager, err := NewManager(ipcPath, logger)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Cleanup()
	
	// Write various event types to the file
	events := []map[string]interface{}{
		{
			"eventType": "testCase",
			"payload": map[string]interface{}{
				"filePath": "/test/file.js",
				"testName": "should work",
				"status":   "PASS",
			},
		},
		{
			"eventType": "runComplete",
			"payload":   map[string]interface{}{},
		},
		{
			"eventType": "unknownEvent",
			"payload": map[string]interface{}{
				"someData": "value",
			},
		},
	}
	
	// Write events to file
	file, err := os.OpenFile(ipcPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}
	
	for _, event := range events {
		data, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("Failed to marshal event: %v", err)
		}
		if _, err := file.Write(data); err != nil {
			t.Fatalf("Failed to write event: %v", err)
		}
		if _, err := file.WriteString("\n"); err != nil {
			t.Fatalf("Failed to write newline: %v", err)
		}
	}
	file.Close()
	
	// Start watching
	if err := manager.WatchEvents(); err != nil {
		t.Fatalf("Failed to start watching: %v", err)
	}
	
	// Wait for events to be processed
	time.Sleep(100 * time.Millisecond)
	
	// Check that we received the testCase event
	var receivedEvents []EventType
	
	// Collect all events with timeout
	timeout := time.After(1 * time.Second)
	for len(receivedEvents) < 2 { // expect testCase + runComplete
		select {
		case event := <-manager.Events:
			receivedEvents = append(receivedEvents, event.Type())
		case <-timeout:
			break
		}
	}
	
	// Verify we got both testCase and runComplete events
	if len(receivedEvents) < 2 {
		t.Errorf("Expected at least 2 events (testCase, runComplete), got %d: %v", 
			len(receivedEvents), receivedEvents)
	} else {
		// Check for testCase event
		foundTestCase := false
		foundRunComplete := false
		for _, eventType := range receivedEvents {
			if eventType == EventTypeTestCase {
				foundTestCase = true
			}
			if eventType == EventTypeRunComplete {
				foundRunComplete = true
			}
		}
		
		if !foundTestCase {
			t.Errorf("Expected testCase event, got events: %v", receivedEvents)
		}
		if !foundRunComplete {
			t.Errorf("Expected runComplete event, got events: %v", receivedEvents)
		}
	}
	
	// Check that unknown events were logged as errors
	// runComplete should NOT be an error anymore, but unknownEvent should be
	foundRunComplete := false
	foundUnknownEvent := false
	
	for _, msg := range logger.errorMessages {
		if strings.Contains(msg, "Unknown event type: runComplete") {
			foundRunComplete = true
		}
		if strings.Contains(msg, "Unknown event type: unknownEvent") {
			foundUnknownEvent = true
		}
	}
	
	if foundRunComplete {
		t.Error("runComplete should be handled gracefully now, not logged as error")
		t.Logf("Error messages: %v", logger.errorMessages)
	}
	
	if !foundUnknownEvent {
		t.Error("Expected error log for unknownEvent event not found")
		t.Logf("Error messages: %v", logger.errorMessages)
	}
}

func TestManager_ShouldHandleRunCompleteGracefully(t *testing.T) {
	// This test verifies that runComplete events should be handled gracefully
	// rather than logged as errors, since they are legitimate completion markers
	
	tempDir := t.TempDir()
	ipcPath := filepath.Join(tempDir, "test.jsonl")
	
	logger := &mockLogger{}
	manager, err := NewManager(ipcPath, logger)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Cleanup()
	
	// Write a runComplete event
	event := map[string]interface{}{
		"eventType": "runComplete",
		"payload":   map[string]interface{}{},
	}
	
	file, err := os.OpenFile(ipcPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}
	
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}
	if _, err := file.Write(data); err != nil {
		t.Fatalf("Failed to write event: %v", err)
	}
	if _, err := file.WriteString("\n"); err != nil {
		t.Fatalf("Failed to write newline: %v", err)
	}
	file.Close()
	
	// Start watching
	if err := manager.WatchEvents(); err != nil {
		t.Fatalf("Failed to start watching: %v", err)
	}
	
	// Wait for events to be processed
	time.Sleep(100 * time.Millisecond)
	
	// After fix: runComplete should NOT generate an error log
	for _, msg := range logger.errorMessages {
		if strings.Contains(msg, "Unknown event type: runComplete") {
			t.Errorf("runComplete event should be handled gracefully, but got error: %s", msg)
		}
	}
}