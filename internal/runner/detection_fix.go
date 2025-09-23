package runner

import "strings"

// hasExplicitRunner checks if a command explicitly specifies any test runner
func hasExplicitRunner(command []string) bool {
	runners := []string{"jest", "vitest", "mocha", "cypress", "pytest"}
	for _, runner := range runners {
		if containsTestRunner(command, runner) {
			return true
		}
	}
	return false
}

// isGenericTestCommand checks if this is a generic test command without explicit runner
func isGenericTestCommand(command []string) bool {
	if len(command) < 2 {
		return false
	}

	// Check for package manager test commands
	packageManagers := map[string]bool{
		"npm":  true,
		"yarn": true,
		"pnpm": true,
		"bun":  true,
		"deno": true,
	}

	cmd := command[0]
	// Extract base name from path
	if idx := strings.LastIndex(cmd, "/"); idx != -1 {
		cmd = cmd[idx+1:]
	}
	if idx := strings.LastIndex(cmd, "\\"); idx != -1 {
		cmd = cmd[idx+1:]
	}

	// Check if it's a package manager running test/run commands
	if packageManagers[cmd] {
		if len(command) >= 2 {
			secondArg := command[1]
			// These are generic test commands
			if secondArg == "test" || secondArg == "run" || secondArg == "start" {
				// Make sure it's not followed by an explicit runner
				if len(command) > 2 {
					return !hasExplicitRunner(command[2:])
				}
				return true
			}
		}
	}

	return false
}

// MatchesWithPrecedence is the improved matching logic that respects explicit runner specification
func MatchesWithPrecedence(command []string, runnerName string, checkPackageJSON func() bool) bool {
	// Step 1: If this runner is explicitly in the command, always match
	if containsTestRunner(command, runnerName) {
		return true
	}

	// Step 2: If another runner is explicitly specified, never match
	if hasExplicitRunner(command) {
		return false
	}

	// Step 3: For generic test commands, use package.json detection
	if isGenericTestCommand(command) {
		return checkPackageJSON()
	}

	// Step 4: For other commands, check package.json only if no explicit runner
	return checkPackageJSON()
}
