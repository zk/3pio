package manyfailures

import "testing"

func TestPass1(t *testing.T) {
	// This test passes
}

func TestFail1(t *testing.T) {
	t.Error("First failure message")
	t.FailNow()
}

func TestFail2(t *testing.T) {
	t.Error("Second failure message")
	t.FailNow()
}

func TestPass2(t *testing.T) {
	// This test also passes
	if false {
		t.Fatal("This should not run")
	}
}

func TestFail3(t *testing.T) {
	t.Error("Third failure message")
	t.FailNow()
}

func TestFail4(t *testing.T) {
	t.Error("Fourth failure message")
	t.FailNow()
}

func TestFail5(t *testing.T) {
	t.Error("Fifth failure message")
	t.FailNow()
}

func TestPass3(t *testing.T) {
	// Another passing test
}
