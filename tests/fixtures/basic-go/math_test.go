package main

import (
	"testing"
)

func TestAdd(t *testing.T) {
	result := add(2, 3)
	if result != 5 {
		t.Errorf("add(2, 3) = %d; want 5", result)
	}
}

func TestSubtract(t *testing.T) {
	result := subtract(5, 3)
	if result != 2 {
		t.Errorf("subtract(5, 3) = %d; want 2", result)
	}
}

func TestMultiply(t *testing.T) {
	result := multiply(3, 4)
	if result != 12 {
		t.Errorf("multiply(3, 4) = %d; want 12", result)
	}
}

func TestDivide(t *testing.T) {
	t.Run("normal division", func(t *testing.T) {
		result := divide(10, 2)
		if result != 5 {
			t.Errorf("divide(10, 2) = %d; want 5", result)
		}
	})

	t.Run("division by zero", func(t *testing.T) {
		result := divide(10, 0)
		if result != 0 {
			t.Errorf("divide(10, 0) = %d; want 0", result)
		}
	})
}

func TestFailingCase(t *testing.T) {
	// This test intentionally fails
	t.Error("This test is supposed to fail")
}

func TestSkippedCase(t *testing.T) {
	t.Skip("Skipping this test for demonstration")
}

// Math functions for testing
func add(a, b int) int {
	return a + b
}

func subtract(a, b int) int {
	return a - b
}

func multiply(a, b int) int {
	return a * b
}

func divide(a, b int) int {
	if b == 0 {
		return 0
	}
	return a / b
}