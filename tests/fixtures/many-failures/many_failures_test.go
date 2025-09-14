package manyfailures

import "testing"

func TestPass1(t *testing.T) {
	// This test passes
}

func TestFail1(t *testing.T) {
	t.Fatal("FORCE FAILURE 1 - this should never pass")
}

func TestFail2(t *testing.T) {
	t.Fatal("FORCE FAILURE 2 - this should never pass")
}

func TestPass2(t *testing.T) {
	// This test also passes
	if false {
		t.Fatal("This should not run")
	}
}

func TestFail3(t *testing.T) {
	t.Fatal("FORCE FAILURE 3 - this should never pass")
}

func TestFail4(t *testing.T) {
	t.Fatal("FORCE FAILURE 4 - this should never pass")
}

func TestFail5(t *testing.T) {
	t.Fatal("FORCE FAILURE 5 - this should never pass")
}

func TestPass3(t *testing.T) {
	// Another passing test
}
