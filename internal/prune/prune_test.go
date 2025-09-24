package prune

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zk/3pio/internal/logger"
)

func TestParseTimestamp(t *testing.T) {
	// Create a pruner with a mock logger
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, ".3pio"), 0755)
	os.Chdir(tmpDir)

	fileLogger, err := logger.NewFileLogger()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer fileLogger.Close()

	p := &Pruner{
		logger:  fileLogger,
		baseDir: ".3pio",
	}

	tests := []struct {
		name      string
		dirname   string
		wantEmpty bool
	}{
		{
			name:      "valid timestamp with memorable name",
			dirname:   "20240315_143025_flying-banana",
			wantEmpty: false,
		},
		{
			name:      "valid timestamp without memorable name",
			dirname:   "20240315_143025",
			wantEmpty: false,
		},
		{
			name:      "invalid format",
			dirname:   "invalid-dirname",
			wantEmpty: true,
		},
		{
			name:      "partial timestamp",
			dirname:   "20240315",
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.parseTimestamp(tt.dirname)
			if tt.wantEmpty && !result.IsZero() {
				t.Errorf("Expected zero time for %s, got %v", tt.dirname, result)
			}
			if !tt.wantEmpty && result.IsZero() {
				t.Errorf("Expected valid time for %s, got zero", tt.dirname)
			}
		})
	}
}

func TestGetRuns(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".3pio")
	runsDir := filepath.Join(baseDir, "runs")
	os.MkdirAll(runsDir, 0755)

	// Change to temp directory
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Create test runs with different timestamps
	testRuns := []string{
		"20240101_120000_oldest-run",
		"20240102_120000_middle-run",
		"20240103_120000_newest-run",
	}

	for _, run := range testRuns {
		runPath := filepath.Join(runsDir, run)
		os.MkdirAll(runPath, 0755)
		// Create a test file in each run
		testFile := filepath.Join(runPath, "test-run.md")
		os.WriteFile(testFile, []byte("test content"), 0644)
	}

	// Create a non-directory file that should be ignored
	os.WriteFile(filepath.Join(runsDir, "not-a-directory.txt"), []byte("ignore me"), 0644)

	// Create logger
	fileLogger, err := logger.NewFileLogger()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer fileLogger.Close()

	p := &Pruner{
		logger:  fileLogger,
		baseDir: ".3pio",
	}

	runs, err := p.getRuns()
	if err != nil {
		t.Fatalf("getRuns failed: %v", err)
	}

	if len(runs) != 3 {
		t.Errorf("Expected 3 runs, got %d", len(runs))
	}

	// Check that runs have correct paths
	for _, run := range runs {
		if run.Size == 0 {
			t.Errorf("Run %s has zero size", run.Path)
		}
		if run.Timestamp.IsZero() {
			t.Errorf("Run %s has zero timestamp", run.Path)
		}
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		result := formatSize(tt.bytes)
		if result != tt.expected {
			t.Errorf("formatSize(%d) = %s, want %s", tt.bytes, result, tt.expected)
		}
	}
}

func TestPruneNoDirectory(t *testing.T) {
	// Create temp directory without .3pio
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Create logger in a temporary .3pio dir
	os.MkdirAll(".3pio", 0755)
	fileLogger, err := logger.NewFileLogger()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer fileLogger.Close()

	// Remove .3pio to test missing directory
	os.RemoveAll(".3pio")

	p := New(fileLogger, false, true)
	err = p.Run()
	if err != nil {
		t.Errorf("Expected no error for missing .3pio directory, got: %v", err)
	}
}

func TestPruneNoRuns(t *testing.T) {
	// Create temp directory with empty .3pio
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".3pio")
	os.MkdirAll(baseDir, 0755)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	fileLogger, err := logger.NewFileLogger()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer fileLogger.Close()

	p := New(fileLogger, false, true)
	err = p.Run()
	if err != nil {
		t.Errorf("Expected no error for no runs, got: %v", err)
	}
}

func TestPruneSingleRun(t *testing.T) {
	// Create temp directory with single run
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".3pio")
	runsDir := filepath.Join(baseDir, "runs")
	runPath := filepath.Join(runsDir, "20240101_120000_single-run")
	os.MkdirAll(runPath, 0755)
	os.WriteFile(filepath.Join(runPath, "test-run.md"), []byte("test"), 0644)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	fileLogger, err := logger.NewFileLogger()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer fileLogger.Close()

	p := New(fileLogger, false, true)
	err = p.Run()
	if err != nil {
		t.Errorf("Expected no error for single run, got: %v", err)
	}

	// Verify the run still exists
	if _, err := os.Stat(runPath); os.IsNotExist(err) {
		t.Errorf("Single run should not be deleted")
	}
}

func TestPruneDryRun(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".3pio")
	runsDir := filepath.Join(baseDir, "runs")
	os.MkdirAll(runsDir, 0755)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Create multiple test runs
	testRuns := []string{
		"20240101_120000_oldest-run",
		"20240102_120000_middle-run",
		"20240103_120000_newest-run",
	}

	for _, run := range testRuns {
		runPath := filepath.Join(runsDir, run)
		os.MkdirAll(runPath, 0755)
		os.WriteFile(filepath.Join(runPath, "test-run.md"), []byte("test"), 0644)
	}

	fileLogger, err := logger.NewFileLogger()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer fileLogger.Close()

	// Run with dry-run flag
	p := New(fileLogger, true, true)
	err = p.Run()
	if err != nil {
		t.Errorf("Dry run failed: %v", err)
	}

	// Verify all runs still exist
	for _, run := range testRuns {
		runPath := filepath.Join(runsDir, run)
		if _, err := os.Stat(runPath); os.IsNotExist(err) {
			t.Errorf("Run %s should not be deleted in dry-run mode", run)
		}
	}
}

func TestPruneMultipleRuns(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".3pio")
	runsDir := filepath.Join(baseDir, "runs")
	os.MkdirAll(runsDir, 0755)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Create multiple test runs
	testRuns := []struct {
		name    string
		keepRun bool
	}{
		{"20240101_120000_oldest-run", false},
		{"20240102_120000_middle-run", false},
		{"20240103_120000_newest-run", true}, // This should be kept
	}

	for _, run := range testRuns {
		runPath := filepath.Join(runsDir, run.name)
		os.MkdirAll(runPath, 0755)
		os.WriteFile(filepath.Join(runPath, "test-run.md"), []byte("test content"), 0644)
	}

	// Create debug log
	debugLogPath := filepath.Join(baseDir, "debug.log")
	os.WriteFile(debugLogPath, []byte("debug log content to clear"), 0644)

	fileLogger, err := logger.NewFileLogger()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	// Close logger before pruning to avoid file lock issues
	fileLogger.Close()

	// Run prune with force flag
	p := New(nil, false, true) // Use nil logger to avoid conflicts
	p.baseDir = ".3pio"
	// Recreate logger after setting baseDir
	newLogger, _ := logger.NewFileLogger()
	p.logger = newLogger
	defer p.logger.Close()

	err = p.Run()
	if err != nil {
		t.Errorf("Prune failed: %v", err)
	}

	// Verify correct runs were deleted/kept
	for _, run := range testRuns {
		runPath := filepath.Join(runsDir, run.name)
		_, err := os.Stat(runPath)
		exists := err == nil
		if run.keepRun && !exists {
			t.Errorf("Run %s should be kept but was deleted", run.name)
		}
		if !run.keepRun && exists {
			t.Errorf("Run %s should be deleted but still exists", run.name)
		}
	}

	// Verify debug log was cleared (should exist but be empty or very small)
	if info, err := os.Stat(debugLogPath); err != nil {
		t.Errorf("Debug log should exist: %v", err)
	} else if info.Size() > 1000 {
		// Allow some size for the new log entry after clearing
		t.Errorf("Debug log should be cleared, but has size %d", info.Size())
	}
}
