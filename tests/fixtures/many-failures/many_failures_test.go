package manyfailures

import "testing"

func TestPass1(t *testing.T) {
	// This test passes
}

func TestFail1(t *testing.T) {
	// Force output to ensure test is actually running
	t.Log("TestFail1 is executing in CI")
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
