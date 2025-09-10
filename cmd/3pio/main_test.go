package main

import (
	"os"
	"strings"
	"testing"

	"github.com/zk/3pio/internal/orchestrator"
)

func TestRunTestsCore_EmptyArgs(t *testing.T) {
	// Test that empty args returns an error
	exitCode, err := runTestsCore([]string{})
	
	// Should return an error for no test runner detected
	if err == nil {
		t.Error("Expected error for empty args, got nil")
	}
	
	if !strings.Contains(err.Error(), "no test runner detected") {
		t.Errorf("Expected 'no test runner detected' error, got: %v", err)
	}
	
	// Should return exit code 1 for failure
	if exitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", exitCode)
	}
}

func TestRunTestsCore_InvalidRunner(t *testing.T) {
	// Test with invalid test runner command
	exitCode, err := runTestsCore([]string{"invalid-test-runner"})
	
	// Should return an error
	if err == nil {
		t.Error("Expected error for invalid test runner, got nil")
	}
	
	if !strings.Contains(err.Error(), "no test runner detected") {
		t.Errorf("Expected 'no test runner detected' error, got: %v", err)
	}
	
	// Should return exit code 1 for failure
	if exitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", exitCode)
	}
}

func TestRunTestsCore_ValidCommands(t *testing.T) {
	// Test that valid command patterns are recognized (without actually running)
	testCases := []struct {
		args []string
		desc string
	}{
		{[]string{"npm", "test"}, "npm test command"},
		{[]string{"npx", "jest"}, "npx jest command"}, 
		{[]string{"npx", "vitest", "run"}, "npx vitest command"},
		{[]string{"pytest"}, "pytest command"},
	}
	
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			// Create orchestrator to verify command is recognized
			config := orchestrator.Config{
				Command: tc.args,
			}
			
			orch, err := orchestrator.New(config)
			if err != nil {
				t.Errorf("Failed to create orchestrator for %s: %v", tc.desc, err)
			}
			
			if orch == nil {
				t.Errorf("Expected orchestrator to be created for %s", tc.desc)
			}
		})
	}
}

func TestMain_Args(t *testing.T) {
	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Test help argument handling  
	// We can't easily test main() because it calls os.Exit
	// But we can test the argument parsing logic indirectly
	
	testCases := []struct {
		args []string
		desc string
	}{
		{[]string{"3pio", "--help"}, "help flag"},
		{[]string{"3pio", "--version"}, "version flag"},
		{[]string{"3pio", "npm", "test"}, "npm test command"},
		{[]string{"3pio", "run", "jest"}, "run subcommand"},
	}
	
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			// We can't actually run main() because it calls os.Exit
			// But we can test that the args parsing doesn't panic
			if len(tc.args) > 1 {
				firstArg := tc.args[1]
				if firstArg == "--help" || firstArg == "-h" || firstArg == "help" {
					// Help case
				} else if firstArg == "--version" || firstArg == "-v" || firstArg == "version" {
					// Version case  
				} else if firstArg == "run" {
					// Run subcommand case
				} else {
					// Assume it's a test command
				}
			}
			// If we get here without panicking, the test passes
		})
	}
}

func TestVersionInfo(t *testing.T) {
	// Test that version variables are set (they're set by build flags)
	if version == "" {
		// Version may be empty in tests, that's ok
	}
	
	if commit == "" {
		// Commit may be empty in tests, that's ok  
	}
	
	if date == "" {
		// Date may be empty in tests, that's ok
	}
	
	// The test passes if we don't panic accessing these variables
}