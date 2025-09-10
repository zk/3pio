package integration_test

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNpmSeparatorHandling(t *testing.T) {
	projectDir := filepath.Join(fixturesDir, "npm-separator-jest")
	
	// Clean output directory
	if err := cleanProjectOutput(projectDir); err != nil {
		t.Fatalf("Failed to clean project output: %v", err)
	}
	
	// Get absolute path to binary
	binaryPath, err := filepath.Abs(threePioBinary)
	if err != nil {
		t.Fatalf("Failed to get absolute path to binary: %v", err)
	}
	
	// Run 3pio with npm test -- format
	cmd := exec.Command(binaryPath, "npm", "test", "--", "example.test.js")
	cmd.Dir = projectDir
	output, err := cmd.CombinedOutput()
	
	// We expect this to work (exit code might be non-zero due to test failures, but that's OK)
	outputStr := string(output)
	
	// Check that the command format is preserved
	if !strings.Contains(outputStr, "npm test -- example.test.js") && 
	   !strings.Contains(outputStr, "example.test.js") {
		t.Errorf("Output should contain the test command or file name. Got: %s", outputStr)
	}
	
	// Verify that files were created
	runDir := getLatestRunDir(t, projectDir)
	
	// Check for basic file structure
	testRunPath := filepath.Join(runDir, "test-run.md")
	if !fileExists(testRunPath) {
		t.Error("npm separator test should create test-run.md")
	}
	
	outputLogPath := filepath.Join(runDir, "output.log")
	if !fileExists(outputLogPath) {
		t.Error("npm separator test should create output.log")
	}
	
	// Check content
	content := readFile(t, testRunPath)
	if !strings.Contains(content, "# 3pio Test Run") {
		t.Error("npm separator test should create proper report structure")
	}
	
	// Check that the output.log contains the correct command
	outputContent := readFile(t, outputLogPath)
	if !strings.Contains(outputContent, "# Command: npm test -- example.test.js") {
		t.Error("output.log should contain the correct command with separator")
	}
}

func TestBasicJestExampleFileHandling(t *testing.T) {
	projectDir := filepath.Join(fixturesDir, "npm-separator-jest")
	
	// Check if the fixture exists
	if !fileExists(projectDir) {
		t.Skip("npm-separator-jest fixture not found, skipping test")
	}
	
	// Clean output directory
	if err := cleanProjectOutput(projectDir); err != nil {
		t.Fatalf("Failed to clean project output: %v", err)
	}
	
	// Get absolute path to binary
	binaryPath, err := filepath.Abs(threePioBinary)
	if err != nil {
		t.Fatalf("Failed to get absolute path to binary: %v", err)
	}
	
	// Run 3pio directly with the test file
	cmd := exec.Command(binaryPath, "npx", "jest", "example.test.js")
	cmd.Dir = projectDir
	// Ignore exit code - test might fail
	_ = cmd.Run()
	
	// Verify that files were created
	runDir := getLatestRunDir(t, projectDir)
	
	// Check for basic file structure
	testRunPath := filepath.Join(runDir, "test-run.md")
	if !fileExists(testRunPath) {
		t.Error("example.test.js should create test-run.md")
	}
	
	outputLogPath := filepath.Join(runDir, "output.log")
	if !fileExists(outputLogPath) {
		t.Error("example.test.js should create output.log")
	}
}

func TestNpmRunTestCommand(t *testing.T) {
	projectDir := filepath.Join(fixturesDir, "basic-vitest") // Use a known working fixture
	
	// Clean output directory
	if err := cleanProjectOutput(projectDir); err != nil {
		t.Fatalf("Failed to clean project output: %v", err)
	}
	
	// Get absolute path to binary
	binaryPath, err := filepath.Abs(threePioBinary)
	if err != nil {
		t.Fatalf("Failed to get absolute path to binary: %v", err)
	}
	
	// Run 3pio with npm run test
	cmd := exec.Command(binaryPath, "npm", "run", "test")
	cmd.Dir = projectDir
	// Ignore exit code - focus on whether we can handle the command
	_ = cmd.Run()
	
	// Check if run directory was created (basic success indicator)
	runsDir := filepath.Join(projectDir, ".3pio", "runs")
	if !fileExists(runsDir) {
		t.Error("npm run test should create runs directory")
	}
}