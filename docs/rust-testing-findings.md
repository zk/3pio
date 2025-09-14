# Rust Testing with 3pio - Initial Findings

## Test Date
2025-09-14

## Projects Tested

### Successfully Cloned
1. **alacritty** - Terminal emulator (requires newer Rust version)
2. **uv** - Python package manager (auto-installs Rust 1.89.0)
3. **sway** - Smart contract language
4. **zed** - Code editor
5. **deno** - JavaScript runtime
6. **rust** - Rust compiler
7. **tauri** - Desktop app framework
8. **rustdesk** - Remote desktop
9. **union** - Blockchain bridging protocol

### Removed (Not Suitable)
- **rustlings** - Educational exercises, not a typical test structure

## Test Results

### Direct cargo test (without 3pio)
Successfully ran on `uv` project:
- Command: `cargo test -p uv-cli --lib`
- Result: 11 tests passed
- Test structure: `module::tests::test_name`

### cargo test with JSON output
Successfully obtained JSON output:
- Command: `RUSTC_BOOTSTRAP=1 cargo test -p uv-cli --lib -- -Z unstable-options --format json`
- Output format matches expected libtest JSON format
- Sample events:
  ```json
  { "type": "suite", "event": "started", "test_count": 11 }
  { "type": "test", "event": "started", "name": "comma::tests::double" }
  { "type": "test", "name": "comma::tests::single", "event": "ok", "exec_time": 0.000047666 }
  ```

### 3pio with cargo test
**Issue Found**: Test runner misdetection
- Command: `3pio cargo test -p uv-cli --lib`
- Problem: Detected as "go test" instead of "cargo test"
- Impact: JSON parsing failed, no tests reported
- Error: `[ERROR] Failed to process native output: error reading cargo test output: read |0: file already closed`

## Technical Analysis

### Code Review
1. **Implementation exists**:
   - `internal/runner/definitions/cargo.go` ✅
   - `internal/runner/definitions/nextest.go` ✅
   - `internal/runner/definitions/cargo_wrapper.go` ✅
   - `internal/runner/definitions/nextest_wrapper.go` ✅

2. **Registration confirmed**:
   - Cargo and nextest are registered in `manager.go` lines 44-48

3. **Detection logic appears correct**:
   - `cargo.go` Detect() checks for "cargo test" command pattern

### Root Cause Hypothesis
The runner detection in `Manager.Detect()` iterates over an unordered map, which may cause "go" test to match before "cargo" test. The Go test detector might be incorrectly matching the cargo command.

## Recommendations

### Immediate Fix Needed
1. Debug why "go test" is matching "cargo test" commands
2. Implement ordered runner detection (check cargo before go)
3. Add more specific detection logic to prevent false matches

### Testing Strategy
1. **Use uv project for testing** - It's well-structured and auto-manages Rust versions
2. **Skip alacritty initially** - Requires Rust 1.85.0+ with edition 2024
3. **Test with smaller packages first** - e.g., `cargo test -p uv-cli --lib`

### Next Steps
1. Fix the runner detection issue
2. Test with successful JSON parsing
3. Verify hierarchical group structure works with Rust's module paths
4. Test workspace support with multiple crates
5. Add integration tests for cargo test

## Sample Commands for Testing

```bash
# Basic test
cd open-source/uv
../../build/3pio cargo test -p uv-cli --lib

# Workspace test
../../build/3pio cargo test --workspace

# With test filtering
../../build/3pio cargo test -p uv-cli --lib comma::tests

# Check reports
cat .3pio/runs/*/test-run.md
```

## Success Criteria
- ✅ JSON output obtained from cargo test
- ✅ Rust support code implemented
- ❌ Correct runner detection
- ❌ Successful test parsing
- ❌ Hierarchical report generation