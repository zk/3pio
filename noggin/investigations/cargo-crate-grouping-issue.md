# Cargo Test Crate Grouping Issue

## Problem
When running `cargo test` in a Rust workspace, 3pio incorrectly merges tests from different crates that have the same test file name.

## Example
In actix-web workspace:
- `actix-http/tests/test_client.rs` contains test `with_query_parameter`
- `awc/tests/test_client.rs` also contains test `with_query_parameter`

**Expected**: 2 separate test groups (actix-http:test_client and awc:test_client) with 2 total tests
**Actual**: 1 merged test group (test_client) with deduplicated tests

## Root Cause
In `/Users/edie/code/3pio/internal/runner/definitions/cargo.go`:

```go
// Line 291-296
if matches := runningIntegrationTestsRegex.FindStringSubmatch(line); matches != nil {
    testName := matches[1]  // Extracts just "test_client" from binary name
    c.mu.Lock()
    c.currentCrate = testName  // Uses test name as crate identifier
    c.logger.Debug("Set current crate to: %s (integration tests)", testName)
    c.mu.Unlock()
```

The regex pattern extracts only the test filename from paths like:
- `Running tests/test_client.rs (target/debug/deps/test_client-ad5dbdac46a0463a)`
- `Running tests/test_client.rs (target/debug/deps/test_client-b7d6393018cb870a)`

Both resolve to `test_client`, losing the crate context.

## Impact
- Test counts may be incorrect (tests appear deduplicated when they shouldn't be)
- Test results from different crates are incorrectly merged into single reports
- Makes it difficult to identify which crate a failing test belongs to

## Solution Options

### Option 1: Parse Cargo Metadata
Run `cargo metadata` before tests to build a map of which test files belong to which crates, then use this to properly assign crate context.

### Option 2: Extract Crate from Path Context
Look for crate indicators in the surrounding output (e.g., "Compiling actix-http" messages that appear before test execution).

### Option 3: Use Binary Hash as Discriminator
Include the binary hash (e.g., `ad5dbdac46a0463a`) in the group name to ensure uniqueness: `test_client-ad5dbdac46a0463a`.

### Option 4: Parse JSON Events for Package Info
The cargo test JSON format may include package information that could be used to properly group tests.

## Workaround
Users can run tests for individual crates separately:
```bash
3pio cargo test -p actix-http
3pio cargo test -p awc
```

## Related Files
- `/Users/edie/code/3pio/internal/runner/definitions/cargo.go` - Cargo test adapter
- `/Users/edie/code/3pio/noggin/reports/open-source/actix-web-20250919-211747/report.md` - Example showing the issue