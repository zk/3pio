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

// extractAdapter extracts an embedded adapter to .3pio/adapters directory
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

	// Use .3pio/adapters directory with version hash
	hash := fmt.Sprintf("%x", md5.Sum(content))[:8]
	adapterDir := filepath.Join(".3pio", "adapters", hash)
	if err := os.MkdirAll(adapterDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create adapter directory: %w", err)
	}

	// Write adapter file
	adapterPath := filepath.Join(adapterDir, filename)
	
	// Check if file already exists with correct content
	if existing, err := os.ReadFile(adapterPath); err == nil {
		if string(existing) == string(content) {
			// Also check if package.json exists for ESM modules
			if isESM {
				pkgPath := filepath.Join(adapterDir, "package.json")
				if _, err := os.Stat(pkgPath); err == nil {
					// Return absolute path
					if absPath, err := filepath.Abs(adapterPath); err == nil {
						return absPath, nil
					}
					return adapterPath, nil
				}
			} else {
				// Return absolute path
				if absPath, err := filepath.Abs(adapterPath); err == nil {
					return absPath, nil
				}
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

// CleanupAdapters removes all extracted adapter files
func CleanupAdapters() error {
	adapterDir := filepath.Join(".3pio", "adapters")
	return os.RemoveAll(adapterDir)
}