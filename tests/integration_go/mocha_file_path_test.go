package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zk/3pio/tests/testutil"
)

// TestMochaFilePathHandling verifies that mocha adapter correctly identifies
// file paths for tests from different files (regression test for the bug where
// all tests were reported as coming from the first file)
func TestMochaFilePathHandling(t *testing.T) {
	// Ensure npm/npx and mocha are available
	if _, err := testutil.LookPath("npm"); err != nil {
		t.Skip("npm not found in PATH")
	}
	if err := testutil.CommandAvailable("npx", "mocha", "--version"); err != nil {
		t.Skipf("mocha command failed: %v", err)
	}

	fixtureDir := filepath.Join("..", "fixtures", "basic-mocha")
	if _, err := os.Stat(fixtureDir); os.IsNotExist(err) {
		t.Skipf("fixture %s not found", fixtureDir)
	}

	// Run 3pio with npx mocha against both spec files
	result := testutil.RunThreepio(t, fixtureDir, []string{"npx", "mocha", "math.spec.js", "string.spec.js"}...)

	runDir := filepath.Join(fixtureDir, ".3pio", "runs", result.RunID)
	ipcFile := filepath.Join(runDir, "ipc.jsonl")

	// Read and parse IPC events
	ipcData, err := os.ReadFile(ipcFile)
	if err != nil {
		t.Fatalf("Failed to read IPC file: %v", err)
	}

	// Count test cases per file
	fileCounts := make(map[string]int)
	mathTestFound := false
	stringTestFound := false

	lines := strings.Split(string(ipcData), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		var event struct {
			EventType string `json:"eventType"`
			Payload   struct {
				TestName    string   `json:"testName"`
				ParentNames []string `json:"parentNames"`
			} `json:"payload"`
		}

		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}

		if event.EventType == "testCase" && len(event.Payload.ParentNames) > 0 {
			filePath := event.Payload.ParentNames[0]
			fileCounts[filePath]++

			// Check if math.spec.js tests are associated with math.spec.js
			if strings.Contains(filePath, "math.spec.js") &&
				strings.Contains(event.Payload.TestName, "adds numbers") {
				mathTestFound = true
			}

			// Check if string.spec.js tests are associated with string.spec.js
			if strings.Contains(filePath, "string.spec.js") &&
				strings.Contains(event.Payload.TestName, "uppercases") {
				stringTestFound = true
			}
		}
	}

	// Verify we have events from both files
	mathFileFound := false
	stringFileFound := false
	for path := range fileCounts {
		if strings.Contains(path, "math.spec.js") {
			mathFileFound = true
		}
		if strings.Contains(path, "string.spec.js") {
			stringFileFound = true
		}
	}

	if !mathFileFound {
		t.Error("Expected to find tests from math.spec.js")
	}
	if !stringFileFound {
		t.Error("Expected to find tests from string.spec.js")
	}

	// Verify specific tests are associated with correct files
	if !mathTestFound {
		t.Error("Math tests should be associated with math.spec.js file")
	}
	if !stringTestFound {
		t.Error("String tests should be associated with string.spec.js file")
	}

	// Verify we don't have all tests attributed to a single file
	if len(fileCounts) < 2 {
		t.Errorf("Expected tests from at least 2 files, but got %d", len(fileCounts))
		for path, count := range fileCounts {
			t.Logf("  %s: %d tests", path, count)
		}
	}
}
