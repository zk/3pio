package manyfailures

import (
	"testing"
	"time"
)

func TestPass1(t *testing.T) {
	// This test passes
}

func TestFail1(t *testing.T) {
	// Always fail with current timestamp to prevent caching
	t.Fatalf("FORCE FAILURE 1 at %v - this should never pass", time.Now().UnixNano())
}

func TestFail2(t *testing.T) {
	t.Fatalf("FORCE FAILURE 2 at %v - this should never pass", time.Now().UnixNano())
}

func TestPass2(t *testing.T) {
	// This test also passes
	if false {
		t.Fatal("This should not run")
	}
}

func TestFail3(t *testing.T) {
	t.Fatalf("FORCE FAILURE 3 at %v - this should never pass", time.Now().UnixNano())
}

func TestFail4(t *testing.T) {
	t.Fatalf("FORCE FAILURE 4 at %v - this should never pass", time.Now().UnixNano())
}

func TestFail5(t *testing.T) {
	t.Fatalf("FORCE FAILURE 5 at %v - this should never pass", time.Now().UnixNano())
}

func TestPass3(t *testing.T) {
	// Another passing test
}
