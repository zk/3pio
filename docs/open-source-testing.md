# Open Source Testing

We run 3pio against open source projects to ensure correct operation on a diverse set of real-world use cases. This document provides comprehensive guidance for testing 3pio against real-world projects across all supported test runners.

## Overview

General guidlines:

- Don't guess at the root cause of issues, always verify with evidence from process output or logs. The 3pio reports are a great place to look.

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


First, run the project's tests without 3pio. This will establish a baseline. Run in background and monitor process.

**CRITICAL REQUIREMENT: RUN ALL TESTS**

Open source testing MUST run the complete test suite, not subsets or samples. The goal is to validate 3pio against real-world complexity and scale. Running partial test suites defeats the purpose of comprehensive validation.

Next, run with 3pio. Run in background and monitor process.

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


### Step 5: Results Analysis and Documentation

Run the following analyses:

1. Compare stats from baseline run to 3pio run including:
- Duration
- Test cases passed / failed / skipped / etc
- Exit code
2. Analyze the 3pio test run output an look for issues.


#### Comparative Analysis
Compare with original test runner results:

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

Add an executive summary to the top of the report based on your findings.

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


## Common Issues and Troubleshooting

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


## Integration with Development Workflow

### Bug Investigation and Feature Development
Open source testing is primarily used for:
1. **Bug Reproduction**: When users report issues, test against their project setup
2. **Feature Development**: Test new features against real codebases to catch edge cases
3. **Performance Investigation**: Identify bottlenecks in realistic scenarios
4. **Compatibility Validation**: Verify support for new test runner versions or configurations
