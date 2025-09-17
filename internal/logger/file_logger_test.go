package logger

import (
	"testing"
)

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected LogLevel
	}{
		{"DEBUG uppercase", "DEBUG", DEBUG},
		{"debug lowercase", "debug", DEBUG},
		{"DEBUG with spaces", "  DEBUG  ", DEBUG},
		{"INFO uppercase", "INFO", INFO},
		{"info lowercase", "info", INFO},
		{"INFO with spaces", "  INFO  ", INFO},
		{"WARN uppercase", "WARN", WARN},
		{"warn lowercase", "warn", WARN},
		{"WARN with spaces", "  WARN  ", WARN},
		{"ERROR uppercase", "ERROR", ERROR},
		{"error lowercase", "error", ERROR},
		{"ERROR with spaces", "  ERROR  ", ERROR},
		{"empty string", "", WARN},
		{"invalid level", "invalid", WARN},
		{"random string", "foobar", WARN},
		{"mixed case", "DeBuG", DEBUG},
		{"mixed case info", "InFo", INFO},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseLogLevel(tt.input)
			if result != tt.expected {
				t.Errorf("parseLogLevel(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLogLevel_String(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARN, "WARN"},
		{ERROR, "ERROR"},
		{LogLevel(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.level.String()
			if result != tt.expected {
				t.Errorf("LogLevel(%d).String() = %q, expected %q", int(tt.level), result, tt.expected)
			}
		})
	}
}
