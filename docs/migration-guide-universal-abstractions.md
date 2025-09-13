# Migration Guide: File-Centric to Universal Group Abstractions

This guide explains the migration from 3pio's file-centric model to the new universal group abstractions system completed in v2.0.0.

## Overview of Changes

### Previous Model (File-Centric)
```
Test Run
└── File (primary abstraction)
    └── Test Case
```

### New Model (Group-Based)
```
Test Run
└── Group (universal abstraction)
    └── Subgroup(s) (nested, optional)
        └── Test Case
```

## Key Differences

### 1. Event Schema Changes

#### Old Events (Removed)
- `testFileStart` - File begins execution
- `testFileResult` - File completes
- `stdoutChunk` - Console output per file
- `stderrChunk` - Error output per file
- `testCase` (old format) - Test with file reference

#### New Events (Current)
- `testGroupDiscovered` - Group hierarchy detected
- `testGroupStart` - Group begins execution
- `testGroupResult` - Group completes with statistics
- `testCase` (new format) - Test with parent hierarchy
- `groupStdout` / `groupStderr` - Output per group (optional)

### 2. Event Structure Examples

#### Old testCase Event
```json
{
  "eventType": "testCase",
  "payload": {
    "filePath": "src/math.test.js",
    "testName": "should add numbers",
    "suiteName": "Math operations",
    "status": "PASS",
    "duration": 0.023
  }
}
```

#### New testCase Event
```json
{
  "eventType": "testCase",
  "payload": {
    "testName": "should add numbers",
    "parentNames": ["src/math.test.js", "Math operations"],
    "status": "PASS",
    "duration": 0.023,
    "stdout": "optional captured stdout",
    "stderr": "optional captured stderr"
  }
}
```

### 3. Report Structure Changes

#### Old Structure
```
.3pio/runs/[runID]/
├── test-run.md
├── output.log
└── reports/
    ├── math.test.js.log      # Flat file reports
    └── string.test.js.log
```

#### New Structure
```
.3pio/runs/[runID]/
├── test-run.md
├── output.log
└── reports/
    ├── src_math_test_js/      # Hierarchical group directories
    │   ├── index.md           # File-level tests
    │   └── math_operations/    # Nested groups
    │       └── index.md
    └── test_string_py/
        └── teststring.md
```

## Migration Steps for Custom Adapters

If you have written custom test runner adapters, follow these steps:

### 1. Update Event Emission

Replace file-based events with group events:

```javascript
// OLD - Don't use
sendEvent({
  eventType: 'testFileStart',
  payload: { filePath: 'src/app.test.js' }
});

// NEW - Use this
sendEvent({
  eventType: 'testGroupDiscovered',
  payload: {
    groupName: 'src/app.test.js',
    parentNames: []
  }
});

sendEvent({
  eventType: 'testGroupStart',
  payload: {
    groupName: 'src/app.test.js',
    parentNames: []
  }
});
```

### 2. Send Complete Hierarchy

For nested test structures, send the full hierarchy:

```javascript
// Discover all ancestor groups
const hierarchy = ['src/app.test.js', 'App Suite', 'Login Tests'];

// Send discovery for each level
for (let i = 0; i < hierarchy.length; i++) {
  sendEvent({
    eventType: 'testGroupDiscovered',
    payload: {
      groupName: hierarchy[i],
      parentNames: hierarchy.slice(0, i)
    }
  });
}
```

### 3. Update Test Case Events

Include parent hierarchy in test case events:

```javascript
// OLD - Don't use
sendEvent({
  eventType: 'testCase',
  payload: {
    filePath: 'src/app.test.js',
    testName: 'should login',
    suiteName: 'Login Tests',
    status: 'PASS'
  }
});

// NEW - Use this
sendEvent({
  eventType: 'testCase',
  payload: {
    testName: 'should login',
    parentNames: ['src/app.test.js', 'App Suite', 'Login Tests'],
    status: 'PASS',
    duration: 0.123
  }
});
```

### 4. Send Group Results

When a group completes, send result with statistics:

```javascript
sendEvent({
  eventType: 'testGroupResult',
  payload: {
    groupName: 'Login Tests',
    parentNames: ['src/app.test.js', 'App Suite'],
    status: 'PASS',
    duration: 2.456,
    totals: {
      passed: 5,
      failed: 0,
      skipped: 1
    }
  }
});
```

## Migration Steps for Report Consumers

If you have tools that consume 3pio reports:

### 1. Update Report Paths

Reports are now in hierarchical directories:

```bash
# OLD path pattern
.3pio/runs/*/reports/math.test.js.log

# NEW path pattern
.3pio/runs/*/reports/src_math_test_js/index.md
.3pio/runs/*/reports/src_math_test_js/math_operations/index.md
```

### 2. Parse New Report Format

Reports now use Markdown with YAML frontmatter:

```markdown
---
group_name: Math operations
parent_path: src_math_test_js
status: PASS
duration: 1.23s
---

# Results for `math.test.js > Math operations`

## Test case results

- ✓ should add numbers (12ms)
- ✓ should subtract numbers (8ms)
```

### 3. Handle Hierarchical Structure

Groups can have both direct tests and subgroups:

```markdown
## Test case results

- ✓ direct test at this level (5ms)

## Subgroups

| Status | Name | Tests | Duration | Report |
|--------|------|-------|----------|--------|
| PASS | Addition | 3 passed | 45ms | ./addition.md |
| PASS | Subtraction | 2 passed | 38ms | ./subtraction.md |
```

## Benefits of the New Model

### 1. Universal Compatibility
- Works naturally with all test runners
- Preserves native test organization
- No forced file-centric mapping

### 2. Better Organization
- Nested test hierarchies preserved
- Natural grouping (describes, suites, classes)
- More intuitive report structure

### 3. Cleaner Architecture
- Single abstraction (groups) instead of multiple concepts
- Simpler event model
- Consistent handling across all runners

### 4. Enhanced Features
- Hierarchical console output
- Group-level statistics
- Better support for parallel execution
- Test-level stdout/stderr capture (when available)

## Backward Compatibility

The system temporarily maintained backward compatibility during migration:
- Old events were converted to new group events
- File paths were treated as root-level groups
- This compatibility layer has been removed in the final version

## Common Patterns

### Jest/Vitest Hierarchy
```
file.test.js (group)
└── describe "Feature" (group)
    └── describe "Scenario" (group)
        └── it "should work" (test)
```

### pytest Hierarchy
```
test_file.py (group)
└── TestClass (group)
    └── test_method (test)
```

### Go Test Hierarchy
```
package (group)
└── TestFunction (group)
    └── t.Run("subtest") (group)
        └── assertion (implicit test)
```

## Troubleshooting

### Missing Groups in Reports
Ensure adapters send `testGroupDiscovered` events for all ancestor groups before sending test results.

### Incorrect Hierarchy
Check that `parentNames` arrays correctly represent the full path from root to immediate parent.

### Performance Issues
Group discovery events are idempotent - sending duplicates is safe and handled efficiently.

## Further Reading

- [Universal Test Abstractions Documentation](./universal-test-abstractions.md)
- [Architecture Overview](./architecture/architecture.md)
- [Writing Adapters Guide](./architecture/writing-adapters.md)