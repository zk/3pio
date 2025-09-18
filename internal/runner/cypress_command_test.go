package runner

import (
    "reflect"
    "testing"
)

func TestCypressBuildCommand(t *testing.T) {
    c := NewCypressDefinition()
    adapter := "/tmp/adapter.js"

    tests := []struct{
        name string
        in []string
        expected []string
    }{
        {
            name: "direct cypress run",
            in: []string{"cypress", "run"},
            expected: []string{"cypress", "run", "--reporter", adapter},
        },
        {
            name: "npx cypress run",
            in: []string{"npx", "cypress", "run"},
            expected: []string{"npx", "cypress", "run", "--reporter", adapter},
        },
        {
            name: "pnpm exec cypress run",
            in: []string{"pnpm", "exec", "cypress", "run"},
            expected: []string{"pnpm", "exec", "cypress", "run", "--reporter", adapter},
        },
        {
            name: "npm test script (needs --)",
            in: []string{"npm", "test"},
            expected: []string{"npm", "test", "--", "--reporter", adapter},
        },
        {
            name: "yarn script (needs --)",
            in: []string{"yarn", "test"},
            expected: []string{"yarn", "test", "--", "--reporter", adapter},
        },
        {
            name: "direct cypress (ensure run added)",
            in: []string{"cypress"},
            expected: []string{"cypress", "run", "--reporter", adapter},
        },
        {
            name: "cypress run with spec",
            in: []string{"cypress", "run", "--spec", "cypress/e2e/sample.cy.ts"},
            expected: []string{"cypress", "run", "--spec", "cypress/e2e/sample.cy.ts", "--reporter", adapter},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := c.BuildCommand(tt.in, adapter)
            if !reflect.DeepEqual(got, tt.expected) {
                t.Fatalf("BuildCommand mismatch\n in:  %#v\n got: %#v\n want: %#v", tt.in, got, tt.expected)
            }
        })
    }
}

