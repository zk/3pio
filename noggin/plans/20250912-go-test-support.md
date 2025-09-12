# Plan: Go Test Support for 3pio

**Date:** 2025-09-12  
**Status:** Ready for Implementation

## Objective

Add native support for `go test` to 3pio, leveraging Go's built-in JSON output format to capture test results without requiring an external adapter.

## Architecture Decisions

### 1. Native JSON Processing
- Parse `go test -json` output directly in the orchestrator
- No external adapter file needed (unlike Jest/Vitest/pytest)
- GoTestDefinition lives in `internal/runner/definitions/` alongside other runners
- ProcessOutput method reads stdout and converts JSON events to IPC events in real-time

### 2. Package to File Mapping
- Run `go list -json ./...` once at the beginning of each 3pio invocation
- Build mapping of package imports to test file paths
- Use mapping to attribute tests to specific files
- Support non-standard patterns (*_integration_test.go, *_e2e_test.go)
- Parse TestGoFiles field from go list for accurate file listing

### 3. Command Modification
- Always ensure `-json` flag is present
- Preserve all user-provided flags and arguments
- If user already provided -json, don't duplicate
- Support common patterns:
  - `go test`
  - `go test ./...`
  - `go test -run TestName`
  - `go test package/path`
  - `go test -v` (verbose works normally)
  - `go test -parallel=N` (parallel works normally)

### 4. Build Failure Handling
- Compilation errors shown in console output only
- No run directory or files created on build failure
- Exit with appropriate error code
- Detect non-JSON lines at start of output for build errors

### 5. Parallel Test Output
- Track pause/cont state transitions for accurate output attribution
- Buffer output between cont→pass events to associate correctly
- Show warning at startup about potential output interleaving
- Implement proper state tracking for parallel tests

### 6. Test Cache Handling
- Detect cached packages via "(cached)" in output
- Show as CACH status in summary table
- Only file-level events for cached tests (no individual test cases)
- No individual report files for cached tests
- Include cache notice in test-run.md

## Implementation Components

### File Structure
```
internal/
├── runner/
│   └── definitions/
│       ├── gotest.go          # GoTestDefinition implementation
│       ├── json_processor.go  # Parse JSON stream to IPC events
│       └── mapper.go          # Map packages to file paths
```

### 1. GoTestDefinition (`gotest.go`)
```go
type GoTestDefinition struct {
    logger *logger.FileLogger
}

func (g *GoTestDefinition) Name() string { return "go" }
func (g *GoTestDefinition) Detect(args []string) bool
func (g *GoTestDefinition) ModifyCommand(cmd []string, ipcPath, runID string) []string
func (g *GoTestDefinition) GetTestFiles(args []string) ([]string, error)
func (g *GoTestDefinition) RequiresAdapter() bool { return false }
func (g *GoTestDefinition) ProcessOutput(stdout io.Reader, ipcPath string) error
```

### 2. JSON Processor (`json_processor.go`)

#### Event Mapping
```
go test JSON Event    →  3pio IPC Event
─────────────────────────────────────────
"start" + Package     →  (run go list)
"run" + Test          →  (track test start)
"pass" + Test         →  testCase {status: "PASS"}
"fail" + Test         →  testCase {status: "FAIL"}
"skip" + Test         →  testCase {status: "SKIP"}
"output" + Test       →  stdoutChunk
"pass/fail" + Package →  testFileResult
```

#### Key Functions
```go
type JSONProcessor struct {
    packageMap  map[string]*PackageInfo
    testStates  map[string]*TestState
    ipcWriter   *IPCWriter
}

func (j *JSONProcessor) ProcessLine(line []byte) error
func (j *JSONProcessor) handleTestEvent(event *GoTestEvent) error
func (j *JSONProcessor) handlePackageEvent(event *GoTestEvent) error
```

### 3. Package Mapper (`mapper.go`)
```go
type PackageInfo struct {
    ImportPath string
    Dir        string
    TestFiles  []string
    IsCached   bool
    Status     string
}

func RunGoList(args []string) (map[string]*PackageInfo, error)
func parseGoListOutput(output []byte) map[string]*PackageInfo
```

## Test Cache Handling

### Detection
- Check for "(cached)" in package output
- Mark package as cached in PackageInfo

### Reporting
```markdown
## Summary
- Total files: 25
- Files cached: 8

**Note:** 8 test files were run from cache (3 packages). 
Cached tests show package-level results only. 
Use `-count=1` flag to force fresh execution.

## Test file results
| Stat | Test | Report file |
| ---- | ---- | ----------- |
| PASS | parser_test.go | ./reports/parser_test.go.md |
| CACH | utils_test.go | (cached) |
| FAIL | main_test.go | ./reports/main_test.go.md |
```

## Special Considerations

### Go Workspaces (go.work)
- Handle multi-module testing
- Parse Module field from go list
- Do NOT group results by module in reports
- Use file system paths (not import paths) in IPC events

### Subtests
- Parse "/" separator in test names
- Flatten subtests (no hierarchical structure)
- Each subtest reported as individual test case with own duration

### Parallel Tests
- Track pause/cont state transitions
- Buffer output between cont→pass for accurate attribution
- Show warning about interleaving at startup

### Build Errors
- Display to console only
- No run directory or files created
- Exit with appropriate error code

### Output Handling
- Both test output (t.Log) and regular output (fmt.Println) go to stdoutChunk events
- Non-JSON lines at start indicate build errors
- Example tests (func ExampleXxx) reported as regular tests

## Limitations to Document

1. **No benchmark support** - `-bench` flag not supported initially
2. **No coverage support** - `-cover` flag not supported initially  
3. **Parallel output interleaving** - Console output may be attributed to wrong test
4. **Cached test details** - Individual test results unavailable for cached packages

## Testing Strategy

### Unit Tests
- Command detection logic
- JSON parsing
- Package mapping
- IPC event generation

### Integration Tests
- Full flow with real go test
- Cached vs fresh runs
- Build failures
- Parallel tests
- Subtests

### Test Fixtures
Create `tests/fixtures/basic-go/` with:
- Simple pass/fail/skip tests
- Parallel tests
- Subtests
- Multiple packages
- Build errors

## Success Criteria

- [x] Correctly detect `go test` commands
- [x] Parse JSON output without external adapter
- [x] Map packages to test files via go list
- [x] Handle cached test results appropriately
- [x] Generate reports consistent with other runners
- [x] Show appropriate warnings for limitations
- [x] Handle build failures gracefully

## Next Steps

1. Create GoTestDefinition implementation
2. Implement JSON stream processor
3. Add package to file mapping
4. Update runner manager registration
5. Write integration tests
6. Update documentation

## Implementation Checklist

### Phase 1: Core Structure
- [ ] Create `GoTestDefinition` in `internal/runner/definitions/gotest.go`
- [ ] Implement `GoTestDefinition` struct with basic interface methods
- [ ] Add detection logic for `go test` commands
- [ ] Register GoTestDefinition in runner manager
- [ ] Test basic detection with unit tests

### Phase 2: Command Modification
- [ ] Implement `-json` flag injection logic
- [ ] Preserve existing user flags
- [ ] Handle various go test patterns (./..., specific packages, -run)
- [ ] Add command modification unit tests

### Phase 3: Package Mapping
- [ ] Implement `go list -json` execution
- [ ] Parse go list output to PackageInfo structs
- [ ] Build ImportPath → TestFiles mapping
- [ ] Handle go workspace (go.work) detection
- [ ] Add mapper unit tests

### Phase 4: JSON Processing
- [ ] Create JSON event parser for go test output
- [ ] Implement test state tracking (run/pause/cont)
- [ ] Map JSON events to IPC events
- [ ] Handle cached package detection
- [ ] Process parallel test output
- [ ] Add JSON processor unit tests

### Phase 5: IPC Integration
- [ ] Send testFileStart events for packages
- [ ] Send testCase events for individual tests
- [ ] Send testFileResult events with proper status
- [ ] Handle stdoutChunk/stderrChunk events
- [ ] Special handling for CACH status

### Phase 6: Report Generation
- [ ] Update report manager to handle CACH status
- [ ] Modify summary to include cached file count
- [ ] Add cache notice to test-run.md
- [ ] Ensure no individual reports for cached tests
- [ ] Test report generation with cached and fresh runs

### Phase 7: Error Handling
- [ ] Handle build failures gracefully
- [ ] Process compilation errors to console
- [ ] Handle missing go binary
- [ ] Test with invalid go code
- [ ] Ensure proper exit codes

### Phase 8: Integration Testing
- [ ] Create `tests/fixtures/basic-go/` test project
- [ ] Add passing, failing, and skipped tests
- [ ] Add parallel tests to fixture
- [ ] Add subtests to fixture
- [ ] Write full integration tests
- [ ] Test with cached vs fresh runs
- [ ] Test with go workspaces

### Phase 9: Documentation
- [ ] Update README with go test support
- [ ] Document limitations (no -bench, no -cover)
- [ ] Add go test examples
- [ ] Document cache behavior
- [ ] Update `docs/architecture/test-runner-adapters.md` with Go section
- [ ] Update `docs/architecture/architecture.md` to include Go test flow
- [ ] Add to supported runners list

### Phase 10: Final Validation
- [ ] Test with real Go projects
- [ ] Verify parallel test warning displays
- [ ] Confirm cache detection works
- [ ] Check report formatting
- [ ] Performance testing with large test suites
- [ ] Edge case testing