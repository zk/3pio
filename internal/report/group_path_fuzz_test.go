package report

import (
    "strings"
    "testing"
    "unicode/utf8"
)

// FuzzSanitizeGroupName checks that SanitizeGroupName is stable and safe for arbitrary input.
func FuzzSanitizeGroupName(f *testing.F) {
    seeds := []string{"", "a", "../etc/passwd", string([]byte{0, 1, 2}), "コンポーネント.test.tsx", "   spaced   name  ", "...", "🐍/🐹/🐙"}
    for _, s := range seeds { f.Add(s) }
    f.Fuzz(func(t *testing.T, in string) {
        got := SanitizeGroupName(in)
        if got == "" {
            t.Fatalf("empty output not allowed")
        }
        if len(got) > MaxComponentLength {
            t.Fatalf("output too long: %d > %d", len(got), MaxComponentLength)
        }
        // Ensure result is valid UTF-8
        if !utf8.ValidString(got) {
            t.Fatalf("output not valid UTF-8")
        }
        // No path separators expected in a single component
        if strings.ContainsAny(got, "/\\") {
            t.Fatalf("output contains path separators: %q", got)
        }
    })
}
