# Verified Libraries

This document lists open source libraries that have been tested with 3pio and shown one-to-one matching results with their native test runners. Only libraries where 3pio produces identical test counts, pass/fail results, and exit codes to the native runner are included.

The verification process involves:
1. Running the test suite without 3pio (baseline)
2. Running the same test suite with 3pio
3. Comparing test counts, results, and exit codes
4. Verifying report generation completeness

## JavaScript/TypeScript Libraries

### jest
- **Repository**: https://github.com/jestjs/jest
- **Date Verified**: 2025-09-16
- **Commit Hash**: `2b49b12a1eebc3ae8a13f0e694fc880a47594298`
- **Test Command**: `yarn jest --ci`
- **Test Results**: 5092 tests (4110 passed, 982 failed) - identical with/without 3pio
- **Notes**: One of the most complex JavaScript test suites. Failed tests are known snapshot issues in Jest codebase itself.

### vueuse
- **Repository**: https://github.com/vueuse/vueuse
- **Date Verified**: 2025-09-15
- **Commit Hash**: Not specified (latest main at time)
- **Test Command**: `pnpm test:unit`
- **Test Results**: 179 test files (177 passed, 2 failed) - identical with/without 3pio
- **Notes**: Large Vue.js monorepo with comprehensive composables testing

## Go Libraries

### uuid (Google)
- **Repository**: https://github.com/google/uuid
- **Date Verified**: 2025-09-19
- **Commit Hash**: `2d3c2a9cc518326daf99a383f07c4d3c44317e4d`
- **Test Command**: `go test -v ./...`
- **Test Results**: 213 tests (212 passed, 1 skipped) - identical with/without 3pio
- **Notes**: UUID generation and parsing library. Includes extensive test coverage with subtests and fuzz tests. The skipped test (TestClockSeqRace) skips regression tests by design.

### gin
- **Repository**: https://github.com/gin-gonic/gin
- **Date Verified**: 2025-09-19
- **Commit Hash**: `2119046230f0119c7c88f86a6b441d9d3aaad03e`
- **Test Command**: `go test -v ./...`
- **Test Results**: 588 tests (586 passed, 1 failed, 1 skipped) - identical with/without 3pio
- **Notes**: Popular Go web framework. The failing test (TestRunEmpty) fails due to port 8080 already in use, which occurs in both baseline and 3pio runs. Demonstrates perfect accuracy with sub-tests and multi-package support (7 packages tested).

### echo
- **Repository**: https://github.com/labstack/echo
- **Date Verified**: 2025-09-19
- **Commit Hash**: `52d2bff1b9ebb7c581304ed2e5d72397ec40ca6d`
- **Test Command**: `go test -v ./...`
- **Test Results**: 1535 tests passed - identical with/without 3pio
- **Notes**: High performance, minimalist Go web framework. Comprehensive test suite includes both core framework tests and middleware tests across 2 packages.

### etcd
- **Repository**: https://github.com/etcd-io/etcd
- **Date Verified**: 2025-09-19
- **Commit Hash**: `b7420c571ed13ae55cfd5b041d83210e005d0f78`
- **Test Command**: `go test -v ./...`
- **Test Results**: 11 tests passed, 1 skipped - identical with/without 3pio
- **Notes**: Distributed key-value store. When run from root with `./...`, most packages have no test files. Only 2 of 12 packages (contrib/raftexample and tools/etcd-dump-logs) contain actual tests. The full etcd test suite requires running tests from specific subdirectories with their own modules.

## Rust Libraries

### serde
- **Repository**: https://github.com/serde-rs/serde
- **Date Verified**: 2025-09-19
- **Commit Hash**: `eed3c7044d6f5ad957d1a8b17de16e983b1bc2ac`
- **Test Command**: `cargo test`
- **Test Results**: 478 tests passed, 1 skipped - identical with/without 3pio
- **Notes**: Rust serialization framework. The skipped test (compiletest::ui) requires nightly compiler. 3pio runs ~22% faster than baseline due to optimized JSON output parsing.

### clap
- **Repository**: https://github.com/clap-rs/clap
- **Date Verified**: 2025-09-19
- **Commit Hash**: `bc9bea5dc4c4f2dcaaa63ce6e5d5c9d801f3c39f`
- **Test Command**: `cargo test`
- **Test Results**: 911 tests (910 passed, 1 failed) - identical with/without 3pio
- **Notes**: Command-line argument parser for Rust. The single failing test is in the ui suite (trycmd tests), which is a pre-existing failure in the repository. 3pio correctly captures and reports all test results with exact match on pass/fail/skip counts and exit code (101).

### actix-web
- **Repository**: https://github.com/actix/actix-web
- **Date Verified**: 2025-09-19
- **Commit Hash**: `41d0176c895dcebdc7b67e0e039b8c0e2bb96bb5`
- **Test Command**: `cargo test`
- **Test Results**: 1251 tests passed, 11 skipped - off by 1 from baseline (1252 passed)
- **Notes**: Popular Rust web framework. Known issue: 3pio incorrectly merges integration tests with same filename from different crates (actix-http/tests/test_client.rs and awc/tests/test_client.rs both contain `with_query_parameter`). See cargo-crate-grouping-issue.md for details. All other test results match correctly.

### tokio
- **Repository**: https://github.com/tokio-rs/tokio
- **Date Verified**: 2025-09-19
- **Commit Hash**: `6d1ae6286880c828c13efb5f11b60c18fb94f947`
- **Test Command**: `cargo test`
- **Test Results**: 2566 tests (2471 passed, 95 skipped) - identical with/without 3pio
- **Notes**: Asynchronous runtime for Rust. Comprehensive test suite across 6 workspace crates including unit tests, integration tests, and extensive doctests. 3pio successfully tracked all tests across 248 test groups with negligible performance overhead.

## Python Libraries

### flask
- **Repository**: https://github.com/pallets/flask
- **Date Verified**: 2025-09-20
- **Commit Hash**: `adf363679da2d9a5ddc564bb2da563c7ca083916`
- **Test Command**: `uv run --group tests pytest tests/`
- **Test Results**: 490 tests passed - identical with/without 3pio
- **Notes**: Micro web framework for Python. Tests that change working directory during execution are correctly handled after fixing pytest adapter to use absolute IPC paths.

### httpie
- **Repository**: https://github.com/httpie/cli
- **Date Verified**: 2025-09-15
- **Commit Hash**: Not specified (latest main at time)
- **Test Command**: `pytest`
- **Test Results**: All tests passed - identical with/without 3pio
- **Notes**: Command-line HTTP client

### pandas
- **Repository**: https://github.com/pandas-dev/pandas
- **Date Verified**: 2025-09-15
- **Commit Hash**: Not specified (latest main at time)
- **Test Command**: `pytest` (subset of tests)
- **Test Results**: Tested subset passed - identical with/without 3pio
- **Notes**: Data analysis library, full suite takes very long

## Verification Criteria

For a library to be included in this list, it must meet the following criteria:

1. **Test Count Match**: The number of discovered and executed tests must be identical between native runner and 3pio
2. **Result Match**: Pass/fail/skip counts must match exactly
3. **Exit Code Match**: The process exit code must be identical
4. **Report Generation**: 3pio must successfully generate all expected reports
5. **No Test Disruption**: 3pio integration must not cause any additional test failures

Libraries that are partially supported or have known issues are documented separately in the project's issue tracker.
