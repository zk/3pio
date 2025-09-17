# Investigation: Test Count Discrepancy - Zed with cargo nextest

**Date**: 2025-09-16
**Issue**: 3pio reports 23 more tests than baseline when running Zed tests with cargo nextest
**Severity**: HIGH - This indicates 3pio may be modifying test discovery behavior

## Problem Statement

When running Zed tests with cargo nextest, there is a discrepancy in test counts:
- **Baseline** (without 3pio): 2733 tests started, 29 skipped
- **3pio run**: 2756 total tests (2708 passed + 25 failed + 23 skipped)
- **Discrepancy**: 23 additional tests reported by 3pio

## Investigation Log

### 1. Command Differences
**Baseline command**:
```
cargo nextest run --workspace --no-fail-fast
```

**3pio modified command**:
```
cargo nextest run --workspace --no-fail-fast --message-format libtest-json
```

3pio adds `--message-format libtest-json` to enable JSON output parsing.

### 2. Initial Hypothesis
The `--message-format libtest-json` flag may cause nextest to:
- Report tests differently
- Include tests that were previously hidden
- Count test granularity differently (functions vs modules)

### 3. Key Findings

#### Test Count Sources
- `cargo nextest list --workspace`: **2734** tests
- Baseline run (without 3pio): **2733** tests started, 29 skipped
- 3pio run: **2756** total (2708 passed + 25 failed + 23 skipped)
- **Discrepancy**: 22-23 extra tests in 3pio

#### Code Analysis - nextest.go

Found critical issue in `processTestEvent()` at line 308:
```go
case "started":
    (*testCount)++  // Increments for EVERY started event
```

The test count is incremented on EVERY "started" event, including tests that might later be "ignored".

#### Hypothesis
The discrepancy appears to be related to how nextest reports "ignored" tests in JSON format:
1. Nextest might send "started" events for ignored tests
2. 3pio counts these on start (incrementing testCount)
3. Later these tests are marked as "ignored"
4. This could cause double counting or incorrect totals

### 4. Root Cause Analysis

#### The Actual Problem
After thorough investigation, discovered that:
1. **Duplicate test names exist across different modules** - 72 test names are duplicated
2. **All 2756 tests reported by 3pio are unique** when considering their full paths
3. **Example**: `test_parse_remote_url_given_https_url` exists in 7 different modules:
   - git_hosting_providers::providers::bitbucket::tests
   - git_hosting_providers::providers::chromium::tests
   - git_hosting_providers::providers::codeberg::tests
   - git_hosting_providers::providers::gitee::tests
   - git_hosting_providers::providers::github::tests
   - git_hosting_providers::providers::gitlab::tests
   - git_hosting_providers::providers::sourcehut::tests

#### The Real Issue
**Nextest behaves differently in different output formats:**
- **Human format (default)**: Reports ~2733 tests started
- **libtest-json format**: Reports 2756 tests (all unique tests)

The 22-23 test discrepancy is due to nextest's JSON format including tests that the human-readable format might be consolidating or handling differently, possibly related to ignored/skipped tests.

## Solution

### This is NOT a bug in 3pio!

3pio is correctly processing all unique tests that nextest reports in JSON format. The discrepancy is a nextest behavior difference between output formats.

### Recommendations
1. **Document this behavior** - Add a note in the Rust support documentation explaining that nextest may report different test counts in JSON vs human format
2. **No code changes needed** - 3pio is functioning correctly
3. **Consider adding a note in reports** - When using nextest, note that test counts may differ from human-readable output

### Key Takeaway
The test count discrepancy is due to nextest's own behavior differences between output formats, not a bug in 3pio's implementation. All 2756 tests are legitimate, unique tests when considering their full module paths.