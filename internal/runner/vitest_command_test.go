package runner

import (
	"testing"
)

func TestVitestBuildCommand(t *testing.T) {
	vitest := NewVitestDefinition()
	adapterPath := "/tmp/adapter.js"

	tests := []struct {
		name     string
		args     []string
		expected []string
	}{
		{
			name: "direct vitest command",
			args: []string{"npx", "vitest", "run"},
			expected: []string{"npx", "vitest", "--reporter", "/tmp/adapter.js", "--reporter", "default", "run"},
		},
		{
			name: "npm test command",
			args: []string{"npm", "test"},
			// For npm test, we should use -- separator to pass flags to underlying script
			expected: []string{"npm", "test", "--", "--reporter", "/tmp/adapter.js", "--reporter", "default"},
		},
		{
			name: "vitest with files",
			args: []string{"npx", "vitest", "run", "test1.js", "test2.js"},
			expected: []string{"npx", "vitest", "--reporter", "/tmp/adapter.js", "--reporter", "default", "run", "test1.js", "test2.js"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := vitest.BuildCommand(tt.args, adapterPath)
			
			if len(result) != len(tt.expected) {
				t.Errorf("Length mismatch: got %d, expected %d\nGot: %v\nExpected: %v", 
					len(result), len(tt.expected), result, tt.expected)
				return
			}
			
			for i, arg := range result {
				if arg != tt.expected[i] {
					t.Errorf("Arg mismatch at index %d: got %q, expected %q\nFull result: %v\nExpected: %v", 
						i, arg, tt.expected[i], result, tt.expected)
					return
				}
			}
		})
	}
}

func TestVitestNpmCommandIssue(t *testing.T) {
	// This test specifically reproduces the bug where npm test fails
	vitest := NewVitestDefinition()
	adapterPath := "/tmp/adapter.js"
	
	// This is the failing case - npm test with vitest in package.json
	args := []string{"npm", "test"}
	result := vitest.BuildCommand(args, adapterPath)
	
	// After the fix, npm test should use -- separator
	expected := []string{"npm", "test", "--", "--reporter", "/tmp/adapter.js", "--reporter", "default"}
	
	if len(result) != len(expected) {
		t.Errorf("Command length mismatch.\nGot: %v\nExpected: %v", result, expected)
		return
	}
	
	for i, arg := range result {
		if arg != expected[i] {
			t.Errorf("Arg mismatch at index %d: got %q, expected %q\nFull result: %v\nExpected: %v", 
				i, arg, expected[i], result, expected)
			return
		}
	}
}

func TestVitestNpmCommandFormats(t *testing.T) {
	// Test various npm command formats that should all work
	vitest := NewVitestDefinition()
	adapterPath := "/tmp/adapter.js"
	
	testCases := []struct {
		name     string
		args     []string
		expected []string
	}{
		{
			name:     "npm test",
			args:     []string{"npm", "test"},
			expected: []string{"npm", "test", "--", "--reporter", "/tmp/adapter.js", "--reporter", "default"},
		},
		{
			name:     "npm run test", 
			args:     []string{"npm", "run", "test"},
			expected: []string{"npm", "run", "test", "--", "--reporter", "/tmp/adapter.js", "--reporter", "default"},
		},
		{
			name:     "npm run custom-test",
			args:     []string{"npm", "run", "custom-test"},
			expected: []string{"npm", "run", "custom-test", "--", "--reporter", "/tmp/adapter.js", "--reporter", "default"},
		},
		{
			name:     "npm test with existing --",
			args:     []string{"npm", "test", "--", "--some-flag"},
			expected: []string{"npm", "test", "--", "--some-flag", "--reporter", "/tmp/adapter.js", "--reporter", "default"},
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := vitest.BuildCommand(tc.args, adapterPath)
			
			if len(result) != len(tc.expected) {
				t.Errorf("Length mismatch for %s: got %d, expected %d\nGot: %v\nExpected: %v", 
					tc.name, len(result), len(tc.expected), result, tc.expected)
				return
			}
			
			for i, arg := range result {
				if arg != tc.expected[i] {
					t.Errorf("Arg mismatch for %s at index %d: got %q, expected %q\nFull result: %v\nExpected: %v", 
						tc.name, i, arg, tc.expected[i], result, tc.expected)
					return
				}
			}
		})
	}
}