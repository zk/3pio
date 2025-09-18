package report

import (
    "strings"
    "testing"
    "unicode/utf8"
)

func FuzzSanitizeGroupName(f *testing.F) {
    seeds := []string{"", "a", "../etc/passwd", string([]byte{0, 1, 2}), "コンポーネント.test.tsx", "   spaced   name  ", "...", "🐍/🐹/🐙"}
    for _, s := range seeds { f.Add(s) }
    f.Fuzz(func(t *testing.T, in string) {
        got := SanitizeGroupName(in)
        if got == "" { t.Fatalf("empty output not allowed") }
        if len(got) > MaxComponentLength { t.Fatalf("too long: %d > %d", len(got), MaxComponentLength) }
        if !utf8.ValidString(got) { t.Fatalf("not valid UTF-8") }
        if strings.ContainsAny(got, "/\\") { t.Fatalf("contains path separators: %q", got) }
    })
}

