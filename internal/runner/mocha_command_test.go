package runner

import (
	"reflect"
	"testing"
)

func TestMochaBuildCommand(t *testing.T) {
	m := NewMochaDefinition()
	adapter := "/tmp/mocha-adapter.js"

	tests := []struct {
		name     string
		in       []string
		expected []string
	}{
		{
			name:     "direct mocha",
			in:       []string{"mocha"},
			expected: []string{"mocha", "--reporter", adapter},
		},
		{
			name:     "direct mocha with glob",
			in:       []string{"mocha", "test/**/*.spec.js"},
			expected: []string{"mocha", "test/**/*.spec.js", "--reporter", adapter},
		},
		{
			name:     "npx mocha",
			in:       []string{"npx", "mocha"},
			expected: []string{"npx", "mocha", "--reporter", adapter},
		},
		{
			name:     "pnpm exec mocha",
			in:       []string{"pnpm", "exec", "mocha"},
			expected: []string{"pnpm", "exec", "mocha", "--reporter", adapter},
		},
		{
			name:     "npm test script (needs --)",
			in:       []string{"npm", "test"},
			expected: []string{"npm", "test", "--", "--reporter", adapter},
		},
		{
			name:     "yarn script (needs --)",
			in:       []string{"yarn", "test"},
			expected: []string{"yarn", "test", "--", "--reporter", adapter},
		},
		{
			name:     "bun script (needs --)",
			in:       []string{"bun", "test"},
			expected: []string{"bun", "test", "--", "--reporter", adapter},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.BuildCommand(tt.in, adapter)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Fatalf("BuildCommand mismatch\n in:  %#v\n got: %#v\n want: %#v", tt.in, got, tt.expected)
			}
		})
	}
}
