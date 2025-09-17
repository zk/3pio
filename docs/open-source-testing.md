# Open Source Testing

We run 3pio against open source projects to ensure correct operation on a diverse set of real-world use cases. This document provides comprehensive guidance for testing 3pio against real-world projects across all supported test runners.

## Overview

Open source testing serves as a critical debugging and quality assurance tool:
- **Bug Discovery**: Identifies issues that don't surface in controlled test fixtures
- **Edge Case Detection**: Uncovers problems with unusual project configurations
- **Real-World Stress Testing**: Tests handling of large, complex codebases
- **Compatibility Issues**: Reveals incompatibilities with specific frameworks or versions
- **Performance Problems**: Identifies bottlenecks under realistic workloads

## Supported Test Runners and Projects

### JavaScript/TypeScript (Jest/Vitest)

#### Tier 1: Primary Test Projects
- **React**: ~2000 tests, excellent Jest coverage
- **Vue.js**: ~3000 tests, comprehensive Vitest usage
- **Next.js**: ~1000 tests, modern Jest configuration
- **Vite**: ~800 tests, extensive Vitest self-testing

#### Tier 2: Secondary Projects
- **Express.js**: Classic Node.js testing patterns
- **Lodash**: Utility library with comprehensive test coverage
- **Styled-components**: Component testing patterns

### Python (pytest)

#### Tier 1: Primary Test Projects
- **FastAPI**: ~1000 tests, modern async testing
- **Django**: ~10000 tests, comprehensive framework testing
- **Requests**: ~500 tests, HTTP library testing patterns
- **Click**: ~300 tests, CLI testing patterns

#### Tier 2: Secondary Projects
- **Flask**: Web framework testing
- **Pandas**: Data processing library tests
- **NumPy**: Scientific computing tests

### Go (go test)

#### Tier 1: Primary Test Projects
- **Kubernetes**: ~5000 tests, large enterprise patterns
- **Docker**: ~2000 tests, system-level testing
- **Hugo**: ~800 tests, static site generator
- **Cobra**: ~200 tests, CLI library testing

#### Tier 2: Secondary Projects
- **Gin**: Web framework testing
- **GORM**: ORM testing patterns
- **Viper**: Configuration library tests

### Rust (cargo test/nextest)

#### Tier 1: Primary Test Projects
- **uv**: Python package manager, fast compilation, excellent test structure
- **Alacritty**: Terminal emulator, 132 tests, good variety
- **Sway**: Smart contract language, workspace testing

#### Tier 2: Secondary Projects
- **Zed**: Editor (large project, slow builds)
- **Deno**: Runtime (very large, requires patience)
- **Tauri**: App framework (complex build process)

## Testing Workflow

### Step 1: Preparation

#### Environment Setup
```bash
# Ensure 3pio is built
cd ~/code/3pio
make build

# Verify binary
./build/3pio --version

# Create testing directory
mkdir -p /tmp/3pio-open-source
cd /tmp/3pio-open-source
```

#### Project Assessment
Before cloning, research the project:
1. **Project size**: Check repository size and complexity
2. **Test runner**: Identify which test framework is used
3. **Dependencies**: Note any special requirements (Python version, Node version, Rust toolchain)
4. **Build time**: Estimate compilation/setup time for planning

### Step 2: Project Setup

#### Clone and Initialize
```bash
# Check if already cloned
if [ -d "/tmp/3pio-open-source/[project-name]" ]; then
    echo "Project already exists, pulling latest changes"
    cd [project-name]
    git pull
else
    echo "Cloning project"
    git clone https://github.com/[org]/[project-name].git
    cd [project-name]
fi
```

#### Dependency Installation

**JavaScript/Node.js Projects:**
```bash
# Check package manager
if [ -f "pnpm-lock.yaml" ]; then
    pnpm install
elif [ -f "yarn.lock" ]; then
    yarn install
else
    npm install
fi
```

**Python Projects:**
```bash
# Prefer uv for speed, fallback to venv
if command -v uv >/dev/null 2>&1; then
    uv venv
    source .venv/bin/activate
    uv pip install -e ".[dev]"  # or requirements-dev.txt
else
    python -m venv .venv
    source .venv/bin/activate
    pip install -e ".[dev]"
fi
```

**Go Projects:**
```bash
# Go modules handle dependencies automatically
go mod download
```

**Rust Projects:**
```bash
# Cargo handles dependencies, but may need toolchain
# Check rust-toolchain.toml or .rust-version
cargo check  # Verify compilation works
```

### Step 3: Test Discovery and Analysis

#### Identify Test Commands
Examine the project to find the correct test commands:

**Package.json (JavaScript/TypeScript):**
```bash
# Look for test scripts
cat package.json | jq '.scripts'

# Common patterns:
# "test": "jest"
# "test": "vitest run"
# "test:unit": "jest src/"
# "test:integration": "jest integration/"
```

**Makefile/Justfile (All languages):**
```bash
# Check for test targets
grep -E "^test|^check" Makefile
grep -E "^test|^check" justfile
```

**CI Configuration:**
```bash
# Check GitHub Actions, GitLab CI, etc.
cat .github/workflows/*.yml | grep -A5 -B5 "test"
```

#### Test Structure Analysis
Document the project's test organization:
- **Test directories**: `tests/`, `test/`, `__tests__/`, `src/**/*.test.js`
- **Test patterns**: Unit vs integration vs e2e
- **Parallel execution**: Check for `--maxWorkers`, `--parallel`, `-j` flags
- **Coverage tools**: Note if coverage is enabled (may conflict with 3pio)

### Step 4: Running Tests with 3pio

#### Create Test Report
```bash
# Create report file
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
REPORT_FILE="noggin/reports/open-source/${PROJECT_NAME}-${TIMESTAMP}.md"
mkdir -p "$(dirname "$REPORT_FILE")"

# Initialize report
cat > "$REPORT_FILE" << EOF
# 3pio Open Source Test Report: ${PROJECT_NAME}

**Date**: $(date)
**Project**: ${PROJECT_NAME}
**Repository**: $(git remote get-url origin)
**Commit**: $(git rev-parse HEAD)
**3pio Version**: $(~/code/3pio/build/3pio --version)

## Test Execution Summary

| Metric | Value |
|--------|-------|
| Test Runner | TBD |
| Total Tests | TBD |
| Passed | TBD |
| Failed | TBD |
| Duration | TBD |
| Report Path | TBD |

## Test Commands Executed

EOF
```

#### Execute Test Runs

**CRITICAL REQUIREMENT: RUN ALL TESTS**

Open source testing MUST run the complete test suite, not subsets or samples. The goal is to validate 3pio against real-world complexity and scale. Running partial test suites defeats the purpose of comprehensive validation.

**Standard Test Execution:**
```bash
# Run ALL tests - use the project's main test command that runs the complete suite
~/code/3pio/build/3pio npm test

# For projects with multiple test commands, run ALL of them
~/code/3pio/build/3pio npm run test:unit
~/code/3pio/build/3pio npm run test:integration
~/code/3pio/build/3pio npm run test:e2e

# Document results
echo "### Complete Test Suite Run" >> "$REPORT_FILE"
echo "\`\`\`bash" >> "$REPORT_FILE"
echo "~/code/3pio/build/3pio npm test" >> "$REPORT_FILE"
echo "\`\`\`" >> "$REPORT_FILE"

# Extract results
if [ -f ".3pio/runs/*/test-run.md" ]; then
    echo "**Results**: $(grep -E 'Total test cases:|Tests cases passed:|Test cases failed:' .3pio/runs/*/test-run.md)" >> "$REPORT_FILE"
fi
```

**Multiple Test Targets:**
```bash
# Test different configurations
for TEST_CMD in "npm run test:unit" "npm run test:integration" "npm run test:e2e"; do
    if npm run | grep -q "${TEST_CMD#npm run }"; then
        echo "### Running: $TEST_CMD" >> "$REPORT_FILE"
        ~/code/3pio/build/3pio $TEST_CMD
        # Document results for each run
    fi
done
```

**Language-Specific Examples:**

*JavaScript/TypeScript:*
```bash
# Jest projects
~/code/3pio/build/3pio npx jest
~/code/3pio/build/3pio npx jest --testPathPattern=unit

# Vitest projects
~/code/3pio/build/3pio npx vitest run
~/code/3pio/build/3pio npx vitest run src/

# Package scripts
~/code/3pio/build/3pio npm test
~/code/3pio/build/3pio npm run test:unit
```

*Python:*
```bash
# pytest projects
~/code/3pio/build/3pio pytest
~/code/3pio/build/3pio pytest tests/
~/code/3pio/build/3pio python -m pytest tests/unit/

# Specific modules
~/code/3pio/build/3pio pytest tests/test_core.py
```

*Go:*
```bash
# Run ALL tests - always use ./... for complete coverage
~/code/3pio/build/3pio go test ./...

# For projects with long-running tests, run complete suite without -short flag
# Only use -short as a fallback if full tests are impractical due to infrastructure requirements
~/code/3pio/build/3pio go test ./...

# AVOID package-specific testing unless documenting why full suite cannot run
```

*Rust:*
```bash
# Run ALL tests - complete workspace coverage
~/code/3pio/build/3pio cargo test --workspace

# If nextest is available, run complete suite with nextest
if command -v cargo-nextest >/dev/null 2>&1; then
    ~/code/3pio/build/3pio cargo nextest run --workspace
fi

# AVOID --lib or package-specific flags unless documenting why full suite cannot run
```

### Step 5: Results Analysis and Documentation

#### Performance Analysis
Extract and analyze performance data:

```bash
# Extract timing data
LATEST_RUN=$(ls -t .3pio/runs/ | head -n1)
DURATION=$(grep "Total duration:" ".3pio/runs/$LATEST_RUN/test-run.md" | cut -d: -f2)
TEST_COUNT=$(grep "Total test cases:" ".3pio/runs/$LATEST_RUN/test-run.md" | cut -d: -f2)

echo "## Performance Analysis" >> "$REPORT_FILE"
echo "- Total Duration: $DURATION" >> "$REPORT_FILE"
echo "- Test Count: $TEST_COUNT" >> "$REPORT_FILE"
echo "- Average per test: $(echo "scale=3; $DURATION / $TEST_COUNT" | bc)s" >> "$REPORT_FILE"

# Check for performance issues
if [ "$DURATION" -gt 300 ]; then
    echo "- ⚠️ **Performance Issue**: Test run took longer than 5 minutes" >> "$REPORT_FILE"
fi
```

#### Issue Detection and Documentation
Systematically check for problems:

```bash
echo "## Issues Found" >> "$REPORT_FILE"

# Check for 3pio-specific errors
if grep -q "Error:" .3pio/runs/*/output.log; then
    echo "### ❌ Error Messages Detected" >> "$REPORT_FILE"
    echo "\`\`\`" >> "$REPORT_FILE"
    grep "Error:" .3pio/runs/*/output.log >> "$REPORT_FILE"
    echo "\`\`\`" >> "$REPORT_FILE"
fi

# Check for adapter issues
if grep -q "adapter" .3pio/debug.log; then
    echo "### 🔧 Adapter Issues" >> "$REPORT_FILE"
    echo "\`\`\`" >> "$REPORT_FILE"
    grep -i "adapter" .3pio/debug.log | tail -10 >> "$REPORT_FILE"
    echo "\`\`\`" >> "$REPORT_FILE"
fi

# Check for IPC communication problems
IPC_FILE_COUNT=$(ls .3pio/ipc/*.jsonl 2>/dev/null | wc -l)
if [ "$IPC_FILE_COUNT" -eq 0 ]; then
    echo "### 📡 IPC Communication Failure" >> "$REPORT_FILE"
    echo "- No IPC files generated - adapter communication failed" >> "$REPORT_FILE"
fi

# Check for missing test discovery
DISCOVERED_FILES=$(grep -c "testGroupStart" .3pio/ipc/*.jsonl 2>/dev/null || echo "0")
if [ "$DISCOVERED_FILES" -eq 0 ]; then
    echo "### 🔍 Test Discovery Issues" >> "$REPORT_FILE"
    echo "- No test groups discovered - check test runner compatibility" >> "$REPORT_FILE"
fi

# Check for incomplete results
COMPLETED_TESTS=$(grep -c "testCase" .3pio/ipc/*.jsonl 2>/dev/null || echo "0")
EXPECTED_TESTS=$(grep "Total test cases:" ".3pio/runs/$LATEST_RUN/test-run.md" | cut -d: -f2 | tr -d ' ')
if [ "$COMPLETED_TESTS" -ne "$EXPECTED_TESTS" ]; then
    echo "### ⚠️ Incomplete Test Results" >> "$REPORT_FILE"
    echo "- Expected: $EXPECTED_TESTS tests, Got: $COMPLETED_TESTS results" >> "$REPORT_FILE"
fi

# Check for coverage mode interference
if grep -q "coverage" .3pio/runs/*/output.log; then
    echo "### 📊 Coverage Mode Detected" >> "$REPORT_FILE"
    echo "- Coverage mode may interfere with 3pio - consider running without coverage" >> "$REPORT_FILE"
fi

# Check for permission issues
if grep -q "permission denied\|EACCES" .3pio/runs/*/output.log; then
    echo "### 🔒 Permission Issues" >> "$REPORT_FILE"
    echo "- File permission errors detected - check write access to project directory" >> "$REPORT_FILE"
fi
```

#### Comparative Analysis
Compare with original test runner results:

```bash
echo "## Comparative Analysis" >> "$REPORT_FILE"

# Run original test command for comparison
echo "Running original test command for comparison..." >> "$REPORT_FILE"
ORIGINAL_START=$(date +%s)
npm test > original_test_output.log 2>&1
ORIGINAL_EXIT_CODE=$?
ORIGINAL_END=$(date +%s)
ORIGINAL_DURATION=$((ORIGINAL_END - ORIGINAL_START))

echo "### Original Test Runner Results" >> "$REPORT_FILE"
echo "- Exit Code: $ORIGINAL_EXIT_CODE" >> "$REPORT_FILE"
echo "- Duration: ${ORIGINAL_DURATION}s" >> "$REPORT_FILE"

# Extract test counts from original output
ORIGINAL_TESTS=$(grep -E "Tests:|passed|failed" original_test_output.log | head -5)
echo "- Results: $ORIGINAL_TESTS" >> "$REPORT_FILE"

# Compare exit codes
THREEPIO_EXIT_CODE=$(echo $?)  # From previous 3pio run
if [ "$ORIGINAL_EXIT_CODE" -ne "$THREEPIO_EXIT_CODE" ]; then
    echo "### ⚠️ Exit Code Mismatch" >> "$REPORT_FILE"
    echo "- Original: $ORIGINAL_EXIT_CODE, 3pio: $THREEPIO_EXIT_CODE" >> "$REPORT_FILE"
fi

# Compare durations
DURATION_DIFF=$(($(echo "$DURATION" | cut -d. -f1) - ORIGINAL_DURATION))
if [ "$DURATION_DIFF" -gt 60 ]; then
    echo "### ⏱️ Performance Impact" >> "$REPORT_FILE"
    echo "- 3pio added ${DURATION_DIFF}s overhead (may indicate performance issue)" >> "$REPORT_FILE"
fi
```

#### Success Validation
Verify 3pio core functionality:

```bash
echo "## Validation Results" >> "$REPORT_FILE"

# Check report generation
if [ -f ".3pio/runs/$LATEST_RUN/test-run.md" ]; then
    echo "- ✅ Main report generated successfully" >> "$REPORT_FILE"
else
    echo "- ❌ **CRITICAL**: Main report missing" >> "$REPORT_FILE"
fi

# Check individual reports
REPORT_COUNT=$(find .3pio/runs/$LATEST_RUN/reports -name "index.md" 2>/dev/null | wc -l)
if [ "$REPORT_COUNT" -gt 0 ]; then
    echo "- ✅ Individual reports: $REPORT_COUNT files generated" >> "$REPORT_FILE"
else
    echo "- ❌ **ISSUE**: No individual test reports generated" >> "$REPORT_FILE"
fi

# Check IPC communication
if [ -f .3pio/ipc/*.jsonl ]; then
    IPC_EVENTS=$(wc -l .3pio/ipc/*.jsonl | tail -n1 | awk '{print $1}')
    echo "- ✅ IPC events: $IPC_EVENTS recorded" >> "$REPORT_FILE"

    # Validate IPC event types
    EVENT_TYPES=$(grep -o '"eventType":"[^"]*"' .3pio/ipc/*.jsonl | sort | uniq -c)
    echo "- Event breakdown: $EVENT_TYPES" >> "$REPORT_FILE"
else
    echo "- ❌ **CRITICAL**: No IPC communication detected" >> "$REPORT_FILE"
fi

# Check debug logs for warnings
WARNING_COUNT=$(grep -c "WARN\|ERROR" .3pio/debug.log 2>/dev/null || echo "0")
if [ "$WARNING_COUNT" -gt 0 ]; then
    echo "- ⚠️ Debug log contains $WARNING_COUNT warnings/errors" >> "$REPORT_FILE"
fi

# Final assessment
echo "" >> "$REPORT_FILE"
if [ -f ".3pio/runs/$LATEST_RUN/test-run.md" ] && [ "$REPORT_COUNT" -gt 0 ] && [ -f .3pio/ipc/*.jsonl ]; then
    echo "**Overall Status**: ✅ 3pio functioned correctly with this project" >> "$REPORT_FILE"
else
    echo "**Overall Status**: ❌ 3pio encountered significant issues - requires investigation" >> "$REPORT_FILE"
fi
```

### Step 6: Report Update and Maintenance

#### Report Organization
Reports should be stored in the project under `noggin/reports/open-source/`:

```
noggin/reports/open-source/
├── react-20250916-1430.md
├── vue-20250916-1445.md
├── fastapi-20250916-1500.md
├── kubernetes-20250916-1530.md
└── alacritty-20250916-1600.md
```

#### Update Existing Reports
When re-testing a project:

```bash
# Check for existing reports
EXISTING=$(ls noggin/reports/open-source/${PROJECT_NAME}-*.md 2>/dev/null | tail -n1)

if [ -n "$EXISTING" ]; then
    echo "## Update $(date)" >> "$EXISTING"
    echo "Previous test results remain below for comparison." >> "$EXISTING"
    echo "" >> "$EXISTING"
    # Append new results
else
    # Create new report as above
fi
```

#### Regression Testing
For ongoing validation:

```bash
# Create regression test script
cat > "test-${PROJECT_NAME}.sh" << 'EOF'
#!/bin/bash
set -e

cd "/tmp/3pio-open-source/${PROJECT_NAME}"
git pull

# Run 3pio test
~/code/3pio/build/3pio npm test

# Check for regressions
LATEST_RUN=$(ls -t .3pio/runs/ | head -n1)
FAILED=$(grep "Test cases failed:" ".3pio/runs/$LATEST_RUN/test-run.md" | cut -d: -f2 | tr -d ' ')

if [ "$FAILED" != "0" ]; then
    echo "❌ Regression detected: $FAILED tests failed"
    exit 1
else
    echo "✅ All tests passed"
fi
EOF

chmod +x "test-${PROJECT_NAME}.sh"
```

## Common Issues and Troubleshooting

### Coverage Mode Conflicts
**Symptom**: 0 test files tracked despite tests running
**Solution**: Disable coverage in test commands
```bash
# Instead of
~/code/3pio/build/3pio npm run test:coverage

# Use
~/code/3pio/build/3pio npm test
```

### Large Test Suites
**Symptom**: Performance degradation or timeouts
**REQUIRED APPROACH**: Run the complete test suite anyway - this is the point of open source testing
**Solutions for genuine issues**:
- Increase timeout limits and be patient
- Monitor memory usage and system resources
- Document performance issues found
- Only use test filtering if the complete suite is literally impossible to run (document why)

### Permission Issues
**Symptom**: Cannot create `.3pio` directory
**Solution**: Ensure write permissions in project directory
```bash
chmod 755 .
# Or run in writable directory
```

### Dependency Installation Failures
**Symptom**: Project won't build or install
**Solutions**:
- Check Node.js/Python/Go/Rust version requirements
- Use appropriate package manager (npm/yarn/pnpm, pip/uv, go mod, cargo)
- Check for system dependencies (native libraries, etc.)

### Test Runner Not Detected
**Symptom**: 3pio doesn't recognize the test framework
**Solutions**:
- Check project's test scripts in package.json
- Look for alternative test commands in Makefile/CI
- Verify test runner is actually installed

## Quality Standards

### Required Documentation
Each test report must include:
- **Project metadata**: Name, repository, commit hash, test date
- **Environment details**: 3pio version, dependency versions
- **Test execution**: Commands run, duration, results
- **Issue tracking**: Any problems encountered
- **Validation**: Confirmation that 3pio worked correctly

### Success Criteria
A successful open source test should demonstrate:
- **No crashes or hangs**: 3pio completes execution without fatal errors
- **Basic functionality**: Tests are discovered and results are captured
- **Issue identification**: Any problems are clearly documented for investigation
- **Comparative analysis**: Results can be compared with original test runner output
- **Debugging information**: Sufficient logs and data to investigate any issues found

### Failure Investigation
When issues occur:
1. **Isolate the problem**: Test with original runner first
2. **Check debug logs**: Review `.3pio/debug.log` for details
3. **Verify IPC communication**: Check `.3pio/ipc/*.jsonl` for events
4. **Compare output**: Diff original vs 3pio captured output
5. **Document findings**: Record issues for future reference

## Integration with Development Workflow

### Bug Investigation and Feature Development
Open source testing is primarily used for:
1. **Bug Reproduction**: When users report issues, test against their project setup
2. **Feature Development**: Test new features against real codebases to catch edge cases
3. **Performance Investigation**: Identify bottlenecks in realistic scenarios
4. **Compatibility Validation**: Verify support for new test runner versions or configurations

## Maintenance and Updates

### Project Maintenance
- **Keep projects current**: Update tested projects when investigating new issues
- **Document findings**: Maintain records of bugs discovered and their resolutions
- **Track compatibility**: Note which project versions work or have issues
- **Archive investigation results**: Keep reports for future reference and debugging

### Documentation Updates
- **Bug patterns**: Document recurring issues and their root causes
- **Workarounds**: Maintain list of known problems and temporary solutions
- **Investigation techniques**: Record effective debugging approaches
- **Issue tracking**: Link open source test results to bug reports and fixes

This comprehensive approach ensures 3pio maintains high quality and compatibility across the diverse ecosystem of testing frameworks and project structures it supports.
