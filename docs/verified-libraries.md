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
- **Date Verified**: 2025-09-15
- **Commit Hash**: Not specified (latest main at time)
- **Test Command**: `go test`
- **Test Results**: 45 tests passed, 1 skipped - identical with/without 3pio
- **Notes**: Simple library demonstrating Go test support

### gin
- **Repository**: https://github.com/gin-gonic/gin
- **Date Verified**: 2025-09-15
- **Commit Hash**: Not specified (latest main at time)
- **Test Command**: `go test ./...`
- **Test Results**: 5 packages passed, 0 failed, 2 skipped - identical with/without 3pio
- **Notes**: Popular Go web framework, demonstrates multi-package support

### echo
- **Repository**: https://github.com/labstack/echo
- **Date Verified**: 2025-09-15
- **Commit Hash**: Not specified (latest main at time)
- **Test Command**: `go test ./...`
- **Test Results**: All tests passed - identical with/without 3pio
- **Notes**: Minimalist Go web framework

### etcd
- **Repository**: https://github.com/etcd-io/etcd
- **Date Verified**: 2025-09-16
- **Commit Hash**: `ad9d69071936b5771830add74a799f2e822e2ffc`
- **Test Command**: `go test ./client/pkg/...`
- **Test Results**: Multiple packages tested successfully - identical with/without 3pio
- **Notes**: Distributed key-value store, complex Go project

## Rust Libraries

### zed (util package)
- **Repository**: https://github.com/zed-industries/zed
- **Date Verified**: 2025-09-16
- **Commit Hash**: Latest main branch (cloned 2025-09-16)
- **Test Command**: `cargo test --lib --bins --package util`
- **Test Results**: 37 tests passed - identical with/without 3pio
- **Notes**: Code editor, tested specific package to avoid GPU dependencies

### serde
- **Repository**: https://github.com/serde-rs/serde
- **Date Verified**: 2025-09-15
- **Commit Hash**: Not specified (latest main at time)
- **Test Command**: `cargo test`
- **Test Results**: All tests passed - identical with/without 3pio
- **Notes**: Popular Rust serialization framework

### clap
- **Repository**: https://github.com/clap-rs/clap
- **Date Verified**: 2025-09-15
- **Commit Hash**: Not specified (latest main at time)
- **Test Command**: `cargo test`
- **Test Results**: All tests passed - identical with/without 3pio
- **Notes**: Command-line argument parser for Rust

### actix-web
- **Repository**: https://github.com/actix/actix-web
- **Date Verified**: 2025-09-15
- **Commit Hash**: Not specified (latest main at time)
- **Test Command**: `cargo test`
- **Test Results**: All tests passed - identical with/without 3pio
- **Notes**: Popular Rust web framework

### tokio
- **Repository**: https://github.com/tokio-rs/tokio
- **Date Verified**: 2025-09-15
- **Commit Hash**: Not specified (latest main at time)
- **Test Command**: `cargo test`
- **Test Results**: All tests passed - identical with/without 3pio
- **Notes**: Asynchronous runtime for Rust

## Python Libraries

### flask
- **Repository**: https://github.com/pallets/flask
- **Date Verified**: 2025-09-15
- **Commit Hash**: Not specified (latest main at time)
- **Test Command**: `pytest`
- **Test Results**: All tests passed - identical with/without 3pio
- **Notes**: Micro web framework for Python

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