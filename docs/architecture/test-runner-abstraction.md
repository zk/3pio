# Test Runner Abstraction Architecture

## 1. Overview

The test runner abstraction layer provides a pluggable architecture for supporting multiple test frameworks (Jest, Vitest, and future runners) without modifying core components. This design follows the strategy pattern with explicit, compile-time known test runners.

## 2. Core Components

### TestRunnerDefinition Interface
Defines the contract for test runner implementations:
- **matches**: Determines if a command uses this test runner
- **getTestFiles**: Discovers test files (static or dynamic mode)
- **buildMainCommand**: Injects adapter into command arguments
- **getAdapterFileName**: Returns the adapter file name
- **interpretExitCode**: Maps exit codes to semantic meanings

### OutputParser Abstract Class
Handles runner-specific console output parsing:
- **parseTestOutput**: Extracts test boundaries from output
- **extractTestFileFromLine**: Identifies file associations
- **isEndOfTestOutput**: Detects test completion markers
- **formatTestHeading**: Processes test section headers

### TestRunnerManager
Central registry and detection logic:
- Static TEST_RUNNERS object with all implementations
- Detection method that checks each runner
- Accessor methods for definitions and parsers
- Type-safe runner name handling

## 3. Implementation Structure

### Directory Organization
```
src/runners/
├── base/
│   ├── TestRunnerDefinition.ts    # Interface definition
│   └── OutputParser.ts             # Abstract parser class
├── jest/
│   ├── JestDefinition.ts          # Jest-specific implementation
│   └── JestOutputParser.ts        # Jest output parsing
└── vitest/
    ├── VitestDefinition.ts        # Vitest-specific implementation
    └── VitestOutputParser.ts      # Vitest output parsing
```

### Registration Pattern
Test runners are explicitly registered in TestRunnerManager:
- Compile-time known set of runners
- Type-safe runner names
- No runtime discovery or plugin loading
- Clear, predictable behavior

## 4. Detection Strategy

### Command Detection
Each runner implements pattern matching for:
- Direct invocation (jest, vitest)
- Package manager invocation (npx, yarn, pnpm)
- npm scripts (npm test, npm run test)

### Package.json Analysis
For abstract commands (npm test):
- Parse scripts section for runner references
- Check dependencies for installed runners
- Fallback to command-line patterns

### Priority Order
Runners checked in specific order:
1. Jest (most common)
2. Vitest (growing adoption)
3. Future runners as added

## 5. Test File Discovery

### Static Discovery
When test files can be determined upfront:
- Jest: Uses --listTests dry run
- Explicit file arguments in command
- Returns complete file list before execution

### Dynamic Discovery
When files discovered during execution:
- Vitest: list command unreliable
- npm run commands without file lists
- Returns empty array, files tracked as they run

## 6. Command Building

### Adapter Injection
Each runner defines how to add its adapter:
- Jest: --reporters flag (adapter only, no default)
- Vitest: --reporter flag with both default and adapter
- Preserves existing reporter configurations if already specified
- Uses absolute paths for adapters

### Argument Preservation
Original command structure maintained:
- User flags preserved
- File arguments kept in position
- Environment variables passed through
- Shell features supported via zx

## 7. Output Parsing

### Runner-Specific Patterns
Each parser handles its runner's format:
- Jest: Worker process output format
- Vitest: stdout/stderr prefixed lines
- Test boundaries and file associations
- Error message extraction

### Common Base Functionality
Abstract OutputParser provides:
- Line-by-line processing
- Buffer management
- Test file path normalization
- Output accumulation strategies

## 8. Integration Points

### CLI Orchestrator
Uses TestRunnerManager for:
- Runner detection from commands
- Test file discovery
- Command modification
- Exit code interpretation

### Report Manager
Uses OutputParser for:
- Parsing console output into test logs
- Associating output with test files
- Extracting test boundaries
- Formatting test results

### Adapters
Remain independent but follow conventions:
- Read adapter path from environment
- Use IPC for communication
- Silent operation (no console output)
- Error resilience

## 9. Adding New Test Runners

### Implementation Steps
1. Create runner directory under src/runners/
2. Implement TestRunnerDefinition interface
3. Implement OutputParser subclass
4. Register in TEST_RUNNERS object
5. Create adapter in src/adapters/
6. Add build configuration
7. Write tests for new components

### Required Components
- Definition class with detection logic
- Parser for output handling
- Adapter for test runner integration
- Unit and integration tests
- Documentation updates

## 10. Benefits

### Maintainability
- Single responsibility per component
- Clear interfaces and contracts
- Isolated test runner logic
- Reduced coupling

### Extensibility
- New runners don't modify existing code
- Explicit registration pattern
- Well-defined extension points
- Type safety throughout

### Testability
- Each component independently testable
- Mock-friendly interfaces
- Isolated business logic
- Clear test boundaries

## 11. Design Decisions

### Static vs Dynamic Registration
Chose static registration for:
- Compile-time type safety
- Predictable behavior
- Easier debugging
- No runtime surprises

### Interface Segregation
Separate interfaces for:
- Test runner operations (Definition)
- Output parsing (Parser)
- Adapter behavior (separate concern)
- Clear separation of concerns

### Explicit Over Implicit
- No auto-discovery of runners
- Clear registration required
- Predictable detection order
- Explicit error messages

## 12. Future Considerations

### Potential Enhancements
- Plugin architecture for external runners
- Configuration file for runner settings
- Custom detection strategies
- Output parser composition

### Scalability
Current design supports:
- 10+ test runners without refactoring
- Custom runners via interface implementation
- Extension through composition
- Performance optimization points identified