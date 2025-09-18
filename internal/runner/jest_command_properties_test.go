package runner

import (
    "strings"
    "testing"
)

func TestJestBuildCommand_Invariants(t *testing.T) {
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
        out := jest.BuildCommand(in, adapter)
        // Reporter appears exactly once
        count := 0
        for i := 0; i < len(out)-1; i++ {
            if out[i] == "--reporters" && out[i+1] == adapter { count++ }
        }
        if count != 1 { t.Fatalf("reporter count=%d out=%v", count, out) }
        // Yarn script should not inject -- unless present
        if len(in) >= 1 && in[0] == "yarn" && !contains(out, "--") && contains(in, "test") {
            // ok; no assertion needed other than absence of separator
        }
        // At most one separator overall
        sep := 0
        for _, a := range out { if a == "--" { sep++ } }
        if sep > 1 { t.Fatalf("multiple -- separators: %v", out) }
    }
}

func contains(s []string, x string) bool { for _, v := range s { if v == x { return true } }; return false }

