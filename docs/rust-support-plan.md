# Rust Test Runner Support Plan

## Overview

This document outlines the plan for adding Rust test runner support to 3pio, covering both `cargo test` (the standard Rust test runner) and `cargo-nextest` (a modern, faster alternative).

## Executive Summary

Rust support will follow the native integration pattern used by Go, processing JSON output directly from test runners without requiring external adapters. Both `cargo test` and `cargo-nextest` will be supported as separate runner definitions, providing comprehensive coverage for the Rust ecosystem.

## Test Runners to Support

### 1. cargo test (Essential)
- **Status**: Default Rust test runner, ships with every Rust installation
- **Adoption**: Universal - every Rust project uses it
- **Key Features**:
  - Runs unit tests, integration tests, and doc tests
  - Built into the Rust toolchain
  - Standard in all CI/CD pipelines
- **JSON Support**: Available via unstable flags (`-Z unstable-options --format json`)

### 2. cargo-nextest (High Value)
- **Status**: Modern test runner, growing adoption
- **Adoption**: Used by major projects (Tokio, Wasmtime, Materialize, Deno)
- **Key Features**:
  - 3x faster execution on average through better parallelization
  - Stable machine-readable output formats
  - Test isolation and better failure reporting
  - Built-in test sharding for distributed CI
- **JSON Support**: First-class support via `--message-format libtest-json`

## Technical Implementation

### Architecture Approach

Following the Go test model, both Rust runners will be implemented as native runners that process JSON output directly, without requiring embedded adapters.

```
User Command → 3pio → Modify Command → Execute → Process JSON → Generate Reports
                       (add JSON flags)           (stream parse)
```

### Runner Definitions

#### CargoTestDefinition (`internal/runner/definitions/cargo.go`)

```go
type CargoTestDefinition struct {
    logger           *logger.FileLogger
    ipcWriter        *IPCWriter
    testStates       map[string]*TestState
    crateGroups      map[string]*CrateGroupInfo
    moduleGroups     map[string]*ModuleGroupInfo
    discoveredGroups map[string]bool
}

// Detect checks for cargo test command
func (c *CargoTestDefinition) Detect(args []string) bool {
    return len(args) >= 2 &&
           args[0] == "cargo" &&
           args[1] == "test"
}

// ModifyCommand adds JSON output flags
func (c *CargoTestDefinition) ModifyCommand(cmd []string) []string {
    // Adds: -- -Z unstable-options --format json --report-time
    // Note: RUSTC_BOOTSTRAP=1 will be set in process environment at spawn time
}
```

#### NextestDefinition (`internal/runner/definitions/nextest.go`)

```go
type NextestDefinition struct {
    logger           *logger.FileLogger
    ipcWriter        *IPCWriter
    testStates       map[string]*TestState
    packageGroups    map[string]*PackageGroupInfo
    discoveredGroups map[string]bool
}

// Detect checks for cargo nextest run command
func (n *NextestDefinition) Detect(args []string) bool {
    return len(args) >= 3 &&
           args[0] == "cargo" &&
           args[1] == "nextest" &&
           args[2] == "run"
}

// ModifyCommand adds JSON output flags
func (n *NextestDefinition) ModifyCommand(cmd []string) []string {
    // Adds: --message-format libtest-json
}
```

### JSON Event Processing

#### cargo test JSON Format
```json
{ "type": "suite", "event": "started", "test_count": 42 }
{ "type": "test", "event": "started", "name": "tests::math::test_addition" }
{ "type": "test", "name": "tests::math::test_addition", "event": "ok", "exec_time": 0.0023 }
{ "type": "test", "name": "tests::math::test_division", "event": "failed", "stdout": "...", "stderr": "..." }
{ "type": "suite", "event": "ok", "passed": 41, "failed": 1, "ignored": 0 }
```

#### cargo-nextest JSON Format
```json
{ "type": "test", "event": "started", "name": "my_crate::tests::test_function" }
{ "type": "test", "event": "ok", "name": "my_crate::tests::test_function", "exec_time": 0.123, "stdout": "", "stderr": "" }
{ "type": "suite", "event": "finished", "passed": 10, "failed": 2, "ignored": 1, "exec_time": 1.5 }
```

### Hierarchical Group Mapping

Both runners will map Rust's test organization to 3pio's universal group abstractions:

#### Rust Test Hierarchy → 3pio Groups

```
crate::module::submodule::test_function
   ↓
Group: crate (root)
  └── Group: module
      └── Group: submodule
          └── Test: test_function
```

Example mappings:

| Rust Test Path | 3pio Group Hierarchy |
|----------------|---------------------|
| `my_app::tests::math::test_add` | `my_app` > `tests` > `math` > test_add |
| `integration::api::test_endpoint` | `integration` > `api` > test_endpoint |
| `doc_test_my_function_0` | `doctests` > doc_test_my_function_0 |

## Implementation Comparison

| Feature | cargo test | cargo-nextest |
|---------|-----------|---------------|
| **JSON Flag** | `-- -Z unstable-options --format json` | `--message-format libtest-json` |
| **Stability** | Unstable (requires RUSTC_BOOTSTRAP=1) | Stable (experimental feature) |
| **Test Discovery** | `cargo test -- --list` | `cargo nextest list --message-format json` |
| **Doctest Support** | Yes | No (must use cargo test) |
| **Workspace Support** | `--workspace` flag | `--workspace` flag |
| **Package Selection** | `-p <package>` | `-p <package>` |
| **Parallel Execution** | Limited control | Fine-grained control |
| **Test Isolation** | Same process | Separate processes |
| **Output Format** | libtest JSON | libtest-json or libtest-json-plus |

## User Experience

### Command Examples

```bash
# Standard cargo test
3pio cargo test                           # All tests in current package
3pio cargo test --workspace               # All tests in workspace
3pio cargo test -p my-crate              # Specific package
3pio cargo test --lib                    # Library tests only
3pio cargo test --doc                    # Doc tests only
3pio cargo test tests::math              # Specific test module

# cargo-nextest
3pio cargo nextest run                   # All tests
3pio cargo nextest run --workspace       # Workspace tests
3pio cargo nextest run -p my-crate      # Specific package
3pio cargo nextest run --partition 1/3   # Sharded execution
```

### Expected Output Structure

```
.3pio/runs/[timestamp]-[name]/
├── test-run.md                           # Main report
├── output.log                            # Complete stdout/stderr
└── reports/
    ├── my_crate/                        # Crate-level group
    │   ├── index.md                     # Crate-level tests
    │   ├── unit_tests/                  # Module group
    │   │   ├── index.md                 # Module tests
    │   │   └── math/                    # Nested module
    │   │       └── index.md             # Math tests
    │   └── integration_tests/           # Integration test group
    │       └── index.md
    └── my_other_crate/                  # Another crate in workspace
        └── index.md
```

## Implementation Phases

### Phase 1: cargo test Support (Week 1-2)
- [ ] Create `CargoTestDefinition` struct
- [ ] Implement command detection and modification
- [ ] Set RUSTC_BOOTSTRAP=1 in subprocess environment (not global)
- [ ] Parse JSON events and map to IPC events
- [ ] Test with single-crate projects
- [ ] Return empty array from `GetTestFiles()` for dynamic discovery

### Phase 2: Hierarchical Support (Week 2-3)
- [ ] Parse module paths into group hierarchy
- [ ] Support nested test modules
- [ ] Handle workspace with multiple crates
- [ ] Track duration and statistics per group
- [ ] Support integration tests (tests/ directory)

### Phase 3: cargo-nextest Support (Week 3-4)
- [ ] Create `NextestDefinition` struct
- [ ] Implement nextest-specific JSON parsing
- [ ] Handle nextest's partition feature
- [ ] Return empty array from `GetTestFiles()` for dynamic discovery
- [ ] Test with large parallel test suites

### Phase 4: Advanced Features (Week 4-5)
- [ ] Doctest support for cargo test
- [ ] Benchmark test handling (`cargo bench`)
- [ ] Custom test harness detection
- [ ] Handle test filtering patterns
- [ ] Support cargo test flags (`--lib`, `--bins`, `--examples`)

### Phase 5: Testing & Polish (Week 5-6)
- [ ] Create comprehensive test fixtures
- [ ] Handle edge cases (panics, timeouts, compilation failures)
- [ ] Performance testing with large test suites
- [ ] Documentation and examples
- [ ] Integration tests for both runners

## Challenges and Solutions

### Challenge 1: Unstable JSON Format
**Problem**: cargo test's JSON output requires unstable features
**Solution**:
- Set `RUSTC_BOOTSTRAP=1` in subprocess environment only (not global)
- Transparent to users - handled automatically by 3pio
- Monitor stabilization progress (rust-lang/rust#49359)
- If JSON fails, suggest cargo-nextest as alternative

### Challenge 2: Test Discovery
**Problem**: No pre-execution test discovery needed
**Solution**:
- Follow the existing pattern: `GetTestFiles()` returns empty array for dynamic discovery
- Tests are discovered as they execute and send events
- Similar to how Jest/Vitest/pytest currently work in 3pio
- For Go, we use `go list` for package metadata, not test discovery - Rust can use `cargo metadata` similarly if needed for workspace structure

### Challenge 3: Workspace Complexity
**Problem**: Multi-crate workspaces need special handling
**Solution**:
- Parse `cargo metadata` to understand workspace structure
- Create crate-level root groups
- Handle cross-crate test dependencies
- Support package-specific test runs

### Challenge 4: Doctest Integration
**Problem**: Doc tests have different naming and structure
**Solution**:
- Detect `--doc` flag in command
- Create separate "doctests" group
- Parse doc test names from generated test names
- Note: Only supported with cargo test, not nextest

## Success Metrics

- Support 90% of common Rust test workflows
- Process 10,000+ test results without performance degradation
- Zero adapter extraction overhead (native processing)
- Compatible with major Rust project structures:
  - Single crate projects
  - Multi-crate workspaces
  - Projects with integration tests
  - Projects with doc tests
- Seamless switching between cargo test and nextest

## Future Enhancements

### Potential Future Support
- **cargo-tarpaulin**: Coverage-focused test runner
- **cargo-fuzz**: Fuzzing framework integration
- **criterion**: Benchmark framework with structured output
- **proptest**: Property-based testing results

### Optimization Opportunities
- Parallel processing of multiple crates in workspace
- Incremental test result updates during long runs
- Test result caching for unchanged code
- Integration with cargo watch for continuous testing

## Risk Mitigation

| Risk | Mitigation Strategy |
|------|-------------------|
| Unstable API changes | Pin to specific Rust version, monitor deprecation notices |
| Performance impact | Benchmark JSON parsing overhead, optimize hot paths |
| Adoption barriers | Clear setup documentation, automated setup detection |
| Compatibility issues | Test with diverse project structures, maintain compatibility matrix |
| Nextest not installed | Detect and provide installation instructions |

## Conclusion

Supporting both cargo test and cargo-nextest provides comprehensive coverage for the Rust ecosystem. The native integration approach aligns with 3pio's architecture for compiled language test runners and avoids the complexity of embedded adapters. This dual support ensures that 3pio works with every Rust project while offering enhanced capabilities for projects using modern tooling.

## References

- [Rust libtest JSON format (unstable)](https://github.com/rust-lang/rust/issues/49359)
- [cargo-nextest documentation](https://nexte.st/)
- [cargo test documentation](https://doc.rust-lang.org/cargo/commands/cargo-test.html)
- [cargo-nextest machine-readable output](https://nexte.st/book/machine-readable.html)