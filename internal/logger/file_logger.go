package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

// String returns the string representation of a log level
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// parseLogLevel converts a string to LogLevel, case-insensitive
func parseLogLevel(level string) LogLevel {
	switch strings.ToUpper(strings.TrimSpace(level)) {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN":
		return WARN
	case "ERROR":
		return ERROR
	default:
		return WARN // Default to WARN for invalid values
	}
}

// FileLogger writes all log messages to .3pio/debug.log
type FileLogger struct {
	mu       sync.Mutex
	file     *os.File
	minLevel LogLevel
}

// NewFileLogger creates a new file-based logger
func NewFileLogger() (*FileLogger, error) {
	// Ensure .3pio directory exists
	if err := os.MkdirAll(".3pio", 0755); err != nil {
		return nil, fmt.Errorf("failed to create .3pio directory: %w", err)
	}

	// Open debug log file in append mode
	logPath := filepath.Join(".3pio", "debug.log")
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open debug log: %w", err)
	}

	// Write session header
	header := fmt.Sprintf("\n=== 3pio Debug Log ===\n"+
		"Session started: %s\n"+
		"PID: %d\n"+
		"Working directory: %s\n"+
		"---\n\n",
		time.Now().Format(time.RFC3339),
		os.Getpid(),
		mustGetwd())

	if _, err := file.WriteString(header); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("failed to write log header: %w", err)
	}

	// Read log level from environment variable, default to WARN
	logLevel := parseLogLevel(os.Getenv("THREEPIO_LOG_LEVEL"))

	return &FileLogger{
		file:     file,
		minLevel: logLevel,
	}, nil
}

// Debug writes a debug message to the log file
func (l *FileLogger) Debug(format string, args ...interface{}) {
	if l.minLevel <= DEBUG {
		l.writeLog("DEBUG", format, args...)
	}
}

// Error writes an error message to the log file and also to stderr
func (l *FileLogger) Error(format string, args ...interface{}) {
	l.writeLog("ERROR", format, args...)
	// Also write errors to stderr so they're visible to the user
	fmt.Fprintf(os.Stderr, "[ERROR] "+format+"\n", args...)
}

// Info writes an info message to the log file
func (l *FileLogger) Info(format string, args ...interface{}) {
	if l.minLevel <= INFO {
		l.writeLog("INFO", format, args...)
	}
}

// Warn writes a warning message to the log file
func (l *FileLogger) Warn(format string, args ...interface{}) {
	if l.minLevel <= WARN {
		l.writeLog("WARN", format, args...)
	}
}

// writeLog writes a timestamped log entry
func (l *FileLogger) writeLog(level, format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file == nil {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	message := fmt.Sprintf(format, args...)
	logLine := fmt.Sprintf("[%s] [%s] %s\n", timestamp, level, message)

	_, _ = l.file.WriteString(logLine)
	_ = l.file.Sync() // Ensure it's written to disk immediately
}

// Close closes the log file
func (l *FileLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		// Write session footer
		footer := fmt.Sprintf("\n--- Session ended: %s ---\n\n",
			time.Now().Format(time.RFC3339))
		_, _ = l.file.WriteString(footer)

		err := l.file.Close()
		l.file = nil
		return err
	}
	return nil
}

// mustGetwd returns the current working directory or "unknown"
func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return "unknown"
	}
	return wd
}
