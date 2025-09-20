package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zk/3pio/internal/logger"
	"github.com/zk/3pio/internal/runner"
)

func TestRunnerDetectionRaceCondition(t *testing.T) {
	// Create a temporary directory with package.json containing both Jest and Vitest
	tempDir := t.TempDir()
	packageJSON := `{
  "name": "test-project",
  "devDependencies": {
    "jest": "^29.0.0",
    "vitest": "^3.0.0"
  },
  "scripts": {
    "test": "vitest"
  }
}`

	err := os.WriteFile(filepath.Join(tempDir, "package.json"), []byte(packageJSON), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Change to test directory
	originalDir, _ := os.Getwd()
	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	// Create logger
	fileLogger, err := logger.NewFileLogger()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = fileLogger.Close() }()

	// Create runner manager
	manager := runner.NewManager(fileLogger)

	// Test explicit Vitest commands - should ALWAYS detect Vitest
	vitestCommands := [][]string{
		{"npx", "vitest", "run"},
		{"yarn", "vitest"},
		{"vitest"},
		{"node", "node_modules/.bin/vitest"},
	}

	for _, cmd := range vitestCommands {
		// Run multiple times to catch race condition
		for i := 0; i < 10; i++ {
			def, err := manager.Detect(cmd)
			if err != nil {
				t.Errorf("Failed to detect runner for command %v: %v", cmd, err)
				continue
			}

			// Check if it's Vitest (by checking adapter file name)
			if def.GetAdapterFileName() != "vitest.js" {
				t.Errorf("Command %v incorrectly detected as %s (expected vitest)", cmd, def.GetAdapterFileName())
			}
		}
	}

	// Test explicit Jest commands - should ALWAYS detect Jest
	jestCommands := [][]string{
		{"npx", "jest"},
		{"yarn", "jest"},
		{"jest"},
		{"node", "node_modules/.bin/jest"},
	}

	for _, cmd := range jestCommands {
		// Run multiple times to catch race condition
		for i := 0; i < 10; i++ {
			def, err := manager.Detect(cmd)
			if err != nil {
				t.Errorf("Failed to detect runner for command %v: %v", cmd, err)
				continue
			}

			// Check if it's Jest (by checking adapter file name)
			if def.GetAdapterFileName() != "jest.js" {
				t.Errorf("Command %v incorrectly detected as %s (expected jest)", cmd, def.GetAdapterFileName())
			}
		}
	}
}

func TestRunnerDetectionPrecedence(t *testing.T) {
	// This test verifies that explicit runner specification takes precedence over package.json

	tempDir := t.TempDir()

	// Create package.json with Jest in dependencies but Vitest in test script
	packageJSON := `{
  "name": "test-project",
  "devDependencies": {
    "jest": "^29.0.0",
    "vitest": "^3.0.0"
  },
  "scripts": {
    "test": "jest"
  }
}`

	err := os.WriteFile(filepath.Join(tempDir, "package.json"), []byte(packageJSON), 0644)
	if err != nil {
		t.Fatal(err)
	}

	originalDir, _ := os.Getwd()
	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	fileLogger, err := logger.NewFileLogger()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = fileLogger.Close() }()

	manager := runner.NewManager(fileLogger)

	// Even though package.json has "test": "jest", explicit vitest command should detect Vitest
	cmd := []string{"npx", "vitest", "run"}

	def, err := manager.Detect(cmd)
	if err != nil {
		t.Fatalf("Failed to detect runner for command %v: %v", cmd, err)
	}

	if def.GetAdapterFileName() != "vitest.js" {
		t.Errorf("Explicit 'npx vitest run' detected as %s, expected vitest.js", def.GetAdapterFileName())
	}

	// Similarly, explicit jest command should detect Jest even if test script uses vitest
	packageJSON2 := `{
  "name": "test-project",
  "devDependencies": {
    "jest": "^29.0.0",
    "vitest": "^3.0.0"
  },
  "scripts": {
    "test": "vitest"
  }
}`

	err = os.WriteFile(filepath.Join(tempDir, "package.json"), []byte(packageJSON2), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cmd = []string{"npx", "jest"}
	def, err = manager.Detect(cmd)
	if err != nil {
		t.Fatalf("Failed to detect runner for command %v: %v", cmd, err)
	}

	if def.GetAdapterFileName() != "jest.js" {
		t.Errorf("Explicit 'npx jest' detected as %s, expected jest.js", def.GetAdapterFileName())
	}
}
