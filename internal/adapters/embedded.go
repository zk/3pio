package adapters

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
)

// Embedded adapter files
var (
	//go:embed jest.js
	jestAdapter []byte

	//go:embed vitest.js
	vitestAdapter []byte

	//go:embed pytest_adapter.py
	pytestAdapter []byte
)

// adapterCache manages extracted adapter paths
type adapterCache struct {
	mu    sync.RWMutex
	paths map[string]string
}

var cache = &adapterCache{
	paths: make(map[string]string),
}

// GetAdapterPath returns the path to an extracted adapter with IPC path injected
func GetAdapterPath(name string, ipcPath string, runID string) (string, error) {
	// No caching needed since each run gets its own adapter
	return extractAdapter(name, ipcPath, runID)
}

// extractAdapter extracts an embedded adapter with IPC path injected
func extractAdapter(name string, ipcPath string, runID string) (string, error) {
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
	default:
		return "", fmt.Errorf("unknown adapter: %s", name)
	}

	// Replace template markers with actual IPC path
	contentStr := string(content)
	
	// For JavaScript adapters, use JSON-like escaping
	if name == "vitest.js" || name == "jest.js" {
		escapedPath := strconv.Quote(ipcPath) // Go's strconv.Quote is similar to JSON.stringify
		// Replace the markers
		pattern := regexp.MustCompile(`/\*__IPC_PATH__\*/".*?"/\*__IPC_PATH__\*/`)
		contentStr = pattern.ReplaceAllString(contentStr, escapedPath)
	}
	
	// For Python adapter, use Python string escaping
	if name == "pytest_adapter.py" {
		// Python uses similar escaping to JSON for basic strings
		escapedPath := strconv.Quote(ipcPath)
		pattern := regexp.MustCompile(`#__IPC_PATH__#".*?"#__IPC_PATH__#`)
		contentStr = pattern.ReplaceAllString(contentStr, escapedPath)
	}
	
	content = []byte(contentStr)
	
	// Use run ID for adapter directory (e.g., "20250911T085108-feisty-han-solo")
	adapterDir := filepath.Join(".3pio", "adapters", runID)
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

// CleanupAdapters removes all extracted adapter files
func CleanupAdapters() error {
	adapterDir := filepath.Join(".3pio", "adapters")
	return os.RemoveAll(adapterDir)
}
