package logger

import (
	"fmt"
	"sync"
)

// TestLogger is a logger for testing that stores messages in memory
type TestLogger struct {
	mu            sync.Mutex
	debugMessages []string
	infoMessages  []string
	errorMessages []string
}

// NewTestLogger creates a new test logger
func NewTestLogger() *TestLogger {
	return &TestLogger{
		debugMessages: []string{},
		infoMessages:  []string{},
		errorMessages: []string{},
	}
}

// Debug writes a debug message to memory
func (l *TestLogger) Debug(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.debugMessages = append(l.debugMessages, fmt.Sprintf(format, args...))
}

// Info writes an info message to memory
func (l *TestLogger) Info(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.infoMessages = append(l.infoMessages, fmt.Sprintf(format, args...))
}

// Error writes an error message to memory
func (l *TestLogger) Error(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.errorMessages = append(l.errorMessages, fmt.Sprintf(format, args...))
}

// Close does nothing for test logger
func (l *TestLogger) Close() error {
	return nil
}

// GetDebugMessages returns all debug messages
func (l *TestLogger) GetDebugMessages() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	result := make([]string, len(l.debugMessages))
	copy(result, l.debugMessages)
	return result
}

// GetInfoMessages returns all info messages
func (l *TestLogger) GetInfoMessages() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	result := make([]string, len(l.infoMessages))
	copy(result, l.infoMessages)
	return result
}

// GetErrorMessages returns all error messages
func (l *TestLogger) GetErrorMessages() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	result := make([]string, len(l.errorMessages))
	copy(result, l.errorMessages)
	return result
}