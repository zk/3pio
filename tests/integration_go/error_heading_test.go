package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNoEmptyErrorHeading(t *testing.T) {
	// This test verifies that we don't show an empty "Error:" heading
	// when tests fail but there's no actual error message
	
	fixtureDir := filepath.Join("..", "fixtures", "basic-vitest")
	if _, err := os.Stat(fixtureDir); err != nil {
		t.Skipf("Fixture directory not found: %s", fixtureDir)
	}

	// Build path to 3pio binary
	cwd, _ := os.Getwd()
	threePioBinary := filepath.Join(filepath.Dir(filepath.Dir(cwd)), "build", "3pio")
	if _, err := os.Stat(threePioBinary); err != nil {
		t.Fatalf("3pio binary not found at %s. Run 'make build' first.", threePioBinary)
	}

	// Run a test that we know will fail
	cmd := exec.Command(threePioBinary, "npx", "vitest", "run", "string.test.js")
	cmd.Dir = fixtureDir
	
	output, _ := cmd.CombinedOutput()
	outputStr := string(output)
	
	// Check if "Error:" appears in the output
	if strings.Contains(outputStr, "Error:") {
		// If it does appear, make sure it's followed by actual content
		errorIndex := strings.Index(outputStr, "Error:")
		afterError := outputStr[errorIndex+6:] // Skip "Error:"
		
		// Check the next line after "Error:"
		lines := strings.Split(afterError, "\n")
		if len(lines) > 0 {
			firstLineAfterError := strings.TrimSpace(lines[0])
			
			// The first line after "Error:" should not be empty
			if firstLineAfterError == "" {
				// Check if the second line has content (in case of formatting)
				if len(lines) > 1 {
					secondLine := strings.TrimSpace(lines[1])
					if secondLine == "" || secondLine == "Test failures!" || strings.HasPrefix(secondLine, "Results:") {
						t.Errorf("Found empty 'Error:' heading with no actual error message\nOutput:\n%s", outputStr)
					}
				} else {
					t.Errorf("Found 'Error:' heading at end of output with no message\nOutput:\n%s", outputStr)
				}
			}
		}
	}
	
	// Verify that test failure message is still shown
	if !strings.Contains(outputStr, "Test failures!") && !strings.Contains(outputStr, "FAIL") {
		t.Errorf("Test failure indication not found in output:\n%s", outputStr)
	}
}

func TestErrorHeadingWithActualError(t *testing.T) {
	// This test verifies that when there IS an actual error (like config error),
	// the Error: heading is shown with the error message
	
	fixtureDir := filepath.Join("..", "fixtures", "jest-config-error")
	if _, err := os.Stat(fixtureDir); err != nil {
		t.Skipf("Fixture directory not found: %s", fixtureDir)
	}

	// Build path to 3pio binary
	cwd, _ := os.Getwd()
	threePioBinary := filepath.Join(filepath.Dir(filepath.Dir(cwd)), "build", "3pio")
	if _, err := os.Stat(threePioBinary); err != nil {
		t.Fatalf("3pio binary not found at %s. Run 'make build' first.", threePioBinary)
	}

	// Run npx jest which should fail with config error
	cmd := exec.Command(threePioBinary, "npx", "jest")
	cmd.Dir = fixtureDir
	
	output, _ := cmd.CombinedOutput()
	outputStr := string(output)
	
	// When there's a real error, Error: should appear with content
	if strings.Contains(outputStr, "Error:") {
		errorIndex := strings.Index(outputStr, "Error:")
		afterError := outputStr[errorIndex+6:] // Skip "Error:"
		
		// Should have actual error content mentioning the config issue
		if !strings.Contains(afterError, "preset") || !strings.Contains(afterError, "non-existent-preset") {
			t.Errorf("Error: heading found but without expected error details\nOutput:\n%s", outputStr)
		}
	} else {
		// For config errors, we expect to see the Error: heading
		t.Errorf("Expected 'Error:' heading for config error but not found\nOutput:\n%s", outputStr)
	}
}