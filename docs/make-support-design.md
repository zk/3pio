# Make Support Design Document

## Design Philosophy

Make support in 3pio follows a **progressive enhancement** approach:
1. Start with the simplest, most common cases
2. Provide clear feedback when cases aren't supported
3. Always have a fallback to standard make behavior
4. Never break existing make workflows

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

## Success Metrics

1. **Coverage**: Successfully handle 80% of common Makefile patterns
2. **Performance**: Parse time < 100ms for typical Makefiles
3. **Reliability**: Zero regressions in existing functionality
4. **Usability**: Clear error messages and fallback behavior

## Open Questions

1. Should we support GNU make extensions or stick to POSIX?
2. How deep should variable resolution go?
3. Should we support `-f` flag for alternate Makefiles?
4. How to handle make's dry-run mode (`-n`)?
5. Should we intercept recursive make calls?

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