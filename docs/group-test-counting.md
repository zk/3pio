# Group Test Counting

## Problem
Reports were showing incorrect test counts at the package/group level. For example, a package with 1 top-level test that had 2 subtests would report "Total tests: 3" instead of correctly showing only the 1 top-level test.

## Solution
1. **Go Test Runner**: Modified to only track top-level tests in package groups, not subtests
2. **Report Terminology**: Changed from "Total tests" to "Group tests" to clarify that we're counting tests at the current group level

## Implementation Details

### Go Test Runner Changes
- Only add tests to `packageGroups[].Tests` if they have no parent hierarchy (`len(suiteChain) == 0`)
- This ensures subtests are not counted in the package-level totals
- Subtests are still tracked separately in their own subgroups

### Report Format Changes
Changed report summary from:
```
- Total tests: N
- Tests passed: M
```

To:
```
- Group tests: N
- Group tests passed: M
```

This makes it clear that the count represents tests at the current group level, not including tests in subgroups.

## Example
For a test file with:
```go
func TestMain(t *testing.T) {
    t.Run("Sub1", func(t *testing.T) { ... })
    t.Run("Sub2", func(t *testing.T) { ... })
}
```

The package report will show:
- Group tests: 1 (only TestMain)
- Subgroups: 1 (TestMain which contains the subtests)

The TestMain subgroup report will show:
- Group tests: 2 (Sub1 and Sub2)