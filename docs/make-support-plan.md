# Make Support Implementation Plan

## Overview

Add support for running tests through Makefile targets (e.g., `3pio make test`) by parsing the Makefile, finding the target, and extracting the test command to run with 3pio's enhancements.

## Core Approach

**Parse → Find → Extract → Transform → Execute**

1. **Parse**: Read and parse the Makefile
2. **Find**: Locate the requested target (e.g., `test`)
3. **Extract**: Extract the test command from the target's recipe
4. **Transform**: Modify the command to include 3pio adapters
5. **Execute**: Run the transformed command with 3pio orchestration

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

## Implementation Strategy

### 1. Makefile Parser
```go
// Parse Makefile and extract targets
type MakefileParser struct {
    content    []byte
    targets    map[string]*Target
    variables  map[string]string  // Simple variable storage
}

type Target struct {
    Name         string
    Dependencies []string
    Commands     []string  // Raw command lines
    LineNumbers  []int     // For debugging
}
```

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

```go
func extractTestCommand(target *Target) (string, error) {
    for _, cmd := range target.Commands {
        // Strip @ and - prefixes
        cleaned := stripMakePrefixes(cmd)

        // Check if this is a test command
        if isTestCommand(cleaned) {
            return cleaned, nil
        }
    }
    return "", ErrNoTestCommand
}
```

### 4. Complexity Decision Tree

#### Simple (Supported)
```makefile
test:
    npm test

test-python:
    @pytest tests/ -v
```
→ Extract and run with 3pio

#### Medium (Supported with warnings)
```makefile
test:
    @echo "Running tests..."
    @npm test
    @echo "Done"
```
→ Extract `npm test`, warn about ignored commands

#### Complex (Passthrough)
```makefile
test: build
    npm test

test-all:
    $(MAKE) test-unit
    $(MAKE) test-integration
```
→ Dependencies or recursive make = passthrough

### 5. Implementation Steps

#### Step 1: Create Makefile Parser
- Build basic Makefile parser in `internal/makefile/`
- Support target extraction with recipes
- Handle @ and - prefixes
- Parse dependencies

#### Step 2: Add Make Runner Definition
- Create `internal/runner/definitions/make.go`
- Implement `Detect()` for make commands
- Implement `BuildCommand()` to extract test command

#### Step 3: Integrate with Runner Manager
- Register MakeDefinition in the manager
- Ensure proper detection order

#### Step 4: Test Command Recognition
- Build pattern matcher for test commands
- Support all common test runners
- Handle variations (npm test, npm run test, etc.)

#### Step 5: Error Handling
- Clear messages for unsupported patterns
- Fallback to regular make execution
- Debug logging for troubleshooting

## File Structure

```
internal/
├── runner/
│   └── definitions/
│       └── make.go          # MakeDefinition implementation
├── makefile/
│   ├── parser.go           # Makefile parsing logic
│   ├── parser_test.go      # Parser tests
│   └── patterns.go         # Common Makefile patterns
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

test-coverage:
	go test -coverprofile=coverage.out ./...
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

## Success Criteria

1. **Basic Support**: Handle 80% of common Makefile patterns
2. **Graceful Degradation**: Clear messaging for unsupported patterns
3. **No Breaking Changes**: Existing 3pio functionality unchanged
4. **Performance**: Minimal overhead for Makefile parsing
5. **Compatibility**: Support GNU make syntax primarily

## Edge Cases to Consider

1. **Recursive Make**: `$(MAKE) -C subdir test`
2. **Variable Expansion**: `$(TEST_CMD)` where TEST_CMD defined elsewhere
3. **Conditionals**: `ifdef` / `ifndef` blocks
4. **Multiple Commands**: Multiple test commands in one target
5. **Shell Features**: Pipes, redirects, command substitution
6. **Include Directives**: Makefile includes other files
7. **Pattern Rules**: `%.test: %.c` style rules
8. **PHONY Targets**: Properly handle .PHONY declarations

## Fallback Strategy

When Makefile is too complex:
1. Detect complexity indicators (recursive make, multiple commands, etc.)
2. Show warning: "Complex Makefile detected. Running make directly..."
3. Option to force extraction: `3pio make test --force-extract`
4. Document limitations clearly

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

## Decision Points

1. **Parser Complexity**: How much Makefile syntax to support initially?
2. **Execution Model**: Direct extraction vs. PATH manipulation?
3. **User Experience**: How to handle unsupported patterns?
4. **Configuration**: Allow .3pio config for make behavior?