# Make Support for 3pio

## Overview

Add support for running tests through Makefile targets (e.g., `3pio make test`) by parsing the Makefile, finding the target, and extracting the test command to run with 3pio's enhancements.

## Design Philosophy

Make support in 3pio follows a **progressive enhancement** approach:
1. Start with the simplest, most common cases
2. Provide clear feedback when cases aren't supported
3. Always have a fallback to standard make behavior
4. Never break existing make workflows

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
        [Runner Manager]
                ↓
        [MakeDefinition]
                ↓
        [Makefile Parser]
                ↓
    [Decision: Simple or Complex?]
         ↙              ↘
    [Simple]          [Complex]
        ↓                ↓
[Extract Command]   [Passthrough]
        ↓                ↓
[Transform with      [Run make
  Adapters]           normally]
        ↓                ↓
    [Execute]        [Execute]
        ↓                ↓
    [3pio Reports]   [Standard Output]
```

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

## Component Design

### 1. MakeDefinition

```go
package definitions

type MakeDefinition struct {
    BaseDefinition
    parser      *makefile.Parser
    target      string
    makefile    string
    complexity  ComplexityLevel
}

type ComplexityLevel int

const (
    Simple ComplexityLevel = iota
    Medium
    Complex
    Unsupported
)

func (m *MakeDefinition) Detect(args []string) bool {
    // Check if command is "make" with test-related target
    return args[0] == "make" && isTestTarget(args)
}

func (m *MakeDefinition) BuildCommand(args []string) ([]string, error) {
    // Parse Makefile
    // Analyze complexity
    // Extract or passthrough based on complexity
}
```

### 2. Makefile Parser

```go
package makefile

type Parser struct {
    content    []byte
    targets    map[string]*Target
    variables  map[string]string
}

type Target struct {
    Name         string
    Dependencies []string
    Commands     []Command
    IsPhony      bool
}

type Command struct {
    Raw        string
    Executable string
    Args       []string
    IsSilent   bool  // @ prefix
    IsIgnored  bool  // - prefix
}

func (p *Parser) Parse() error {
    // Parse Makefile syntax
    // Build target dependency graph
    // Resolve simple variables
}

func (p *Parser) ExtractTestCommand(target string) (*TestCommand, error) {
    // Find target
    // Analyze commands
    // Return extracted test command or error
}
```

### 3. Complexity Analyzer

```go
package makefile

type ComplexityAnalyzer struct {
    target *Target
}

func (ca *ComplexityAnalyzer) Analyze() ComplexityLevel {
    // Check for complexity indicators:
    // - Multiple commands
    // - Recursive make calls
    // - Complex shell constructs
    // - Dependencies on other targets
    // Return appropriate complexity level
}

// Complexity indicators
func hasRecursiveMake(cmd Command) bool
func hasMultipleTestCommands(target *Target) bool
func hasComplexShellConstructs(cmd Command) bool
func hasDependencies(target *Target) bool
```

### 4. Command Extractor

```go
package makefile

type CommandExtractor struct {
    parser *Parser
}

func (ce *CommandExtractor) Extract(target string) ([]string, error) {
    t := ce.parser.targets[target]

    // Simple case: single test command
    if len(t.Commands) == 1 {
        return ce.extractSingleCommand(t.Commands[0])
    }

    // Medium case: test command with setup
    if testCmd := ce.findTestCommand(t.Commands); testCmd != nil {
        return ce.extractSingleCommand(*testCmd)
    }

    return nil, ErrComplexTarget
}

func (ce *CommandExtractor) findTestCommand(cmds []Command) *Command {
    // Identify the actual test command among setup commands
    // Look for npm test, pytest, go test, etc.
}
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

```go
// internal/runner/manager.go

func NewManager() *Manager {
    return &Manager{
        definitions: []Definition{
            // Existing definitions...
            &definitions.MakeDefinition{}, // Add make support
        },
    }
}
```

### 2. Command Detection

```go
// internal/runner/definitions/make.go

var testTargetPatterns = []string{
    "test", "tests", "check", "test-unit", "test-integration",
    "test-all", "unittest", "unit-test", "integration-test",
}

func isTestTarget(args []string) bool {
    for _, arg := range args[1:] {
        if !strings.HasPrefix(arg, "-") {
            return matchesTestPattern(arg)
        }
    }
    return false
}
```

## Execution Flow

### Simple Case Flow

1. User runs: `3pio make test`
2. MakeDefinition detects make command
3. Parser reads Makefile
4. Analyzer determines complexity: Simple
5. Extractor gets: `npm test`
6. Transform to: `npm test -- --reporters /path/to/adapter`
7. Execute with 3pio orchestrator
8. Generate full 3pio reports

### Complex Case Flow

1. User runs: `3pio make test`
2. MakeDefinition detects make command
3. Parser reads Makefile
4. Analyzer determines complexity: Complex
5. Show warning:
   ```
   Complex Makefile target detected.
   Running 'make test' directly without 3pio enhancements.
   Reason: Target has dependencies and multiple commands.
   ```
6. Execute: `make test` (passthrough)
7. Capture output normally

## Implementation Steps

### Step 1: Create Makefile Parser
- Build basic Makefile parser in `internal/makefile/`
- Support target extraction with recipes
- Handle @ and - prefixes
- Parse dependencies

### Step 2: Add Make Runner Definition
- Create `internal/runner/definitions/make.go`
- Implement `Detect()` for make commands
- Implement `BuildCommand()` to extract test command

### Step 3: Integrate with Runner Manager
- Register MakeDefinition in the manager
- Ensure proper detection order

### Step 4: Test Command Recognition
- Build pattern matcher for test commands
- Support all common test runners
- Handle variations (npm test, npm run test, etc.)

### Step 5: Error Handling
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

### Parse Errors
```go
if err := parser.Parse(); err != nil {
    if errors.Is(err, makefile.ErrNoMakefile) {
        return nil, fmt.Errorf("No Makefile found in current directory")
    }
    if errors.Is(err, makefile.ErrSyntax) {
        // Fall back to passthrough
        log.Debug("Makefile syntax too complex, using passthrough")
        return args, nil
    }
}
```

### Target Not Found
```go
if _, exists := parser.targets[target]; !exists {
    return nil, fmt.Errorf("Target '%s' not found in Makefile", target)
}
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
```go
// internal/makefile/parser_test.go
func TestParseSimpleMakefile(t *testing.T)
func TestParseComplexMakefile(t *testing.T)
func TestExtractTestCommand(t *testing.T)
func TestComplexityAnalysis(t *testing.T)

// internal/runner/definitions/make_test.go
func TestMakeDetection(t *testing.T)
func TestBuildCommand(t *testing.T)
```

### Integration Tests
```go
// tests/integration_go/make_test.go
func TestMakeSimpleNPM(t *testing.T)
func TestMakeWithDependencies(t *testing.T)
func TestMakeComplexFallback(t *testing.T)
func TestMakeNoMakefile(t *testing.T)
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
- Basic make detection
- Simple Makefile parser
- Support single-command test targets
- Passthrough for complex cases

### Phase 2: Enhancement (Week 3-4)
- Medium complexity support
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

### Alternative 2: Make Wrapper Binary
- Pro: Full control
- Con: Requires installation step
- Con: May conflict with system make

### Alternative 3: LD_PRELOAD/DYLD_INSERT_LIBRARIES
- Pro: Transparent interception
- Con: Platform-specific
- Con: Security concerns

## Decision: Progressive Parser
We chose the progressive parser approach because:
1. Simplest to implement and understand
2. Handles common cases well
3. Clear upgrade path for complexity
4. No system modifications required
5. Safe fallback behavior