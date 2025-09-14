package main

import (
	"strings"
	"testing"
)

func TestStringOperations(t *testing.T) {
	t.Run("concatenation", func(t *testing.T) {
		result := strings.Join([]string{"hello", "world"}, " ")
		if result != "hello world" {
			t.Errorf("got %q, want %q", result, "hello world")
		}
	})

	t.Run("uppercase", func(t *testing.T) {
		result := strings.ToUpper("hello")
		if result != "HELLO" {
			t.Errorf("got %q, want %q", result, "HELLO")
		}
	})

	t.Run("contains", func(t *testing.T) {
		if !strings.Contains("hello world", "world") {
			t.Error("expected string to contain 'world'")
		}
	})
}

func TestParallelTests(t *testing.T) {
	t.Run("parallel test 1", func(t *testing.T) {
		t.Parallel()
		// Simulate some work
		result := len("test")
		if result != 4 {
			t.Errorf("got %d, want 4", result)
		}
	})

	t.Run("parallel test 2", func(t *testing.T) {
		t.Parallel()
		// Simulate some work
		result := strings.Count("hello", "l")
		if result != 2 {
			t.Errorf("got %d, want 2", result)
		}
	})

	t.Run("parallel test 3", func(t *testing.T) {
		t.Parallel()
		// Simulate some work
		result := strings.Index("world", "r")
		if result != 2 {
			t.Errorf("got %d, want 2", result)
		}
	})
}
