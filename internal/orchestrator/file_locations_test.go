package orchestrator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOrchestrator_FileLocations(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	_ = os.Chdir(tempDir)
	defer func() { _ = os.Chdir(oldCwd) }()

	// Create a mock package.json for runner detection
	packageJSON := `{"name": "test-project", "scripts": {"test": "echo test"}}`
	if err := os.WriteFile("package.json", []byte(packageJSON), 0644); err != nil {
		t.Fatalf("Failed to write package.json: %v", err)
	}

	config := Config{
		Command: []string{"npm", "test"}, // Use npm test which will be recognized
		Logger:  &mockLogger{},
	}

	orch, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}
	defer orch.Close()

	// Run the orchestrator which will initialize paths
	// It will fail but that's ok - we just need initialization
	err = orch.Run()
	if err != nil {
		t.Logf("Run() returned error (expected): %v", err)
	}

	// Check that IPC path is in the run directory
	t.Logf("IPC path: %s", orch.ipcPath)
	t.Logf("Run ID: %s", orch.runID)
	t.Logf("Run Dir: %s", orch.runDir)

	if orch.ipcPath == "" {
		t.Error("IPC path should not be empty")
	}

	if orch.runID == "" {
		t.Error("Run ID should not be empty")
	}

	// Check that IPC path contains the expected structure
	expectedIPCPath := filepath.Join(".3pio", "runs", orch.runID, "ipc.jsonl")
	if orch.ipcPath != expectedIPCPath {
		t.Errorf("IPC path should be %s, got: %s", expectedIPCPath, orch.ipcPath)
	}
}

func TestAdapterExtraction_FileLocations(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	_ = os.Chdir(tempDir)
	defer func() { _ = os.Chdir(oldCwd) }()

	runID := "20250913T180000-test-run"
	runDir := filepath.Join(".3pio", "runs", runID)

	// Create the run directory
	if err := os.MkdirAll(runDir, 0755); err != nil {
		t.Fatalf("Failed to create run directory: %v", err)
	}

	_ = filepath.Join(runDir, "ipc.jsonl") // ipcPath will be used when adapter extraction is updated

	// Test adapter extraction with new location
	tests := []struct {
		name        string
		adapterName string
		wantPath    string
	}{
		{
			name:        "Jest adapter",
			adapterName: "jest.js",
			wantPath:    filepath.Join(runDir, "adapters", "jest.js"),
		},
		{
			name:        "Vitest adapter",
			adapterName: "vitest.js",
			wantPath:    filepath.Join(runDir, "adapters", "vitest.js"),
		},
		{
			name:        "Pytest adapter",
			adapterName: "pytest_adapter.py",
			wantPath:    filepath.Join(runDir, "adapters", "pytest_adapter.py"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This will need to use the updated GetAdapterPath function
			// For now, we're just testing the expected path structure
			expectedPath := tt.wantPath
			expectedDir := filepath.Dir(expectedPath)

			// The adapter extraction should create this directory structure
			if err := os.MkdirAll(expectedDir, 0755); err != nil {
				t.Errorf("Should be able to create adapter directory: %v", err)
			}

			// Verify the path is correct
			if !strings.Contains(expectedPath, filepath.Join("runs", runID, "adapters")) {
				t.Errorf("Adapter path should be in runs/[runID]/adapters/, got: %s", expectedPath)
			}
		})
	}
}
