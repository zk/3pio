# Rust Test Projects for 3pio Validation

This document outlines the top Rust projects on GitHub organized by testing difficulty tiers, with specific verification strategies for validating 3pio's Rust support.

## Testing Tiers

### 🟢 Tier 1: Easy (Start Here)

These projects have simple test structures and standard cargo test usage, perfect for initial 3pio validation.

#### 1. **rustlings** (59,941⭐)
- **Why Easy**: Educational exercises with simple, independent tests
- **Test Structure**: Individual exercise files with inline tests
- **Verification Strategy**:
  ```bash
  git clone https://github.com/rust-lang/rustlings
  cd rustlings
  3pio cargo test
  # Verify: Basic test discovery, simple pass/fail reporting
  # Check: .3pio/runs/*/test-run.md shows all exercises
  # Validate: Individual test results match cargo test output
  ```

#### 2. **alacritty** (60,275⭐)
- **Why Easy**: Terminal emulator with standard test patterns
- **Test Structure**: Clean unit tests and integration tests
- **Verification Strategy**:
  ```bash
  git clone https://github.com/alacritty/alacritty
  cd alacritty
  3pio cargo test
  # Verify: Module hierarchy (alacritty::config::tests::*)
  # Check: Integration tests in separate groups
  # Validate: Test durations and console output capture
  ```

### 🟡 Tier 2: Medium

These projects have workspace structures, multiple crates, or moderate complexity.

#### 3. **uv** (67,439⭐)
- **Why Medium**: Multi-crate workspace with extensive testing
- **Test Structure**: Workspace with multiple packages
- **Verification Strategy**:
  ```bash
  git clone https://github.com/astral-sh/uv
  cd uv
  3pio cargo test --workspace
  # Verify: Workspace support with multiple crate groups
  # Check: Hierarchical reports (workspace > crate > module > test)
  # Validate: Performance with 1000+ tests
  # Test: 3pio cargo test -p uv-resolver (single package)
  ```

#### 4. **sway** (62,141⭐)
- **Why Medium**: Smart contract language with complex test organization
- **Test Structure**: Multiple crates with domain-specific tests
- **Verification Strategy**:
  ```bash
  git clone https://github.com/FuelLabs/sway
  cd sway
  3pio cargo test --workspace
  # Verify: Complex workspace with 20+ crates
  # Check: Nested test modules (sway_core::semantic_analysis::tests::*)
  # Validate: Doc test handling
  # Test: Partial runs with -p flags
  ```

#### 5. **zed** (65,522⭐)
- **Why Medium**: Large codebase with comprehensive testing
- **Test Structure**: Multi-crate editor with UI and backend tests
- **Verification Strategy**:
  ```bash
  git clone https://github.com/zed-industries/zed
  cd zed
  3pio cargo test --workspace --lib
  # Verify: Large workspace handling (50+ crates)
  # Check: Library-only test filtering
  # Validate: Parallel test execution reporting
  # Test: 3pio cargo nextest run (if they use nextest)
  ```

### 🔴 Tier 3: Hard

These projects have custom test frameworks, massive test suites, or special requirements.

#### 6. **deno** (104,200⭐)
- **Why Hard**: Custom test harnesses and massive test suite
- **Test Structure**: Mix of Rust tests and JavaScript runtime tests
- **Verification Strategy**:
  ```bash
  git clone https://github.com/denoland/deno
  cd deno
  3pio cargo test --workspace
  # Verify: Handling of 5000+ tests
  # Check: Custom test macros and generated tests
  # Validate: Performance and memory usage
  # Test: Integration test handling in cli/tests/
  # Challenge: May timeout or need special flags
  ```

#### 7. **rust-lang/rust** (106,433⭐)
- **Why Hard**: The Rust compiler with custom test framework
- **Test Structure**: Compiletest framework, UI tests, run-pass tests
- **Verification Strategy**:
  ```bash
  git clone https://github.com/rust-lang/rust
  cd rust
  # Standard library tests only (compiler tests use custom framework)
  3pio cargo test --lib -p std
  # Verify: Stdlib test handling
  # Check: Massive nested module structure
  # Note: Full test suite requires x.py and won't work with 3pio initially
  # Challenge: Custom test framework (compiletest) needs special adapter
  ```

#### 8. **tauri** (96,517⭐)
- **Why Hard**: Desktop framework with platform-specific tests
- **Test Structure**: Core Rust tests plus webview integration tests
- **Verification Strategy**:
  ```bash
  git clone https://github.com/tauri-apps/tauri
  cd tauri
  3pio cargo test --workspace --lib
  # Verify: Cross-platform test handling
  # Check: Conditional compilation tests (#[cfg(target_os = ...)])
  # Validate: Feature-gated test discovery
  # Challenge: Some tests may require specific platform setup
  ```

### 🔵 Tier 4: Validation Projects

These projects are good for specific feature validation.

#### 9. **rustdesk** (98,187⭐)
- **Why Validation**: Tests 3pio with GUI/system integration tests
- **Test Focus**: Network and GUI component testing
- **Verification Strategy**:
  ```bash
  git clone https://github.com/rustdesk/rustdesk
  cd rustdesk
  3pio cargo test --workspace --lib
  # Verify: Filtering out platform-specific tests
  # Check: Network test handling
  # Validate: Binary crate vs library test separation
  ```

#### 10. **unionlabs/union** (74,816⭐)
- **Why Validation**: Blockchain project with specialized testing
- **Test Focus**: Cryptographic and consensus tests
- **Verification Strategy**:
  ```bash
  git clone https://github.com/unionlabs/union
  cd union
  3pio cargo test --workspace
  # Verify: Long-running test handling
  # Check: Specialized test output (crypto proofs, benchmarks)
  # Validate: Test timeout handling
  ```

## Verification Checklist

For each project, validate these 3pio features:

### Basic Functionality
- [ ] Test discovery and counting matches cargo test
- [ ] Pass/fail status correctly reported
- [ ] Exit codes match original test runner
- [ ] Test durations captured accurately

### Hierarchical Reporting
- [ ] Workspace structure preserved
- [ ] Crate grouping correct
- [ ] Module nesting maintained
- [ ] Test names fully qualified

### Output Handling
- [ ] stdout/stderr captured per test
- [ ] Console output in output.log
- [ ] Panic messages preserved
- [ ] Assert messages included in failures

### Performance
- [ ] Reasonable overhead (<10% slower than direct cargo test)
- [ ] Memory usage stable for large test suites
- [ ] Incremental report writing works
- [ ] Handles interruption (Ctrl+C) gracefully

### Advanced Features
- [ ] cargo test flags work (--lib, --bins, --tests, --doc)
- [ ] Package selection works (-p package_name)
- [ ] Workspace testing (--workspace)
- [ ] Test filtering (test_name patterns)
- [ ] cargo-nextest support (where applicable)

## Testing Progression

### Week 1-2: Tier 1
Start with rustlings and alacritty to validate basic cargo test support.

### Week 3-4: Tier 2
Test workspace support with uv, sway, and zed. Focus on hierarchical reporting.

### Week 5-6: Tier 3
Attempt deno and rust-lang/rust for stress testing. Document limitations.

### Ongoing: Tier 4
Use validation projects to test specific features as they're implemented.

## Success Metrics

- **Tier 1**: 100% test compatibility, all features work
- **Tier 2**: 95% compatibility, minor edge cases documented
- **Tier 3**: 80% compatibility, major features work with known limitations
- **Tier 4**: Feature-specific validation successful

## Notes

- Start testing with `RUSTC_BOOTSTRAP=1` for JSON output
- Document any projects that require cargo-nextest
- Track performance metrics for each project
- Note any custom test patterns that need special handling
- Create GitHub issues for any incompatibilities found