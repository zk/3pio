package adapters

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

// Embedded adapter files
var (
	//go:embed jest.js
	jestAdapter []byte

	//go:embed vitest.js
	vitestAdapter []byte

	//go:embed pytest_adapter.py
	pytestAdapter []byte

	//go:embed cypress.js
	cypressAdapter []byte

	//go:embed mocha.js
	mochaAdapter []byte
)

// GetAdapterPath returns the path to an extracted adapter with IPC path and log level injected
func GetAdapterPath(name string, ipcPath string, runDir string, logLevel string) (string, error) {
	// No caching needed since each run gets its own adapter
	return extractAdapter(name, ipcPath, runDir, logLevel)
}

// extractAdapter extracts an embedded adapter with IPC path and log level injected
func extractAdapter(name string, ipcPath string, runDir string, logLevel string) (string, error) {
	var content []byte
	var filename string
	var isESM bool

	switch name {
	case "jest.js":
		content = jestAdapter
		// Check if target project is ES module
		if isProjectESM() {
			filename = "jest.cjs" // Use .cjs extension for ES module projects
		} else {
			filename = "jest.js"
		}
		isESM = false
	case "vitest.js":
		content = vitestAdapter
		filename = "vitest.js"
		isESM = true // Vitest adapter is ESM
	case "pytest_adapter.py":
		content = pytestAdapter
		filename = "pytest_adapter.py"
		isESM = false
	case "cypress.js":
		content = cypressAdapter
		// Cypress reporter is CommonJS
		filename = "cypress.js"
		isESM = false
	case "mocha.js":
		content = mochaAdapter
		// Mocha reporter is CommonJS
		filename = "mocha.js"
		isESM = false
	default:
		return "", fmt.Errorf("unknown adapter: %s", name)
	}

	// Replace template markers with actual IPC path
	contentStr := string(content)

	// For JavaScript adapters, inject as single-quoted strings for ESLint consistency
	if name == "vitest.js" || name == "jest.js" || name == "cypress.js" || name == "mocha.js" {
		// Quote using JSON, then convert to single-quoted JS literal
		jsonQuoted := strconv.Quote(ipcPath)
		if len(jsonQuoted) >= 2 {
			jsonQuoted = jsonQuoted[1 : len(jsonQuoted)-1] // strip outer quotes
		}
		singleQuoted := "'" + jsonQuoted + "'"
		// Replace both compact and spaced markers, and both quote styles
		patterns := []*regexp.Regexp{
			regexp.MustCompile(`/\*__IPC_PATH__\*/\s*[\"'][^\"']*[\"']\s*;?\s*/\*__IPC_PATH__\*/`),
			regexp.MustCompile(`/\*\s*__IPC_PATH__\s*\*/\s*[\"'][^\"']*[\"']\s*;?\s*/\*\s*__IPC_PATH__\s*\*/`),
		}
		for _, pattern := range patterns {
			contentStr = pattern.ReplaceAllString(contentStr, singleQuoted)
		}
	}

	// For Python adapter, use Python string escaping
	if name == "pytest_adapter.py" {
		// Python uses similar escaping to JSON for basic strings
		escapedPath := strconv.Quote(ipcPath)
		pattern := regexp.MustCompile(`#__IPC_PATH__#".*?"#__IPC_PATH__#`)
		contentStr = pattern.ReplaceAllString(contentStr, escapedPath)
	}

	// Inject log level into all adapters
	// For JavaScript adapters, inject log level as single-quoted strings
	if name == "vitest.js" || name == "jest.js" || name == "cypress.js" || name == "mocha.js" {
		jsonQuoted := strconv.Quote(logLevel)
		if len(jsonQuoted) >= 2 {
			jsonQuoted = jsonQuoted[1 : len(jsonQuoted)-1]
		}
		singleQuoted := "'" + jsonQuoted + "'"
		patterns := []*regexp.Regexp{
			regexp.MustCompile(`/\*__LOG_LEVEL__\*/\s*[\"'][^\"']*[\"']\s*;?\s*/\*__LOG_LEVEL__\*/`),
			regexp.MustCompile(`/\*\s*__LOG_LEVEL__\s*\*/\s*[\"'][^\"']*[\"']\s*;?\s*/\*\s*__LOG_LEVEL__\s*\*/`),
		}
		for _, pattern := range patterns {
			contentStr = pattern.ReplaceAllString(contentStr, singleQuoted)
		}
	}

	// For Python adapter, use Python string escaping for log level
	if name == "pytest_adapter.py" {
		escapedLogLevel := strconv.Quote(logLevel)
		logPattern := regexp.MustCompile(`#__LOG_LEVEL__#".*?"#__LOG_LEVEL__#`)
		contentStr = logPattern.ReplaceAllString(contentStr, escapedLogLevel)
	}

	content = []byte(contentStr)

	// Use run directory for adapter location
	adapterDir := filepath.Join(runDir, "adapters")
	if err := os.MkdirAll(adapterDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create adapter directory: %w", err)
	}

	// Write adapter file
	adapterPath := filepath.Join(adapterDir, filename)

	// No need to check if file exists - each run gets its own adapter

	// Write the adapter file
	if err := os.WriteFile(adapterPath, content, 0644); err != nil {
		return "", fmt.Errorf("failed to write adapter file: %w", err)
	}

	// For ESM modules, create a package.json with type: module
	if isESM {
		packageJSON := `{"type": "module"}`
		pkgPath := filepath.Join(adapterDir, "package.json")
		if err := os.WriteFile(pkgPath, []byte(packageJSON), 0644); err != nil {
			return "", fmt.Errorf("failed to write package.json: %w", err)
		}
	}

	// Make Python adapter executable
	if name == "pytest_adapter.py" {
		_ = os.Chmod(adapterPath, 0755)
	}

	// Return absolute path for compatibility with all test runners
	absPath, err := filepath.Abs(adapterPath)
	if err != nil {
		return adapterPath, nil // Fall back to relative path if abs fails
	}
	return absPath, nil
}

// isProjectESM checks if the current project is configured as an ES module
func isProjectESM() bool {
	// Check if package.json exists and has "type": "module"
	packagePath := "package.json"
	if _, err := os.Stat(packagePath); err != nil {
		return false // No package.json found
	}

	data, err := os.ReadFile(packagePath)
	if err != nil {
		return false // Can't read package.json
	}

	var pkg struct {
		Type string `json:"type"`
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
		return false // Invalid JSON
	}

	return pkg.Type == "module"
}
