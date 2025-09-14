package adapters

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGetAdapterPath_IPCPathInjection(t *testing.T) {
	tests := []struct {
		name        string
		adapterName string
		ipcPath     string
		runDir      string
		wantErr     bool
		checkFunc   func(t *testing.T, path string, content []byte)
	}{
		{
			name:        "Jest adapter with simple IPC path",
			adapterName: "jest.js",
			ipcPath:     "/tmp/test.jsonl",
			runDir:       ".3pio/runs/20250911T085108-test-run",
			wantErr:     false,
			checkFunc: func(t *testing.T, path string, content []byte) {
				contentStr := string(content)
				// Check that the IPC path was injected
				if !strings.Contains(contentStr, `"/tmp/test.jsonl"`) {
					t.Errorf("Expected injected IPC path '/tmp/test.jsonl' not found in adapter content")
				}
				// Check that template markers were removed
				if strings.Contains(contentStr, "/*__IPC_PATH__*/") {
					t.Errorf("Template markers still present in adapter content")
				}
				// Check that WILL_BE_REPLACED was replaced
				if strings.Contains(contentStr, "WILL_BE_REPLACED") {
					t.Errorf("WILL_BE_REPLACED placeholder still present in adapter content")
				}
			},
		},
		{
			name:        "Vitest adapter with special characters in path",
			adapterName: "vitest.js",
			ipcPath:     "/home/user's files/.3pio/ipc/test.jsonl",
			runDir:       ".3pio/runs/20250911T085108-special-chars",
			wantErr:     false,
			checkFunc: func(t *testing.T, path string, content []byte) {
				contentStr := string(content)
				// Check that the path was properly escaped
				if !strings.Contains(contentStr, `"/home/user's files/.3pio/ipc/test.jsonl"`) &&
					!strings.Contains(contentStr, `"/home/user\'s files/.3pio/ipc/test.jsonl"`) {
					t.Errorf("Expected escaped IPC path not found in adapter content")
				}
				// Check that template markers were removed
				if strings.Contains(contentStr, "/*__IPC_PATH__*/") {
					t.Errorf("Template markers still present in adapter content")
				}
			},
		},
		{
			name:        "Python adapter with IPC path injection",
			adapterName: "pytest_adapter.py",
			ipcPath:     "/var/tmp/.3pio/ipc/test.jsonl",
			runDir:       ".3pio/runs/20250911T085108-python-test",
			wantErr:     false,
			checkFunc: func(t *testing.T, path string, content []byte) {
				contentStr := string(content)
				// Check that the IPC path was injected
				if !strings.Contains(contentStr, `"/var/tmp/.3pio/ipc/test.jsonl"`) {
					t.Errorf("Expected injected IPC path '/var/tmp/.3pio/ipc/test.jsonl' not found in adapter content")
				}
				// Check that template markers were removed
				if strings.Contains(contentStr, "#__IPC_PATH__#") {
					t.Errorf("Template markers still present in adapter content")
				}
				// Check that WILL_BE_REPLACED was replaced
				if strings.Contains(contentStr, "WILL_BE_REPLACED") {
					t.Errorf("WILL_BE_REPLACED placeholder still present in adapter content")
				}
			},
		},
		{
			name:        "Windows-style path with backslashes",
			adapterName: "jest.js",
			ipcPath:     `C:\Users\test\.3pio\ipc\test.jsonl`,
			runDir:       ".3pio/runs/20250911T085108-windows-test",
			wantErr:     false,
			checkFunc: func(t *testing.T, path string, content []byte) {
				contentStr := string(content)
				// Check that backslashes were properly escaped
				if !strings.Contains(contentStr, `"C:\\Users\\test\\.3pio\\ipc\\test.jsonl"`) {
					t.Errorf("Expected escaped Windows path not found in adapter content")
				}
			},
		},
		{
			name:        "Path with Unicode characters",
			adapterName: "vitest.js",
			ipcPath:     "/home/用户/.3pio/ipc/test.jsonl",
			runDir:       ".3pio/runs/20250911T085108-unicode-test",
			wantErr:     false,
			checkFunc: func(t *testing.T, path string, content []byte) {
				contentStr := string(content)
				// Check that Unicode characters are preserved
				if !strings.Contains(contentStr, `"/home/用户/.3pio/ipc/test.jsonl"`) &&
					!strings.Contains(contentStr, `"/home/\u7528\u6237/.3pio/ipc/test.jsonl"`) {
					t.Errorf("Expected Unicode path not found in adapter content")
				}
			},
		},
		{
			name:        "Unknown adapter should fail",
			adapterName: "unknown.js",
			ipcPath:     "/tmp/test.jsonl",
			runDir:       ".3pio/runs/20250911T085108-unknown",
			wantErr:     true,
			checkFunc:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up any existing adapter directory for this run
			_ = os.RemoveAll(tt.runDir)
			defer func() { _ = os.RemoveAll(tt.runDir) }()

			// Call GetAdapterPath with IPC path and run directory
			path, err := GetAdapterPath(tt.adapterName, tt.ipcPath, tt.runDir)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAdapterPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return // Expected error, test passed
			}

			// Verify the adapter was created in the correct directory
			expectedDir := filepath.Join(tt.runDir, "adapters")
			if !strings.Contains(path, expectedDir) {
				t.Errorf("Adapter path %s does not contain expected directory %s", path, expectedDir)
			}

			// Read the adapter content
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("Failed to read adapter file: %v", err)
			}

			// Run custom check function
			if tt.checkFunc != nil {
				tt.checkFunc(t, path, content)
			}

			// Verify file exists
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Errorf("Adapter file does not exist at path: %s", path)
			}
		})
	}
}

func TestGetAdapterPath_UniqueAdaptersPerRun(t *testing.T) {
	ipcPath1 := "/tmp/run1.jsonl"
	ipcPath2 := "/tmp/run2.jsonl"
	runDir1 := ".3pio/runs/20250911T085108-run1"
	runDir2 := ".3pio/runs/20250911T085108-run2"

	// Clean up
	defer func() { _ = os.RemoveAll(runDir1) }()
	defer func() { _ = os.RemoveAll(runDir2) }()

	// Get adapter for first run
	path1, err := GetAdapterPath("jest.js", ipcPath1, runDir1)
	if err != nil {
		t.Fatalf("Failed to get adapter for run1: %v", err)
	}

	// Get adapter for second run
	path2, err := GetAdapterPath("jest.js", ipcPath2, runDir2)
	if err != nil {
		t.Fatalf("Failed to get adapter for run2: %v", err)
	}

	// Verify different paths
	if path1 == path2 {
		t.Errorf("Expected different adapter paths for different runs, got same path: %s", path1)
	}

	// Verify different content (due to different IPC paths)
	content1, _ := os.ReadFile(path1)
	content2, _ := os.ReadFile(path2)

	if strings.Contains(string(content1), ipcPath2) {
		t.Errorf("Adapter for run1 contains IPC path from run2")
	}

	if strings.Contains(string(content2), ipcPath1) {
		t.Errorf("Adapter for run2 contains IPC path from run1")
	}

	// Verify each has correct IPC path
	if !strings.Contains(string(content1), `"/tmp/run1.jsonl"`) {
		t.Errorf("Adapter for run1 does not contain correct IPC path")
	}

	if !strings.Contains(string(content2), `"/tmp/run2.jsonl"`) {
		t.Errorf("Adapter for run2 does not contain correct IPC path")
	}
}

func TestGetAdapterPath_ESMHandling(t *testing.T) {
	ipcPath := "/tmp/test.jsonl"
	runDir := ".3pio/runs/20250911T085108-esm-test"

	// Clean up
	defer func() { _ = os.RemoveAll(runDir) }()

	// Test Vitest adapter (ESM)
	path, err := GetAdapterPath("vitest.js", ipcPath, runDir)
	if err != nil {
		t.Fatalf("Failed to get Vitest adapter: %v", err)
	}

	// Check for package.json with type: module
	pkgPath := filepath.Join(filepath.Dir(path), "package.json")
	if _, err := os.Stat(pkgPath); os.IsNotExist(err) {
		t.Errorf("package.json not created for ESM adapter")
	}

	pkgContent, _ := os.ReadFile(pkgPath)
	if !strings.Contains(string(pkgContent), `"type": "module"`) {
		t.Errorf("package.json does not contain type: module for ESM adapter")
	}
}

func TestGetAdapterPath_PythonExecutable(t *testing.T) {
	ipcPath := "/tmp/test.jsonl"
	runDir := ".3pio/runs/20250911T085108-python-exec-test"

	// Clean up
	defer func() { _ = os.RemoveAll(runDir) }()

	// Test Python adapter
	path, err := GetAdapterPath("pytest_adapter.py", ipcPath, runDir)
	if err != nil {
		t.Fatalf("Failed to get Python adapter: %v", err)
	}

	// Check file permissions (should be executable)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Failed to stat Python adapter: %v", err)
	}

	// Check if executable bit is set (Unix-like systems only)
	// On Windows, executable permissions work differently
	if runtime.GOOS != "windows" {
		if info.Mode()&0111 == 0 {
			t.Errorf("Python adapter is not executable")
		}
	}
}
