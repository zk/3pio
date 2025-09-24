package prune

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zk/3pio/internal/logger"
)

func mkdirAll(t *testing.T, path string, perm os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(path, perm); err != nil {
		t.Fatalf("Failed to create directory %s: %v", path, err)
	}
}

func writeFile(t *testing.T, path string, data []byte, perm os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, data, perm); err != nil {
		t.Fatalf("Failed to write file %s: %v", path, err)
	}
}

func chdirTo(t *testing.T, dir string) {
	t.Helper()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Failed to change directory to %s: %v", dir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Fatalf("Failed to restore working directory: %v", err)
		}
	})
}

func closeLogger(t *testing.T, l *logger.FileLogger) {
	t.Helper()
	if l == nil {
		return
	}
	if err := l.Close(); err != nil {
		t.Fatalf("Failed to close logger: %v", err)
	}
}

func TestParseTimestamp(t *testing.T) {
	// Create a pruner with a mock logger
	tmpDir := t.TempDir()
	mkdirAll(t, filepath.Join(tmpDir, ".3pio"), 0755)
	chdirTo(t, tmpDir)

	fileLogger, err := logger.NewFileLogger()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer closeLogger(t, fileLogger)

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
	mkdirAll(t, runsDir, 0755)

	// Change to temp directory
	chdirTo(t, tmpDir)

	// Create test runs with different timestamps
	testRuns := []string{
		"20240101_120000_oldest-run",
		"20240102_120000_middle-run",
		"20240103_120000_newest-run",
	}

	for _, run := range testRuns {
		runPath := filepath.Join(runsDir, run)
		mkdirAll(t, runPath, 0755)
		// Create a test file in each run
		testFile := filepath.Join(runPath, "test-run.md")
		writeFile(t, testFile, []byte("test content"), 0644)
	}

	// Create a non-directory file that should be ignored
	writeFile(t, filepath.Join(runsDir, "not-a-directory.txt"), []byte("ignore me"), 0644)

	// Create logger
	fileLogger, err := logger.NewFileLogger()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer closeLogger(t, fileLogger)

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
	chdirTo(t, tmpDir)

	// Create logger in a temporary .3pio dir
	mkdirAll(t, ".3pio", 0755)
	fileLogger, err := logger.NewFileLogger()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer closeLogger(t, fileLogger)

	// Remove .3pio to test missing directory
	closeLogger(t, fileLogger)
	if err := os.RemoveAll(".3pio"); err != nil {
		t.Fatalf("Failed to remove .3pio directory: %v", err)
	}

	p := New(fileLogger, false, true)
	err = p.Run()
	if err != nil {
		t.Errorf("Expected no error for missing .3pio directory, got: %v", err)
	}
	closeLogger(t, p.logger)
}

func TestPruneNoRuns(t *testing.T) {
	// Create temp directory with empty .3pio
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".3pio")
	mkdirAll(t, baseDir, 0755)

	chdirTo(t, tmpDir)

	fileLogger, err := logger.NewFileLogger()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer closeLogger(t, fileLogger)

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
	mkdirAll(t, runPath, 0755)
	writeFile(t, filepath.Join(runPath, "test-run.md"), []byte("test"), 0644)

	chdirTo(t, tmpDir)

	fileLogger, err := logger.NewFileLogger()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer closeLogger(t, fileLogger)

	p := New(fileLogger, false, true)
	err = p.Run()
	if err != nil {
		t.Errorf("Expected no error for single run, got: %v", err)
	}
	closeLogger(t, p.logger)

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
	mkdirAll(t, runsDir, 0755)

	chdirTo(t, tmpDir)

	// Create multiple test runs
	testRuns := []string{
		"20240101_120000_oldest-run",
		"20240102_120000_middle-run",
		"20240103_120000_newest-run",
	}

	for _, run := range testRuns {
		runPath := filepath.Join(runsDir, run)
		mkdirAll(t, runPath, 0755)
		writeFile(t, filepath.Join(runPath, "test-run.md"), []byte("test"), 0644)
	}

	fileLogger, err := logger.NewFileLogger()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer closeLogger(t, fileLogger)

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
	mkdirAll(t, runsDir, 0755)

	chdirTo(t, tmpDir)

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
		mkdirAll(t, runPath, 0755)
		writeFile(t, filepath.Join(runPath, "test-run.md"), []byte("test content"), 0644)
	}

	// Create debug log
	debugLogPath := filepath.Join(baseDir, "debug.log")
	writeFile(t, debugLogPath, []byte("debug log content to clear"), 0644)

	fileLogger, err := logger.NewFileLogger()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	// Close logger before pruning to avoid file lock issues
	closeLogger(t, fileLogger)

	// Run prune with force flag
	p := New(nil, false, true) // Use nil logger to avoid conflicts
	p.baseDir = ".3pio"
	// Recreate logger after setting baseDir
	newLogger, err := logger.NewFileLogger()
	if err != nil {
		t.Fatalf("Failed to recreate logger: %v", err)
	}
	p.logger = newLogger

	err = p.Run()
	if err != nil {
		t.Errorf("Prune failed: %v", err)
	}
	closeLogger(t, p.logger)

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
