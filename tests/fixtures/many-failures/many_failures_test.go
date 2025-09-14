package manyfailures

import "testing"

func TestPass1(t *testing.T) {
	// This test passes
}

func TestFail1(t *testing.T) {
	t.Fatal("First failure message")
}

func TestFail2(t *testing.T) {
	t.Fatal("Second failure message")
}

func TestPass2(t *testing.T) {
	// This test also passes
}

func TestFail3(t *testing.T) {
	t.Fatal("Third failure message")
}

func TestFail4(t *testing.T) {
	t.Fatal("Fourth failure message")
}

func TestFail5(t *testing.T) {
	t.Fatal("Fifth failure message")
}

func TestPass3(t *testing.T) {
	// Another passing test
}
