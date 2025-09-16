package runner

import (
	"testing"
)

func TestJestDefinition_Matches_FalsePositives(t *testing.T) {
	jest := &JestDefinition{}

	tests := []struct {
		name     string
		command  []string
		expected bool
	}{
		// Should match
		{"jest command", []string{"jest"}, true},
		{"npx jest", []string{"npx", "jest"}, true},
		{"jest with args", []string{"jest", "--coverage"}, true},
		{"path to jest", []string{"/node_modules/.bin/jest"}, true},

		// Should NOT match
		{"jest-like tool", []string{"jest-codemods"}, false},
		{"path containing jest", []string{"/home/jest-user/mytool"}, false},
		{"jester command", []string{"jester"}, false},
		{"test-jest-helper", []string{"test-jest-helper"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := jest.Matches(tt.command)
			if result != tt.expected {
				t.Errorf("JestDefinition.Matches(%v) = %v, want %v", tt.command, result, tt.expected)
			}
		})
	}
}

func TestVitestDefinition_Matches_FalsePositives(t *testing.T) {
	vitest := &VitestDefinition{}

	tests := []struct {
		name     string
		command  []string
		expected bool
	}{
		// Should match
		{"vitest command", []string{"vitest"}, true},
		{"npx vitest", []string{"npx", "vitest"}, true},
		{"vitest with args", []string{"vitest", "run"}, true},
		{"path to vitest", []string{"/node_modules/.bin/vitest"}, true},

		// Should NOT match
		{"vitest-like tool", []string{"vitest-ui"}, false},
		{"path containing vitest", []string{"/home/vitest-user/mytool"}, false},
		{"vitester command", []string{"vitester"}, false},
		{"test-vitest-helper", []string{"test-vitest-helper"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := vitest.Matches(tt.command)
			if result != tt.expected {
				t.Errorf("VitestDefinition.Matches(%v) = %v, want %v", tt.command, result, tt.expected)
			}
		})
	}
}

func TestPytestDefinition_Matches_FalsePositives(t *testing.T) {
	pytest := &PytestDefinition{}

	tests := []struct {
		name     string
		command  []string
		expected bool
	}{
		// Should match
		{"pytest command", []string{"pytest"}, true},
		{"py.test command", []string{"py.test"}, true},
		{"pytest with args", []string{"pytest", "-v"}, true},
		{"path to pytest", []string{"/usr/local/bin/pytest"}, true},
		{"python -m pytest", []string{"python", "-m", "pytest"}, true},

		// Should NOT match
		{"pytest-like tool", []string{"pytest-cov"}, false},
		{"path containing pytest", []string{"/home/pytest-user/mytool"}, false},
		{"mypytest command", []string{"mypytest"}, false},
		{"test-pytest-helper", []string{"test-pytest-helper"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pytest.Matches(tt.command)
			if result != tt.expected {
				t.Errorf("PytestDefinition.Matches(%v) = %v, want %v", tt.command, result, tt.expected)
			}
		})
	}
}
