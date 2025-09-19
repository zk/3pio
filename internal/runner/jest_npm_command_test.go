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
		// NPM variations
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
			name:     "npm run test:unit custom script",
			args:     []string{"npm", "run", "test:unit"},
			expected: []string{"npm", "run", "test:unit", "--", "--reporters", "/fake/adapter/path"},
		},
		{
			name:     "npm exec jest command",
			args:     []string{"npm", "exec", "jest"},
			expected: []string{"npm", "exec", "jest", "--reporters", "/fake/adapter/path"},
		},
		{
			name:     "npm test with existing -- and coverage",
			args:     []string{"npm", "test", "--", "--coverage"},
			expected: []string{"npm", "test", "--", "--coverage", "--reporters", "/fake/adapter/path"},
		},
		{
			name:     "npm test with multiple flags after --",
			args:     []string{"npm", "test", "--", "--watch", "--coverage", "--verbose"},
			expected: []string{"npm", "test", "--", "--watch", "--coverage", "--verbose", "--reporters", "/fake/adapter/path"},
		},

		// Yarn variations - Yarn doesn't need -- separator for scripts
		{
			name:     "yarn test command should NOT use -- separator",
			args:     []string{"yarn", "test"},
			expected: []string{"yarn", "test", "--reporters", "/fake/adapter/path"},
		},
		{
			name:     "yarn run test",
			args:     []string{"yarn", "run", "test"},
			expected: []string{"yarn", "run", "test", "--reporters", "/fake/adapter/path"},
		},
		{
			name:     "yarn test:ci custom script",
			args:     []string{"yarn", "test:ci"},
			expected: []string{"yarn", "test:ci", "--reporters", "/fake/adapter/path"},
		},
		{
			name:     "yarn jest direct",
			args:     []string{"yarn", "jest"},
			expected: []string{"yarn", "jest", "--reporters", "/fake/adapter/path"},
		},
		{
			name:     "yarn test with watch disabled",
			args:     []string{"yarn", "test", "--watchAll=false"},
			expected: []string{"yarn", "test", "--watchAll=false", "--reporters", "/fake/adapter/path"},
		},

		// PNPM variations
		{
			name:     "pnpm test",
			args:     []string{"pnpm", "test"},
			expected: []string{"pnpm", "test", "--", "--reporters", "/fake/adapter/path"},
		},
		{
			name:     "pnpm run test",
			args:     []string{"pnpm", "run", "test"},
			expected: []string{"pnpm", "run", "test", "--", "--reporters", "/fake/adapter/path"},
		},
		{
			name:     "pnpm exec jest",
			args:     []string{"pnpm", "exec", "jest"},
			expected: []string{"pnpm", "exec", "jest", "--", "--reporters", "/fake/adapter/path"},
		},
		{
			name:     "pnpm test with -- and test pattern",
			args:     []string{"pnpm", "test", "--", "src/**/*.test.js"},
			expected: []string{"pnpm", "test", "--", "src/**/*.test.js", "--reporters", "/fake/adapter/path"},
		},

		// Bun variations
		{
			name:     "bun test (might use bun's test runner)",
			args:     []string{"bun", "test"},
			expected: []string{"bun", "test", "--", "--reporters", "/fake/adapter/path"},
		},
		{
			name:     "bun run test",
			args:     []string{"bun", "run", "test"},
			expected: []string{"bun", "run", "test", "--", "--reporters", "/fake/adapter/path"},
		},
		{
			name:     "bunx jest",
			args:     []string{"bunx", "jest"},
			expected: []string{"bunx", "jest", "--reporters", "/fake/adapter/path"},
		},
		{
			name:     "bun jest direct",
			args:     []string{"bun", "jest"},
			expected: []string{"bun", "jest", "--", "--reporters", "/fake/adapter/path"},
		},
		{
			name:     "bunx jest with config",
			args:     []string{"bunx", "jest", "--config=jest.config.js"},
			expected: []string{"bunx", "jest", "--reporters", "/fake/adapter/path", "--config=jest.config.js"},
		},

		// Direct Jest invocations
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
		{
			name:     "npx jest with multiple test files",
			args:     []string{"npx", "jest", "math.test.js", "string.test.js"},
			expected: []string{"npx", "jest", "--reporters", "/fake/adapter/path", "--", "math.test.js", "string.test.js"},
		},
		{
			name:     "npx jest with flags and files",
			args:     []string{"npx", "jest", "--coverage", "math.test.js"},
			expected: []string{"npx", "jest", "--reporters", "/fake/adapter/path", "--coverage", "--", "math.test.js"},
		},
		{
			name:     "npx jest with watch mode",
			args:     []string{"npx", "jest", "--watch"},
			expected: []string{"npx", "jest", "--reporters", "/fake/adapter/path", "--watch"},
		},
		{
			name:     "npx jest with maxWorkers",
			args:     []string{"npx", "jest", "--maxWorkers=4"},
			expected: []string{"npx", "jest", "--reporters", "/fake/adapter/path", "--maxWorkers=4"},
		},
		{
			name:     "npx with --no-install flag before jest",
			args:     []string{"npx", "--no-install", "jest"},
			expected: []string{"npx", "--no-install", "jest", "--reporters", "/fake/adapter/path"},
		},
		{
			name:     "npx with package version",
			args:     []string{"npx", "jest@29"},
			expected: []string{"npx", "jest@29", "--reporters", "/fake/adapter/path"},
		},

		// Node direct execution
		{
			name:     "node with jest from node_modules",
			args:     []string{"node", "node_modules/.bin/jest"},
			expected: []string{"node", "node_modules/.bin/jest", "--reporters", "/fake/adapter/path"},
		},
		{
			name:     "node with jest and test pattern",
			args:     []string{"node", "node_modules/.bin/jest", "src/**/*.spec.js"},
			expected: []string{"node", "node_modules/.bin/jest", "--reporters", "/fake/adapter/path", "--", "src/**/*.spec.js"},
		},
		{
			name:     "node with jest CLI path",
			args:     []string{"node", "./node_modules/jest/bin/jest.js"},
			expected: []string{"node", "./node_modules/jest/bin/jest.js", "--reporters", "/fake/adapter/path"},
		},

		// Complex real-world scenarios
		{
			name:     "npm test with bail and coverage",
			args:     []string{"npm", "test", "--", "--bail", "--coverage", "--coverageDirectory=./coverage"},
			expected: []string{"npm", "test", "--", "--bail", "--coverage", "--coverageDirectory=./coverage", "--reporters", "/fake/adapter/path"},
		},
		{
			name:     "yarn test with specific test suite pattern",
			args:     []string{"yarn", "test", "--", "--testNamePattern=Auth", "--verbose"},
			expected: []string{"yarn", "test", "--", "--testNamePattern=Auth", "--verbose", "--reporters", "/fake/adapter/path"},
		},
		{
			name:     "pnpm with updateSnapshot",
			args:     []string{"pnpm", "test", "--", "-u"},
			expected: []string{"pnpm", "test", "--", "-u", "--reporters", "/fake/adapter/path"},
		},
		{
			name:     "npm run test:integration with env var style",
			args:     []string{"npm", "run", "test:integration", "--", "--runInBand"},
			expected: []string{"npm", "run", "test:integration", "--", "--runInBand", "--reporters", "/fake/adapter/path"},
		},
		{
			name:     "direct jest binary",
			args:     []string{"jest"},
			expected: []string{"jest", "--reporters", "/fake/adapter/path"},
		},
		{
			name:     "jest with only changed files",
			args:     []string{"jest", "-o"},
			expected: []string{"jest", "--reporters", "/fake/adapter/path", "-o"},
		},
		{
			name:     "jest with test path pattern",
			args:     []string{"jest", "src/components"},
			expected: []string{"jest", "--reporters", "/fake/adapter/path", "--", "src/components"},
		},

		// Edge cases
		{
			name:     "npm test with no additional args",
			args:     []string{"npm", "t"}, // npm t is alias for npm test
			expected: []string{"npm", "t", "--", "--reporters", "/fake/adapter/path"},
		},
		{
			name:     "yarn with workspace",
			args:     []string{"yarn", "workspace", "@myapp/client", "test"},
			expected: []string{"yarn", "workspace", "@myapp/client", "test", "--reporters", "/fake/adapter/path"},
		},
		{
			name:     "npm with silent flag",
			args:     []string{"npm", "test", "--silent"},
			expected: []string{"npm", "test", "--silent", "--", "--reporters", "/fake/adapter/path"},
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
			expected: []string{"npm", "test", "--", "--verbose", "--reporters", "/fake/adapter/path"},
		},
		{
			name:     "npm test with jest options after --",
			args:     []string{"npm", "test", "--", "--watchAll", "false"},
			expected: []string{"npm", "test", "--", "--watchAll", "false", "--reporters", "/fake/adapter/path"},
		},
		{
			name:     "yarn test (no jest in command) should NOT use -- separator",
			args:     []string{"yarn", "test"},
			expected: []string{"yarn", "test", "--reporters", "/fake/adapter/path"},
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
