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
- Jest, Vitest, Mocha, Cypress, pytest, go test, cargo test, cargo nextest

## Testing Workflow

### Step 1: Preparation

- Ensure 3pio is built `cd ~/code/3pio && make build`
- Verify binary `build/3pio --version`
- Create project clone directory if needed `mkdir -p /tmp/3pio-open-source`
- The user will indicate the project to test, clone it to the `tmp/3pio-open-source` directory.
  - Identify the test framework used
  - Identify test commands
  - Check if dependencies have been installed, fi not install dependencies.
- Create report directory and file in `~/code/3pio/noggin/reports/open-source/[project name]-[timestamp]/report.md`
  - Be terse
  - Create the following sections
    - Overview
      - Name of project and brief description
      - Date / time of test
      - Path to project
      - Github repository url
      - Test runner
      - Identified test commands
    - Dependency Setup
      - List what you did to install dependencies, if any.
    - Baseline Run
      - To be filled in later
    - 3pio Run
      - To be filled in later
    - Results
      - To be filled in later
- Run a baseline test
  - Run project tests with the identified test command
  - **CRITICAL REQUIREMENT: RUN ALL TESTS** Open source testing MUST run the complete test suite, not subsets or samples. The goal is to validate 3pio against real-world complexity and scale. Running partial test suites defeats the purpose of comprehensive validation.
  - Write console output to a file called `baseline.out` in report directory for later analysis
  - Run tests in background and check periodically. Some test suites take many minutes to complete.
  - Update Baseline Run section of report
    - Record pass / fail / skipped / notest, duration, and exit code.
- Note any anomalies
- Run 3pio test
  - Run project tests with identified test command prefixed with `3pio`. ex. `3pio npm test`.
  - Write console output to a file called `3pio.out` in report directory for later analysis
  - Run tests in background and check periodically. Some test suites take many minutes to complete.
  - Update 3pio Run section of report
    - Record pass / fail / skipped / notest, duration, and exit code
    - Record 3pio run output directory (`.3pio/runs/[runId])
    - Analyze 3pio run output directory and ensure:
      - test-run.md
        - test-run.md was created an is well formed
        - Sanity check yaml header
        - test-run.md reports COMPLETE status
        - Number of test groups reported seems right
        - Paths to individual report files are valid
      - individual report files
        - Sanity check yaml header
        - Paths to subgroup report files are valid
    - Check debug.log for errors specifically for this run (in `.3pio` directory)
      - In general, look for contradicting or obviously wrong information
- Fill in Results section
  - Summarize and compare baseline and 3pio runs
  - Note any major anomalies or errors found in analysis.
  - Compare number of tests and type between baseline and 3pio runs
  - Compare exit codes
  - Compare performance (durations) of baseline and 3pio runs.
