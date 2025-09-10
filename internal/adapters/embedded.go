package adapters

import (
	_ "embed"
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
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

// GetAdapterPath returns the path to an extracted adapter
func GetAdapterPath(name string) (string, error) {
	// Check cache first
	cache.mu.RLock()
	if path, ok := cache.paths[name]; ok {
		cache.mu.RUnlock()
		// Verify file still exists
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	cache.mu.RUnlock()

	// Extract adapter
	cache.mu.Lock()
	defer cache.mu.Unlock()

	path, err := extractAdapter(name)
	if err != nil {
		return "", err
	}

	cache.paths[name] = path
	return path, nil
}

// extractAdapter extracts an embedded adapter to a temporary directory
func extractAdapter(name string) (string, error) {
	var content []byte
	var filename string
	var isESM bool

	switch name {
	case "jest.js":
		content = jestAdapter
		filename = "jest.js"
		isESM = false
	case "vitest.js":
		content = vitestAdapter
		filename = "vitest.js"
		isESM = true  // Vitest adapter is ESM
	case "pytest_adapter.py":
		content = pytestAdapter
		filename = "pytest_adapter.py"
		isESM = false
	default:
		return "", fmt.Errorf("unknown adapter: %s", name)
	}

	// Create temp directory with version hash
	hash := fmt.Sprintf("%x", md5.Sum(content))[:8]
	tempDir := filepath.Join(os.TempDir(), "3pio-adapters", hash)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Write adapter file
	adapterPath := filepath.Join(tempDir, filename)
	
	// Check if file already exists with correct content
	if existing, err := os.ReadFile(adapterPath); err == nil {
		if string(existing) == string(content) {
			// Also check if package.json exists for ESM modules
			if isESM {
				pkgPath := filepath.Join(tempDir, "package.json")
				if _, err := os.Stat(pkgPath); err == nil {
					return adapterPath, nil
				}
			} else {
				return adapterPath, nil
			}
		}
	}

	// Write the adapter file
	if err := os.WriteFile(adapterPath, content, 0644); err != nil {
		return "", fmt.Errorf("failed to write adapter file: %w", err)
	}

	// For ESM modules, create a package.json with type: module
	if isESM {
		packageJSON := `{"type": "module"}`
		pkgPath := filepath.Join(tempDir, "package.json")
		if err := os.WriteFile(pkgPath, []byte(packageJSON), 0644); err != nil {
			return "", fmt.Errorf("failed to write package.json: %w", err)
		}
	}

	// Make Python adapter executable
	if name == "pytest_adapter.py" {
		os.Chmod(adapterPath, 0755)
	}

	return adapterPath, nil
}

// CleanupAdapters removes all extracted adapter files
func CleanupAdapters() error {
	tempDir := filepath.Join(os.TempDir(), "3pio-adapters")
	return os.RemoveAll(tempDir)
}