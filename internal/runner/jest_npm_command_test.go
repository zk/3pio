package runner

import (
	"testing"
)

func TestJestDefinition_BuildCommand_NPM(t *testing.T) {
	jest := NewJestDefinition()
	
	tests := []struct {
		name     string
		args     []string
		expected []string
	}{
		{
			name:     "npm test command should use -- separator",
			args:     []string{"npm", "test"},
			expected: []string{"npm", "test", "--", "--reporters", "/fake/adapter/path"},
		},
		{
			name:     "npm run test command should use -- separator",
			args:     []string{"npm", "run", "test"},
			expected: []string{"npm", "run", "test", "--", "--reporters", "/fake/adapter/path"},
		},
		{
			name:     "yarn test command should use -- separator",
			args:     []string{"yarn", "test"},
			expected: []string{"yarn", "test", "--", "--reporters", "/fake/adapter/path"},
		},
		{
			name:     "direct jest command should not use -- separator",
			args:     []string{"npx", "jest"},
			expected: []string{"npx", "jest", "--reporters", "/fake/adapter/path"},
		},
		{
			name:     "jest with test files should use -- separator before files",
			args:     []string{"npx", "jest", "math.test.js"},
			expected: []string{"npx", "jest", "--reporters", "/fake/adapter/path", "--", "math.test.js"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := jest.BuildCommand(tt.args, "/fake/adapter/path")
			
			if len(result) != len(tt.expected) {
				t.Errorf("BuildCommand() length = %v, want %v", len(result), len(tt.expected))
				t.Errorf("Got:      %v", result)
				t.Errorf("Expected: %v", tt.expected)
				return
			}
			
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("BuildCommand() result[%d] = %v, want %v", i, v, tt.expected[i])
					t.Errorf("Got:      %v", result)
					t.Errorf("Expected: %v", tt.expected)
					break
				}
			}
		})
	}
}

func TestJestDefinition_BuildCommand_Existing_Args(t *testing.T) {
	jest := NewJestDefinition()
	
	tests := []struct {
		name     string
		args     []string
		expected []string
	}{
		{
			name:     "npm test with existing -- separator",
			args:     []string{"npm", "test", "--", "--verbose"},
			expected: []string{"npm", "test", "--", "--reporters", "/fake/adapter/path", "--verbose"},
		},
		{
			name:     "npm test with jest options after --",
			args:     []string{"npm", "test", "--", "--watchAll", "false"},
			expected: []string{"npm", "test", "--", "--reporters", "/fake/adapter/path", "--watchAll", "false"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := jest.BuildCommand(tt.args, "/fake/adapter/path")
			
			if len(result) != len(tt.expected) {
				t.Errorf("BuildCommand() length = %v, want %v", len(result), len(tt.expected))
				t.Errorf("Got:      %v", result)
				t.Errorf("Expected: %v", tt.expected)
				return
			}
			
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("BuildCommand() result[%d] = %v, want %v", i, v, tt.expected[i])
					t.Errorf("Got:      %v", result)
					t.Errorf("Expected: %v", tt.expected)
					break
				}
			}
		})
	}
}