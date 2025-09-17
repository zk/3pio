package runner

import (
	"testing"

	"github.com/zk/3pio/internal/logger"
)

func TestIsPackageManager_FalsePositives(t *testing.T) {
	tests := []struct {
		name     string
		cmd      string
		expected bool
	}{
		// Legitimate package managers
		{"npm", "npm", true},
		{"yarn", "yarn", true},
		{"pnpm", "pnpm", true},
		{"bun", "bun", true},
		{"npm with path", "/usr/local/bin/npm", true},
		{"yarn with path", "/usr/local/bin/yarn", true},

		// False positives that should NOT match
		{"npm in middle of filename", "mynpm-tool", false},
		{"npm in path but different command", "/npm-workspace/mytool", false},
		{"yarn in middle of filename", "yarn-helper", false},
		{"pnpm substring", "mypnpm", false},
		{"bun substring", "bundle", false},
		{"npm-like directory in path", "/home/npm-user/scripts/test.sh", false},
		{"compound command with npm in name", "npm-audit-fix", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPackageManager(tt.cmd)
			if result != tt.expected {
				t.Errorf("isPackageManager(%q) = %v, want %v", tt.cmd, result, tt.expected)
			}
		})
	}
}

func TestManager_Detect_FalsePositives(t *testing.T) {
	// Create a test logger
	testLogger, err := logger.NewFileLogger()
	if err != nil {
		t.Fatalf("Failed to create test logger: %v", err)
	}
	defer func() { _ = testLogger.Close() }()

	m := NewManager(testLogger)

	tests := []struct {
		name        string
		command     []string
		shouldMatch bool
		description string
	}{
		// These should NOT be detected as package managers
		{
			name:        "npm-like tool name",
			command:     []string{"npm-audit", "fix"},
			shouldMatch: false,
			description: "Tool with npm in name shouldn't be detected as npm",
		},
		{
			name:        "path with npm directory",
			command:     []string{"/home/npm-workspace/run-tests.sh"},
			shouldMatch: false,
			description: "Path containing npm shouldn't be detected as npm",
		},
		{
			name:        "yarn-like command",
			command:     []string{"yarn-lock-check", "--strict"},
			shouldMatch: false,
			description: "Command with yarn prefix shouldn't be detected as yarn",
		},
		{
			name:        "bundler not bun",
			command:     []string{"bundle", "exec", "rspec"},
			shouldMatch: false,
			description: "bundle command shouldn't be detected as bun",
		},

		// Note: npm/yarn/pnpm by themselves won't match any test runner
		// They need a matching test runner in package.json
		// These tests are just verifying that false positives don't occur
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := m.Detect(tt.command)
			matched := err == nil

			if matched != tt.shouldMatch {
				if tt.shouldMatch {
					t.Errorf("Expected to match %v (%s), but got error: %v", tt.command, tt.description, err)
				} else {
					t.Errorf("Expected NO match for %v (%s), but it was incorrectly detected", tt.command, tt.description)
				}
			}
		})
	}
}
