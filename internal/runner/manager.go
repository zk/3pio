package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/zk/3pio/internal/logger"
	"github.com/zk/3pio/internal/runner/definitions"
)

// Manager manages test runner definitions
type Manager struct {
	// Use slice for deterministic iteration order
	runners []struct {
		name string
		def  Definition
	}
	logger *logger.FileLogger
}

// Close closes the manager and its resources
func (m *Manager) Close() error {
	if m.logger != nil {
		return m.logger.Close()
	}
	return nil
}

// NewManager creates a new runner manager
func NewManager(fileLogger *logger.FileLogger) *Manager {
	m := &Manager{
		runners: make([]struct {
			name string
			def  Definition
		}, 0),
		logger: fileLogger,
	}

	// Register built-in runners in priority order
	// More specific runners should come before generic ones
	m.Register("vitest", NewVitestDefinition()) // Check Vitest before Jest
	m.Register("jest", NewJestDefinition())
	m.Register("cypress", NewCypressDefinition())
	m.Register("mocha", NewMochaDefinition())
	m.Register("pytest", NewPytestDefinition())

	// Register Go test runner (native, no adapter)
	m.Register("go", definitions.NewGoTestWrapper(fileLogger))

	// Register Rust test runners (native, no adapters)
	cargoImpl := definitions.NewCargoTestDefinition(fileLogger)
	m.Register("cargo", definitions.NewCargoTestWrapper(cargoImpl))

	nextestImpl := definitions.NewNextestDefinition(fileLogger)
	m.Register("nextest", definitions.NewNextestWrapper(nextestImpl))

	return m
}

// Register adds a new test runner definition
func (m *Manager) Register(name string, def Definition) {
	m.runners = append(m.runners, struct {
		name string
		def  Definition
	}{name, def})
}

// Detect identifies the test runner from command and returns its definition
func (m *Manager) Detect(command []string) (Definition, error) {
	// FIRST: Check if this is a package manager script command
	if len(command) > 0 {
		packageManager := command[0]
		if isPackageManager(packageManager) && len(command) >= 2 {
			// Check if we're running a script (not installing or other npm commands)
			isScriptCommand := command[1] == "run" || command[1] == "test" ||
				(packageManager != "npm" && !strings.HasPrefix(command[1], "-")) // yarn/pnpm allow direct script names

			if isScriptCommand {
				// Extract the script name
				scriptName := ""
				if command[1] == "run" && len(command) > 2 {
					scriptName = command[2]
				} else if command[1] != "run" {
					scriptName = command[1]
				}

				if scriptName != "" {
					// Read package.json and check what the script actually runs
					if scriptCommand := resolvePackageScript(scriptName); scriptCommand != "" {
						// Check if the resolved script directly invokes a known test runner
						supportedRunners := []string{"jest", "vitest", "mocha", "cypress", "pytest"}
						directInvocation := false

						// Split the command to check the first part
						scriptParts := strings.Fields(scriptCommand)
						if len(scriptParts) > 0 {
							firstCmd := scriptParts[0]

							// Handle cross-env specifically - it's just a env var setter
							if firstCmd == "cross-env" && len(scriptParts) > 1 {
								// Skip cross-env and any environment variables (KEY=VALUE pairs)
								for i := 1; i < len(scriptParts); i++ {
									if !strings.Contains(scriptParts[i], "=") {
										// This is the actual command after cross-env
										firstCmd = scriptParts[i]
										// Check if there's another arg for npx/yarn/pnpm cases
										if i+1 < len(scriptParts) && (firstCmd == "npx" || firstCmd == "yarn" || firstCmd == "pnpm") {
											// Check the next argument for the test runner
											nextCmd := scriptParts[i+1]
											for _, runner := range supportedRunners {
												if nextCmd == runner {
													directInvocation = true
													break
												}
											}
										} else {
											// Check if this is a direct test runner
											for _, runner := range supportedRunners {
												if firstCmd == runner {
													directInvocation = true
													break
												}
											}
										}
										break
									}
								}
							} else {
								// Original logic for non-cross-env commands
								// Check for direct invocation or via npx/yarn/pnpm
								for _, runner := range supportedRunners {
									if firstCmd == runner ||
										(firstCmd == "npx" && len(scriptParts) > 1 && scriptParts[1] == runner) ||
										(firstCmd == "yarn" && len(scriptParts) > 1 && scriptParts[1] == runner) ||
										(firstCmd == "pnpm" && len(scriptParts) > 1 && scriptParts[1] == runner) {
										directInvocation = true
										break
									}
								}
							}
						}

						// If the script doesn't directly invoke a test runner, fail immediately
						if !directInvocation {
							// Get any additional arguments safely
							var additionalArgs string
							if len(command) > 3 {
								additionalArgs = strings.Join(command[3:], " ")
							}

							return nil, fmt.Errorf("3pio error: npm script '%s' uses a custom wrapper ('%s').\n"+
								"3pio cannot inject adapters into custom wrapper scripts.\n"+
								"Please either:\n"+
								"  1. Run the test command directly: npx jest %s\n"+
								"  2. Modify the script to directly invoke the test runner\n"+
								"  3. Update the wrapper script to pass through all arguments",
								scriptName, scriptCommand, additionalArgs)
						}
					}
				}
			}
		}
	}

	// SECOND: Now check normal runner detection
	for _, runner := range m.runners {
		if runner.def.Matches(command) {
			return runner.def, nil
		}
	}

	return nil, fmt.Errorf("no test runner detected for command: %s", strings.Join(command, " "))
}

// GetDefinition returns a specific runner definition by name
func (m *Manager) GetDefinition(name string) (Definition, bool) {
	// Linear search is fine for ~8 items
	for _, runner := range m.runners {
		if runner.name == name {
			return runner.def, true
		}
	}
	return nil, false
}

// isPackageManager checks if a command is a package manager
func isPackageManager(cmd string) bool {
	// Extract the base command name from the full path
	baseName := cmd
	if idx := strings.LastIndex(cmd, "/"); idx != -1 {
		baseName = cmd[idx+1:]
	}
	if idx := strings.LastIndex(baseName, "\\"); idx != -1 {
		baseName = baseName[idx+1:]
	}

	// Check for exact matches of package manager names
	managers := []string{"npm", "yarn", "pnpm", "bun"}
	for _, m := range managers {
		if baseName == m {
			return true
		}
	}
	return false
}

// resolvePackageScript reads package.json and returns the actual command for a script
func resolvePackageScript(scriptName string) string {
	data, err := os.ReadFile("package.json")
	if err != nil {
		return ""
	}

	var pkg map[string]interface{}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return ""
	}

	scripts, ok := pkg["scripts"].(map[string]interface{})
	if !ok {
		return ""
	}

	script, ok := scripts[scriptName].(string)
	if !ok {
		return ""
	}

	return script
}

// OutputParser interface for parsing test output
type OutputParser interface {
	ParseTestOutput(output string) map[string][]string
}

// BaseOutputParser provides common parsing functionality
type BaseOutputParser struct{}

// ParseTestOutput provides basic output parsing
func (b *BaseOutputParser) ParseTestOutput(output string) map[string][]string {
	// Basic implementation - can be overridden by specific parsers
	result := make(map[string][]string)
	lines := strings.Split(output, "\n")

	currentFile := ""
	for _, line := range lines {
		// Look for file patterns
		if strings.Contains(line, "PASS") || strings.Contains(line, "FAIL") {
			// Extract file name if present
			parts := strings.Fields(line)
			for _, part := range parts {
				if strings.HasSuffix(part, ".js") || strings.HasSuffix(part, ".ts") ||
					strings.HasSuffix(part, ".jsx") || strings.HasSuffix(part, ".tsx") ||
					strings.HasSuffix(part, ".py") {
					currentFile = part
					break
				}
			}
		}

		if currentFile != "" {
			result[currentFile] = append(result[currentFile], line)
		}
	}

	return result
}

// JestOutputParser parses Jest output
type JestOutputParser struct {
	BaseOutputParser
}

// NewJestOutputParser creates a new Jest output parser
func NewJestOutputParser() *JestOutputParser {
	return &JestOutputParser{}
}

// ParseTestOutput parses Jest-specific output format
func (j *JestOutputParser) ParseTestOutput(output string) map[string][]string {
	result := make(map[string][]string)
	lines := strings.Split(output, "\n")

	currentFile := ""
	collectingOutput := false

	for _, line := range lines {
		// Jest file markers
		if strings.HasPrefix(line, "PASS ") || strings.HasPrefix(line, "FAIL ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				currentFile = parts[1]
				collectingOutput = true
				result[currentFile] = append(result[currentFile], line)
			}
		} else if collectingOutput && currentFile != "" {
			// Continue collecting output for current file
			if strings.TrimSpace(line) == "" && strings.HasPrefix(lines[0], "Test Suites:") {
				// End of file output
				collectingOutput = false
				currentFile = ""
			} else {
				result[currentFile] = append(result[currentFile], line)
			}
		}
	}

	return result
}

// VitestOutputParser parses Vitest output
type VitestOutputParser struct {
	BaseOutputParser
}

// NewVitestOutputParser creates a new Vitest output parser
func NewVitestOutputParser() *VitestOutputParser {
	return &VitestOutputParser{}
}

// ParseTestOutput parses Vitest-specific output format
func (v *VitestOutputParser) ParseTestOutput(output string) map[string][]string {
	result := make(map[string][]string)
	lines := strings.Split(output, "\n")

	currentFile := ""

	for _, line := range lines {
		// Vitest file markers (with checkmarks or X)
		if strings.Contains(line, "✓") || strings.Contains(line, "✗") || strings.Contains(line, "↓") {
			// Look for file paths
			parts := strings.Fields(line)
			for _, part := range parts {
				if strings.HasSuffix(part, ".js") || strings.HasSuffix(part, ".ts") ||
					strings.HasSuffix(part, ".jsx") || strings.HasSuffix(part, ".tsx") {
					currentFile = strings.TrimPrefix(part, "./")
					break
				}
			}
		}

		if currentFile != "" {
			result[currentFile] = append(result[currentFile], line)

			// Check if this is the end of output for this file
			if strings.TrimSpace(line) == "" {
				currentFile = ""
			}
		}
	}

	return result
}

// PytestOutputParser parses pytest output
type PytestOutputParser struct {
	BaseOutputParser
}

// NewPytestOutputParser creates a new pytest output parser
func NewPytestOutputParser() *PytestOutputParser {
	return &PytestOutputParser{}
}

// ParseTestOutput parses pytest-specific output format
func (p *PytestOutputParser) ParseTestOutput(output string) map[string][]string {
	result := make(map[string][]string)
	lines := strings.Split(output, "\n")

	currentFile := ""

	for _, line := range lines {
		// pytest file markers
		if strings.HasSuffix(line, ".py") && (strings.Contains(line, "PASSED") ||
			strings.Contains(line, "FAILED") || strings.Contains(line, "SKIPPED")) {
			// Extract file name
			parts := strings.Split(line, "::")
			if len(parts) > 0 {
				currentFile = strings.TrimSpace(parts[0])
			}
		}

		if currentFile != "" {
			result[currentFile] = append(result[currentFile], line)

			// Check for section separators
			if strings.HasPrefix(line, "=") || strings.HasPrefix(line, "-") {
				currentFile = ""
			}
		}
	}

	return result
}

// GetParser returns the appropriate output parser for a runner
func (m *Manager) GetParser(runnerName string) OutputParser {
	switch runnerName {
	case "jest", "jest.js":
		return NewJestOutputParser()
	case "vitest", "vitest.js":
		return NewVitestOutputParser()
	case "pytest", "pytest_adapter.py":
		return NewPytestOutputParser()
	case "cypress", "cypress.js":
		return NewCypressOutputParser()
	case "mocha", "mocha.js":
		// Mocha output is similar enough for fallback parsing
		return NewCypressOutputParser()
	default:
		return &BaseOutputParser{}
	}
}

// CypressOutputParser parses Cypress (Mocha-style) output
type CypressOutputParser struct {
	BaseOutputParser
}

// NewCypressOutputParser creates a new Cypress output parser
func NewCypressOutputParser() *CypressOutputParser {
	return &CypressOutputParser{}
}

// ParseTestOutput parses Cypress/Mocha-style output. This is best-effort only
// since Cypress output can vary; the adapter is the primary source of truth.
func (c *CypressOutputParser) ParseTestOutput(output string) map[string][]string {
	result := make(map[string][]string)
	lines := strings.Split(output, "\n")

	currentFile := ""
	for _, line := range lines {
		// Heuristics: look for spec file paths and test result markers
		if strings.Contains(line, ".cy.") || strings.Contains(line, ".spec.") {
			// naive token search for a file-like segment
			parts := strings.Fields(line)
			for _, part := range parts {
				if strings.Contains(part, ".cy.") || strings.Contains(part, ".spec.") {
					currentFile = strings.TrimPrefix(part, "./")
					break
				}
			}
		}

		if currentFile != "" {
			result[currentFile] = append(result[currentFile], line)
		}
	}
	return result
}
