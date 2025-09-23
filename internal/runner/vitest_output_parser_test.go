package runner

import (
	"strings"
	"testing"
)

func TestVitestOutputParser_V3Style(t *testing.T) {
	parser := NewVitestOutputParser()

	sample := "" +
		"✓ tests/math.test.ts  (2)\n" +
		"  ✓ adds numbers\n" +
		"  ✓ multiplies numbers\n" +
		"\n" +
		"✗ tests/string.test.ts  (2)\n" +
		"  ✓ concatenates strings\n" +
		"  ✗ should fail intentionally\n"

	got := parser.ParseTestOutput(sample)

	// Expect both files to be present as keys
	if _, ok := got["tests/math.test.ts"]; !ok {
		t.Fatalf("expected key for tests/math.test.ts, got: %#v", got)
	}
	if _, ok := got["tests/string.test.ts"]; !ok {
		t.Fatalf("expected key for tests/string.test.ts, got: %#v", got)
	}
}

func TestVitestOutputParser_V2Style(t *testing.T) {
	parser := NewVitestOutputParser()

	// Simulate a slightly different spacing/marker but containing checkmark/X and file tokens
	sample := "" +
		"✓ src/math.test.js (3) 3 passed\n" +
		"  ✓ division\n" +
		"  ✓ addition\n" +
		"  ✓ multiplication\n" +
		"\n" +
		"✗ src/string.test.js (2) 1 failed\n" +
		"  ✓ concatenation\n" +
		"  ✗ toUpperCase\n"

	got := parser.ParseTestOutput(sample)

	wantKeys := []string{"src/math.test.js", "src/string.test.js"}
	for _, k := range wantKeys {
		if _, ok := got[k]; !ok {
			t.Fatalf("expected key for %s, got: %#v", k, got)
		}
	}

	// Ensure lines were captured per file
	if len(got["src/math.test.js"]) == 0 || len(got["src/string.test.js"]) == 0 {
		t.Fatalf("expected non-empty output slices per file, got: %#v", got)
	}

	// Basic sanity: first line for string.test.js contains the file marker
	if got := got["src/string.test.js"][0]; !containsAny(got, []string{"✗", "✓"}) {
		t.Fatalf("expected marker in first grouped line, got: %q", got)
	}
}

func containsAny(s string, tokens []string) bool {
	for _, tok := range tokens {
		if strings.Contains(s, tok) {
			return true
		}
	}
	return false
}
