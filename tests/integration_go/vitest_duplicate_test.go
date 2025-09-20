package integration_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestVitestNoDuplicateTestEvents verifies that Vitest adapter doesn't emit duplicate test events
func TestVitestNoDuplicateTestEvents(t *testing.T) {
	t.Parallel()

	// Use the basic-vitest fixture
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	fixtureDir := filepath.Join(filepath.Dir(filepath.Dir(cwd)), "tests", "fixtures", "basic-vitest")

	// Clean up any previous runs
	_ = os.RemoveAll(filepath.Join(fixtureDir, ".3pio"))

	// Run 3pio with Vitest
	_, _, exitCode := runBinary(t, fixtureDir, "npx", "vitest", "run")

	t.Logf("Exit code: %d", exitCode)

	// Should succeed (string.test.js has a failing test but we're not checking that here)
	// We just want to analyze the IPC events

	// Find the most recent run directory
	runsDir := filepath.Join(fixtureDir, ".3pio", "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		t.Fatalf("Failed to read runs directory: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("No run directories found")
	}

	// Get the most recent run (last entry)
	runDir := entries[len(entries)-1].Name()
	ipcPath := filepath.Join(runsDir, runDir, "ipc.jsonl")

	// Read and parse IPC events
	ipcData, err := os.ReadFile(ipcPath)
	if err != nil {
		t.Fatalf("Failed to read IPC file: %v", err)
	}

	// Count test events by unique identifier
	// Use a string key for the map
	testEvents := make(map[string]int)
	testDetails := make(map[string]struct {
		TestName    string
		ParentNames []string
	})
	lines := strings.Split(string(ipcData), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		var event map[string]interface{}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue // Skip malformed lines
		}

		if event["eventType"] == "testCase" {
			payload := event["payload"].(map[string]interface{})
			testName := payload["testName"].(string)

			// Convert parentNames to string slice
			parentNamesRaw := payload["parentNames"].([]interface{})
			parentNames := make([]string, len(parentNamesRaw))
			for i, name := range parentNamesRaw {
				parentNames[i] = name.(string)
			}

			// Create unique identifier
			identifier := fmt.Sprintf("%s::%s", strings.Join(parentNames, "::"), testName)
			testEvents[identifier]++
			testDetails[identifier] = struct {
				TestName    string
				ParentNames []string
			}{
				TestName:    testName,
				ParentNames: parentNames,
			}
		}
	}

	// Check for duplicates
	var duplicates []string
	for identifier, count := range testEvents {
		if count > 1 {
			duplicates = append(duplicates, identifier)
		}
	}

	if len(duplicates) > 0 {
		t.Errorf("Found %d duplicate test events:", len(duplicates))
		for _, identifier := range duplicates {
			details := testDetails[identifier]
			t.Errorf("  Test '%s' in '%v' emitted %d times",
				details.TestName, details.ParentNames, testEvents[identifier])
		}
	}

	// Verify we have the expected number of unique tests
	// basic-vitest has: 3 tests in math.test.js + 4 tests in string.test.js = 7 tests
	expectedTests := 7
	t.Logf("Found %d unique test events", len(testEvents))

	// Log all test events for debugging
	for identifier, count := range testEvents {
		t.Logf("  Test: %s (count: %d)", identifier, count)
	}

	if len(testEvents) != expectedTests {
		t.Errorf("Expected %d unique test events, got %d", expectedTests, len(testEvents))
	}
}