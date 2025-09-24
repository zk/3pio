package integration_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestVitestV2NoDuplicateTestEvents verifies that Vitest v2 adapter path doesn't emit duplicate test events
func TestVitestV2NoDuplicateTestEvents(t *testing.T) {
	t.Parallel()

	// Use an isolated copy of the basic-vitest-v2 fixture
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	sourceFixtureDir := filepath.Join(filepath.Dir(filepath.Dir(cwd)), "tests", "fixtures", "basic-vitest-v2")
	if _, err := os.Stat(sourceFixtureDir); err != nil {
		t.Skipf("Fixture directory not found: %s", sourceFixtureDir)
	}

	tempDir := t.TempDir()
	fixtureDir := filepath.Join(tempDir, "basic-vitest-v2")
	if err := copyDir(sourceFixtureDir, fixtureDir); err != nil {
		t.Fatalf("Failed to copy fixture: %v", err)
	}

	// Clean up any previous runs
	if err := os.RemoveAll(filepath.Join(fixtureDir, ".3pio")); err != nil {
		t.Fatalf("Failed to remove fixture .3pio directory: %v", err)
	}

	// Run 3pio with Vitest v2 fixture
	_, _, exitCode := runBinary(t, fixtureDir, "npx", "vitest", "run")
	t.Logf("Exit code: %d", exitCode)

	// Find latest run dir and load IPC file
	runsDir := filepath.Join(fixtureDir, ".3pio", "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		t.Fatalf("Failed to read runs directory: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("No run directories found")
	}
	runDir := entries[len(entries)-1].Name()
	ipcPath := filepath.Join(runsDir, runDir, "ipc.jsonl")

	data, err := os.ReadFile(ipcPath)
	if err != nil {
		t.Fatalf("Failed to read IPC file: %v", err)
	}
	if len(data) == 0 {
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		t.Fatalf("IPC file at %s is empty; entries: %v", ipcPath, names)
	}

	// Count testCase events by unique identifier
	testEvents := make(map[string]int)
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var event map[string]any
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		if et, ok := event["eventType"].(string); !ok || et != "testCase" {
			continue
		}
		payload, ok := event["payload"].(map[string]any)
		if !ok {
			continue
		}
		tn, _ := payload["testName"].(string)
		pnRaw, _ := payload["parentNames"].([]any)
		parentNames := make([]string, 0, len(pnRaw))
		for _, v := range pnRaw {
			if s, ok := v.(string); ok {
				parentNames = append(parentNames, s)
			}
		}
		id := fmt.Sprintf("%s::%s", strings.Join(parentNames, "::"), tn)
		testEvents[id]++
	}

	// Ensure no duplicates
	for id, count := range testEvents {
		if count > 1 {
			t.Fatalf("duplicate testCase events for %s: %d", id, count)
		}
	}

	// Expected unique tests: math (2), string (3 including skipped) = 5
	if got := len(testEvents); got != 5 {
		t.Fatalf("expected 5 unique testCase events, got %d", got)
	}
}
