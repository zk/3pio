# Unified Vitest Adapter Plan

## Goal
Create a single Vitest adapter that supports versions 1.x, 2.x, and 3.x to maximize compatibility with the ecosystem.

## Current State
- Adapter supports Vitest 2.x and 3.x (runtime version detection)
- Uses Vitest 3-specific reporter APIs when available; falls back to v2 processing on `onFinished`
- Major frameworks still on older versions (Redux and Svelte use 2.x)

## Ecosystem Analysis

### Version Distribution
- **Vitest 3.x**: Vue.js Core (3.2.4), TanStack Query (3.1.3), VueUse
- **Vitest 2.x**: Redux (2.1.9), Svelte (2.1.9) - significant usage
- **Vitest 1.x**: Mostly deprecated, few major projects remain

### Market Impact
- Supporting v2.x would immediately enable testing for Redux (61k stars) and Svelte (79k stars)
- v3.x support covers cutting-edge projects
- v1.x support provides legacy compatibility

## Implementation Strategy

### 1. Version Detection
```javascript
const vitestVersion = require('vitest/package.json').version;
const majorVersion = parseInt(vitestVersion.split('.')[0], 10);

// Determine which API surface to use
const useV2Api = majorVersion < 3;
const useV3Api = majorVersion >= 3;
```

### 2. API Strategy Mapping

**CRITICAL**: Vitest 3.0 completely removed `onTaskUpdate` and other v2 methods. We must implement both API surfaces completely.

#### Vitest 1.x Strategy (if needed)
- Primary: `onTaskUpdate` for all test events
- Extract test results from task tree structure
- Limited API surface

#### Vitest 2.x Strategy
- **Available Methods**: `onTaskUpdate`, `onFinished`, `onCollected`, `onPathsCollected`
- Primary: `onTaskUpdate(packs: TaskResultPack[])` for test events
- Extract test results from `TaskResultPack[]` with nested task trees
- Use `onFinished` for final cleanup

#### Vitest 3.x Strategy
- **Available Methods**: `onTestCaseResult`, `onTestModuleEnd`, `onTestRunStart/End`, `onTestSuiteResult`
- **REMOVED**: `onTaskUpdate` (does not exist in v3)
- Primary: `onTestCaseResult(testCase)` for individual test results
- Direct access via `testCase.result()` and `testCase.diagnostic()`
- Use `onTestModuleEnd` for file completion

### 3. Unified Event Handler
Create abstraction layer that normalizes events across versions:

```javascript
class UnifiedTestHandler {
  handleTestResult(testData, version) {
    // Convert version-specific format to 3pio IPC events
    const normalized = this.normalizeTestData(testData, version);
    IPCSender.sendEvent(normalized);
  }

  normalizeTestData(data, version) {
    switch(version) {
      case 1: return this.normalizeV1(data);
      case 2: return this.normalizeV2(data);
      case 3: return this.normalizeV3(data);
    }
  }
}
```

### 4. Reporter Implementation

#### Dual API Surface Implementation
Must implement BOTH v2 and v3 API surfaces since methods don't overlap:

```javascript
class ThreePioVitestReporter {
  constructor() {
    this.version = this.detectVersion();
    this.useV2Api = this.version < 3;
    this.useV3Api = this.version >= 3;
    this.handler = new UnifiedTestHandler();
  }

  // ===== V2 API Methods (completely removed in v3) =====

  onTaskUpdate(packs) {
    if (!this.useV2Api) return; // Method doesn't exist in v3

    packs.forEach(pack => {
      // Extract test results from TaskResultPack
      this.processV2TaskPack(pack);
    });
  }

  onFinished(files, errors) {
    if (!this.useV2Api) return;
    // Handle v2 test run completion
  }

  onCollected(files) {
    if (!this.useV2Api) return;
    // Handle v2 file collection
  }

  // ===== V3 API Methods (new in v3) =====

  onTestCaseResult(testCase) {
    if (!this.useV3Api) return; // Method doesn't exist in v2

    const result = testCase.result();
    const diagnostic = testCase.diagnostic();
    this.handler.handleTestResult({
      name: testCase.name,
      status: result.state,
      duration: diagnostic?.duration,
      error: result.errors?.[0]
    }, 3);
  }

  onTestModuleEnd(testModule) {
    if (!this.useV3Api) return;
    // Handle v3 module completion
  }

  onTestRunEnd(modules, errors) {
    if (!this.useV3Api) return;
    // Handle v3 test run completion
  }
}
```

### 5. Version-Specific Handling

#### Key Differences to Address

**V1 → V2 Changes:**
- Task structure improvements
- Hook execution timing
- Error reporting format

**V2 → V3 Breaking Changes:**
- **REMOVED APIs**: `onTaskUpdate`, `onFinished`, `onCollected`, `onPathsCollected`, `onSpecsCollected`
- **NEW APIs**: `onTestCaseResult`, `onTestModuleEnd`, `onTestRunStart/End`, `onTestSuiteResult`
- Completely different data structures:
  - v2: `TaskResultPack[]` with nested task trees
  - v3: Direct `testCase.result()` and `testCase.diagnostic()` methods
- Different event flow and lifecycle hooks
- Bundled chai as ESM
- Default pool configuration

#### Compatibility Layer
- **Cannot polyfill** - v2 and v3 APIs are completely different
- Must implement both API surfaces natively
- Normalize data internally before sending to IPC:
  - Extract test data from v2 `TaskResultPack`
  - Extract test data from v3 `testCase.result()`
  - Convert both to unified IPC format

### 6. Comprehensive Testing Strategy

#### Test Categories

##### 1. Unit Tests
Test individual components in isolation:

**Version Detection Tests**
```javascript
describe('Version Detection', () => {
  test('correctly identifies Vitest 2.x', () => {
    mockVitestVersion('2.1.9');
    const reporter = new ThreePioVitestReporter();
    expect(reporter.useV2Api).toBe(true);
    expect(reporter.useV3Api).toBe(false);
  });

  test('correctly identifies Vitest 3.x', () => {
    mockVitestVersion('3.2.4');
    const reporter = new ThreePioVitestReporter();
    expect(reporter.useV2Api).toBe(false);
    expect(reporter.useV3Api).toBe(true);
  });
});
```

**API Surface Tests**
```javascript
describe('API Surface Activation', () => {
  test('v2 adapter ignores v3 methods', () => {
    const reporter = new ThreePioVitestReporter();
    reporter.useV2Api = true;
    reporter.useV3Api = false;

    const spy = jest.spyOn(reporter.handler, 'handleTestResult');
    reporter.onTestCaseResult(mockTestCase); // v3 method
    expect(spy).not.toHaveBeenCalled();
  });

  test('v3 adapter ignores v2 methods', () => {
    const reporter = new ThreePioVitestReporter();
    reporter.useV2Api = false;
    reporter.useV3Api = true;

    const spy = jest.spyOn(reporter, 'processV2TaskPack');
    reporter.onTaskUpdate([mockTaskPack]); // v2 method
    expect(spy).not.toHaveBeenCalled();
  });
});
```

##### 2. Data Extraction Tests

**V2 TaskResultPack Processing**
```javascript
describe('V2 Data Extraction', () => {
  test('extracts test results from TaskResultPack', () => {
    const taskPack = {
      id: 'test-1',
      type: 'test',
      name: 'should add numbers',
      state: 'passed',
      duration: 5,
      result: { state: 'passed' }
    };

    const result = extractV2TestData(taskPack);
    expect(result).toEqual({
      name: 'should add numbers',
      status: 'PASS',
      duration: 5,
      error: null
    });
  });

  test('handles nested suite structure', () => {
    const taskPack = {
      type: 'suite',
      name: 'Math Utils',
      tasks: [
        { type: 'test', name: 'addition', state: 'passed' },
        { type: 'test', name: 'subtraction', state: 'failed' }
      ]
    };

    const results = extractV2SuiteData(taskPack);
    expect(results).toHaveLength(2);
    expect(results[0].parentNames).toContain('Math Utils');
  });
});
```

**V3 TestCase Processing**
```javascript
describe('V3 Data Extraction', () => {
  test('extracts test results from testCase.result()', () => {
    const mockTestCase = {
      name: 'should add numbers',
      result: () => ({ state: 'passed', errors: [] }),
      diagnostic: () => ({ duration: 5 })
    };

    const result = extractV3TestData(mockTestCase);
    expect(result).toEqual({
      name: 'should add numbers',
      status: 'PASS',
      duration: 5,
      error: null
    });
  });

  test('handles error extraction', () => {
    const mockTestCase = {
      name: 'should fail',
      result: () => ({
        state: 'failed',
        errors: [{
          message: 'Expected 2 to be 3',
          stack: 'Error: Expected 2 to be 3\n  at test.js:10'
        }]
      })
    };

    const result = extractV3TestData(mockTestCase);
    expect(result.error.message).toBe('Expected 2 to be 3');
  });
});
```

##### 3. Integration Tests

**Test Fixture Structure**
```
tests/fixtures/
├── vitest-2.x/
│   ├── basic-tests/
│   ├── nested-suites/
│   ├── parallel-tests/
│   └── failing-tests/
├── vitest-3.x/
│   ├── basic-tests/
│   ├── nested-suites/
│   ├── concurrent-tests/
│   └── snapshot-tests/
└── cross-version/
    └── identical-tests/  # Same tests for both versions
```

**Cross-Version Consistency Tests**
```javascript
describe('Cross-Version Consistency', () => {
  const testFile = 'tests/fixtures/cross-version/math.test.js';

  test('produces identical IPC events for v2 and v3', async () => {
    // Run with v2
    const v2Events = await runWithVitest('2.1.9', testFile);

    // Run with v3
    const v3Events = await runWithVitest('3.2.4', testFile);

    // Normalize timestamps and durations
    const normalize = (events) => events.map(e => ({
      ...e,
      payload: { ...e.payload, duration: undefined }
    }));

    expect(normalize(v2Events)).toEqual(normalize(v3Events));
  });
});
```

##### 4. Real-World Project Tests

**Redux (v2.1.9) Test Suite**
```javascript
describe('Redux Integration (v2)', () => {
  beforeAll(async () => {
    await setupReduxProject();
  });

  test('runs Redux test suite with 3pio adapter', async () => {
    const result = await exec('3pio yarn vitest --run');
    expect(result.exitCode).toBe(0);

    const report = await readReport('.3pio/runs/latest/test-run.md');
    expect(report).toContain('✓ combineReducers');
    expect(report).toContain('✓ createStore');
  });

  test('captures all Redux test files', async () => {
    const ipcEvents = await getIPCEvents();
    const fileGroups = ipcEvents
      .filter(e => e.eventType === 'testGroupDiscovered')
      .filter(e => e.payload.parentNames.length === 0);

    expect(fileGroups.length).toBeGreaterThan(10); // Redux has many test files
  });
});
```

**Vue Core (v3.2.4) Test Suite**
```javascript
describe('Vue Core Integration (v3)', () => {
  test('handles Vue Core's complex test structure', async () => {
    const result = await exec('3pio yarn test-unit');

    // Vue Core has thousands of tests
    const ipcEvents = await getIPCEvents();
    const testCases = ipcEvents.filter(e => e.eventType === 'testCase');
    expect(testCases.length).toBeGreaterThan(1000);
  });
});
```

##### 5. Performance & Memory Tests

```javascript
describe('Performance', () => {
  test('handles 10,000 tests without memory leak', async () => {
    const initialMemory = process.memoryUsage().heapUsed;

    await runLargeTestSuite(10000);

    global.gc(); // Force garbage collection
    const finalMemory = process.memoryUsage().heapUsed;

    // Memory should not grow more than 50MB
    expect(finalMemory - initialMemory).toBeLessThan(50 * 1024 * 1024);
  });

  test('IPC writing performance', async () => {
    const start = Date.now();

    // Simulate 1000 test events
    for (let i = 0; i < 1000; i++) {
      await IPCSender.sendEvent({
        eventType: 'testCase',
        payload: { /* ... */ }
      });
    }

    const duration = Date.now() - start;
    expect(duration).toBeLessThan(1000); // Should complete within 1 second
  });
});
```

##### 6. Edge Case Tests

```javascript
describe('Edge Cases', () => {
  test('handles missing version info gracefully', () => {
    delete require.cache[require.resolve('vitest/package.json')];
    const reporter = new ThreePioVitestReporter();
    // Should default to v3 behavior
    expect(reporter.useV3Api).toBe(true);
  });

  test('handles malformed TaskResultPack', () => {
    const malformed = { /* missing required fields */ };
    expect(() => extractV2TestData(malformed)).not.toThrow();
  });

  test('handles concurrent test execution', async () => {
    // Test with maxThreads > 1
    const result = await exec('3pio vitest --run --pool=threads --poolOptions.threads.maxThreads=4');
    expect(result.exitCode).toBe(0);
  });
});
```

##### 7. CI/CD Pipeline Configuration

**GitHub Actions Workflow**
```yaml
name: Test Unified Adapter

on: [push, pull_request]

jobs:
  test-matrix:
    strategy:
      matrix:
        vitest-version: ['2.0.5', '2.1.9', '3.0.0', '3.1.3', '3.2.4']
        node-version: ['18', '20']
        os: [ubuntu-latest, windows-latest, macos-latest]

    runs-on: ${{ matrix.os }}

    steps:
      - uses: actions/checkout@v3

      - name: Setup Node.js
        uses: actions/setup-node@v3
        with:
          node-version: ${{ matrix.node-version }}

      - name: Install Vitest version
        run: npm install vitest@${{ matrix.vitest-version }}

      - name: Run adapter tests
        run: npm test

      - name: Test with real projects
        run: |
          if [[ "${{ matrix.vitest-version }}" == "2."* ]]; then
            ./scripts/test-redux.sh
            ./scripts/test-svelte.sh
          else
            ./scripts/test-vue.sh
            ./scripts/test-tanstack.sh
          fi
```

##### 8. Regression Testing

```javascript
describe('Regression Tests', () => {
  test('v3 support not broken by v2 additions', async () => {
    // Run existing v3 test suite
    const result = await exec('npm run test:v3-only');
    expect(result.exitCode).toBe(0);
  });

  test('maintains backward compatibility', async () => {
    // Test that old IPC event format still works
    const legacyEvent = { /* v0.1 format */ };
    expect(() => processIPCEvent(legacyEvent)).not.toThrow();
  });
});
```

#### Test Execution Strategy

```bash
# Local development testing
make test-adapter           # Run all adapter tests
make test-adapter-v2        # Test only v2 compatibility
make test-adapter-v3        # Test only v3 compatibility
make test-adapter-projects  # Test with real projects

# CI testing
npm run test:ci             # Full test suite with coverage
npm run test:integration    # Integration tests only
npm run test:performance    # Performance benchmarks

# Manual testing script
./scripts/test-all-versions.sh
```

#### Coverage Requirements
- Unit tests: 90% coverage minimum
- Integration tests: Must cover all reporter methods
- Real-world tests: Redux, Svelte (v2), Vue, TanStack (v3)
- Cross-platform: Linux, macOS, Windows

#### Critical Test Scenarios

##### Version-Specific Failure Modes

**V2 Specific Challenges**
```javascript
describe('V2 Specific Challenges', () => {
  test('handles deeply nested task trees', () => {
    // V2 uses nested TaskResultPack structures
    const deeplyNested = createNestedTaskPack(10); // 10 levels deep
    const results = extractV2TestData(deeplyNested);
    expect(results).toBeDefined();
    expect(results.parentNames).toHaveLength(10);
  });

  test('processes task updates in correct order', () => {
    // V2 may send multiple onTaskUpdate calls
    const updates = [];
    reporter.onTaskUpdate([pack1]); // First update
    reporter.onTaskUpdate([pack2]); // Second update

    // Ensure updates are processed sequentially
    expect(ipcEvents[0].payload.testName).toBe('test1');
    expect(ipcEvents[1].payload.testName).toBe('test2');
  });

  test('handles incomplete TaskResultPack', () => {
    // V2 might send partial updates
    const incomplete = { type: 'test', name: 'test' }; // Missing state
    expect(() => extractV2TestData(incomplete)).not.toThrow();
  });
});
```

**V3 Specific Challenges**
```javascript
describe('V3 Specific Challenges', () => {
  test('handles rapid test execution', () => {
    // V3 may call onTestCaseReady and onTestCaseResult in same tick
    const testCase = createMockTestCase();

    reporter.onTestCaseReady(testCase);
    reporter.onTestCaseResult(testCase); // Immediately after

    // Should handle both without issues
    expect(ipcEvents).toHaveLength(1); // Only one test event
  });

  test('extracts errors from multiple error formats', () => {
    // V3 has different error structures
    const errors = [
      new Error('Standard error'),
      { message: 'Object error', stack: 'stack' },
      'String error'
    ];

    errors.forEach(error => {
      const result = extractV3Error(error);
      expect(result.message).toBeDefined();
    });
  });

  test('handles missing diagnostic data', () => {
    const testCase = {
      result: () => ({ state: 'passed' }),
      diagnostic: () => null // No diagnostic
    };

    const result = extractV3TestData(testCase);
    expect(result.duration).toBeUndefined();
  });
});
```

##### Cross-Version Validation

```javascript
describe('Cross-Version IPC Consistency', () => {
  const testScenarios = [
    { name: 'simple passing test', v2File: 'pass.v2.json', v3File: 'pass.v3.json' },
    { name: 'failing with assertion', v2File: 'fail.v2.json', v3File: 'fail.v3.json' },
    { name: 'skipped test', v2File: 'skip.v2.json', v3File: 'skip.v3.json' },
    { name: 'nested suites', v2File: 'nested.v2.json', v3File: 'nested.v3.json' },
    { name: 'concurrent tests', v2File: 'concurrent.v2.json', v3File: 'concurrent.v3.json' }
  ];

  testScenarios.forEach(scenario => {
    test(`${scenario.name} produces consistent IPC events`, () => {
      const v2Result = processV2Data(loadFixture(scenario.v2File));
      const v3Result = processV3Data(loadFixture(scenario.v3File));

      // Normalize and compare
      expect(normalizeIPCEvents(v2Result)).toEqual(normalizeIPCEvents(v3Result));
    });
  });
});
```

##### Failure Recovery Tests

```javascript
describe('Failure Recovery', () => {
  test('continues after reporter method throws', () => {
    const reporter = new ThreePioVitestReporter();

    // Simulate error in v2 processing
    jest.spyOn(reporter, 'processV2TaskPack').mockImplementation(() => {
      throw new Error('Processing failed');
    });

    expect(() => reporter.onTaskUpdate([pack])).not.toThrow();
    // Should log error but continue
  });

  test('handles IPC write failures gracefully', () => {
    // Simulate file system error
    jest.spyOn(fs, 'appendFileSync').mockImplementation(() => {
      throw new Error('ENOSPC: no space left');
    });

    const reporter = new ThreePioVitestReporter();
    expect(() => reporter.sendIPCEvent(event)).not.toThrow();
  });

  test('recovers from version detection failure', () => {
    // Simulate missing package.json
    jest.mock('vitest/package.json', () => {
      throw new Error('Cannot find module');
    });

    const reporter = new ThreePioVitestReporter();
    // Should default to v3
    expect(reporter.useV3Api).toBe(true);
  });
});
```

#### Validation Criteria

1. **IPC Event Consistency**: Same test produces identical IPC events in v2 and v3
2. **No Data Loss**: All test results are captured regardless of version
3. **Performance Parity**: v2 and v3 adapters have similar performance characteristics
4. **Error Resilience**: Adapter continues functioning despite errors
5. **Backward Compatibility**: Existing v3 functionality unchanged

### 7. Rollout Plan

#### Phase 1: v2.x Support (Priority)
- Extend adapter to support v2.x
- Test with Redux and Svelte
- Document version differences

#### Phase 2: v1.x Support (Optional)
- Add v1.x compatibility if needed
- Focus on common v1.6 patterns
- Provide migration guidance

#### Phase 3: Future-Proofing
- Design for v4+ extensibility
- Monitor Vitest roadmap
- Add version strategies as needed

## Benefits
- **Wider Adoption**: Works with majority of Vitest projects
- **Single Codebase**: Easier maintenance than multiple adapters
- **Graceful Degradation**: Falls back to compatible APIs
- **Future-Proof**: Extensible architecture for new versions

## Trade-offs
- **Complexity**: Must maintain two complete API implementations (not just strategies)
- **Code Size**: Adapter will be larger due to dual API surfaces
- **Testing Burden**: Must test both v2 and v3 APIs thoroughly
- **Bug Surface**: Completely different code paths for v2 vs v3
- **Maintenance**: Breaking changes between versions require separate implementations

## Success Criteria
- [ ] Redux tests run successfully with v2.x
- [ ] Svelte tests run successfully with v2.x
- [ ] Vue.js Core tests run successfully with v3.x
- [ ] No regression for existing v3.x support
- [ ] Clear version compatibility documentation
- [ ] Automated tests for all major versions

## Decision
Proceed with unified adapter supporting v2.x and v3.x initially, with v1.x as stretch goal based on user demand.

**Critical Implementation Note**: The removal of `onTaskUpdate` in v3.x means we cannot use a simple strategy pattern. We must implement both v2 and v3 reporter APIs as completely separate code paths within the same adapter class.
