# Research: Adding Deno Support to 3pio

## Executive Summary

This document analyzes the feasibility and requirements for adding Deno test runner support to 3pio. Based on research into Deno's testing capabilities and 3pio's architecture, **Deno support should be implemented using the TAP reporter approach** - a hybrid between native processing (like Go/Rust) and adapter injection (like Jest/Vitest), without requiring external adapter files.

## Key Findings

### Current Deno Test Runner Limitations

1. **No JSON Output Format**: Unlike Go test (`-json`) and cargo test (`--format json`), Deno test does not provide native JSON output
2. **No Custom Reporter API**: Deno lacks the extensible reporter interfaces found in Jest/Vitest/pytest
3. **Limited Structured Output Options**:
   - TAP format (`--reporter=tap`) - Best option for integration
   - JUnit XML (`--reporter=junit`) - Alternative structured format
   - Pretty/Dot reporters - Human-readable only

### Available Integration Points

#### TAP Reporter (Recommended Approach)
- **Command**: `deno test --reporter=tap`
- **Format**: Line-based Test Anything Protocol
- **Advantages**:
  - Machine-parseable format
  - Real-time streaming output
  - Hierarchical test organization support via test descriptions
  - Standard protocol with well-defined specification
- **Example Output**:
  ```
  1..4
  ok 1 - Math operations > should add numbers
  not ok 2 - String utils > should concatenate
      Error: Expected "foobar" but got "foo"
  ok 3 - API tests > should return 200
  # skip 4 - Database tests > should connect
  ```

#### JUnit Reporter (Alternative)
- **Command**: `deno test --reporter=junit` or `--junit-path=report.xml`
- **Format**: XML-based structured output
- **Disadvantages**:
  - XML parsing complexity
  - Batch output (not streaming)
  - Less natural mapping to 3pio's group hierarchy

## Proposed Implementation Strategy

### Architecture: Native TAP Processing

Following the pattern of Go test and cargo test, implement Deno support as a **native runner with TAP processing**:

```
User Command → 3pio → Modify Command → Execute → Parse TAP → Generate IPC Events
                      (add --reporter=tap)         (stream)
```

### Implementation Components

#### 1. DenoTestDefinition (`internal/runner/definitions/deno.go`)

```go
type DenoTestDefinition struct {
    logger       *logger.FileLogger
    ipcWriter    *IPCWriter
    testGroups   map[string]*TestGroupInfo
    currentGroup string
}

func (d *DenoTestDefinition) Detect(args []string) bool {
    return len(args) >= 2 &&
           args[0] == "deno" &&
           args[1] == "test"
}

func (d *DenoTestDefinition) ModifyCommand(cmd []string) []string {
    // Add --reporter=tap if not present
    // Preserve existing arguments
}

func (d *DenoTestDefinition) ProcessOutput(line string) {
    // Parse TAP format
    // Generate IPC events
}
```

#### 2. TAP Parser Component

Parse TAP output and map to 3pio's universal group abstractions:

##### TAP Line Types to Handle
- **Plan**: `1..N` - Total test count
- **Test Result**: `ok|not ok N - description`
- **Diagnostic**: `# comment` - Skip/todo markers
- **YAML Block**: Error details between `---` and `...`
- **Bailout**: `Bail out!` - Critical failure

##### Parser Implementation (Pseudocode)

```go
type TAPParser struct {
    ipcWriter       *IPCWriter
    groups          map[string]*GroupInfo
    yamlBuffer      []string
    inYAML          bool
    testNumber      int
}

func (p *TAPParser) ProcessLine(line string) {
    switch {
    case p.inYAML:
        if line == "  ..." {
            p.processYAMLBlock()
            p.inYAML = false
        } else {
            p.yamlBuffer = append(p.yamlBuffer, line)
        }

    case strings.HasPrefix(line, "ok ") || strings.HasPrefix(line, "not ok "):
        p.processTestResult(line)

    case strings.HasPrefix(line, "  ---"):
        p.inYAML = true
        p.yamlBuffer = []string{}

    case strings.HasPrefix(line, "1.."):
        p.processPlan(line)

    case strings.HasPrefix(line, "# "):
        p.processDiagnostic(line)

    case strings.HasPrefix(line, "Bail out!"):
        p.processBailout(line)
    }
}

func (p *TAPParser) processTestResult(line string) {
    // Parse: "ok 1 - Math > addition > should add"
    parts := regexp.MustCompile(`^(ok|not ok)\s+(\d+)\s+-\s+(.+?)(\s+#.*)?$`)
    matches := parts.FindStringSubmatch(line)

    status := matches[1]
    description := matches[3]
    directive := matches[4] // # SKIP, # TODO

    // Extract hierarchy from description
    hierarchy := p.parseHierarchy(description)
    parentNames := hierarchy[:len(hierarchy)-1]
    testName := hierarchy[len(hierarchy)-1]

    // Ensure parent groups exist
    for i := range parentNames {
        groupPath := strings.Join(parentNames[:i+1], " > ")
        if !p.groups[groupPath] {
            p.sendGroupDiscovered(parentNames[:i+1])
            p.groups[groupPath] = &GroupInfo{started: false}
        }
    }

    // Start groups if needed
    for i := range parentNames {
        groupPath := strings.Join(parentNames[:i+1], " > ")
        if !p.groups[groupPath].started {
            p.sendGroupStart(parentNames[:i+1])
            p.groups[groupPath].started = true
        }
    }

    // Send test case event
    testStatus := "PASS"
    if status == "not ok" {
        testStatus = "FAIL"
    }
    if directive != "" {
        testStatus = "SKIP"
    }

    p.ipcWriter.WriteTestCase(testName, parentNames, testStatus, nil)
}

func (p *TAPParser) parseHierarchy(description string) []string {
    // Try common separators in order of preference
    separators := []string{" > ", " :: ", " / ", "."}

    for _, sep := range separators {
        if strings.Contains(description, sep) {
            return strings.Split(description, sep)
        }
    }

    // No separator found, treat as single test in root
    return []string{"deno-tests", description}
}
```

##### Hierarchical Group Extraction

Deno test descriptions often follow patterns that indicate hierarchy:
```
"Math operations > addition > should add positive numbers"
"API tests > authentication > login > should accept valid credentials"
```

Parse using common separators (`>`, `::`, `/`, `.`) to build group hierarchy:
```
Group: Math operations
  └── Group: addition
      └── Test: should add positive numbers
```

#### 3. Test Step Support

Deno's `TestContext.step()` API creates nested test steps that should map to 3pio's group hierarchy:

```javascript
Deno.test("User API", async (t) => {
  await t.step("authentication", async (t) => {
    await t.step("login", async () => {
      // Test login
    });
    await t.step("logout", async () => {
      // Test logout
    });
  });
});
```

TAP output would show flattened hierarchy that needs reconstruction:
```
ok 1 - User API > authentication > login
ok 2 - User API > authentication > logout
```

## Comparison with Existing Runners

| Runner | Type | Output Format | Adapter Required | Command Modification |
|--------|------|--------------|------------------|---------------------|
| Jest | Adapter-based | Via reporter API | Yes (jest.js) | `--reporters path/to/adapter` |
| Vitest | Adapter-based | Via reporter API | Yes (vitest.js) | `--reporter path/to/adapter` |
| pytest | Adapter-based | Via plugin API | Yes (pytest_adapter.py) | `-p adapter_name` |
| Go test | Native | JSON stream | No | `-json` |
| cargo test | Native | JSON stream | No | `--format json` |
| **Deno test** | **Native** | **TAP stream** | **No** | `--reporter=tap` |

## Implementation Phases

### Phase 1: Basic TAP Support (MVP)
- [ ] Create `DenoTestDefinition` struct
- [ ] Implement TAP line parser for basic ok/not ok
- [ ] Map flat test names to single-level groups
- [ ] Generate testCase events with pass/fail status
- [ ] Handle test counts and summary

### Phase 2: Hierarchical Support
- [ ] Parse test descriptions for hierarchy markers (`>`, `::`, `/`)
- [ ] Build nested group structure from test names
- [ ] Support Deno's test steps (nested descriptions)
- [ ] Generate testGroupStart/Result events
- [ ] Handle skip and todo directives

### Phase 3: Error Handling
- [ ] Parse YAML error blocks in TAP output
- [ ] Extract stack traces and error messages
- [ ] Handle test timeouts and bailouts
- [ ] Support diagnostic comments
- [ ] Process beforeAll/afterAll hook failures

### Phase 4: Advanced Features
- [ ] Support test filtering (`deno test --filter`)
- [ ] Handle parallel test execution
- [ ] Process benchmark tests (`deno bench` integration)
- [ ] Support coverage (if `--coverage` doesn't conflict)
- [ ] Integration with deno.json configuration

### Phase 5: Testing & Polish
- [ ] Create test fixtures for various Deno project structures
- [ ] Test with Deno standard library tests
- [ ] Test with popular Deno frameworks (Fresh, Oak, etc.)
- [ ] Performance testing with large test suites
- [ ] Documentation and examples

## Alternative Approaches Considered

### 1. JUnit XML Processing (Not Recommended)
- **Pros**: Structured data, comprehensive test information
- **Cons**:
  - Requires full XML parsing
  - Output only available after completion
  - More complex implementation
  - Less efficient than streaming TAP

### 2. Post-Processing Pretty Output (Not Recommended)
- **Pros**: No command modification needed
- **Cons**:
  - Fragile parsing of human-readable format
  - Format may change between Deno versions
  - Difficult to extract structured data
  - No reliable error information

### 3. Custom Test Framework Wrapper (Not Recommended)
- **Pros**: Full control over output format
- **Cons**:
  - Requires users to modify their tests
  - Not compatible with existing Deno tests
  - Maintenance burden for wrapper code
  - Goes against 3pio's zero-config philosophy

### 4. Wait for JSON Reporter (Not Viable)
- GitHub discussions indicate JSON output is "inevitable" but no timeline
- Cannot wait indefinitely for feature that may take years
- TAP provides adequate structured output today

## TAP Format Challenges and 3pio-Specific Solutions

### Challenge 1: Flat Output vs Hierarchical Structure
**Issue**: TAP output is flat while 3pio needs hierarchical groups
**Example**:
```tap
ok 1 - Math > addition > should add positive numbers
ok 2 - Math > subtraction > should subtract
```

**3pio Solution**:
- Parse common separators (`>`, `::`, `/`) to build group hierarchy dynamically
- Create groups on-demand as tests are discovered (consistent with all 3pio runners)
- Generate `testGroupDiscovered` events for parent groups before test execution
- File boundaries inferred from test execution order and description patterns

### Challenge 2: Limited Metadata (Duration, Types, Paths)
**Issue**: TAP lacks standard fields for duration, file paths, test types
**3pio's Approach**:
- **Duration**: Not critical - 3pio already handles missing durations gracefully (see Go test handling)
- **File paths**: Use test description prefixes or maintain state from command execution
- **Test types**: All tests treated uniformly as "test cases" (same as other runners)
- **Key insight**: 3pio only needs pass/fail status and hierarchical organization, which TAP provides

### Challenge 3: Inconsistent Error Formatting
**Issue**: TAP error details in optional YAML blocks with varying formats
**Example**:
```tap
not ok 2 - should parse JSON
  ---
  message: 'Unexpected token'
  severity: fail
  ...
```

**3pio Solution**:
- Parse YAML blocks when present, gracefully degrade when absent
- Extract error message and stack trace using regex patterns
- Fallback to test description line if no YAML block
- Similar to how 3pio handles varying error formats from different runners

### Challenge 4: No Test Discovery Phase
**Issue**: TAP only reports tests as they run, no upfront discovery
**3pio's Standard Approach**:
- **This is not a problem** - 3pio uses dynamic discovery for ALL runners
- Test files discovered as they execute (same as Jest, Vitest, pytest, Go, Rust)
- `GetTestFiles()` returns empty array to enable dynamic discovery
- Progress shown as "Test files will be discovered and reported as they run"

### Challenge 5: Streaming and State Management
**Issue**: Must maintain state while parsing line-by-line TAP stream
**3pio Implementation**:
```go
type TAPParser struct {
    groups          map[string]*GroupInfo  // Track discovered groups
    currentFile     string                 // Inferred from test patterns
    testCount       int                    // Running count
    pendingError    *YAMLBlock            // Buffer for YAML blocks
}
```
- Similar complexity to existing `CargoTestDefinition` and `GoTestDefinition`
- Reuse existing patterns for state management and event generation

### Challenge 6: Test Step Attribution
**Issue**: Nested Deno test steps become flattened in TAP
**Example**:
```javascript
Deno.test("API", async (t) => {
  await t.step("auth", async (t) => {
    await t.step("login", () => {});
  });
});
// Becomes: ok 1 - API > auth > login
```

**3pio Solution**:
- Parse separator patterns to identify group boundaries
- "API" and "auth" become groups, "login" becomes test case
- Send `testGroupDiscovered` for "API" and "API > auth" before test case
- Consistent with how 3pio handles Jest describe blocks

### Challenge 7: No Duration Information
**Issue**: TAP doesn't standardize timing information
**3pio's Approach**:
- Duration is optional in 3pio's data model
- Many successful 3pio integrations work without precise timing
- If present in comments (`# time=123ms`), parse it; otherwise omit
- Reports show "N/A" for duration when unavailable

### Challenge 8: Parallel Execution and Out-of-Order Tests
**Issue**: Parallel tests may produce non-sequential TAP output
```tap
ok 2 - test from file B
ok 1 - test from file A
```

**3pio Solution**:
- Don't rely on test numbers for ordering
- Use test descriptions to determine file/group context
- Buffer and reorder within groups if needed
- Similar to how 3pio handles Jest worker process output

### Challenge 9: Skip/Todo Information
**Issue**: TAP skip/todo directives are basic
```tap
ok 3 - test name # SKIP
ok 4 - test name # TODO not implemented
```

**3pio Solution**:
- Map `# SKIP` to status "SKIP" in testCase event
- Map `# TODO` to status "SKIP" with different message
- Sufficient for 3pio's reporting needs (shows as skipped in reports)

### Challenge 10: File Association
**Issue**: TAP doesn't indicate which file tests belong to
**3pio Solution**:
- Infer files from test description patterns (e.g., "path/to/file.test.ts > test name")
- Maintain state from Deno test command execution context
- Use heuristics based on test naming conventions
- Fallback to single "deno-tests" group if file detection fails

## Why These Solutions Work for 3pio

1. **3pio's Architecture is Flexible**: Designed to handle varying levels of information from different runners
2. **Dynamic Discovery is Standard**: All runners use dynamic test discovery, TAP fits this model
3. **Essential Data is Present**: Pass/fail status and test names are sufficient for core functionality
4. **Graceful Degradation**: Missing metadata (duration, detailed errors) handled gracefully
5. **Proven Patterns**: Similar challenges solved for Go test and cargo test integrations

## Benefits of Deno Support

### For 3pio Users
- **Unified Test Experience**: Same 3pio interface for Deno as other runners
- **Persistent Results**: Test results saved to filesystem
- **Structured Reports**: Hierarchical organization of test results
- **CI/CD Integration**: Machine-readable outputs for automation

### For the Deno Ecosystem
- **Adoption Driver**: Better tooling encourages Deno adoption
- **Cross-Runtime Testing**: Projects using both Node.js and Deno
- **Modern JavaScript**: Deno represents future of server-side JS
- **TypeScript-First**: Natural fit for TypeScript projects

## Implementation Complexity Assessment

**Estimated Effort**: Medium (2-3 weeks)

**Complexity Factors**:
- ✅ **Simple**: TAP format is well-documented and straightforward
- ✅ **Simple**: No adapter extraction needed (native processing)
- ✅ **Simple**: Similar to existing Go/Rust implementations
- ⚠️ **Medium**: Parsing hierarchical structure from flat output
- ⚠️ **Medium**: Handling various test description formats
- ✅ **Simple**: Reuse existing IPC event generation code

## Success Metrics

- Support 90% of common Deno test patterns
- Process test suites with 1000+ tests efficiently
- Maintain real-time output during test execution
- Generate reports compatible with existing 3pio structure
- Zero configuration required for basic usage

## Recommendation

**Proceed with TAP-based implementation** as it provides:

1. **Immediate feasibility** - Can be implemented today with current Deno
2. **Streaming output** - Real-time test progress like other 3pio runners
3. **Sufficient structure** - TAP provides pass/fail, counts, and basic errors
4. **Future-compatible** - Can add JSON support later when available
5. **Consistent architecture** - Follows native runner pattern (Go/Rust)

The TAP approach balances implementation simplicity with functional completeness, making it the optimal choice for adding Deno support to 3pio.

## Next Steps

1. Create proof-of-concept TAP parser
2. Test with various Deno project structures
3. Implement DenoTestDefinition in Go
4. Add integration tests
5. Document usage and limitations
6. Consider future JSON reporter when available

## References

- [Deno Testing Documentation](https://docs.deno.com/runtime/fundamentals/testing/)
- [TAP Specification](https://testanything.org/tap-specification.html)
- [Deno Test CLI Reference](https://docs.deno.com/runtime/reference/cli/test/)
- [3pio Architecture Documentation](../docs/architecture/architecture.md)
- [3pio Test Runner Adapters](../docs/architecture/test-runner-adapters.md)