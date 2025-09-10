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
	hasTestFiles := false
	testFileIndex := -1
	
	// First pass: find jest and check for test files
	for i, arg := range args {
		if strings.Contains(arg, "jest") {
			foundJest = true
		}
		// Check if this looks like a test file (not starting with -)
		if foundJest && !strings.HasPrefix(arg, "-") && strings.Contains(arg, ".") {
			hasTestFiles = true
			if testFileIndex == -1 {
				testFileIndex = i
			}
		}
	}
	
	// Build the command
	for i, arg := range args {
		if strings.Contains(arg, "jest") {
			result = append(result, arg)
			result = append(result, "--reporters", adapterPath)
			// If there are test files, add -- separator before them
			if hasTestFiles && i+1 == testFileIndex {
				result = append(result, "--")
			}
		} else {
			result = append(result, arg)
		}
	}
	
	// If jest wasn't found in args (e.g., npm test), add reporter at the end
	if !foundJest {
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
	result := make([]string, 0, len(args)+4)
	
	foundVitest := false
	for i, arg := range args {
		result = append(result, arg)
		
		// After vitest command, inject reporter
		if !foundVitest && strings.Contains(arg, "vitest") {
			foundVitest = true
			// Add reporter flags after vitest
			if i == len(args)-1 || !strings.HasPrefix(args[i+1], "--") {
				result = append(result, "--reporter", adapterPath, "--reporter", "default")
			}
		}
	}
	
	// If vitest wasn't found in args, add reporter at the end
	if !foundVitest {
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
	os.Setenv("PYTHONPATH", pythonPath)
	
	foundPytest := false
	for i, arg := range args {
		result = append(result, arg)
		
		// After pytest command, inject plugin
		if !foundPytest && (strings.Contains(arg, "pytest") || strings.Contains(arg, "py.test")) {
			foundPytest = true
			// Add plugin flag after pytest
			if i == len(args)-1 || !strings.HasPrefix(args[i+1], "-") {
				result = append(result, "-p", "pytest_adapter")
			}
		}
	}
	
	// If pytest wasn't found, add plugin at the end
	if !foundPytest {
		result = append(result, "-p", "pytest_adapter")
	}
	
	return result
}