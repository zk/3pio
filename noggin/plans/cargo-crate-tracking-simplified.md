# Simplified Cargo Crate Tracking Implementation Plan

## Problem Statement
When running `cargo test` in Rust workspaces, integration tests with identical filenames from different crates are incorrectly merged into a single test group, causing inaccurate test counts and loss of crate context.

## Solution: Track Crate Context from Unit Test Lines

### Key Discovery
Cargo ALWAYS runs unit tests before integration tests for each crate, even with parallel execution. The unit test line contains the actual crate name, which we can use to qualify subsequent integration tests.

### Implementation Strategy

## Progress Checklist

### Phase 1: Test-Driven Development Setup
- [ ] Write failing unit test for crate name extraction from unit test line
- [ ] Write failing unit test for integration test qualification
- [ ] Write failing integration test with multi-crate workspace fixture
- [ ] Verify all tests fail as expected

### Phase 2: Core Implementation (20 lines)
- [ ] Add `lastUnitTestCrate` field to `CargoTestDefinition` struct
- [ ] Update `processLineData` to track crate from unit test lines
- [ ] Modify integration test handling to use crate qualification
- [ ] Ensure all existing tests still pass

### Phase 3: Test Verification
- [ ] Unit test: Crate name extraction passes
- [ ] Unit test: Integration test qualification passes
- [ ] Integration test: Multi-crate workspace passes
- [ ] Manual test with actix-web repository

### Phase 4: Edge Case Testing
- [ ] Test with crate having only unit tests
- [ ] Test with crate having only integration tests
- [ ] Test with multiple integration test files per crate
- [ ] Test with doc tests

### Phase 5: Polish & Documentation
- [ ] Update code comments
- [ ] Update architecture documentation
- [ ] Close related issue
- [ ] Remove TODO comment in `loadCargoMetadata`

## TDD Test Cases

### 1. Unit Test: Extract Crate Name from Unit Test Line
```go
func TestCargoExtractCrateFromUnitTest(t *testing.T) {
    def := NewCargoTestDefinition(logger)

    line := "     Running unittests src/lib.rs (target/debug/deps/actix_http-abc123def456)"
    def.processLineData(line, &jsonCount)

    assert.Equal(t, "actix_http", def.lastUnitTestCrate)
    assert.Equal(t, "actix_http", def.currentCrate)
}
```

### 2. Unit Test: Qualify Integration Test with Crate
```go
func TestCargoQualifyIntegrationTest(t *testing.T) {
    def := NewCargoTestDefinition(logger)
    def.lastUnitTestCrate = "actix_http"

    line := "     Running tests/test_client.rs (target/debug/deps/test_client-xyz789)"
    def.processLineData(line, &jsonCount)

    assert.Equal(t, "actix_http::test_client", def.currentCrate)
}
```

### 3. Integration Test: Multi-Crate Workspace
```go
func TestCargoMultiCrateWorkspace(t *testing.T) {
    // Create fixture with two crates having test_client.rs
    // Run cargo test through 3pio
    // Verify two distinct test groups created
}
```

## Implementation Code (Simplified)

```go
// Add field to struct
type CargoTestDefinition struct {
    // ... existing fields ...
    lastUnitTestCrate string  // Track crate from most recent unit test
}

// In processLineData function:
func (c *CargoTestDefinition) processLineData(line string, jsonEventCount *int) {
    // Extract crate name from unit test line
    if matches := runningUnittestsRegex.FindStringSubmatch(line); matches != nil {
        crateName := matches[1]  // e.g., "actix_http" from "actix_http-abc123"
        c.mu.Lock()
        c.currentCrate = crateName
        c.lastUnitTestCrate = crateName  // Save for integration tests
        c.mu.Unlock()
        return
    }

    // For integration tests, qualify with the last crate
    if matches := runningIntegrationTestsRegex.FindStringSubmatch(line); matches != nil {
        testName := matches[1]  // e.g., "test_client"
        c.mu.Lock()
        // Create unique identifier using crate::test format
        if c.lastUnitTestCrate != "" {
            c.currentCrate = fmt.Sprintf("%s::%s", c.lastUnitTestCrate, testName)
        } else {
            // Fallback if no unit test was seen (shouldn't happen)
            c.currentCrate = testName
        }
        c.mu.Unlock()
        return
    }

    // ... rest of existing code ...
}
```

## Expected Outcomes

1. **actix-web test count**: Should show 1252 tests (currently shows 1251 due to merging)
2. **Distinct test groups**: `actix_http::test_client` and `awc::test_client` instead of single `test_client`
3. **No performance impact**: Uses existing stderr output, no additional subprocess
4. **100% backwards compatible**: Existing single-crate projects unaffected

## Risk Mitigation

- If unit test line is missing (shouldn't happen), fall back to current behavior
- Test extensively with parallel execution flags
- Verify with multiple real-world Rust workspaces

## Success Metrics

- ✅ actix-web shows correct 1252 test count
- ✅ All existing cargo tests pass
- ✅ New tests for crate qualification pass
- ✅ No performance regression
- ✅ Works with all cargo flags and options