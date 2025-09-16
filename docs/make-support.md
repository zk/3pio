# Make Support for 3pio

## Overview

Add support for discovering test commands through Makefile targets (e.g., `3pio make test`) by parsing the Makefile, finding the target, and extracting the test command to run with 3pio's enhancements. Make is treated as a command discovery mechanism (like package.json), not as an execution mechanism.

## Design Philosophy

Make support in 3pio follows a **command extraction** approach:
1. Parse Makefiles to discover test commands
2. Extract and run the actual test command directly
3. Provide clear feedback when extraction isn't possible
4. Never actually execute `make` - only use it for discovery
5. Always warn users that upstream tasks are skipped

## Core Approach

**Parse → Find → Extract → Transform → Execute**

1. **Parse**: Read and parse the Makefile
2. **Find**: Locate the requested target (e.g., `test`)
3. **Extract**: Extract the test command from the target's recipe
4. **Transform**: Modify the command to include 3pio adapters
5. **Execute**: Run the transformed command with 3pio orchestration

## Architecture Overview

```
User Command: 3pio make test
                ↓
        [Makefile Parser]
                ↓
        [Find Target]
                ↓
    [Can Extract Command?]
         ↙              ↘
      [Yes]            [No]
        ↓                ↓
[Extract Command]   [Error: Cannot
        ↓            extract test
[Identify Runner]     command]
        ↓
[Use Appropriate
  Definition]
        ↓
[Execute with
  Adapters]
        ↓
    [3pio Reports]
```

## Implementation Strategy

### 1. Makefile Parser

The parser reads Makefiles and extracts:
- Target names and their recipes
- Dependencies between targets
- Simple variable substitutions
- Command prefixes (@ for silent, - for error ignoring)

### 2. Command Extraction

The parser will identify test commands by looking for known test runners:

**JavaScript/Node.js patterns:**
- `npm test`, `npm run test`
- `yarn test`, `yarn run test`
- `pnpm test`, `pnpm run test`
- `npx jest`, `npx vitest`
- `node test.js`

**Python patterns:**
- `pytest`, `python -m pytest`, `py.test`
- `python -m unittest`
- `nose2`, `python -m nose`

**Go patterns:**
- `go test`

**Other patterns:**
- `cargo test` (Rust)
- `dotnet test` (.NET)
- `mvn test` (Java/Maven)

### 3. Extraction Algorithm

The extraction process:
1. Strip Make-specific prefixes (@ and -)
2. Identify which command is the test runner
3. Return the clean test command
4. Error if no test command found or extraction not possible

### 4. Extraction Decision Tree

#### Extractable (Supported)
```makefile
test:
    npm test

test-python:
    @pytest tests/ -v
```
→ Extract and run with 3pio

#### Extractable with Additional Warnings
```makefile
test:
    @echo "Running tests..."
    @npm test
    @echo "Done"
```
→ Extract `npm test`, warn about ignored echo commands in addition to the standard warning

#### Not Extractable (Not Supported)
```makefile
test: build
    npm test

test-all:
    $(MAKE) test-unit
    $(MAKE) test-integration
```
→ Error: "Cannot extract test command from Makefile target with dependencies. Run the test command directly: `3pio npm test`"

## Component Design

### 1. Make Command Extraction

Make is handled as a pre-processing step, not a runner definition:

```pseudo
If command starts with "make":
    Parse Makefile
    Find target (e.g., "test")
    Extract actual test command
    Return extracted command for normal processing
Else:
    Continue with normal runner detection
```

Usage example:
```bash
# User runs:
3pio make test

# 3pio internally:
# 1. Parses Makefile, finds "test:" target
# 2. Extracts "npm test" from the recipe
# 3. Processes as if user ran: 3pio npm test
```

### 2. Makefile Parser

The parser component handles:
- Reading and tokenizing Makefile syntax
- Building a map of targets and their recipes
- Tracking dependencies between targets
- Simple variable resolution (e.g., `$(TEST_CMD)`)
- Identifying command prefixes (@ for silent, - for ignore errors)

### 3. Extraction Analyzer

Determines if a test command can be extracted by checking for blockers:
- Dependencies on other targets (e.g., `test: build`)
- Recursive make calls (e.g., `$(MAKE) test-unit`)
- Complex shell constructs (pipes, loops, conditionals)
- No identifiable test command

Returns either success or a clear error message explaining why extraction failed.

### 4. Command Extractor

Extracts the actual test command from a Makefile target:
- For single-command targets: direct extraction
- For multi-command targets: finds the test command among echo/setup commands
- Strips Make-specific prefixes (@, -)
- Returns clean command ready for 3pio processing

Example extraction:
```makefile
# Input Makefile target:
test:
    @echo "Running tests..."
    npm test
    @echo "Done"

# Extracted command: npm test
```

## Challenges

### 1. Makefile Syntax
- Comments (`#`) and line continuations (`\`)
- Silent commands (`@` prefix)
- Error-ignoring commands (`-` prefix)
- Variable references (`$(VAR)` and `${VAR}`)
- Automatic variables (`$@`, `$<`, etc.)

### 2. Test Command Extraction
- Test commands may be one of several commands in a recipe
- Need to identify which command is the actual test runner
- Some targets may have only echo/setup commands before the test

### 3. Complexity Boundaries
- Recursive make calls (`$(MAKE)` or `make`)
- Target dependencies that must run first
- Complex shell constructs (pipes, conditionals)
- Multiple test commands in one target

## Integration Points

### 1. Runner Manager Integration

Make command extraction happens before runner detection:
```pseudo
1. Check if command starts with "make"
2. If yes, extract actual command from Makefile
3. Pass extracted command to normal runner detection
4. Continue with standard 3pio flow
```

### 2. Command Detection

Common test target patterns to recognize:
- `test`, `tests`, `check`
- `test-unit`, `test-integration`, `test-all`
- `unittest`, `unit-test`, `integration-test`
- Language-specific: `test-go`, `test-python`, `test-js`

## Execution Flow

### Simple Case Flow

1. User runs: `3pio make test`
2. Show warning:
   ```
   Warning: `make` support is experimental in 3pio.
   We'll attempt to extract and run your test command directly,
   but this skips any upstream tasks defined in your Makefile.
   ```
3. Parser reads Makefile
4. Analyzer determines extraction is possible
5. Extractor gets: `npm test`
6. Transform to: `npm test -- --reporters /path/to/adapter`
7. Execute with 3pio orchestrator
8. Generate full 3pio reports

### Non-Extractable Case Flow

1. User runs: `3pio make test`
2. Show warning (same as above)
3. Parser reads Makefile
4. Analyzer finds blockers (dependencies, recursive make, etc.)
5. Show error:
   ```
   Error: Cannot extract test command from Makefile.
   Reason: Target 'test' has dependencies that must run first.

   To use 3pio, run the test command directly:
     3pio npm test
   ```
5. Exit with error code

## Implementation Steps

### Step 1: Create Makefile Parser
- Build basic Makefile parser in `internal/makefile/`
- Support target extraction with recipes
- Handle @ and - prefixes
- Parse dependencies

### Step 2: Add Make Command Extractor
- Create `internal/makefile/extractor.go`
- Implement pre-processing step for make commands
- Return extracted command for normal runner detection

### Step 3: Integrate with Command Processing
- Add make extraction as pre-processing step
- Ensure extraction happens before runner detection

### Step 4: Test Command Recognition
- Build pattern matcher for test commands
- Support all common test runners
- Handle variations (npm test, npm run test, etc.)

### Step 5: Error Handling
- Clear messages for unsupported patterns
- Suggest direct command usage when extraction fails
- Debug logging for troubleshooting

## File Structure

```
internal/
├── makefile/
│   ├── parser.go           # Makefile parsing logic
│   ├── extractor.go        # Command extraction logic
│   ├── analyzer.go         # Extraction feasibility analyzer
│   └── patterns.go         # Test command patterns
```

## Test Fixtures

```
tests/fixtures/
├── make-simple/
│   ├── Makefile           # Simple test target
│   └── tests/
├── make-npm/
│   ├── Makefile           # NPM-based tests
│   └── package.json
├── make-multi/
│   ├── Makefile           # Multiple test targets
│   └── tests/
└── make-complex/
    ├── Makefile           # Complex with dependencies
    └── tests/
```

## Example Makefiles to Support

### Simple NPM
```makefile
test:
	npm test

test-watch:
	npm test -- --watch
```

### Go Project
```makefile
test:
	go test -v ./...

# Note: Coverage targets would not work with 3pio as coverage mode is unsupported
# test-coverage:
# 	go test -coverprofile=coverage.out ./...
```

### Python Project
```makefile
test:
	pytest tests/ -v

test-unit:
	pytest tests/unit -v

test-integration:
	pytest tests/integration -v
```

### Complex Example
```makefile
.PHONY: test test-unit test-integration

test: test-unit test-integration

test-unit:
	@echo "Running unit tests..."
	npm run test:unit

test-integration: start-services
	@echo "Running integration tests..."
	npm run test:integration
	@$(MAKE) stop-services
```

## Error Handling

### Common Error Messages

**No Makefile:**
```
Error: No Makefile found in current directory
```

**Target not found:**
```
Error: Target 'test' not found in Makefile
Available targets: build, clean, install
```

**Cannot extract:**
```
Error: Cannot extract test command from target 'test'
Reason: Target has dependencies that must run first

To use 3pio, run the test command directly:
  3pio npm test
```

**Complex syntax:**
```
Error: Makefile too complex to parse
Reason: Conditional statements not supported

Run your test command directly instead.
```

## Configuration

### Future: .3pio.yaml Configuration
```yaml
make:
  # Force extraction even for complex targets
  force_extraction: false

  # Custom test target names
  test_targets:
    - test
    - check
    - validate

  # Complexity threshold
  max_complexity: medium

  # Variable substitutions
  variables:
    TEST_RUNNER: "npm test"
```

## Testing Strategy

### Unit Tests
- Parse simple Makefiles
- Parse complex Makefiles
- Extract test commands
- Analyze extraction feasibility
- Handle edge cases

### Integration Tests
- Simple NPM test extraction
- Python test extraction
- Go test extraction
- Error on dependencies
- Error on recursive make
- Error on missing Makefile
- Error on missing target

## Success Criteria

1. **Basic Support**: Extract test commands from simple Makefile patterns
2. **Clear Errors**: Explicit messages when extraction isn't possible
3. **No Breaking Changes**: Existing 3pio functionality unchanged
4. **Performance**: Minimal overhead for Makefile parsing
5. **User Guidance**: Always suggest how to proceed when extraction fails

## Edge Cases to Consider

1. **Recursive Make**: `$(MAKE) -C subdir test`
2. **Variable Expansion**: `$(TEST_CMD)` where TEST_CMD defined elsewhere
3. **Conditionals**: `ifdef` / `ifndef` blocks
4. **Multiple Commands**: Multiple test commands in one target
5. **Shell Features**: Pipes, redirects, command substitution
6. **Include Directives**: Makefile includes other files
7. **Pattern Rules**: `%.test: %.c` style rules
8. **PHONY Targets**: Properly handle .PHONY declarations

## Error Strategy

When extraction isn't possible:
1. Detect extraction blockers (dependencies, recursive make, etc.)
2. Show clear error with reason
3. Suggest running the test command directly
4. Document supported patterns clearly

## Performance Considerations

1. **Lazy Parsing**: Only parse Makefile when make command detected
2. **Caching**: Cache parsed Makefile for multiple target analysis
3. **Fast Path**: Quick detection for non-make commands
4. **Timeout**: Set parsing timeout for extremely large Makefiles

## Security Considerations

1. **Shell Injection**: Carefully handle command extraction
2. **Path Traversal**: Validate Makefile paths
3. **Variable Expansion**: Don't execute arbitrary shell commands
4. **Resource Limits**: Limit Makefile size and parsing time

## Rollout Plan

### Phase 1: MVP (Week 1-2)
- Basic make command detection
- Simple Makefile parser
- Support single-command test targets
- Clear errors for unsupported cases

### Phase 2: Enhancement (Week 3-4)
- Handle multi-command targets (find test command among echos)
- Better error messages
- More test patterns
- Integration tests

### Phase 3: Polish (Week 5)
- Documentation
- Real-world testing
- Performance optimization
- Edge case handling

## Future Enhancements

### Near-term
1. **Variable Resolution**: Expand simple variables like `$(TEST_CMD)`
2. **Working Directory**: Handle `cd` commands before test execution
3. **Environment Variables**: Preserve make's environment setup
4. **Include Files**: Support included Makefiles

### Long-term
1. **Multiple Test Commands**: Support running multiple test commands in a single 3pio session
   - Example: `test: unit integration e2e` where each runs different test suites
   - Would require 3pio to support multiple concurrent test runners
   - Each would get its own report section
2. **Dependency Execution**: Run target dependencies before test
3. **Advanced Parser**: Full Makefile grammar support
4. **Shell Injection**: Modify PATH to intercept test commands
5. **Make Wrapper**: Custom make binary that integrates with 3pio

## Risk Mitigation

1. **Start Conservative**: Only support simplest cases initially
2. **Clear Documentation**: Document supported Makefile patterns
3. **Escape Hatch**: Always allow falling back to regular make
4. **User Feedback**: Collect real-world Makefiles for testing
5. **Incremental Rollout**: Beta feature flag initially

## Timeline Estimate

- Week 1: Basic detection and simple parser
- Week 2: Command extraction and transformation
- Week 3: Test fixtures and integration tests
- Week 4: Edge cases and documentation
- Week 5: Real-world testing and refinement

## Alternatives Considered

### Alternative 1: Shell PATH Manipulation
- Pro: Catches all test invocations
- Con: Complex, platform-specific
- Con: May break make's assumptions

### Alternative 2: Execute Make Directly
- Pro: Preserves all Make functionality
- Con: Cannot inject adapters
- Con: No structured reports

### Alternative 3: Make Wrapper Binary
- Pro: Full control
- Con: Requires installation step
- Con: May conflict with system make

## Decision: Command Extraction
We chose the command extraction approach because:
1. Treats Make like package.json - as a command source
2. Maintains full 3pio functionality for supported cases
3. Clear boundaries on what's supported
4. No system modifications required
5. Consistent with 3pio's architecture