package pkg_zebra

import "testing"

// This package comes last alphabetically and has failures
func TestZebraPass(t *testing.T) {
	// This test passes
}

func TestZebraFail1(t *testing.T) {
	t.Fatal("First failure in zebra package")
}

func TestZebraFail2(t *testing.T) {
	t.Fatal("Second failure in zebra package")
}

func TestZebraFail3(t *testing.T) {
	t.Fatal("Third failure in zebra package")
}
