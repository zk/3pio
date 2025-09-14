package runner

import (
	"fmt"
	"strings"

	"github.com/zk/3pio/internal/logger"
	"github.com/zk/3pio/internal/runner/definitions"
)

// Manager manages test runner definitions
type Manager struct {
	runners map[string]Definition
	logger  *logger.FileLogger
}

// NewManager creates a new runner manager
func NewManager() *Manager {
	// Create a logger for the manager
	fileLogger, _ := logger.NewFileLogger()

	m := &Manager{
		runners: make(map[string]Definition),
		logger:  fileLogger,
	}

	// Register built-in runners
	m.Register("jest", NewJestDefinition())
	m.Register("vitest", NewVitestDefinition())
	m.Register("pytest", NewPytestDefinition())

	// Register Go test runner (native, no adapter)
	m.Register("go", definitions.NewGoTestWrapper(fileLogger))

	return m
}

// Register adds a new test runner definition
func (m *Manager) Register(name string, def Definition) {
	m.runners[name] = def
}

// Detect identifies the test runner from command and returns its definition
func (m *Manager) Detect(command []string) (Definition, error) {
	// Check each runner to see if it matches
	for _, def := range m.runners {
		if def.Matches(command) {
			return def, nil
		}
	}

	// Special handling for npm/yarn/pnpm commands
	if len(command) > 0 {
		packageManager := command[0]
		if isPackageManager(packageManager) {
			// Try to detect from package.json test script
			for _, def := range m.runners {
				if def.Matches([]string{"test"}) {
					return def, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("no test runner detected for command: %s", strings.Join(command, " "))
}

// GetDefinition returns a specific runner definition by name
func (m *Manager) GetDefinition(name string) (Definition, bool) {
	def, ok := m.runners[name]
	return def, ok
}

// isPackageManager checks if a command is a package manager
func isPackageManager(cmd string) bool {
	managers := []string{"npm", "yarn", "pnpm", "bun"}
	for _, m := range managers {
		if strings.Contains(cmd, m) {
			return true
		}
	}
	return false
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
	case "jest":
		return NewJestOutputParser()
	case "vitest":
		return NewVitestOutputParser()
	case "pytest":
		return NewPytestOutputParser()
	default:
		return &BaseOutputParser{}
	}
}
