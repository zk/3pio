# Implementation Plan: Cargo Metadata Integration for Proper Crate Grouping

## Executive Summary
Integrate `cargo metadata` parsing into 3pio's cargo test adapter to correctly identify which crate each test belongs to, solving the issue where tests with the same filename from different crates are incorrectly merged.

## Current Problem
- Integration tests `test_client.rs` exist in both `actix-http` and `awc` crates
- Current regex extracts only "test_client" from both, losing crate context
- Tests are incorrectly grouped together, causing inaccurate counts

## Solution Overview
Parse `cargo metadata --format-version 1` output before test execution to build a comprehensive mapping of test names to their owning crates.

## Implementation Details

### Phase 1: Data Structure Updates

Add to `cargo.go`:

```go
// WorkspaceMetadata represents parsed cargo metadata
type WorkspaceMetadata struct {
    WorkspaceRoot    string                 `json:"workspace_root"`
    WorkspaceMembers []string               `json:"workspace_members"`
    Packages        []PackageMetadata       `json:"packages"`
}

// PackageMetadata represents a single package/crate
type PackageMetadata struct {
    ID          string           `json:"id"`           // Full package ID
    Name        string           `json:"name"`          // Crate name
    Version     string           `json:"version"`
    ManifestPath string          `json:"manifest_path"`
    Targets     []TargetMetadata `json:"targets"`
}

// TargetMetadata represents a build target
type TargetMetadata struct {
    Name    string   `json:"name"`     // Target name (e.g., "test_client")
    Kind    []string `json:"kind"`     // ["test"], ["bin"], ["lib"], etc.
    SrcPath string   `json:"src_path"`  // Full path to source file
}

// Update CargoTestDefinition struct
type CargoTestDefinition struct {
    // ... existing fields ...

    // New fields for metadata integration
    workspaceMetadata   *WorkspaceMetadata
    testNameToCrate     map[string]string  // test_client -> actix_http
    isWorkspace         bool
    metadataLoadError   error              // Track if metadata loading failed
}
```

### Phase 2: Metadata Loading Implementation

Replace the placeholder `loadCargoMetadata()` function:

```go
func (c *CargoTestDefinition) loadCargoMetadata() {
    // Set timeout for metadata command
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    cmd := exec.CommandContext(ctx, "cargo", "metadata", "--format-version", "1", "--no-deps")
    output, err := cmd.Output()

    if err != nil {
        c.logger.Debug("Failed to load cargo metadata: %v (falling back to basic detection)", err)
        c.metadataLoadError = err
        return
    }

    var metadata WorkspaceMetadata
    if err := json.Unmarshal(output, &metadata); err != nil {
        c.logger.Debug("Failed to parse cargo metadata: %v", err)
        c.metadataLoadError = err
        return
    }

    c.workspaceMetadata = &metadata
    c.buildTestMappings()
}

func (c *CargoTestDefinition) buildTestMappings() {
    c.testNameToCrate = make(map[string]string)

    // Create set of workspace members for filtering
    workspaceMembers := make(map[string]bool)
    for _, memberID := range c.workspaceMetadata.WorkspaceMembers {
        workspaceMembers[memberID] = true
    }

    // Build mappings only for workspace packages
    for _, pkg := range c.workspaceMetadata.Packages {
        if !workspaceMembers[pkg.ID] {
            continue // Skip dependencies
        }

        for _, target := range pkg.Targets {
            // Check if this is a test target
            for _, kind := range target.Kind {
                if kind == "test" {
                    // Map both with and without underscores (cargo normalizes)
                    testName := target.Name
                    c.testNameToCrate[testName] = pkg.Name

                    // Also map the normalized form (test-client -> test_client)
                    normalized := strings.ReplaceAll(testName, "-", "_")
                    c.testNameToCrate[normalized] = pkg.Name

                    c.logger.Debug("Mapped test %s -> crate %s", testName, pkg.Name)
                }
            }
        }
    }

    c.isWorkspace = len(c.workspaceMetadata.WorkspaceMembers) > 1
    c.logger.Debug("Loaded metadata: %d packages, %d test mappings, workspace=%v",
        len(c.workspaceMetadata.Packages), len(c.testNameToCrate), c.isWorkspace)
}
```

### Phase 3: Integration with Test Processing

Update the test detection logic:

```go
func (c *CargoTestDefinition) processLineData(line string, jsonEventCount *int) {
    // ... existing code ...

    // Enhanced integration test detection
    if matches := runningIntegrationTestsRegex.FindStringSubmatch(line); matches != nil {
        testName := matches[1]  // e.g., "test_client"
        c.mu.Lock()

        // Try to resolve crate from metadata first
        if crateName, ok := c.testNameToCrate[testName]; ok {
            // Use properly qualified name
            groupName := fmt.Sprintf("%s::%s", crateName, testName)
            c.currentCrate = groupName
            c.logger.Debug("Set current crate to: %s (integration test with metadata)", groupName)
        } else if c.metadataLoadError == nil && len(c.testNameToCrate) > 0 {
            // Metadata loaded but test not found - might be from a non-workspace crate
            c.currentCrate = testName
            c.logger.Debug("Test %s not in metadata, using unqualified name", testName)
        } else {
            // Fallback: Use test name as-is if metadata unavailable
            c.currentCrate = testName
            c.logger.Debug("Set current crate to: %s (integration test, no metadata)", testName)
        }

        c.mu.Unlock()
        return
    }

    // ... rest of existing code ...
}
```

### Phase 4: Group Reporting Enhancement

Update group discovery/start events to use proper names:

```go
func (c *CargoTestDefinition) sendGroupDiscovered(groupName string, parentNames []string) {
    // For integration tests with crate prefix, extract clean display name
    displayName := groupName
    if strings.Contains(groupName, "::") {
        parts := strings.SplitN(groupName, "::", 2)
        displayName = fmt.Sprintf("%s/%s", parts[0], parts[1])
    }

    event := map[string]interface{}{
        "eventType": "testGroupDiscovered",
        "payload": map[string]interface{}{
            "groupName":    displayName,
            "parentNames":  parentNames,
            "metadata": map[string]interface{}{
                "fullName": groupName,
                "type":     "integration",
            },
        },
    }
    c.ipcWriter.WriteEvent(event)
}
```

## Error Handling Strategy

1. **Metadata Command Fails**:
   - Log warning, continue with existing behavior
   - Tests still run but without crate qualification

2. **Timeout**:
   - 5-second timeout on metadata command
   - Prevents hanging on large projects

3. **Non-Workspace Projects**:
   - Single-crate projects work normally
   - No qualification needed when there's no ambiguity

4. **Missing Test in Metadata**:
   - Could happen with dynamically generated tests
   - Fall back to unqualified name

## Testing Plan

1. **Unit Tests**:
   - Mock cargo metadata output parsing
   - Test mapping construction
   - Test fallback behavior

2. **Integration Tests**:
   - Test with actix-web (known problematic case)
   - Test with single-crate projects
   - Test with projects that have no tests
   - Test with --package filter

3. **Performance Tests**:
   - Measure overhead of metadata command
   - Ensure timeout works correctly

## Rollout Strategy

1. **Feature Flag** (optional):
   - Add `THREEPIO_USE_CARGO_METADATA` env var
   - Default to enabled, allow disabling if issues arise

2. **Gradual Enhancement**:
   - Phase 1: Basic metadata loading and mapping
   - Phase 2: Enhanced error messages showing crate context
   - Phase 3: Support for benchmark and example tests

## Success Criteria

- ✅ actix-web shows 1252 tests (matching baseline exactly)
- ✅ Tests from different crates maintain separate groups
- ✅ Group names clearly indicate owning crate
- ✅ No performance regression (< 100ms overhead)
- ✅ Graceful fallback when metadata unavailable

## Timeline Estimate

- Implementation: 4-6 hours
- Testing: 2-3 hours
- Documentation: 1 hour
- Total: ~1 day of focused work

## Alternative Considerations

If full metadata integration proves problematic:
1. Quick fix with binary hash remains available
2. Could use simplified heuristic based on Compiling output
3. Could make crate grouping optional via flag

## Dependencies

- No new external dependencies required
- Uses standard `cargo metadata` command (stable since Rust 1.0)
- JSON parsing with Go's standard library

## Backwards Compatibility

- Fully backwards compatible
- Falls back to existing behavior if metadata unavailable
- No changes to IPC protocol or report format (just better group names)