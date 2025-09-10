# Plan: Add pytest Support to 3pio

## Objective
Add full pytest support to 3pio, enabling Python test suites to benefit from the same AI-first structured reporting as Jest and Vitest tests.

## Success Criteria
- [ ] pytest test runs generate structured reports at `.3pio/runs/*/test-run.md`
- [ ] Individual test files get separate log files with test boundaries
- [ ] Console output is captured and stored
- [ ] Exit codes are properly mirrored
- [ ] Support for common pytest invocation patterns (pytest, python -m pytest)
- [ ] Silent adapter with no interference to normal pytest output

## Key Decisions Made

### Distribution Strategy
**Decision**: Same 3pio package distributed via both npm and pip
- **npm install -g 3pio** - For JavaScript developers (includes Jest/Vitest/pytest support)
- **pip install 3pio** - For Python developers (same functionality, Python-native distribution)
- **Rationale**: One tool, two distribution channels. Developers use their ecosystem's native package manager. Both installations provide the full 3pio CLI with all test runner support built-in.

### Plugin Injection Method
**Decision**: Use `-p` flag (`pytest -p threepio_pytest`)
- **Rationale**: Cleanest approach, no filesystem pollution, standard pytest mechanism

### Test Discovery Approach  
**Decision**: Start simple - extract test files from args (like Vitest)
- **Rationale**: `pytest --collect-only` can be added later if needed, simpler initial implementation

### Python Version Support
**Decision**: Minimum Python 3.9+
- **Rationale**: Aligns with pytest's own requirements, stable hook API, wide compatibility

### Environment Handling
**Decision**: Use whatever Python/environment is active in the shell
- **Rationale**: 3pio just shells out like with Jest/Vitest, respects user's environment

## Implementation Tasks

### Phase 1: Core pytest Support
- [ ] Create `src/runners/pytest/PyTestDefinition.ts` for command detection
- [ ] Create `src/runners/pytest/PyTestOutputParser.ts` for output parsing
- [ ] Register pytest in `TestRunnerManager.ts`
- [ ] Add pytest detection patterns (pytest, python -m pytest)
- [ ] Create `src/adapters/pytest/pytest_adapter.py` - Python adapter embedded in package

### Phase 2: Python Adapter Development
- [ ] Implement pytest plugin hooks in Python adapter
- [ ] Handle IPC communication from Python to TypeScript
- [ ] Implement silent mode (suppress all output)
- [ ] Bundle Python adapter with the npm package
- [ ] Handle adapter injection via `-p` flag

### Phase 3: Distribution Setup
- [ ] Add Python adapter to npm package build
- [ ] Create pyproject.toml for pip distribution
- [ ] Set up Python package structure that includes:
  - The Node.js CLI (via node binary or rewritten in Python)
  - The pytest adapter
  - All JavaScript adapters
- [ ] Ensure both npm and pip installations provide identical functionality

### Phase 4: Command Building
- [ ] Locate bundled pytest adapter within 3pio installation
- [ ] Build command: inject `-p /path/to/3pio/adapters/pytest_adapter.py` into pytest args
- [ ] Pass THREEPIO_IPC_PATH environment variable
- [ ] Handle edge cases (existing -p flags, plugin conflicts)

### Phase 5: Test Discovery
- [ ] Extract test files from command args (like Vitest approach)
- [ ] Support common patterns: `test_*.py`, `*_test.py`
- [ ] Future: Add `pytest --collect-only` for comprehensive discovery

### Phase 6: Output Parsing
- [ ] Parse pytest output format for test boundaries
- [ ] Extract test file paths from output lines
- [ ] Identify test case markers (PASSED, FAILED, SKIPPED)
- [ ] Handle pytest-specific formats (assertions, tracebacks)

### Phase 7: Testing
- [ ] Unit tests for PyTestDefinition
- [ ] Unit tests for PyTestOutputParser
- [ ] Create pytest fixture projects in `tests/fixtures/`:
  - [ ] `basic-pytest` - Simple passing tests (equivalent to basic-jest)
  - [ ] `empty-pytest` - Tests with failures and skips (equivalent to empty-jest)
  - [ ] `long-names-pytest` - Tests with long file/test names
  - [ ] `npm-separator-pytest` - Tests for npm script with -- separator
- [ ] End-to-end test with real pytest execution
- [ ] Test with common pytest plugins (pytest-xdist, pytest-cov)
- [ ] Test both npm and pip installations work identically

### Phase 8: Documentation
- [ ] Document installation options:
  - JavaScript developers: `npm install -g 3pio`
  - Python developers: `pip install 3pio`
  - Both provide identical functionality
- [ ] Update README with pytest examples
- [ ] Add pytest section to CLAUDE.md
- [ ] Create troubleshooting guide for common issues

## Technical Approach

### Package Structure (bundled in 3pio)
```
3pio/
├── dist/
│   ├── cli.js           # Main CLI (TypeScript compiled)
│   ├── adapters/
│   │   ├── jest.js      # Jest adapter
│   │   ├── vitest.js    # Vitest adapter
│   │   └── pytest_adapter.py  # Python pytest adapter
│   └── runners/
│       ├── jest/
│       ├── vitest/
│       └── pytest/
└── package.json / pyproject.toml  # Dual distribution metadata
```

### TypeScript Runner Definition
```typescript
// src/runners/pytest/PyTestDefinition.ts
class PyTestDefinition implements TestRunnerDefinition {
    - Detect pytest: "pytest", "python -m pytest"
    - Find adapter: path.resolve(__dirname, '../adapters/pytest_adapter.py')
    - Build command: pytest -p /absolute/path/to/adapter [original args]
    - Test discovery: Extract .py files from args (like Vitest)
    - Exit code: Mirror pytest's exit codes
}
```

### Command Injection Strategy
```bash
# Original command
pytest tests/test_math.py -v

# Modified command (3pio finds and injects bundled adapter)
# The actual path is resolved at runtime relative to the CLI location
THREEPIO_IPC_PATH=/tmp/xyz.jsonl pytest -p /absolute/path/to/3pio/dist/adapters/pytest_adapter.py tests/test_math.py -v
```

## Simplified Approach

Since 3pio shells out to run tests (like Jest/Vitest), many complexities are eliminated:

### What We DON'T Need to Handle
- Python version management (uses active Python)
- Virtual environments (uses active environment)  
- pytest configuration files (pytest reads them normally)
- Plugin compatibility (all plugins work normally)
- Tox/other wrappers (just pass through the command)
- Cross-language dependencies (pip and npm packages are independent)

### What We DO Need to Handle
1. **Detect pytest usage** in the command
2. **Find bundled pytest adapter** within 3pio installation
3. **Inject `-p /path/to/adapter.py`** into the command
4. **Set THREEPIO_IPC_PATH** environment variable
5. **Parse pytest output** for test boundaries

### Installation Model
- **JavaScript developers**: `npm install -g 3pio` - Full support for Jest, Vitest, and pytest
- **Python developers**: `pip install 3pio` - Same package, Python-native installation
- **Both installations are identical** - Same CLI, same adapters, same functionality

## Testing Strategy

### Unit Testing
- Mock pytest output for parser testing
- Test command detection with various patterns
- Verify IPC event generation
- Test error handling and edge cases

### Integration Testing
- Create minimal pytest projects for testing
- Test with different pytest plugins (pytest-xdist, pytest-cov)
- Verify report generation accuracy
- Test with parametrized and fixture-based tests

### End-to-End Testing
- Real pytest projects with various configurations
- Complex test suites with multiple files
- Performance testing with large test suites
- Cross-platform testing (macOS, Linux, Windows)

## Potential Challenges

### 1. Locating Bundled Adapter
- **Challenge**: The CLI needs to find the bundled pytest adapter relative to its own location
- **Solution**: Use consistent relative path (`../adapters/pytest_adapter.py` from `dist/cli.js`) regardless of installation method. The adapter is always in the same position relative to the CLI in both npm and pip installations.

### 2. Output Format Variations  
- **Challenge**: pytest output varies with plugins/versions
- **Solution**: Robust parsing patterns, focus on standard markers (PASSED/FAILED/SKIPPED)

### 3. Cross-Language IPC
- **Challenge**: Python plugin writing to same IPC file as TypeScript
- **Solution**: JSON Lines format with atomic writes, same as JS adapters

### 4. Silent Mode in Python
- **Challenge**: Suppressing all pytest output from plugin
- **Solution**: Redirect sys.stdout/stderr, disable pytest terminal reporter

## Next Steps

1. **Create TypeScript side first** - PyTestDefinition and OutputParser
2. **Create fixture projects** - Test fixtures in `tests/fixtures/` for testing
3. **Bundle Python adapter** - Include pytest_adapter.py in build
4. **Test integration** - Use fixture projects for end-to-end tests
5. **Set up pip distribution** - Create pyproject.toml for PyPI
6. **Update documentation** - Installation and usage guides

## Impact

This pytest implementation establishes the pattern for expanding 3pio to other language ecosystems. Each new test runner follows the same approach:
- TypeScript runner definition for detection and command building
- Language-specific adapter bundled with the package
- Distribution through that language's native package manager
- Same CLI, same functionality, ecosystem-appropriate installation

See `docs/future-vision.md` for the complete roadmap of planned test runners and distribution strategies.