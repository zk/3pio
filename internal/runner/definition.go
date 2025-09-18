package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Definition interface for test runner implementations
type Definition interface {
	// Matches determines if this runner can handle the given command
	Matches(command []string) bool

	// GetTestFiles returns list of test files (empty for dynamic discovery)
	GetTestFiles(args []string) ([]string, error)

	// BuildCommand builds the command with adapter injection
	BuildCommand(args []string, adapterPath string) []string

	// GetAdapterFileName returns the adapter file name
	GetAdapterFileName() string

	// InterpretExitCode maps exit codes to success/failure
	InterpretExitCode(code int) string
}

// NativeRunner interface for runners that process output directly without adapters
type NativeRunner interface {
	Definition
	// IsNative returns true if this runner processes output directly
	IsNative() bool
	// GetNativeDefinition returns the underlying native definition
	GetNativeDefinition() interface{}
}

// BaseDefinition provides common functionality for test runners
type BaseDefinition struct {
	name        string
	adapterFile string
}

// GetAdapterFileName returns the adapter file name
func (b *BaseDefinition) GetAdapterFileName() string {
	return b.adapterFile
}

// InterpretExitCode provides default exit code interpretation
func (b *BaseDefinition) InterpretExitCode(code int) string {
	if code == 0 {
		return "success"
	}
	return "failure"
}

// containsTestRunner checks if a command contains a specific test runner
// It's more strict than strings.Contains to avoid false positives
func containsTestRunner(command []string, runner string) bool {
	for _, arg := range command {
		// Extract base name from path
		baseName := arg
		if idx := strings.LastIndex(arg, "/"); idx != -1 {
			baseName = arg[idx+1:]
		}
		if idx := strings.LastIndex(baseName, "\\"); idx != -1 {
			baseName = baseName[idx+1:]
		}

		// Check for exact match or as part of a known pattern
		if baseName == runner {
			return true
		}

		// Handle common patterns like "python -m pytest"
		if runner == "pytest" && arg == "-m" {
			// Check if the next argument is pytest
			idx := indexOf(command, arg)
			if idx >= 0 && idx < len(command)-1 && command[idx+1] == "pytest" {
				return true
			}
		}
	}
	return false
}

// indexOf finds the index of a string in a slice
func indexOf(slice []string, item string) int {
	for i, s := range slice {
		if s == item {
			return i
		}
	}
	return -1
}

// JestDefinition implements Definition for Jest
type JestDefinition struct {
	BaseDefinition
}

// NewJestDefinition creates a new Jest definition
func NewJestDefinition() *JestDefinition {
	return &JestDefinition{
		BaseDefinition: BaseDefinition{
			name:        "jest",
			adapterFile: "jest.js",
		},
	}
}

// Matches checks if the command is for Jest
func (j *JestDefinition) Matches(command []string) bool {
	return containsTestRunner(command, "jest") || j.isJestInPackageJSON()
}

// GetTestFiles gets test files for Jest
func (j *JestDefinition) GetTestFiles(args []string) ([]string, error) {
	// Try to use --listTests for static discovery
	// This is simplified - actual implementation would run jest --listTests
	return []string{}, nil // Dynamic discovery
}

// BuildCommand builds Jest command with adapter
func (j *JestDefinition) BuildCommand(args []string, adapterPath string) []string {
	result := make([]string, 0, len(args)+5)

	foundJest := false
	jestIndex := -1
	isPackageManagerCommand := false

	// Check if this is a package manager command that needs -- separator
	if len(args) > 0 {
		cmd := args[0]
		isPackageManagerCommand = (cmd == "npm" || strings.HasPrefix(cmd, "npm ")) ||
			(cmd == "yarn" || strings.HasPrefix(cmd, "yarn ")) ||
			(cmd == "pnpm" || strings.HasPrefix(cmd, "pnpm ")) ||
			(cmd == "bun" || strings.HasPrefix(cmd, "bun "))
	}

	// Special handling for package manager commands that directly call jest
	// npm exec jest and yarn jest should NOT use -- separator (they're direct invocations)
	// pnpm exec jest and bun jest SHOULD use -- separator (they need it)
	isDirectJestCall := false
	if len(args) >= 2 && isPackageManagerCommand {
		cmd := args[0]
		// npm exec and yarn jest are direct invocations (no -- needed)
		if (cmd == "npm" && args[1] == "exec" && len(args) > 2 && strings.Contains(args[2], "jest")) ||
			(cmd == "yarn" && args[1] == "jest") {
			isDirectJestCall = true
		}
		// pnpm exec jest and bun jest need -- separator
		// They are NOT marked as direct calls, so they'll use the -- separator
	}

	// Check for bunx/npx which are treated like direct invocations
	if len(args) > 0 && (strings.Contains(args[0], "npx") || strings.Contains(args[0], "bunx")) {
		isDirectJestCall = true
		isPackageManagerCommand = false
	}

	// Find jest position and check for test files
	for i, arg := range args {
		if strings.Contains(arg, "jest") {
			foundJest = true
			if jestIndex == -1 {
				jestIndex = i
			}
		}
	}

	// Determine if we have test files after jest command
	hasTestFiles := false
	if foundJest && jestIndex >= 0 {
		for i := jestIndex + 1; i < len(args); i++ {
			arg := args[i]
			// Check if this looks like a test file/pattern (not a flag)
			if !strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") {
				// Could be a file, directory, or glob pattern
				if strings.Contains(arg, ".") || strings.Contains(arg, "/") || !strings.Contains(arg, "=") {
					hasTestFiles = true
					break
				}
			}
		}
	}

	// Handle package manager commands that need -- separator (npm test, yarn test, pnpm exec jest, bun jest, etc.)
	// This includes pnpm exec jest and bun jest even though they contain "jest"
	if isPackageManagerCommand && !isDirectJestCall {
		// Check if -- separator already exists
		hasSeparator := false
		for _, arg := range args {
			if arg == "--" {
				hasSeparator = true
				break
			}
		}

		// Yarn (v1.x) doesn't need -- separator for script commands (yarn test, yarn run test, etc.)
		// It automatically forwards flags to the script. Using -- with yarn can cause issues
		// where arguments are incorrectly concatenated with pipe characters.
		isYarnScript := len(args) >= 2 && args[0] == "yarn" &&
			(args[1] == "test" || args[1] == "run" || !strings.Contains(args[1], "jest"))

		if hasSeparator {
			// Append reporter flags at the end (after all other Jest flags)
			result = append(result, args...)
			result = append(result, "--reporters", adapterPath)
		} else if isYarnScript {
			// For yarn scripts, don't use -- separator
			result = append(result, args...)
			result = append(result, "--reporters", adapterPath)
		} else {
			// Add all args, then -- separator, then reporter flags (for npm, pnpm, bun)
			result = append(result, args...)
			result = append(result, "--", "--reporters", adapterPath)
		}

		return result
	}

	// Handle direct jest invocations and direct calls through package managers
	reporterAdded := false
	separatorAdded := false

	for i, arg := range args {
		// Check if we need to add -- separator before this arg
		// This happens when we've added the reporter, we have test files,
		// and this arg is a test file (not a flag)
		if reporterAdded && !separatorAdded && hasTestFiles &&
			!strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") &&
			i > jestIndex && (strings.Contains(arg, ".") || strings.Contains(arg, "/")) {
			result = append(result, "--")
			separatorAdded = true
		}

		result = append(result, arg)

		// Add reporter after jest command
		if !reporterAdded && strings.Contains(arg, "jest") {
			result = append(result, "--reporters", adapterPath)
			reporterAdded = true
		}
	}

	// If jest wasn't found in args (fallback case), add reporter at the end
	if !foundJest && !reporterAdded {
		result = append(result, "--reporters", adapterPath)
	}

	return result
}

// isJestInPackageJSON checks if Jest is configured in package.json
func (j *JestDefinition) isJestInPackageJSON() bool {
	data, err := os.ReadFile("package.json")
	if err != nil {
		return false
	}

	var pkg map[string]interface{}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return false
	}

	// Check test script
	if scripts, ok := pkg["scripts"].(map[string]interface{}); ok {
		if test, ok := scripts["test"].(string); ok {
			if strings.Contains(test, "jest") {
				return true
			}
		}
	}

	// Check dependencies
	for _, depKey := range []string{"dependencies", "devDependencies"} {
		if deps, ok := pkg[depKey].(map[string]interface{}); ok {
			if _, hasJest := deps["jest"]; hasJest {
				return true
			}
		}
	}

	return false
}

// VitestDefinition implements Definition for Vitest
type VitestDefinition struct {
    BaseDefinition
}

// NewVitestDefinition creates a new Vitest definition
func NewVitestDefinition() *VitestDefinition {
	return &VitestDefinition{
		BaseDefinition: BaseDefinition{
			name:        "vitest",
			adapterFile: "vitest.js",
		},
	}
}

// Matches checks if the command is for Vitest
func (v *VitestDefinition) Matches(command []string) bool {
	return containsTestRunner(command, "vitest") || v.isVitestInPackageJSON()
}

// GetTestFiles gets test files for Vitest
func (v *VitestDefinition) GetTestFiles(args []string) ([]string, error) {
	// Check if specific files are provided as arguments
	files := []string{}
	for _, arg := range args {
		if strings.HasSuffix(arg, ".js") || strings.HasSuffix(arg, ".ts") ||
			strings.HasSuffix(arg, ".jsx") || strings.HasSuffix(arg, ".tsx") {
			files = append(files, arg)
		}
	}

	if len(files) > 0 {
		return files, nil
	}

	// Otherwise use dynamic discovery
	return []string{}, nil
}

// BuildCommand builds Vitest command with adapter
func (v *VitestDefinition) BuildCommand(args []string, adapterPath string) []string {
	foundVitest := false
	isPackageManagerCommand := false
	isDirectVitestCall := false

	// Check if this is a package manager command
	if len(args) > 0 {
		cmd := args[0]
		isPackageManagerCommand = (cmd == "npm" || strings.HasPrefix(cmd, "npm ")) ||
			(cmd == "yarn" || strings.HasPrefix(cmd, "yarn ")) ||
			(cmd == "pnpm" || strings.HasPrefix(cmd, "pnpm ")) ||
			(cmd == "bun" || strings.HasPrefix(cmd, "bun ")) ||
			(cmd == "deno" || strings.HasPrefix(cmd, "deno "))
	}

	// Check for vitest executable in the command (not in config file names)
	for _, arg := range args {
		// Check if this is actually the vitest command or its CLI entry point
		if arg == "vitest" ||
			strings.HasSuffix(arg, "/vitest") ||
			strings.HasSuffix(arg, ".bin/vitest") ||
			strings.Contains(arg, "vitest@") ||
			strings.Contains(arg, "vitest/dist/cli") { // Handle node execution of vitest CLI
			foundVitest = true
			break
		}
	}

	// Determine if this is a direct vitest invocation
	// Direct invocations include: npx/bunx vitest, yarn vitest, pnpm exec vitest, etc.
	if len(args) >= 2 && isPackageManagerCommand {
		cmd := args[0]
		secondArg := args[1]

		// These are direct vitest invocations (no -- separator needed)
		if (cmd == "yarn" && secondArg == "vitest") ||
			(cmd == "yarn" && secondArg == "dlx" && len(args) > 2 && strings.Contains(args[2], "vitest")) ||
			(cmd == "pnpm" && secondArg == "exec" && len(args) > 2 && strings.Contains(args[2], "vitest")) ||
			(cmd == "pnpm" && secondArg == "dlx" && len(args) > 2 && strings.Contains(args[2], "vitest")) ||
			(cmd == "bun" && secondArg == "x" && len(args) > 2 && strings.Contains(args[2], "vitest")) {
			isDirectVitestCall = true
		}
	}

	// npx and bunx are always direct invocations
	if len(args) > 0 && (strings.Contains(args[0], "npx") || strings.Contains(args[0], "bunx")) {
		isDirectVitestCall = true
		isPackageManagerCommand = false
	}

	// Handle package manager commands that need special handling
	if isPackageManagerCommand && !isDirectVitestCall && !foundVitest {
		// Commands like: npm test, yarn test, pnpm test, bun test, deno task test
		result := make([]string, 0, len(args)+6)

		// Different package managers handle flags differently:
		// - pnpm: passes unknown flags directly to the script (no -- needed)
		// - npm/yarn/bun/deno: need -- separator to pass flags to the script
		cmd := args[0]
		needsSeparator := (cmd == "npm" || strings.HasPrefix(cmd, "npm ")) ||
			(cmd == "yarn" || strings.HasPrefix(cmd, "yarn ")) ||
			(cmd == "bun" || strings.HasPrefix(cmd, "bun ")) ||
			(cmd == "deno" || strings.HasPrefix(cmd, "deno "))

		// Check if -- separator already exists
		hasSeparator := false
		for _, arg := range args {
			if arg == "--" {
				hasSeparator = true
				break
			}
		}

		result = append(result, args...)

		if needsSeparator && !hasSeparator {
			result = append(result, "--")
		}

		result = append(result, "--reporter", adapterPath, "--reporter", "default")

		return result
	}

	// Handle direct vitest invocations
	result := make([]string, 0, len(args)+5) // Extra space for potential 'run' command
	reporterAdded := false
	needsRunCommand := false

	for i, arg := range args {
		// Only check for vitest command or its CLI entry point
		if !reporterAdded && (arg == "vitest" ||
			strings.HasSuffix(arg, "/vitest") ||
			strings.HasSuffix(arg, ".bin/vitest") ||
			strings.Contains(arg, "vitest@") ||
			strings.Contains(arg, "vitest/dist/cli")) {
			result = append(result, arg)
			// Add reporter flags immediately after vitest command
			result = append(result, "--reporter", adapterPath, "--reporter", "default")
			reporterAdded = true

			// Check if next argument is a vitest subcommand
			// If not, we need to add 'run' to prevent watch mode
			if i+1 < len(args) {
				nextArg := args[i+1]
				// Check if next arg is a known vitest subcommand
				isSubcommand := nextArg == "run" || nextArg == "watch" ||
					nextArg == "bench" || nextArg == "typecheck" ||
					nextArg == "list" || nextArg == "related" ||
					nextArg == "init"

				if !isSubcommand && !strings.HasPrefix(nextArg, "-") {
					// Next arg is not a subcommand or flag, so we need to add 'run'
					needsRunCommand = true
				} else if strings.HasPrefix(nextArg, "-") {
					// Next arg is a flag, we need 'run' before it
					needsRunCommand = true
				}
			} else {
				// No more args after vitest, need to add 'run'
				needsRunCommand = true
			}

			if needsRunCommand {
				result = append(result, "run")
			}
		} else {
			result = append(result, arg)
		}
	}

	// Fallback: if vitest wasn't found and reporter not added, add at the end
	if !foundVitest && !reporterAdded {
		result = append(result, "--reporter", adapterPath, "--reporter", "default")
	}

	return result
}

// isVitestInPackageJSON checks if Vitest is configured in package.json
func (v *VitestDefinition) isVitestInPackageJSON() bool {
	data, err := os.ReadFile("package.json")
	if err != nil {
		return false
	}

	var pkg map[string]interface{}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return false
	}

	// Check test script
	if scripts, ok := pkg["scripts"].(map[string]interface{}); ok {
		if test, ok := scripts["test"].(string); ok {
			if strings.Contains(test, "vitest") {
				return true
			}
		}
	}

	// Check dependencies
	for _, depKey := range []string{"dependencies", "devDependencies"} {
		if deps, ok := pkg[depKey].(map[string]interface{}); ok {
			if _, hasVitest := deps["vitest"]; hasVitest {
				return true
			}
		}
	}

	return false
}

// PytestDefinition implements Definition for pytest
type PytestDefinition struct {
    BaseDefinition
}

// NewPytestDefinition creates a new pytest definition
func NewPytestDefinition() *PytestDefinition {
	return &PytestDefinition{
		BaseDefinition: BaseDefinition{
			name:        "pytest",
			adapterFile: "pytest_adapter.py",
		},
	}
}

// Matches checks if the command is for pytest
func (p *PytestDefinition) Matches(command []string) bool {
	return containsTestRunner(command, "pytest") || containsTestRunner(command, "py.test")
}

// GetTestFiles gets test files for pytest
func (p *PytestDefinition) GetTestFiles(args []string) ([]string, error) {
	// Check if specific files are provided
	files := []string{}
	for _, arg := range args {
		if strings.HasSuffix(arg, ".py") && !strings.HasPrefix(arg, "-") {
			files = append(files, arg)
		}
	}

	if len(files) > 0 {
		return files, nil
	}

	// Could run pytest --collect-only for static discovery
	// For now, use dynamic discovery
	return []string{}, nil
}

// BuildCommand builds pytest command with adapter
func (p *PytestDefinition) BuildCommand(args []string, adapterPath string) []string {
	result := make([]string, 0, len(args)+2)

	// Set Python path to include adapter directory
	adapterDir := filepath.Dir(adapterPath)
	pythonPath := os.Getenv("PYTHONPATH")
	if pythonPath != "" {
		pythonPath = fmt.Sprintf("%s%c%s", adapterDir, os.PathListSeparator, pythonPath)
	} else {
		pythonPath = adapterDir
	}
	_ = os.Setenv("PYTHONPATH", pythonPath)

	foundPytest := false
	for _, arg := range args {
		result = append(result, arg)

		// After pytest command, inject plugin immediately
		if !foundPytest && (strings.Contains(arg, "pytest") || strings.Contains(arg, "py.test")) {
			foundPytest = true
			// Always add plugin flag immediately after pytest
			result = append(result, "-p", "pytest_adapter")
		}
	}

	// If pytest wasn't found, add plugin at the end
	if !foundPytest {
		result = append(result, "-p", "pytest_adapter")
	}

	return result
}

// CypressDefinition implements Definition for Cypress
type CypressDefinition struct {
    BaseDefinition
}

// NewCypressDefinition creates a new Cypress definition
func NewCypressDefinition() *CypressDefinition {
    return &CypressDefinition{
        BaseDefinition: BaseDefinition{
            name:        "cypress",
            adapterFile: "cypress.js",
        },
    }
}

// Matches checks if the command is for Cypress
func (c *CypressDefinition) Matches(command []string) bool {
    return containsTestRunner(command, "cypress") || c.isCypressInPackageJSON()
}

// GetTestFiles gets test files for Cypress (dynamic by default)
func (c *CypressDefinition) GetTestFiles(args []string) ([]string, error) {
    // Cypress discovers specs dynamically; we return empty to indicate that.
    return []string{}, nil
}

// BuildCommand builds Cypress command with reporter injection
func (c *CypressDefinition) BuildCommand(args []string, adapterPath string) []string {
    // Strategy similar to Vitest/Jest:
    // - If package manager script (npm/yarn/pnpm/bun), add reporter flags after '--' for npm/yarn/bun; pnpm passes directly.
    // - If direct invocation (cypress ...), ensure 'run' subcommand is present, and inject '--reporter <adapterPath>'.

    result := make([]string, 0, len(args)+4)

    isPackageManagerCommand := false
    if len(args) > 0 {
        cmd := args[0]
        isPackageManagerCommand = (cmd == "npm" || strings.HasPrefix(cmd, "npm ")) ||
            (cmd == "yarn" || strings.HasPrefix(cmd, "yarn ")) ||
            (cmd == "pnpm" || strings.HasPrefix(cmd, "pnpm ")) ||
            (cmd == "bun" || strings.HasPrefix(cmd, "bun "))
    }

    // Handle package manager scripts like `npm test` when script runs cypress
    if isPackageManagerCommand {
        // Check for script forms that need separator
        cmd := args[0]
        needsSeparator := (cmd == "npm" || strings.HasPrefix(cmd, "npm ")) ||
            (cmd == "yarn" || strings.HasPrefix(cmd, "yarn ")) ||
            (cmd == "bun" || strings.HasPrefix(cmd, "bun "))

        // If args already contain '--', just append reporter flags
        hasSeparator := false
        for _, a := range args {
            if a == "--" { hasSeparator = true; break }
        }

        result = append(result, args...)
        if hasSeparator || !needsSeparator {
            result = append(result, "--reporter", adapterPath)
        } else {
            result = append(result, "--", "--reporter", adapterPath)
        }
        return result
    }

    // Direct invocations (cypress run ... or npx/bunx cypress run ...)
    // Build by preserving arg order and appending reporter at the end.
    // Ensure 'run' is present after the cypress token.
    hasCypress := false
    hasRun := false
    for _, a := range args {
        if a == "cypress" || strings.HasSuffix(a, "/cypress") || strings.HasSuffix(a, ".bin/cypress") {
            hasCypress = true
        } else if hasCypress && a == "run" {
            hasRun = true
        }
    }

    result = append(result, args...)
    if hasCypress && !hasRun {
        result = append(result, "run")
    }
    result = append(result, "--reporter", adapterPath)
    return result
}

// isCypressInPackageJSON checks if Cypress is configured in package.json
func (c *CypressDefinition) isCypressInPackageJSON() bool {
    data, err := os.ReadFile("package.json")
    if err != nil {
        return false
    }

    var pkg map[string]interface{}
    if err := json.Unmarshal(data, &pkg); err != nil {
        return false
    }

    // Check test script
    if scripts, ok := pkg["scripts"].(map[string]interface{}); ok {
        if test, ok := scripts["test"].(string); ok {
            if strings.Contains(test, "cypress") {
                return true
            }
        }
    }

    // Check dependencies
    for _, depKey := range []string{"dependencies", "devDependencies"} {
        if deps, ok := pkg[depKey].(map[string]interface{}); ok {
            if _, has := deps["cypress"]; has {
                return true
            }
        }
    }

    return false
}

// MochaDefinition implements Definition for Mocha
type MochaDefinition struct {
    BaseDefinition
}

// NewMochaDefinition creates a new Mocha definition
func NewMochaDefinition() *MochaDefinition {
    return &MochaDefinition{
        BaseDefinition: BaseDefinition{
            name:        "mocha",
            adapterFile: "mocha.js",
        },
    }
}

// Matches checks if the command is for Mocha
func (m *MochaDefinition) Matches(command []string) bool {
    return containsTestRunner(command, "mocha") || m.isMochaInPackageJSON()
}

// GetTestFiles gets test files for Mocha (dynamic by default; CLI often passes globs)
func (m *MochaDefinition) GetTestFiles(args []string) ([]string, error) {
    // Mocha usually accepts globs; rely on dynamic discovery
    return []string{}, nil
}

// BuildCommand builds Mocha command with reporter injection
func (m *MochaDefinition) BuildCommand(args []string, adapterPath string) []string {
    result := make([]string, 0, len(args)+4)

    isPackageManagerCommand := false
    if len(args) > 0 {
        cmd := args[0]
        isPackageManagerCommand = (cmd == "npm" || strings.HasPrefix(cmd, "npm ")) ||
            (cmd == "yarn" || strings.HasPrefix(cmd, "yarn ")) ||
            (cmd == "pnpm" || strings.HasPrefix(cmd, "pnpm ")) ||
            (cmd == "bun" || strings.HasPrefix(cmd, "bun "))
    }

    if isPackageManagerCommand {
        // Similar strategy to Cypress: npm/yarn/bun need '--' for script flags; pnpm passes directly
        cmd := args[0]
        needsSeparator := (cmd == "npm" || strings.HasPrefix(cmd, "npm ")) ||
            (cmd == "yarn" || strings.HasPrefix(cmd, "yarn ")) ||
            (cmd == "bun" || strings.HasPrefix(cmd, "bun "))

        hasSeparator := false
        for _, a := range args {
            if a == "--" { hasSeparator = true; break }
        }

        result = append(result, args...)
        if hasSeparator || !needsSeparator {
            result = append(result, "--reporter", adapterPath)
        } else {
            result = append(result, "--", "--reporter", adapterPath)
        }
        return result
    }

    // Direct mocha invocation or via npx/bunx
    result = append(result, args...)
    result = append(result, "--reporter", adapterPath)
    return result
}

// isMochaInPackageJSON checks if Mocha is configured in package.json
func (m *MochaDefinition) isMochaInPackageJSON() bool {
    data, err := os.ReadFile("package.json")
    if err != nil {
        return false
    }

    var pkg map[string]interface{}
    if err := json.Unmarshal(data, &pkg); err != nil {
        return false
    }

    if scripts, ok := pkg["scripts"].(map[string]interface{}); ok {
        if test, ok := scripts["test"].(string); ok {
            if strings.Contains(test, "mocha") {
                return true
            }
        }
    }

    for _, depKey := range []string{"dependencies", "devDependencies"} {
        if deps, ok := pkg[depKey].(map[string]interface{}); ok {
            if _, has := deps["mocha"]; has {
                return true
            }
        }
    }
    return false
}
