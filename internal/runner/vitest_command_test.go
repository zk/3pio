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
		// NPM variations (10 examples)
		{
			name:     "npm test command",
			args:     []string{"npm", "test"},
			expected: []string{"npm", "test", "--", "--reporter", "/tmp/adapter.js", "--reporter", "default"},
		},
		{
			name:     "npm run test",
			args:     []string{"npm", "run", "test"},
			expected: []string{"npm", "run", "test", "--", "--reporter", "/tmp/adapter.js", "--reporter", "default"},
		},
		{
			name:     "npm run test:unit custom script",
			args:     []string{"npm", "run", "test:unit"},
			expected: []string{"npm", "run", "test:unit", "--", "--reporter", "/tmp/adapter.js", "--reporter", "default"},
		},
		{
			name:     "npm run test:watch",
			args:     []string{"npm", "run", "test:watch"},
			expected: []string{"npm", "run", "test:watch", "--", "--reporter", "/tmp/adapter.js", "--reporter", "default"},
		},
		{
			name:     "npm test with existing -- and run flag",
			args:     []string{"npm", "test", "--", "--run"},
			expected: []string{"npm", "test", "--", "--run", "--reporter", "/tmp/adapter.js", "--reporter", "default"},
		},
		{
			name:     "npm test with coverage",
			args:     []string{"npm", "test", "--", "--coverage"},
			expected: []string{"npm", "test", "--", "--coverage", "--reporter", "/tmp/adapter.js", "--reporter", "default"},
		},
		{
			name:     "npm run test:ci",
			args:     []string{"npm", "run", "test:ci"},
			expected: []string{"npm", "run", "test:ci", "--", "--reporter", "/tmp/adapter.js", "--reporter", "default"},
		},
		{
			name:     "npm test with multiple flags",
			args:     []string{"npm", "test", "--", "--run", "--coverage", "--silent"},
			expected: []string{"npm", "test", "--", "--run", "--coverage", "--silent", "--reporter", "/tmp/adapter.js", "--reporter", "default"},
		},
		{
			name:     "npm t shorthand",
			args:     []string{"npm", "t"},
			expected: []string{"npm", "t", "--", "--reporter", "/tmp/adapter.js", "--reporter", "default"},
		},
		{
			name:     "npm test with silent flag",
			args:     []string{"npm", "test", "--silent"},
			expected: []string{"npm", "test", "--silent", "--", "--reporter", "/tmp/adapter.js", "--reporter", "default"},
		},

		// Direct Vitest invocations (10 examples)
		{
			name:     "direct vitest command",
			args:     []string{"vitest"},
			expected: []string{"vitest", "--reporter", "/tmp/adapter.js", "--reporter", "default", "run"},
		},
		{
			name:     "vitest run",
			args:     []string{"vitest", "run"},
			expected: []string{"vitest", "--reporter", "/tmp/adapter.js", "--reporter", "default", "run"},
		},
		{
			name:     "npx vitest",
			args:     []string{"npx", "vitest"},
			expected: []string{"npx", "vitest", "--reporter", "/tmp/adapter.js", "--reporter", "default", "run"},
		},
		{
			name:     "npx vitest run",
			args:     []string{"npx", "vitest", "run"},
			expected: []string{"npx", "vitest", "--reporter", "/tmp/adapter.js", "--reporter", "default", "run"},
		},
		{
			name:     "npx vitest with test files",
			args:     []string{"npx", "vitest", "run", "test1.js", "test2.js"},
			expected: []string{"npx", "vitest", "--reporter", "/tmp/adapter.js", "--reporter", "default", "run", "test1.js", "test2.js"},
		},
		{
			name:     "npx vitest with path pattern",
			args:     []string{"npx", "vitest", "run", "src/components"},
			expected: []string{"npx", "vitest", "--reporter", "/tmp/adapter.js", "--reporter", "default", "run", "src/components"},
		},
		{
			name:     "npx vitest with coverage",
			args:     []string{"npx", "vitest", "run", "--coverage"},
			expected: []string{"npx", "vitest", "--reporter", "/tmp/adapter.js", "--reporter", "default", "run", "--coverage"},
		},
		{
			name:     "npx vitest with threads disabled",
			args:     []string{"npx", "vitest", "run", "--no-threads"},
			expected: []string{"npx", "vitest", "--reporter", "/tmp/adapter.js", "--reporter", "default", "run", "--no-threads"},
		},
		{
			name:     "npx --no-install vitest",
			args:     []string{"npx", "--no-install", "vitest", "run"},
			expected: []string{"npx", "--no-install", "vitest", "--reporter", "/tmp/adapter.js", "--reporter", "default", "run"},
		},
		{
			name:     "npx with package version",
			args:     []string{"npx", "vitest@latest", "run"},
			expected: []string{"npx", "vitest@latest", "--reporter", "/tmp/adapter.js", "--reporter", "default", "run"},
		},

		// Yarn variations (10 examples)
		{
			name:     "yarn test",
			args:     []string{"yarn", "test"},
			expected: []string{"yarn", "test", "--", "--reporter", "/tmp/adapter.js", "--reporter", "default"},
		},
		{
			name:     "yarn run test",
			args:     []string{"yarn", "run", "test"},
			expected: []string{"yarn", "run", "test", "--", "--reporter", "/tmp/adapter.js", "--reporter", "default"},
		},
		{
			name:     "yarn vitest",
			args:     []string{"yarn", "vitest"},
			expected: []string{"yarn", "vitest", "--reporter", "/tmp/adapter.js", "--reporter", "default", "run"},
		},
		{
			name:     "yarn vitest run",
			args:     []string{"yarn", "vitest", "run"},
			expected: []string{"yarn", "vitest", "--reporter", "/tmp/adapter.js", "--reporter", "default", "run"},
		},
		{
			name:     "yarn test:unit",
			args:     []string{"yarn", "test:unit"},
			expected: []string{"yarn", "test:unit", "--", "--reporter", "/tmp/adapter.js", "--reporter", "default"},
		},
		{
			name:     "yarn workspace test",
			args:     []string{"yarn", "workspace", "@myapp/client", "test"},
			expected: []string{"yarn", "workspace", "@myapp/client", "test", "--", "--reporter", "/tmp/adapter.js", "--reporter", "default"},
		},
		{
			name:     "yarn test with existing --",
			args:     []string{"yarn", "test", "--", "--run"},
			expected: []string{"yarn", "test", "--", "--run", "--reporter", "/tmp/adapter.js", "--reporter", "default"},
		},
		{
			name:     "yarn dlx vitest",
			args:     []string{"yarn", "dlx", "vitest", "run"},
			expected: []string{"yarn", "dlx", "vitest", "--reporter", "/tmp/adapter.js", "--reporter", "default", "run"},
		},
		{
			name:     "yarn berry test",
			args:     []string{"yarn", "berry", "test"},
			expected: []string{"yarn", "berry", "test", "--", "--reporter", "/tmp/adapter.js", "--reporter", "default"},
		},
		{
			name:     "yarn test with config",
			args:     []string{"yarn", "test", "--", "--config=vitest.config.ts"},
			expected: []string{"yarn", "test", "--", "--config=vitest.config.ts", "--reporter", "/tmp/adapter.js", "--reporter", "default"},
		},

		// PNPM variations (10 examples)
		{
			name:     "pnpm test",
			args:     []string{"pnpm", "test"},
			expected: []string{"pnpm", "test", "--", "--reporter", "/tmp/adapter.js", "--reporter", "default"},
		},
		{
			name:     "pnpm run test",
			args:     []string{"pnpm", "run", "test"},
			expected: []string{"pnpm", "run", "test", "--", "--reporter", "/tmp/adapter.js", "--reporter", "default"},
		},
		{
			name:     "pnpm run test:watch",
			args:     []string{"pnpm", "run", "test:watch"},
			expected: []string{"pnpm", "run", "test:watch", "--", "--reporter", "/tmp/adapter.js", "--reporter", "default"},
		},
		{
			name:     "pnpm exec vitest",
			args:     []string{"pnpm", "exec", "vitest"},
			expected: []string{"pnpm", "exec", "vitest", "--reporter", "/tmp/adapter.js", "--reporter", "default", "run"},
		},
		{
			name:     "pnpm exec vitest run",
			args:     []string{"pnpm", "exec", "vitest", "run"},
			expected: []string{"pnpm", "exec", "vitest", "--reporter", "/tmp/adapter.js", "--reporter", "default", "run"},
		},
		{
			name:     "pnpm dlx vitest",
			args:     []string{"pnpm", "dlx", "vitest", "run"},
			expected: []string{"pnpm", "dlx", "vitest", "--reporter", "/tmp/adapter.js", "--reporter", "default", "run"},
		},
		{
			name:     "pnpm test with filter",
			args:     []string{"pnpm", "--filter", "backend", "test"},
			expected: []string{"pnpm", "--filter", "backend", "test", "--", "--reporter", "/tmp/adapter.js", "--reporter", "default"},
		},
		{
			name:     "pnpm recursive test",
			args:     []string{"pnpm", "-r", "test"},
			expected: []string{"pnpm", "-r", "test", "--", "--reporter", "/tmp/adapter.js", "--reporter", "default"},
		},
		{
			name:     "pnpm test with existing separator",
			args:     []string{"pnpm", "test", "--", "--ui"},
			expected: []string{"pnpm", "test", "--", "--ui", "--reporter", "/tmp/adapter.js", "--reporter", "default"},
		},
		{
			name:     "pnpm test in workspace",
			args:     []string{"pnpm", "--workspace-root", "test"},
			expected: []string{"pnpm", "--workspace-root", "test", "--", "--reporter", "/tmp/adapter.js", "--reporter", "default"},
		},

		// Bun variations (10 examples)
		{
			name:     "bun test (might use bun's test runner)",
			args:     []string{"bun", "test"},
			expected: []string{"bun", "test", "--", "--reporter", "/tmp/adapter.js", "--reporter", "default"},
		},
		{
			name:     "bun run test",
			args:     []string{"bun", "run", "test"},
			expected: []string{"bun", "run", "test", "--", "--reporter", "/tmp/adapter.js", "--reporter", "default"},
		},
		{
			name:     "bunx vitest",
			args:     []string{"bunx", "vitest"},
			expected: []string{"bunx", "vitest", "--reporter", "/tmp/adapter.js", "--reporter", "default", "run"},
		},
		{
			name:     "bunx vitest run",
			args:     []string{"bunx", "vitest", "run"},
			expected: []string{"bunx", "vitest", "--reporter", "/tmp/adapter.js", "--reporter", "default", "run"},
		},
		{
			name:     "bunx --bun vitest",
			args:     []string{"bunx", "--bun", "vitest", "run"},
			expected: []string{"bunx", "--bun", "vitest", "--reporter", "/tmp/adapter.js", "--reporter", "default", "run"},
		},
		{
			name:     "bun run test:unit",
			args:     []string{"bun", "run", "test:unit"},
			expected: []string{"bun", "run", "test:unit", "--", "--reporter", "/tmp/adapter.js", "--reporter", "default"},
		},
		{
			name:     "bun x vitest",
			args:     []string{"bun", "x", "vitest"},
			expected: []string{"bun", "x", "vitest", "--reporter", "/tmp/adapter.js", "--reporter", "default", "run"},
		},
		{
			name:     "bun test with existing --",
			args:     []string{"bun", "test", "--", "--run"},
			expected: []string{"bun", "test", "--", "--run", "--reporter", "/tmp/adapter.js", "--reporter", "default"},
		},
		{
			name:     "bunx vitest with config",
			args:     []string{"bunx", "vitest", "run", "--config", "vitest.config.mjs"},
			expected: []string{"bunx", "vitest", "--reporter", "/tmp/adapter.js", "--reporter", "default", "run", "--config", "vitest.config.mjs"},
		},
		{
			name:     "bun run dev:test",
			args:     []string{"bun", "run", "dev:test"},
			expected: []string{"bun", "run", "dev:test", "--", "--reporter", "/tmp/adapter.js", "--reporter", "default"},
		},

		// Node direct execution
		{
			name:     "node with vitest from node_modules",
			args:     []string{"node", "node_modules/.bin/vitest", "run"},
			expected: []string{"node", "node_modules/.bin/vitest", "--reporter", "/tmp/adapter.js", "--reporter", "default", "run"},
		},
		{
			name:     "node with vitest CLI path",
			args:     []string{"node", "./node_modules/vitest/dist/cli.mjs", "run"},
			expected: []string{"node", "./node_modules/vitest/dist/cli.mjs", "--reporter", "/tmp/adapter.js", "--reporter", "default", "run"},
		},

		// Deno variations
		{
			name:     "deno task test",
			args:     []string{"deno", "task", "test"},
			expected: []string{"deno", "task", "test", "--", "--reporter", "/tmp/adapter.js", "--reporter", "default"},
		},

		// Edge cases
		{
			name:     "vitest bench",
			args:     []string{"vitest", "bench"},
			expected: []string{"vitest", "--reporter", "/tmp/adapter.js", "--reporter", "default", "bench"},
		},
		{
			name:     "vitest with inline config",
			args:     []string{"vitest", "run", "--globals", "--dom"},
			expected: []string{"vitest", "--reporter", "/tmp/adapter.js", "--reporter", "default", "run", "--globals", "--dom"},
		},
		{
			name:     "vitest typecheck",
			args:     []string{"vitest", "typecheck"},
			expected: []string{"vitest", "--reporter", "/tmp/adapter.js", "--reporter", "default", "typecheck"},
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
