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
	cmdStr := strings.Join(command, " ")
	return strings.Contains(cmdStr, "jest") || j.isJestInPackageJSON()
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
		isPackageManagerCommand = strings.Contains(cmd, "npm") ||
			strings.Contains(cmd, "yarn") ||
			strings.Contains(cmd, "pnpm") ||
			strings.Contains(cmd, "bun")
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

		if hasSeparator {
			// Append reporter flags at the end (after all other Jest flags)
			result = append(result, args...)
			result = append(result, "--reporters", adapterPath)
		} else {
			// Add all args, then -- separator, then reporter flags
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
	cmdStr := strings.Join(command, " ")
	return strings.Contains(cmdStr, "vitest") || v.isVitestInPackageJSON()
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
		isPackageManagerCommand = strings.Contains(cmd, "npm") ||
			strings.Contains(cmd, "yarn") ||
			strings.Contains(cmd, "pnpm") ||
			strings.Contains(cmd, "bun") ||
			strings.Contains(cmd, "deno")
	}

	// Check for vitest in the command
	for _, arg := range args {
		if strings.Contains(arg, "vitest") {
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

	// Handle package manager commands that need -- separator
	if isPackageManagerCommand && !isDirectVitestCall && !foundVitest {
		// Commands like: npm test, yarn test, pnpm test, bun test, deno task test
		result := make([]string, 0, len(args)+6)

		// Check if -- separator already exists
		hasSeparator := false
		for _, arg := range args {
			if arg == "--" {
				hasSeparator = true
				break
			}
		}

		if hasSeparator {
			// Append everything up to and including --, then add reporters
			result = append(result, args...)
			result = append(result, "--reporter", adapterPath, "--reporter", "default")
		} else {
			// Add all args, then -- separator, then reporter flags
			result = append(result, args...)
			result = append(result, "--", "--reporter", adapterPath, "--reporter", "default")
		}

		return result
	}

	// Handle direct vitest invocations
	result := make([]string, 0, len(args)+4)
	reporterAdded := false

	for _, arg := range args {
		if !reporterAdded && strings.Contains(arg, "vitest") {
			result = append(result, arg)
			// Add reporter flags immediately after vitest command
			result = append(result, "--reporter", adapterPath, "--reporter", "default")
			reporterAdded = true
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
	cmdStr := strings.Join(command, " ")
	return strings.Contains(cmdStr, "pytest") || strings.Contains(cmdStr, "py.test")
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
