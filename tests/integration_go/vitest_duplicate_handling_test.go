package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestVitestDuplicateHandling verifies that duplicate test names are handled correctly
// by appending unique suffixes to subsequent occurrences
func TestVitestDuplicateHandling(t *testing.T) {
	t.Parallel()

	// Use the vitest-duplicates fixture
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	fixtureDir := filepath.Join(filepath.Dir(filepath.Dir(cwd)), "tests", "fixtures", "vitest-duplicates")

	// Clean up any previous runs
	_ = os.RemoveAll(filepath.Join(fixtureDir, ".3pio"))

	// Run 3pio with Vitest
	_, _, exitCode := runBinary(t, fixtureDir, "npx", "vitest", "run")

	if exitCode != 0 {
		t.Logf("Exit code: %d (expected 0)", exitCode)
	}

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

	// Track all test names to verify unique suffixes were added
	testNames := make(map[string]bool)
	var allTestNames []string
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

			// Convert parentNames to identify the context
			parentNamesRaw := payload["parentNames"].([]interface{})
			parentNames := make([]string, len(parentNamesRaw))
			for i, name := range parentNamesRaw {
				parentNames[i] = name.(string)
			}

			// Create full identifier
			fullName := strings.Join(append(parentNames, testName), " > ")

			if testNames[fullName] {
				t.Errorf("Duplicate test identifier found: %s", fullName)
			}
			testNames[fullName] = true
			allTestNames = append(allTestNames, testName)
		}
	}

	// Expected test patterns
	expectedPatterns := []struct {
		base           string
		count          int
		desc           string
		expectSuffixes bool // Whether we expect suffixes for duplicates
	}{
		{"duplicate test name", 3, "within same describe block", true},
		{"another duplicate", 2, "another pair in same describe block", true},
		{"cross-suite duplicate", 2, "across different describe blocks", false}, // Different suites = different tests
		{"unique test name", 1, "unique test should appear once", false},
	}

	// Verify each pattern
	for _, pattern := range expectedPatterns {
		var found []string
		for _, name := range allTestNames {
			if strings.HasPrefix(name, pattern.base) {
				found = append(found, name)
			}
		}

		t.Logf("Pattern '%s': found %d occurrences", pattern.base, len(found))
		for _, name := range found {
			t.Logf("  - %s", name)
		}

		if len(found) != pattern.count {
			t.Errorf("Expected %d tests matching '%s' (%s), found %d",
				pattern.count, pattern.base, pattern.desc, len(found))
		}

		// For duplicates, verify they have unique suffixes if expected
		if pattern.count > 1 {
			if pattern.expectSuffixes {
				// For same-suite duplicates, all names should be unique (due to suffixes)
				uniqueNames := make(map[string]bool)
				for _, name := range found {
					if uniqueNames[name] {
						t.Errorf("Duplicate test name not properly suffixed: %s", name)
					}
					uniqueNames[name] = true
				}
				// Check that we have the base name and suffixed versions
				hasBase := false
				hasSuffixed := false
				for _, name := range found {
					if name == pattern.base {
						hasBase = true
					} else if strings.HasPrefix(name, pattern.base+" [") {
						hasSuffixed = true
					}
				}

				if !hasBase {
					t.Errorf("Missing base test name '%s' (first occurrence should keep original name)", pattern.base)
				}
				if !hasSuffixed {
					t.Errorf("Missing suffixed versions of '%s'", pattern.base)
				}
			} else {
				// For cross-suite duplicates, all should have the same base name (no suffixes)
				for _, name := range found {
					if name != pattern.base {
						t.Errorf("Cross-suite test should not have suffix, got '%s' instead of '%s'", name, pattern.base)
					}
				}
			}
		}
	}

	// Total expected tests: 10
	// - 3x "duplicate test name"
	// - 1x "unique test name"
	// - 2x "another duplicate"
	// - 2x "cross-suite duplicate"
	expectedTotal := 8
	if len(testNames) != expectedTotal {
		t.Errorf("Expected %d unique test identifiers, got %d", expectedTotal, len(testNames))
	}
}
