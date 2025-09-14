package manyfailures

import (
	"os"
	"testing"
)

func TestPass1(t *testing.T) {
	// This test passes
}

func TestFail1(t *testing.T) {
	// Create a marker file to verify this version is running
	_ = os.WriteFile("CI_TEST_MARKER_v3.txt", []byte("TestFail1 executed"), 0644)
	t.Errorf("First failure message - this test must fail")
	t.Fatal("FORCE FAILURE - this should never pass")
}

func TestFail2(t *testing.T) {
	t.Errorf("Second failure message - this test must fail")
	t.Fatal("FORCE FAILURE - this should never pass")
}

func TestPass2(t *testing.T) {
	// This test also passes
	if false {
		t.Fatal("This should not run")
	}
}

func TestFail3(t *testing.T) {
	t.Errorf("Third failure message - this test must fail")
	t.Fatal("FORCE FAILURE - this should never pass")
}

func TestFail4(t *testing.T) {
	t.Errorf("Fourth failure message - this test must fail")
	t.Fatal("FORCE FAILURE - this should never pass")
}

func TestFail5(t *testing.T) {
	t.Errorf("Fifth failure message - this test must fail")
	t.Fatal("FORCE FAILURE - this should never pass")
}

func TestPass3(t *testing.T) {
	// Another passing test
}
