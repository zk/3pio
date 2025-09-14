# Go Test Subgroup Duration Handling

## Problem
Deeply nested test hierarchies in Go (using `t.Run()`) were not properly reporting durations for intermediate subgroups. Only the top-level test and package-level groups were sending `testGroupResult` events, leaving intermediate groups with 0 duration in reports.

## Example Case
```go
func TestParsed(t *testing.T) {
    t.Run("SubA", func(t *testing.T) {          // Intermediate subgroup
        t.Run("", func(t *testing.T) {          // Empty-named subgroup
            t.Run("DeepTest", func(t *testing.T) {
                // Actual test
            })
        })
    })
}
```

In this hierarchy:
- `TestParsed` would get a group result ✓
- `SubA` would NOT get a group result ✗
- Empty-named subgroups would NOT get group results ✗
- Package-level group would get a result ✓

## Solution
Modified the Go test runner definition (`internal/runner/definitions/gotest.go`) to send `testGroupResult` events for ALL intermediate subgroups when they complete, not just top-level tests.

### Key Changes
1. When any test completes (pass/fail/skip), check if it's a parent group of other tests
2. If it is a parent group (has subtests), send a `testGroupResult` event for it
3. This applies to both top-level tests AND intermediate subgroups

### Implementation Details
- Track subgroup statistics in `subgroupStats` map
- Key format: `package/Test/SubA/SubB/...`
- When a test completes, check if its key exists in the stats map
- If it exists, it means this test has subtests, so send a group result
- Works for arbitrarily deep nesting and empty group names

## Testing
Added `TestGoTestDefinition_DeeplyNestedSubgroups` test that verifies:
- All intermediate subgroups get result events
- Empty-named groups are handled correctly
- Durations are properly set for all groups

## Impact
This ensures that all test groups in Go test output have proper duration reporting, improving the accuracy of test reports for complex test suites with deeply nested structures.