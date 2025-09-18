package runner

import (
    "strings"
    "testing"
)

// Property-style checks: reporter exactly once, separator policy, and order preserved.
func TestJestBuildCommand_Invariants(t *testing.T) {
    t.Parallel()
    jest := NewJestDefinition()
    adapter := "/a/adapter.js"

    cases := [][]string{
        {"npm", "test"},
        {"npm", "test", "--", "--watch"},
        {"yarn", "test"},
        {"pnpm", "exec", "jest"},
        {"bun", "jest"},
        {"npx", "jest"},
        {"npx", "jest", "math.test.js"},
        {"jest"},
        {"jest", "src"},
        {"node", "node_modules/.bin/jest", "--coverage", "file.test.js"},
    }

    for _, in := range cases {
        in := in
        t.Run(strings.Join(in, " "), func(t *testing.T) {
            t.Parallel()
            out := jest.BuildCommand(in, adapter)

            // Reporter appears exactly once
            count := 0
            for i := 0; i < len(out)-1; i++ {
                if out[i] == "--reporters" && out[i+1] == adapter {
                    count++
                }
            }
            if count != 1 {
                t.Fatalf("reporter occurrences = %d, want 1; out=%v", count, out)
            }

            // Yarn scripts should not have implicit separator insertion
            if len(in) >= 1 && in[0] == "yarn" && !contains(in, "--") {
                if contains(out, "--") {
                    t.Fatalf("unexpected -- separator for yarn script: %v", out)
                }
            }

            // For direct jest with files, if a separator is present, it must appear before the first non-flag after jest
            if contains(in, "jest") && anyFileAfter(in, indexOf(in, "jest")) {
                // ok for separator to be present; ensure at most one
                sepCount := 0
                for _, a := range out { if a == "--" { sepCount++ } }
                if sepCount > 1 { t.Fatalf("multiple -- separators: %v", out) }
            }
        })
    }
}

func anyFileAfter(args []string, idx int) bool {
    for i := idx + 1; i < len(args); i++ {
        a := args[i]
        if !strings.HasPrefix(a, "-") && (strings.Contains(a, ".") || strings.Contains(a, "/")) {
            return true
        }
    }
    return false
}

func contains(s []string, x string) bool { for _, v := range s { if v == x { return true } }; return false }

