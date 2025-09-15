# Test Organization in 3pio

## Overview

This document covers how 3pio organizes, counts, and handles hierarchical test structures across different test runners. It includes path sanitization for filesystem compatibility, test counting methodologies, and handling of complex nested test hierarchies.

## Path Sanitization

### Overview

3pio sanitizes file paths when creating report **directories** to ensure filesystem compatibility across different operating systems. The sanitized names are used for directory names only - all report files within these directories are named `index.md` (except for the top-level `test-run.md`).

### Sanitization Rules

The `report.SanitizeGroupName` function applies the following transformations:

1. **Convert to lowercase** - All paths are converted to lowercase for consistency
2. **Replace path separators** - Forward slashes (`/`) and backslashes (`\`) are replaced with underscores (`_`)
3. **Replace ALL dots** - All dots (`.`) including in file extensions are replaced with underscores (`_`)
4. **Replace dashes** - All dashes (`-`) are replaced with underscores (`_`)
5. **Handle special characters** - Invalid filesystem characters are replaced with underscores
6. **Windows reserved names** - Reserved names like CON, PRN, AUX are wrapped with underscores
7. **Collapse multiple underscores** - Multiple consecutive underscores are collapsed to a single underscore

### Examples

| Original Path | Sanitized Path |
|--------------|----------------|
| `./math.test.js` | `_math_test_js` |
| `./test/system/api.test.ts` | `_test_system_api_test_ts` |
| `test/unit/helper.spec.js` | `test_unit_helper_spec_js` |
| `./src/utils.test.integration.js` | `_src_utils_test_integration_js` |
| `./my-component.spec.tsx` | `_my_component_spec_tsx` |
| `./test-utils/mock-data/users.test.js` | `_test_utils_mock_data_users_test_js` |

### Report File Structure

**Important**: The sanitized names shown above are used for **directory names only**. The actual report structure is:

```
.3pio/runs/[runID]/
├── test-run.md                                    # Top-level report (only non-index.md file)
└── reports/
    ├── _math_test_js/
    │   └── index.md                               # Report for math.test.js
    ├── _test_system_api_test_ts/
    │   └── index.md                               # Report for test/system/api.test.ts
    └── _my_component_spec_tsx/
        └── index.md                               # Report for my-component.spec.tsx
```

All report files are named `index.md` within their respective directories. This provides:
- Consistent file naming
- Clean URLs when served via web server
- No conflicts between directory and file names
- Easy navigation structure

### Architecture

#### Centralized Sanitization

As of September 2025, path sanitization has been centralized to use a single function: `report.SanitizeGroupName`. This ensures that:

1. **Consistency** - The displayed "See..." path in console output matches the actual directory created
2. **Maintainability** - Only one sanitization function to maintain and test
3. **Predictability** - Users can rely on consistent path transformations
4. **No Duplication** - Eliminates the risk of different sanitization logic causing mismatches

#### Components

- **Report Manager** (`internal/report/`) - Uses `SanitizeGroupName` when creating report directories
- **Orchestrator** (`internal/orchestrator/`) - Uses the same `SanitizeGroupName` when displaying report paths in console output

#### Implementation Details

The console output generation in `orchestrator.go:701` uses:
```go
reportPath := fmt.Sprintf(".3pio/runs/%s/reports/%s/index.md", o.runID, report.SanitizeGroupName(group.Name))
```

The directory creation in `group_path.go` uses the same `SanitizeGroupName` function, ensuring perfect consistency between what users see and what actually exists on disk.

### Testing

Path sanitization consistency is verified through multiple test layers:

1. **Unit Tests**: `TestSanitizePathConsistency` in `internal/orchestrator/path_sanitization_test.go` verifies that path sanitization produces expected results
2. **Integration Tests**: `TestConsoleOutputMatchesActualDirectoryStructure` in `tests/integration_go/path_consistency_test.go` verifies that console output exactly matches actual directory structure by:
   - Running a failing test to trigger the "See..." message
   - Extracting the displayed path from console output
   - Verifying the actual directory exists with the same name
   - Ensuring perfect match between displayed and actual paths

### Migration Notes

Previously, the orchestrator had its own `sanitizePathForFilesystem` function which could produce different results than the report manager's sanitization. This has been removed in favor of using the centralized `report.SanitizeGroupName` function everywhere.

## Test Counting

### Problem

Reports were showing incorrect test counts at the package/group level. For example, a package with 1 top-level test that had 2 subtests would report "Total tests: 3" instead of correctly showing only the 1 top-level test.

### Solution

1. **Go Test Runner**: Modified to only track top-level tests in package groups, not subtests
2. **Report Terminology**: Changed from "Total tests" to "Group tests" to clarify that we're counting tests at the current group level

### Implementation Details

#### Go Test Runner Changes
- Only add tests to `packageGroups[].Tests` if they have no parent hierarchy (`len(suiteChain) == 0`)
- This ensures subtests are not counted in the package-level totals
- Subtests are still tracked separately in their own subgroups

#### Report Format Changes
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

### Example

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

## Subgroup Handling

### Go Test Subgroup Duration Handling

#### Problem
Deeply nested test hierarchies in Go (using `t.Run()`) were not properly reporting durations for intermediate subgroups. Only the top-level test and package-level groups were sending `testGroupResult` events, leaving intermediate groups with 0 duration in reports.

#### Example Case
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

#### Solution
Modified the Go test runner definition (`internal/runner/definitions/gotest.go`) to send `testGroupResult` events for ALL intermediate subgroups when they complete, not just top-level tests.

##### Key Changes
1. When any test completes (pass/fail/skip), check if it's a parent group of other tests
2. If it is a parent group (has subtests), send a `testGroupResult` event for it
3. This applies to both top-level tests AND intermediate subgroups

##### Implementation Details
- Track subgroup statistics in `subgroupStats` map
- Key format: `package/Test/SubA/SubB/...`
- When a test completes, check if its key exists in the stats map
- If it exists, it means this test has subtests, so send a group result
- Works for arbitrarily deep nesting and empty group names

#### Testing
Added `TestGoTestDefinition_DeeplyNestedSubgroups` test that verifies:
- All intermediate subgroups get result events
- Empty-named groups are handled correctly
- Durations are properly set for all groups

#### Impact
This ensures that all test groups in Go test output have proper duration reporting, improving the accuracy of test reports for complex test suites with deeply nested structures.

## Universal Group Abstractions

### Overview

3pio uses a universal group abstraction model that works across all test runners. This hierarchical model supports:

- **Files** → Top-level test files or modules
- **Describes/Suites** → Test grouping constructs (describe blocks, test classes)
- **Nested Groups** → Arbitrary nesting depth
- **Individual Tests** → Actual test cases

### Cross-Runner Mapping

| Test Runner | File Level | Group Level | Test Level |
|-------------|------------|-------------|------------|
| **Jest/Vitest** | `math.test.js` | `describe("Math operations")` | `it("should add numbers")` |
| **Go** | Package (`github.com/user/pkg`) | `TestMath` | `t.Run("addition")` |
| **pytest** | `test_math.py` | `class TestMathOperations` | `def test_addition` |
| **Rust** | Crate (`my_crate`) | `mod tests` | `fn test_addition` |

### Deterministic Group IDs

Each group gets a deterministic SHA256-based ID generated from its full hierarchy path:
- Enables consistent cross-run references
- Filesystem-safe path generation
- Collision-resistant for complex hierarchies

### Group Status Propagation

- **Failed**: If any child test fails, parent groups inherit failed status
- **Passed**: If all child tests pass, parent groups get passed status
- **Skipped**: If all child tests are skipped, parent groups get skipped status
- **Mixed**: For mixed results, overall status is based on presence of failures

## Best Practices

### For Test Organization
1. **Use consistent naming patterns** across test files
2. **Group related tests** using framework-specific constructs (describe, test classes, modules)
3. **Avoid deeply nested hierarchies** when possible for cleaner reports
4. **Use descriptive test and group names** for better report readability

### For Path Compatibility
1. **Avoid special characters** in test file names when possible
2. **Use standard file extensions** (.test.js, .spec.ts, _test.py, etc.)
3. **Be aware of filesystem limitations** on different operating systems
4. **Test with long paths** to ensure sanitization works correctly

### For Performance
1. **Limit extremely deep nesting** (>5 levels) for better performance
2. **Use meaningful group names** to reduce processing overhead
3. **Consider test execution time** when organizing hierarchies
4. **Monitor report generation times** for large test suites

## Troubleshooting

### Common Issues

1. **Mismatched console paths and actual directories**
   - Verify centralized sanitization is being used
   - Check that both orchestrator and report manager use `SanitizeGroupName`

2. **Incorrect test counts in reports**
   - Ensure subtests are not being double-counted at parent levels
   - Verify "Group tests" terminology is used instead of "Total tests"

3. **Missing duration data for subgroups**
   - Check that `testGroupResult` events are sent for all group levels
   - Verify intermediate subgroups are tracked in statistics maps

4. **Path sanitization inconsistencies**
   - Run path consistency integration tests
   - Verify single sanitization function is used throughout codebase

### Debugging Tips

1. **Check debug logs** for path sanitization details
2. **Verify IPC events** include complete group hierarchies
3. **Test with edge cases** like empty group names and special characters
4. **Use integration tests** to verify end-to-end behavior