package integration_test

import (
	"io"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestInterruptedTestRun(t *testing.T) {
	projectDir := filepath.Join(fixturesDir, "basic-jest")

	// Clean output directory
	if err := cleanProjectOutput(projectDir); err != nil {
		t.Fatalf("Failed to clean project output: %v", err)
	}

	// Get absolute path to binary
	binaryPath, err := filepath.Abs(threePioBinary)
	if err != nil {
		t.Fatalf("Failed to get absolute path to binary: %v", err)
	}

	// Start 3pio process
	cmd := exec.Command(binaryPath, "npx", "jest")
	cmd.Dir = projectDir

	// Set up pipes to prevent deadlock
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	// Start the command
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start command: %v", err)
	}

	// Wait a brief moment to let the process start and begin execution
	time.Sleep(200 * time.Millisecond)

	// Send SIGTERM to interrupt the process
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("Failed to send SIGTERM: %v", err)
	}

	// Wait for the process to terminate
	_ = cmd.Wait()

	// Check if any output was generated despite being interrupted
	threePioDir := filepath.Join(projectDir, ".3pio")
	if !fileExists(threePioDir) {
		// This is OK - the process might have been killed before creating any files
		t.Log("No .3pio directory found - process was killed very early")
		return
	}

	runsDir := filepath.Join(threePioDir, "runs")
	if !fileExists(runsDir) {
		// This is also OK - process might have been killed before creating runs
		t.Log("No runs directory found - process was killed before test execution")
		return
	}

	// If we did create some output, verify basic structure exists
	runDir := getLatestRunDir(t, projectDir)

	// Check if partial files were created
	testRunPath := filepath.Join(runDir, "test-run.md")
	outputLogPath := filepath.Join(runDir, "output.log")

	// These files might exist in partial form
	if fileExists(testRunPath) {
		t.Log("test-run.md was created before interruption")
	}

	if fileExists(outputLogPath) {
		t.Log("output.log was created before interruption")
	}

	// The key test is that the process can be interrupted gracefully
	// without hanging or causing errors
	t.Log("Process was successfully interrupted")
}

func TestGracefulShutdownOnSIGINT(t *testing.T) {
	projectDir := filepath.Join(fixturesDir, "basic-vitest")

	// Clean output directory
	if err := cleanProjectOutput(projectDir); err != nil {
		t.Fatalf("Failed to clean project output: %v", err)
	}

	// Get absolute path to binary
	binaryPath, err := filepath.Abs(threePioBinary)
	if err != nil {
		t.Fatalf("Failed to get absolute path to binary: %v", err)
	}

	// Start 3pio process
	cmd := exec.Command(binaryPath, "npx", "vitest", "run", "math.test.js")
	cmd.Dir = projectDir

	// Set up pipes to prevent deadlock
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	// Start the command
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start command: %v", err)
	}

	// Wait a brief moment to let the process start
	time.Sleep(100 * time.Millisecond)

	// Send SIGINT (Ctrl+C equivalent)
	if err := cmd.Process.Signal(syscall.SIGINT); err != nil {
		t.Fatalf("Failed to send SIGINT: %v", err)
	}

	// Wait for the process to terminate with a timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-done:
		// Process terminated successfully
		t.Log("Process terminated gracefully after SIGINT")
	case <-time.After(5 * time.Second):
		// Process didn't terminate within timeout, force kill
		_ = cmd.Process.Kill()
		t.Error("Process did not terminate within timeout after SIGINT")
	}
}

func TestQuickProcessTermination(t *testing.T) {
	projectDir := filepath.Join(fixturesDir, "basic-jest")

	// Clean output directory
	if err := cleanProjectOutput(projectDir); err != nil {
		t.Fatalf("Failed to clean project output: %v", err)
	}

	// Get absolute path to binary
	binaryPath, err := filepath.Abs(threePioBinary)
	if err != nil {
		t.Fatalf("Failed to get absolute path to binary: %v", err)
	}

	// Start and immediately kill the process (very quick termination)
	cmd := exec.Command(binaryPath, "npx", "jest")
	cmd.Dir = projectDir

	// Set up pipes to prevent deadlock
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	// Start the command
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start command: %v", err)
	}

	// Kill almost immediately
	time.Sleep(10 * time.Millisecond)

	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("Failed to kill process: %v", err)
	}

	// Wait for termination
	_ = cmd.Wait()

	// The main test is that this doesn't hang or cause issues
	t.Log("Quick termination completed successfully")
}
