package definitions

import (
	"testing"

	"github.com/zk/3pio/internal/logger"
)

func TestGoTestDefinition_Detect_FalsePositives(t *testing.T) {
	logger, _ := logger.NewFileLogger()
	defer func() { _ = logger.Close() }()

	g := NewGoTestDefinition(logger)

	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		// Should match
		{"go test", []string{"go", "test"}, true},
		{"go test with package", []string{"go", "test", "./..."}, true},
		{"go test with flags", []string{"go", "test", "-v", "./..."}, true},
		{"full path to go", []string{"/usr/local/bin/go", "test"}, true},
		{"go test specific file", []string{"go", "test", "main_test.go"}, true},

		// Should NOT match - potential false positives
		{"django command", []string{"/usr/bin/django", "test"}, false},
		{"mygo command", []string{"/home/user/mygo", "test"}, false},
		{"go-like command", []string{"go-wrapper", "test"}, false},
		{"gogo test", []string{"gogo", "test"}, false},
		{"test go", []string{"test", "go"}, false},
		{"go without test", []string{"go", "build"}, false},
		{"single go", []string{"go"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.Detect(tt.args)
			if result != tt.expected {
				t.Errorf("GoTestDefinition.Detect(%v) = %v, want %v", tt.args, result, tt.expected)
			}
		})
	}
}

func TestCargoTestDefinition_Detect_FalsePositives(t *testing.T) {
	logger, _ := logger.NewFileLogger()
	defer func() { _ = logger.Close() }()

	c := NewCargoTestDefinition(logger)

	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		// Should match
		{"cargo test", []string{"cargo", "test"}, true},
		{"cargo test with args", []string{"cargo", "test", "--", "--nocapture"}, true},
		{"cargo with toolchain", []string{"cargo", "+nightly", "test"}, true},
		{"full path cargo", []string{"/usr/local/bin/cargo", "test"}, true},
		{"cargo test specific test", []string{"cargo", "test", "test_name"}, true},

		// Should NOT match - potential false positives
		{"mycargo command", []string{"/home/user/mycargo", "test"}, false},
		{"discargo test", []string{"/usr/bin/discargo", "test"}, false},
		{"path ending in cargo dir", []string{"/home/cargo/mycargo", "test"}, false},
		{"cargo-like command", []string{"cargo-wrapper", "test"}, false},
		{"cargogo test", []string{"cargogo", "test"}, false},
		{"test cargo", []string{"test", "cargo"}, false},
		{"cargo without test", []string{"cargo", "build"}, false},
		{"cargo nextest", []string{"cargo", "nextest"}, false}, // This is nextest, not cargo test
		{"single cargo", []string{"cargo"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Detect(tt.args)
			if result != tt.expected {
				t.Errorf("CargoTestDefinition.Detect(%v) = %v, want %v", tt.args, result, tt.expected)
			}
		})
	}
}

func TestNextestDefinition_Detect_FalsePositives(t *testing.T) {
	logger, _ := logger.NewFileLogger()
	defer func() { _ = logger.Close() }()

	n := NewNextestDefinition(logger)

	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		// Should match
		{"cargo nextest", []string{"cargo", "nextest"}, true},
		{"cargo nextest run", []string{"cargo", "nextest", "run"}, true},
		{"cargo with toolchain nextest", []string{"cargo", "+stable", "nextest"}, true},
		{"full path cargo nextest", []string{"/usr/local/bin/cargo", "nextest"}, true},
		{"cargo nextest with args", []string{"cargo", "nextest", "run", "--no-fail-fast"}, true},

		// Should NOT match - potential false positives
		{"mycargo nextest", []string{"/home/user/mycargo", "nextest"}, false},
		{"discargo nextest", []string{"/usr/bin/discargo", "nextest"}, false},
		{"cargo-like nextest", []string{"cargo-wrapper", "nextest"}, false},
		{"cargogo nextest", []string{"cargogo", "nextest"}, false},
		{"nextest cargo", []string{"nextest", "cargo"}, false},
		{"cargo test", []string{"cargo", "test"}, false}, // This is cargo test, not nextest
		{"cargo without nextest", []string{"cargo", "build"}, false},
		{"single cargo", []string{"cargo"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := n.Detect(tt.args)
			if result != tt.expected {
				t.Errorf("NextestDefinition.Detect(%v) = %v, want %v", tt.args, result, tt.expected)
			}
		})
	}
}
