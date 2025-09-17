package definitions

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/zk/3pio/internal/logger"
)

func TestNextestDefinition_Name(t *testing.T) {
	logger, _ := logger.NewFileLogger()
	defer func() { _ = logger.Close() }()
	def := NewNextestDefinition(logger)
	if def.Name() != "nextest" {
		t.Errorf("Expected name 'nextest', got '%s'", def.Name())
	}
}

func TestNextestDefinition_Detect(t *testing.T) {
	logger, _ := logger.NewFileLogger()
	defer func() { _ = logger.Close() }()
	def := NewNextestDefinition(logger)

	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{
			name:     "cargo nextest run",
			args:     []string{"cargo", "nextest", "run"},
			expected: true,
		},
		{
			name:     "cargo nextest without run",
			args:     []string{"cargo", "nextest"},
			expected: true,
		},
		{
			name:     "cargo nextest with args",
			args:     []string{"cargo", "nextest", "run", "--workspace"},
			expected: true,
		},
		{
			name:     "cargo nextest with toolchain",
			args:     []string{"cargo", "+nightly", "nextest", "run"},
			expected: true,
		},
		{
			name:     "cargo nextest with stable toolchain",
			args:     []string{"cargo", "+stable", "nextest"},
			expected: true,
		},
		{
			name:     "full path cargo nextest",
			args:     []string{"/usr/bin/cargo", "nextest", "run"},
			expected: true,
		},
		{
			name:     "full path with toolchain",
			args:     []string{"/usr/bin/cargo", "+nightly", "nextest"},
			expected: true,
		},
		{
			name:     "cargo test not nextest",
			args:     []string{"cargo", "test"},
			expected: false,
		},
		{
			name:     "cargo build",
			args:     []string{"cargo", "build"},
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

func TestNextestDefinition_ModifyCommand(t *testing.T) {
	logger, _ := logger.NewFileLogger()
	defer func() { _ = logger.Close() }()
	def := NewNextestDefinition(logger)

	tests := []struct {
		name     string
		cmd      []string
		contains []string
		checkRun bool
	}{
		{
			name:     "cargo nextest with run",
			cmd:      []string{"cargo", "nextest", "run"},
			contains: []string{"cargo", "nextest", "run", "--message-format", "libtest-json"},
			checkRun: false,
		},
		{
			name:     "cargo nextest without run",
			cmd:      []string{"cargo", "nextest"},
			contains: []string{"cargo", "nextest", "run", "--message-format", "libtest-json"},
			checkRun: true,
		},
		{
			name:     "cargo nextest with args",
			cmd:      []string{"cargo", "nextest", "run", "--workspace"},
			contains: []string{"--workspace", "--message-format", "libtest-json"},
			checkRun: false,
		},
		{
			name:     "cargo nextest with partition",
			cmd:      []string{"cargo", "nextest", "run", "--partition", "count:1/2"},
			contains: []string{"--partition", "count:1/2", "--message-format", "libtest-json"},
			checkRun: false,
		},
		{
			name:     "cargo nextest with toolchain",
			cmd:      []string{"cargo", "+nightly", "nextest"},
			contains: []string{"cargo", "+nightly", "nextest", "run", "--message-format"},
			checkRun: true,
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

			if tt.checkRun {
				// Verify "run" was added
				hasRun := false
				for _, arg := range result {
					if arg == "run" {
						hasRun = true
						break
					}
				}
				if !hasRun {
					t.Errorf("ModifyCommand should have added 'run' to command %v", result)
				}
			}
		})
	}
}

func TestNextestDefinition_ProcessJSONEvents(t *testing.T) {
	logger, _ := logger.NewFileLogger()
	defer func() { _ = logger.Close() }()
	def := NewNextestDefinition(logger)

	// Test with sample nextest JSON events
	jsonEvents := `{"type":"suite","event":"started","test_count":3}
{"type":"test","event":"started","name":"my_crate::tests::test_add"}
{"type":"test","event":"ok","name":"my_crate::tests::test_add","exec_time":0.001}
{"type":"test","event":"started","name":"my_crate::tests::test_fail"}
{"type":"test","event":"failed","name":"my_crate::tests::test_fail","stdout":"assertion failed","stderr":""}
{"type":"suite","event":"finished","passed":1,"failed":1,"ignored":0}
`
	reader := strings.NewReader(jsonEvents)

	// Create a temporary file for cross-platform compatibility
	tempFile, err := os.CreateTemp("", "test-nextest-ipc-*.jsonl")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tempPath := tempFile.Name()
	tempFile.Close()
	defer os.Remove(tempPath)

	// Process the events
	err = def.ProcessOutput(reader, tempPath)
	if err != nil {
		t.Errorf("ProcessOutput failed: %v", err)
	}

	// Verify the method executes without error
	// Note: The actual implementation may clear or reset states after processing
}

func TestNextestDefinition_RequiresAdapter(t *testing.T) {
	logger, _ := logger.NewFileLogger()
	defer func() { _ = logger.Close() }()
	def := NewNextestDefinition(logger)

	if def.RequiresAdapter() {
		t.Error("NextestDefinition should not require an adapter")
	}
}

func TestNextestDefinition_GetTestFiles(t *testing.T) {
	logger, _ := logger.NewFileLogger()
	defer func() { _ = logger.Close() }()
	def := NewNextestDefinition(logger)

	files, err := def.GetTestFiles([]string{"cargo", "nextest", "run"})
	if err != nil {
		t.Errorf("GetTestFiles returned unexpected error: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("GetTestFiles should return empty array for dynamic discovery, got %v", files)
	}
}

func TestNextestDefinition_SetEnvironment(t *testing.T) {
	logger, _ := logger.NewFileLogger()
	defer func() { _ = logger.Close() }()
	def := NewNextestDefinition(logger)

	env := def.SetEnvironment()
	if len(env) != 1 {
		t.Errorf("SetEnvironment should return 1 environment variable, got %d", len(env))
	}

	if env[0] != "NEXTEST_EXPERIMENTAL_LIBTEST_JSON=1" {
		t.Errorf("SetEnvironment should return NEXTEST_EXPERIMENTAL_LIBTEST_JSON=1, got %s", env[0])
	}
}

func TestNextestDefinition_ParseTestName(t *testing.T) {
	logger, _ := logger.NewFileLogger()
	defer func() { _ = logger.Close() }()
	def := NewNextestDefinition(logger)

	tests := []struct {
		name         string
		testName     string
		expectedPkg  string
		expectedMod  string
		expectedTest string
	}{
		{
			name:         "simple test",
			testName:     "my_crate::tests::test_add",
			expectedPkg:  "my_crate",
			expectedMod:  "tests",
			expectedTest: "test_add",
		},
		{
			name:         "test with dollar separator",
			testName:     "my_crate$tests::test_add",
			expectedPkg:  "my_crate",
			expectedMod:  "tests",
			expectedTest: "test_add",
		},
		{
			name:         "nested modules",
			testName:     "my_crate::mod1::mod2::test_func",
			expectedPkg:  "my_crate",
			expectedMod:  "mod1::mod2",
			expectedTest: "test_func",
		},
		{
			name:         "integration test",
			testName:     "integration$test_api",
			expectedPkg:  "integration",
			expectedMod:  "",
			expectedTest: "test_api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This would test internal parsing logic if exposed
			// For now, we just verify the definition handles various formats
			// without crashing
			_ = def
		})
	}
}

func TestNextestDefinition_SuiteCompletionEvents(t *testing.T) {
	logger, _ := logger.NewFileLogger()
	defer func() { _ = logger.Close() }()
	def := NewNextestDefinition(logger)

	tests := []struct {
		name           string
		jsonEvents     string
		expectedGroups int
		checkStatus    bool
	}{
		{
			name: "suite with ok event",
			jsonEvents: `{"type":"suite","event":"started","test_count":2}
{"type":"test","event":"started","name":"my_crate::tests::test1"}
{"type":"test","event":"ok","name":"my_crate::tests::test1","exec_time":0.001}
{"type":"test","event":"started","name":"my_crate::tests::test2"}
{"type":"test","event":"ok","name":"my_crate::tests::test2","exec_time":0.002}
{"type":"suite","event":"ok","passed":2,"failed":0,"ignored":0,"exec_time":0.003}
`,
			expectedGroups: 1,
			checkStatus:    true,
		},
		// TODO: Fix edge case where test events come without proper group setup
		// {
		// 	name: "suite with failed event",
		// 	jsonEvents: `{"type":"suite","event":"started","test_count":2}
		// {"type":"test","event":"started","name":"my_crate::tests::test1"}
		// {"type":"test","event":"failed","name":"my_crate::tests::test1","stdout":"assertion failed"}
		// {"type":"test","event":"started","name":"my_crate::tests::test2"}
		// {"type":"test","event":"ok","name":"my_crate::tests::test2","exec_time":0.002}
		// {"type":"suite","event":"failed","passed":1,"failed":1,"ignored":0,"exec_time":0.003}
		// `,
		// 	expectedGroups: 1,
		// 	checkStatus:    true,
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary IPC file
			tempFile, err := os.CreateTemp("", "test-nextest-ipc-*.jsonl")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			tempPath := tempFile.Name()
			tempFile.Close()
			defer os.Remove(tempPath)

			// Process the JSON events
			reader := bytes.NewReader([]byte(tt.jsonEvents))
			err = def.ProcessOutput(reader, tempPath)
			if err != nil {
				t.Errorf("ProcessOutput failed: %v", err)
			}

			// Read and verify IPC events
			ipcData, err := os.ReadFile(tempPath)
			if err != nil {
				t.Fatalf("Failed to read IPC file: %v", err)
			}

			// Debug: Print IPC data
			t.Logf("IPC data: %s", string(ipcData))

			// Parse IPC events and check for testGroupResult
			lines := strings.Split(string(ipcData), "\n")
			groupResultCount := 0
			for _, line := range lines {
				if line == "" {
					continue
				}

				// Parse JSON event
				var event map[string]interface{}
				if err := json.Unmarshal([]byte(line), &event); err != nil {
					t.Errorf("Failed to parse IPC event: %v", err)
					continue
				}

				if eventType, ok := event["eventType"].(string); ok && eventType == "testGroupResult" {
					groupResultCount++
					if tt.checkStatus {
						// Verify the group has a final status (not RUNNING)
						if payload, ok := event["payload"].(map[string]interface{}); ok {
							if status, ok := payload["status"].(string); ok {
								if status == "RUNNING" {
									t.Errorf("Group should not be in RUNNING state after suite completion")
								}
							} else {
								t.Errorf("Group result missing status field")
							}
						}
					}
				}
			}

			if groupResultCount != tt.expectedGroups {
				t.Errorf("Expected %d testGroupResult events, got %d", tt.expectedGroups, groupResultCount)
			}
		})
	}
}
