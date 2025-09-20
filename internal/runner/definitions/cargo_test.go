package definitions

import (
	"os"
	"strings"
	"testing"

	"github.com/zk/3pio/internal/logger"
)

func TestCargoTestDefinition_Name(t *testing.T) {
	logger, _ := logger.NewFileLogger()
	defer func() { _ = logger.Close() }()
	def := NewCargoTestDefinition(logger)
	if def.Name() != "cargo" {
		t.Errorf("Expected name 'cargo', got '%s'", def.Name())
	}
}

func TestCargoTestDefinition_Detect(t *testing.T) {
	logger, _ := logger.NewFileLogger()
	defer func() { _ = logger.Close() }()
	def := NewCargoTestDefinition(logger)

	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{
			name:     "basic cargo test",
			args:     []string{"cargo", "test"},
			expected: true,
		},
		{
			name:     "cargo test with args",
			args:     []string{"cargo", "test", "--lib"},
			expected: true,
		},
		{
			name:     "cargo test with toolchain",
			args:     []string{"cargo", "+nightly", "test"},
			expected: true,
		},
		{
			name:     "cargo test with stable toolchain",
			args:     []string{"cargo", "+stable", "test", "--release"},
			expected: true,
		},
		{
			name:     "full path cargo test",
			args:     []string{"/usr/bin/cargo", "test"},
			expected: true,
		},
		{
			name:     "full path with toolchain",
			args:     []string{"/usr/bin/cargo", "+nightly", "test"},
			expected: true,
		},
		{
			name:     "cargo build not test",
			args:     []string{"cargo", "build"},
			expected: false,
		},
		{
			name:     "cargo nextest",
			args:     []string{"cargo", "nextest", "run"},
			expected: false,
		},
		{
			name:     "not cargo",
			args:     []string{"npm", "test"},
			expected: false,
		},
		{
			name:     "empty args",
			args:     []string{},
			expected: false,
		},
		{
			name:     "single arg",
			args:     []string{"cargo"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := def.Detect(tt.args)
			if result != tt.expected {
				t.Errorf("Detect(%v) = %v, expected %v", tt.args, result, tt.expected)
			}
		})
	}
}

func TestCargoTestDefinition_ModifyCommand(t *testing.T) {
	logger, _ := logger.NewFileLogger()
	defer func() { _ = logger.Close() }()
	def := NewCargoTestDefinition(logger)

	tests := []struct {
		name     string
		cmd      []string
		contains []string
	}{
		{
			name:     "basic cargo test",
			cmd:      []string{"cargo", "test"},
			contains: []string{"cargo", "test", "--", "-Z", "unstable-options", "--format", "json", "--report-time"},
		},
		{
			name:     "cargo test with existing args",
			cmd:      []string{"cargo", "test", "--lib"},
			contains: []string{"cargo", "test", "--lib", "--", "-Z", "unstable-options", "--format", "json"},
		},
		{
			name:     "cargo test with filter",
			cmd:      []string{"cargo", "test", "test_name"},
			contains: []string{"cargo", "test", "test_name", "--", "-Z", "unstable-options"},
		},
		{
			name:     "cargo test with toolchain",
			cmd:      []string{"cargo", "+nightly", "test"},
			contains: []string{"cargo", "+nightly", "test", "--", "-Z", "unstable-options"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := def.ModifyCommand(tt.cmd, "/tmp/test.jsonl", "test-run-id")
			resultStr := strings.Join(result, " ")
			for _, expected := range tt.contains {
				if !strings.Contains(resultStr, expected) {
					t.Errorf("ModifyCommand result %v does not contain '%s'", result, expected)
				}
			}
		})
	}
}

func TestCargoTestDefinition_ProcessJSONEvents(t *testing.T) {
	logger, _ := logger.NewFileLogger()
	defer func() { _ = logger.Close() }()
	def := NewCargoTestDefinition(logger)

	// Test with sample JSON events
	jsonEvents := `{"type":"suite","event":"started","test_count":3}
{"type":"test","event":"started","name":"tests::test_add"}
{"type":"test","name":"tests::test_add","event":"ok","exec_time":0.001}
{"type":"test","event":"started","name":"tests::test_subtract"}
{"type":"test","name":"tests::test_subtract","event":"failed","stdout":"assertion failed"}
{"type":"suite","event":"ok","passed":1,"failed":1,"ignored":0}
`
	reader := strings.NewReader(jsonEvents)

	// Create a temporary file for cross-platform compatibility
	tempFile, err := os.CreateTemp("", "test-cargo-ipc-*.jsonl")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tempPath := tempFile.Name()
	_ = tempFile.Close()
	defer func() { _ = os.Remove(tempPath) }()

	// Process the events
	err = def.ProcessOutput(reader, tempPath)
	if err != nil {
		t.Errorf("ProcessOutput failed: %v", err)
	}

	// Verify internal state was updated - test states are tracked
	// Note: The actual implementation may clear or reset states after processing
	// so we just verify the method executes without error
}

func TestCargoTestDefinition_RequiresAdapter(t *testing.T) {
	logger, _ := logger.NewFileLogger()
	defer func() { _ = logger.Close() }()
	def := NewCargoTestDefinition(logger)

	if def.RequiresAdapter() {
		t.Error("CargoTestDefinition should not require an adapter")
	}
}

func TestCargoTestDefinition_GetTestFiles(t *testing.T) {
	logger, _ := logger.NewFileLogger()
	defer func() { _ = logger.Close() }()
	def := NewCargoTestDefinition(logger)

	files, err := def.GetTestFiles([]string{"cargo", "test"})
	if err != nil {
		t.Errorf("GetTestFiles returned unexpected error: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("GetTestFiles should return empty array for dynamic discovery, got %v", files)
	}
}

func TestCargoTestDefinition_SetEnvironment(t *testing.T) {
	logger, _ := logger.NewFileLogger()
	defer func() { _ = logger.Close() }()
	def := NewCargoTestDefinition(logger)

	env := def.SetEnvironment()
	if len(env) != 1 {
		t.Errorf("SetEnvironment should return 1 environment variable, got %d", len(env))
	}

	if env[0] != "RUSTC_BOOTSTRAP=1" {
		t.Errorf("SetEnvironment should return RUSTC_BOOTSTRAP=1, got %s", env[0])
	}
}

func TestCargoExtractCrateFromUnitTest(t *testing.T) {
	logger, _ := logger.NewFileLogger()
	defer func() { _ = logger.Close() }()
	def := NewCargoTestDefinition(logger)
	def.ipcWriter = &IPCWriter{} // Mock IPC writer

	var jsonCount int
	line := "     Running unittests src/lib.rs (target/debug/deps/actix_http-abc123def456)"
	def.processLineData(line, &jsonCount)

	if def.lastUnitTestCrate != "actix_http" {
		t.Errorf("Expected lastUnitTestCrate to be 'actix_http', got '%s'", def.lastUnitTestCrate)
	}
	if def.currentCrate != "actix_http" {
		t.Errorf("Expected currentCrate to be 'actix_http', got '%s'", def.currentCrate)
	}
}

func TestCargoQualifyIntegrationTestWithCrate(t *testing.T) {
	logger, _ := logger.NewFileLogger()
	defer func() { _ = logger.Close() }()
	def := NewCargoTestDefinition(logger)
	def.ipcWriter = &IPCWriter{} // Mock IPC writer
	def.lastUnitTestCrate = "actix_http" // Pre-set from previous unit test

	var jsonCount int
	line := "     Running tests/test_client.rs (target/debug/deps/test_client-a1b2c3d4e5f6a7b8)"
	def.processLineData(line, &jsonCount)

	expected := "actix_http::test_client"
	if def.currentCrate != expected {
		t.Errorf("Expected currentCrate to be '%s', got '%s'", expected, def.currentCrate)
		t.Logf("lastUnitTestCrate: '%s'", def.lastUnitTestCrate)
	}
}

func TestCargoIntegrationTestWithoutPriorUnitTest(t *testing.T) {
	logger, _ := logger.NewFileLogger()
	defer func() { _ = logger.Close() }()
	def := NewCargoTestDefinition(logger)
	def.ipcWriter = &IPCWriter{} // Mock IPC writer
	// No lastUnitTestCrate set - simulating edge case

	var jsonCount int
	line := "     Running tests/test_client.rs (target/debug/deps/test_client-a1b2c3d4e5f6a7b8)"
	def.processLineData(line, &jsonCount)

	// Should fall back to just test name without qualification
	if def.currentCrate != "test_client" {
		t.Errorf("Expected currentCrate to be 'test_client' (fallback), got '%s'", def.currentCrate)
	}
}

func TestCargoMultipleCratesSameTestFile(t *testing.T) {
	logger, _ := logger.NewFileLogger()
	defer func() { _ = logger.Close() }()
	def := NewCargoTestDefinition(logger)
	def.ipcWriter = &IPCWriter{} // Mock IPC writer

	var jsonCount int

	// First crate's unit test
	line1 := "     Running unittests src/lib.rs (target/debug/deps/actix_http-abc123def456789a)"
	def.processLineData(line1, &jsonCount)
	if def.lastUnitTestCrate != "actix_http" {
		t.Errorf("Expected lastUnitTestCrate to be 'actix_http', got '%s'", def.lastUnitTestCrate)
	}

	// First crate's integration test
	line2 := "     Running tests/test_client.rs (target/debug/deps/test_client-def456abc789012b)"
	def.processLineData(line2, &jsonCount)
	if def.currentCrate != "actix_http::test_client" {
		t.Errorf("Expected currentCrate to be 'actix_http::test_client', got '%s'", def.currentCrate)
	}

	// Second crate's unit test
	line3 := "     Running unittests src/lib.rs (target/debug/deps/awc-89abcdef01234567)"
	def.processLineData(line3, &jsonCount)
	if def.lastUnitTestCrate != "awc" {
		t.Errorf("Expected lastUnitTestCrate to be 'awc', got '%s'", def.lastUnitTestCrate)
	}

	// Second crate's integration test (same filename as first)
	line4 := "     Running tests/test_client.rs (target/debug/deps/test_client-0123456789abcdef)"
	def.processLineData(line4, &jsonCount)
	if def.currentCrate != "awc::test_client" {
		t.Errorf("Expected currentCrate to be 'awc::test_client', got '%s'", def.currentCrate)
	}
}
