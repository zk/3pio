# Path Sanitization

## Overview

3pio sanitizes file paths when creating report directories to ensure filesystem compatibility across different operating systems. This document describes how path sanitization works and recent changes made to ensure consistency.

## Sanitization Rules

The `report.SanitizeGroupName` function applies the following transformations:

1. **Convert to lowercase** - All paths are converted to lowercase for consistency
2. **Replace path separators** - Forward slashes (`/`) and backslashes (`\`) are replaced with underscores (`_`)
3. **Replace ALL dots** - All dots (`.`) including in file extensions are replaced with underscores (`_`)
4. **Replace dashes** - All dashes (`-`) are replaced with underscores (`_`)
5. **Handle special characters** - Invalid filesystem characters are replaced with underscores
6. **Windows reserved names** - Reserved names like CON, PRN, AUX are wrapped with underscores
7. **Collapse multiple underscores** - Multiple consecutive underscores are collapsed to a single underscore

## Examples

| Original Path | Sanitized Path |
|--------------|----------------|
| `./math.test.js` | `_math_test_js` |
| `./test/system/api.test.ts` | `_test_system_api_test_ts` |
| `test/unit/helper.spec.js` | `test_unit_helper_spec_js` |
| `./src/utils.test.integration.js` | `_src_utils_test_integration_js` |
| `./my-component.spec.tsx` | `_my_component_spec_tsx` |
| `./test-utils/mock-data/users.test.js` | `_test_utils_mock_data_users_test_js` |

## Architecture

### Centralized Sanitization

As of September 2025, path sanitization has been centralized to use a single function: `report.SanitizeGroupName`. This ensures that:

1. **Consistency** - The displayed "See..." path in console output matches the actual directory created
2. **Maintainability** - Only one sanitization function to maintain and test
3. **Predictability** - Users can rely on consistent path transformations
4. **No Duplication** - Eliminates the risk of different sanitization logic causing mismatches

### Components

- **Report Manager** (`internal/report/`) - Uses `SanitizeGroupName` when creating report directories
- **Orchestrator** (`internal/orchestrator/`) - Uses the same `SanitizeGroupName` when displaying report paths in console output

### Implementation Details

The console output generation in `orchestrator.go:701` uses:
```go
reportPath := fmt.Sprintf(".3pio/runs/%s/reports/%s/index.md", o.runID, report.SanitizeGroupName(group.Name))
```

The directory creation in `group_path.go` uses the same `SanitizeGroupName` function, ensuring perfect consistency between what users see and what actually exists on disk.

## Testing

Path sanitization consistency is verified through multiple test layers:

1. **Unit Tests**: `TestSanitizePathConsistency` in `internal/orchestrator/path_sanitization_test.go` verifies that path sanitization produces expected results
2. **Integration Tests**: `TestConsoleOutputMatchesActualDirectoryStructure` in `tests/integration_go/path_consistency_test.go` verifies that console output exactly matches actual directory structure by:
   - Running a failing test to trigger the "See..." message
   - Extracting the displayed path from console output
   - Verifying the actual directory exists with the same name
   - Ensuring perfect match between displayed and actual paths

## Migration Notes

Previously, the orchestrator had its own `sanitizePathForFilesystem` function which could produce different results than the report manager's sanitization. This has been removed in favor of using the centralized `report.SanitizeGroupName` function everywhere.