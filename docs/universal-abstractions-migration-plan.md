# Universal Abstractions Migration Plan

## Executive Summary

This plan outlines the migration from 3pio's current file-centric model to a universal group abstraction that better supports diverse test runners and languages. The new model introduces a flexible hierarchy: **Test Run** > **Groups** (with optional nested subgroups) > **Test Cases**.

## Goals

1. **Universal Compatibility**: Support all test runners with their natural organization
2. **Accurate Representation**: Preserve nested test hierarchies where available (Jest describes, Go subtests, RSpec contexts)
3. **Clean Architecture**: Full replacement of file-centric model with group abstractions
4. **Simplicity**: Minimize complexity - no group types, pragmatic path handling

## Current State vs Target State

### Current State (File-Centric)
```
Test Run
└── File (primary abstraction)
    └── Test Case
```
- Works well: JavaScript, Python, Ruby, PHP
- Problematic: Go, Rust, Java, C++

### Target State (Group-Centric)
```
Test Run
└── Group (no types, just hierarchical organization)
    └── Subgroup(s) (nested, optional)
        └── Test Case
```
- Groups have no types - simpler, more universal
- Test runners without hierarchy support default to flat structure
- Path truncation ensures cross-platform compatibility (260 char limit)

## Group Naming Convention

**Key Principle**: Use test runner's native names without transformation

| Test Runner | Source | Group Name |
|------------|--------|------------|
| Jest | `describe("Button Component")` | "Button Component" |
| pytest | `class TestButton` | "TestButton" |
| Go | `func TestButton` with `t.Run("Component")` | "TestButton" > "Component" |
| Vitest | `describe("API tests")` | "API tests" |

No standardization or transformation - each runner's natural naming is preserved.

## Hierarchy Detection Strategy

**For runners without native hierarchy support:**
- Create **one group per file** (using filename as group name)
- All tests in that file become direct children of the file group
- No attempt to parse test names for hierarchy hints
- Simple, predictable structure

**Example:**
```
Test Run
└── math.test.js (group)
    ├── should add numbers (test case)
    ├── should subtract numbers (test case)
    └── should multiply numbers (test case)
```

## Implementation Phases

### Phase 1: Foundation (Week 1-2)
**Goal**: Establish new data structures for universal abstractions

#### 1.1 Define New IPC Event Schema
```json
// New group events
{
  "eventType": "testGroupStart",
  "payload": {
    "groupName": "Button Component",     // Group's own name
    "parentNames": [                     // Full hierarchy from root
      "src/components/Button.test.js",   // Root (file)
      "UI Components"                    // Parent describe block
    ],
    "metadata": {}                       // runner-specific data
  }
}

// Group discovery event - sent when adapter discovers hierarchy
{
  "eventType": "testGroupDiscovered",
  "payload": {
    "groupName": "Button Component",     // Group's own name
    "parentNames": [                     // Full hierarchy from root to parent
      "src/components/Button.test.js",   // Root (typically file path)
      "UI Components"                    // Parent group
    ]
    // Note: CLI will generate deterministic IDs from the full path
  }
}

{
  "eventType": "testGroupResult",
  "payload": {
    "groupName": "Button Component",
    "parentNames": [                     // Full hierarchy for identification
      "src/components/Button.test.js",
      "UI Components"
    ],
    "status": "PASS|FAIL|SKIP",
    "duration": 1.23,
    "totals": {
      "passed": 10,
      "failed": 2,
      "skipped": 1
    }
  }
}

// Updated test case event
{
  "eventType": "testCase",
  "payload": {
    "testName": "should add numbers",
    "parentNames": [                     // Full hierarchy to parent group
      "src/components/Button.test.js",
      "UI Components",
      "Button Component"
    ],
    "status": "PASS|FAIL|SKIP",
    "duration": 0.023,
    "error": null,
    "stdout": "optional captured stdout",  // optional
    "stderr": "optional captured stderr"   // optional
  }
}
```

#### 1.2 Update Core Data Structures

**internal/report/types.go**:
```go
type TestGroup struct {
    ID           string
    Name         string
    Path         string        // filesystem path if applicable
    ParentID     string        // for nested groups
    Depth        int
    Status       TestStatus
    Duration     time.Duration
    TestCases    []TestCase
    Subgroups    []TestGroup
    StartTime    time.Time
    EndTime      time.Time
}

type TestCase struct {
    ID           string
    GroupID   string        // parent group
    Name         string
    Status       TestStatus
    Duration     time.Duration
    Error        *TestError
    Stdout       string        // optional captured stdout
    Stderr       string        // optional captured stderr
    StartTime    time.Time
    EndTime      time.Time
}

// GroupType removed - groups are type-agnostic for simplicity
```


### Phase 2: Report Manager Updates (Week 2-3)
**Goal**: Support hierarchical group in report generation

#### 2.1 Refactor Report State Management
- Replace file-centric maps with group trees
- Support arbitrary nesting depth
- Maintain group hierarchy in memory
- Store stdout/stderr at test case level when available

#### 2.1.1 Event Processing Strategy
- **Order-independent processing**: Events can arrive in any order
- **On-demand group creation**: Groups created when first referenced
- **Eventually consistent model**: Groups updated incrementally as events arrive
- **No buffering or reordering**: Process events immediately as they come
- **Orphaned test cases**: Log error but continue processing
- **testGroupResult before testGroupStart**: Valid - create/update report with available info

#### 2.1.2 Report Regeneration Strategy
- **When a group completes**: Regenerate its own report file with final statistics
- **Parent update logic**:
  - Check if this was the last pending child of parent
  - If yes: Parent is now complete → mark complete and regenerate parent report (recursive)
  - If no: Parent remains incomplete → no parent update needed
- **Efficiency**: Ancestors only regenerated when they complete, not on every descendant update

#### 2.1.3 Edge Cases & Implementation Details
- **Empty groups**: Still create report file showing "0 total" tests
- **Console output**: output.log remains as complete catchall, no group attribution needed
- **Memory management**: Keep full hierarchy in memory (~82MB for 100k tests)
- **Source of truth**: Hierarchy built incrementally from events as they arrive

#### 2.1.4 Hierarchy Discovery Strategy
Different test runners expose hierarchy differently:
- **Vitest/Jest**: Can walk parent chain from test case
- **pytest**: May only know immediate parent
- **Go**: May only know package name

Adapters should send `testGroupDiscovered` events when they encounter groups to ensure complete hierarchy:
1. When test starts, check if all ancestor groups exist
2. If missing, send `testGroupDiscovered` events to fill gaps
3. This ensures hierarchy is complete even if runner doesn't provide full context

#### 2.1.5 Deterministic Group ID Generation by CLI

##### General Algorithm and Requirements

The CLI (Report Manager) generates deterministic IDs from the group names provided by adapters. Adapters only need to send the complete hierarchy of names:

```go
// CLI-side ID generation from names
func generateGroupId(groupName string, parentNames []string) string {
    // Build full path from root to this group
    fullPath := append(parentNames, groupName)

    // Generate deterministic ID from full path
    pathString := strings.Join(fullPath, ":")
    hash := sha256.Sum256([]byte(pathString))
    return hex.EncodeToString(hash[:16]) // Use first 16 bytes for ID
}
```

**Requirements for adapters:**
1. Extract complete hierarchy of names from test object
2. Send group name and all parent names (from root to immediate parent)
3. Send `testGroupDiscovered` events for all ancestors
4. Events are idempotent - duplicates from parallel workers are safe

**Example adapter pattern:**
```javascript
// Adapter only sends names, not IDs
function sendGroupHierarchy(testObject) {
  const fullPath = extractFullPath(testObject); // ['file.test.js', 'Parent Suite', 'Child Suite']

  // Send discovery events for each level
  for (let i = 0; i < fullPath.length; i++) {
    const groupName = fullPath[i];
    const parentNames = fullPath.slice(0, i); // All ancestors

    sendEvent({
      eventType: 'testGroupDiscovered',
      payload: {
        groupName: groupName,
        parentNames: parentNames
      }
    });
  }
}
```

##### Test Runner Specific Implementations

###### Jest/Vitest
Test object contains complete hierarchy in `ancestorTitles`:
```javascript
// Worker receives test with full context
test = {
  ancestorTitles: ['App Suite', 'Components', 'Button Tests'],
  title: 'should render',
  testPath: '/src/app.test.js'
}

// Send hierarchy with names only
const filePath = test.testPath;  // Root group

// Send discovery for each level
sendEvent({
  eventType: 'testGroupDiscovered',
  payload: {
    groupName: filePath,
    parentNames: []  // File is root
  }
});

for (let i = 0; i < test.ancestorTitles.length; i++) {
  sendEvent({
    eventType: 'testGroupDiscovered',
    payload: {
      groupName: test.ancestorTitles[i],
      parentNames: [filePath, ...test.ancestorTitles.slice(0, i)]
    }
  });
}
```

###### pytest
Test item contains node ID with full path:
```python
# Worker receives item with complete path
item.nodeid = "tests/test_math.py::TestCalculator::test_addition"

# Parse hierarchy
parts = item.nodeid.split("::")
# Result: ['tests/test_math.py', 'TestCalculator', 'test_addition']

file_path = parts[0]
class_name = parts[1] if len(parts) > 2 else None
test_name = parts[-1]

# Send hierarchy with names only
send_event({
    "eventType": "testGroupDiscovered",
    "payload": {
        "groupName": file_path,
        "parentNames": []  # File is root
    }
})

if class_name:
    send_event({
        "eventType": "testGroupDiscovered",
        "payload": {
            "groupName": class_name,
            "parentNames": [file_path]
        }
    })
```

###### Go test
Package and test name available, subtests use slash notation:
```go
// JSON output includes full test path
{
  "Package": "github.com/user/project/math",
  "Test": "TestCalculator/Addition/Positive"
}

// Parse hierarchy
packageName := "github.com/user/project/math"
testParts := strings.Split("TestCalculator/Addition/Positive", "/")
// Result: ['TestCalculator', 'Addition', 'Positive']

// Send hierarchy with names only
sendEvent(Event{
    EventType: "testGroupDiscovered",
    Payload: map[string]interface{}{
        "groupName": packageName,
        "parentNames": []string{}, // Package is root
    },
})

parentNames := []string{packageName}
for _, part := range testParts[:len(testParts)-1] {
    sendEvent(Event{
        EventType: "testGroupDiscovered",
        Payload: map[string]interface{}{
            "groupName": part,
            "parentNames": parentNames,
        },
    })
    parentNames = append(parentNames, part)
}
```

###### RSpec
Example object contains full description hierarchy:
```ruby
# Worker receives example with full context
example.metadata[:full_description] = "App when logged in displays dashboard"
example.metadata[:described_class] = App
example.metadata[:example_group] = {
  :description => "when logged in",
  :parent_example_group => {
    :description => "App"
  }
}

# Walk parent chain to build full path
path = []
group = example.metadata[:example_group]
while group
  path.unshift(group[:description])
  group = group[:parent_example_group]
end
# Result: ['App', 'when logged in']
```

##### Parallel Safety Guarantees

1. **Name-Based Identity**: Adapters only send names, CLI generates IDs
   - Same name hierarchy always produces same ID in the CLI
   - Workers don't need to coordinate ID generation

2. **Idempotent Events**: Multiple `testGroupDiscovered` events for same group are safe
   - CLI (Report Manager) deduplicates based on generated IDs
   - Same name hierarchy → same ID → deduplicated

3. **No Shared State**: Each worker independently sends complete name hierarchy
   - No need for inter-worker communication or shared memory
   - Each worker can discover and report the same groups

4. **Order Independence**: Groups can be discovered in any order
   - Parent groups created on-demand when referenced
   - CLI builds hierarchy incrementally as events arrive

This approach ensures consistent group identification across parallel workers without any coordination overhead. The CLI is the single source of truth for ID generation.

#### 2.2 Filesystem-Safe Naming Strategy

**Sanitization Rules**:
```go
// internal/report/sanitizer.go
func SanitizeGroupName(name string) string {
    // Convert to lowercase
    name = strings.ToLower(name)

    // Replace filesystem-unsafe characters
    replacements := map[string]string{
        " ":  "_",     // Spaces to underscores
        "/":  "_",     // Path separator
        "\\": "_",     // Windows path separator
        ":":  "_",     // Drive letter separator (Windows)
        "*":  "_star", // Wildcard
        "?":  "_q",    // Wildcard
        "\"": "_dq",   // Quote
        "<":  "_lt",   // Less than
        ">":  "_gt",   // Greater than
        "|":  "_pipe", // Pipe
        "\n": "_nl",   // Newline
        "\r": "",      // Carriage return
        "\t": "_tab",  // Tab
        "-":  "_",     // Normalize hyphens to underscores
    }

    // Apply replacements
    safe := name
    for old, new := range replacements {
        safe = strings.ReplaceAll(safe, old, new)
    }

    // Universal handling - replace common separators
    safe = strings.ReplaceAll(safe, "/", "_")
    safe = strings.ReplaceAll(safe, ".", "_")

    // Enforce length limit for all groups
    if len(safe) > 100 {
        safe = safe[:97] + "..."
    }

    // Remove leading/trailing dots and spaces
    safe = strings.Trim(safe, ". ")

    // Ensure non-empty
    if safe == "" {
        safe = "unnamed"
    }

    return safe
}

func GenerateGroupPath(group *TestGroup) string {
    // For nested groups, create hierarchical paths
    parts := []string{}
    current := group

    for current != nil {
        sanitized := SanitizeGroupName(current.Name)
        parts = append([]string{sanitized}, parts...)
        current = current.Parent
    }

    // Build full path
    fullPath := filepath.Join(parts...)

    // Only enforce 260 character limit on Windows
    if runtime.GOOS == "windows" && len(fullPath) > 260 {
        // Truncate from left side, preserve rightmost (most specific) parts
        hash := sha256.Sum256([]byte(fullPath))
        hashPrefix := hex.EncodeToString(hash[:4])
        truncated := fullPath[len(fullPath)-250:]
        fullPath = hashPrefix + "_" + truncated
    }

    return fullPath
}
```

#### 2.3 Output File Structure

**Directory Layout**:
```
.3pio/runs/[runID]/
├── test-run.md                         # Main report
├── output.log                          # Complete stdout/stderr
└── reports/                            # Hierarchical test reports
    ├── github_com_user_project/
    │   └── math.md                     # Contains all tests in math package
    ├── src_components_button_test_js/  # Directory for file group
    │   ├── index.md                    # File-level tests (outside describes)
    │   ├── button_rendering.md         # Tests directly in this describe + links to nested
    │   └── button_rendering/
    │       ├── with_props.md          # Tests in this nested describe
    │       └── without_props.md       # Tests in this nested describe
    └── test_math_py/                   # Directory for file group
        ├── index.md                     # File-level tests (outside classes)
        └── testmathoperations.md       # All test methods in class
```

**Important**: To avoid filesystem conflicts between files and directories with the same name, every group becomes a directory. Direct tests at any level are stored in `index.md` within that directory.

**Key Principle**: Every group gets a report file that:
- Lists all direct test cases (if any)
- Links to all subgroups (if any)
- Shows statistics based on content:
  - **Direct tests only**: Show test statistics only
  - **Both direct tests AND subgroups**: Show test statistics AND subgroup statistics
  - **Subgroups only**: Show subgroup statistics only

**Summary Section Format Rules**:
- **Groups with direct tests only**: Show test counts (Total tests, Tests passed, Tests failed, Tests skipped)
- **Groups with both direct tests AND subgroups**: Show test counts AND subgroup counts (Subgroups, Subgroups passed, Subgroups failed)
- **Groups with subgroups only**: Show subgroup counts only (Subgroups, Subgroups passed, Subgroups failed)

**Report Content Structure**:
```markdown
---
group_name: with props
parent_path: src_components_button_test_js/button_rendering
status: PASS
duration: 1.23s
created: 2025-02-15T12:30:00.000Z
updated: 2025-02-15T12:31:11.000Z
---

# Results for `button.test.js > button rendering > with props`

## Test case results

- ✓ should render correctly (0.01s)
- ✓ should handle click events (0.01s)
- ✕ should apply custom styles (0.02s)
```
Error: Expected color to be 'red' but got 'blue'
  at line 45 in Button.test.js
```

## stdout/stderr
```
console.log: Rendering button with props
console.log: Click handler triggered
```
```

**Example: Group with Both Direct Tests and Nested Subgroups**:
```markdown
---
group_name: Button rendering
parent_path: src_components_button_test_js
status: FAIL
duration: 2.5s
created: 2025-02-15T12:30:00.000Z
updated: 2025-02-15T12:31:11.000Z
---

# Results for `button.test.js > button rendering`

## Summary

- Total tests: 2
- Tests passed: 1
- Tests failed: 1
- Subgroups: 2
- Subgroups passed: 2

## Test case results

- ✓ should have default aria-label (0.01s)
- ✕ should handle disabled state (0.01s)
```
Expected disabled attribute to be present
  at line 23 in Button.test.js
```

## Subgroups

| Status | Name | Tests | Duration | Report |
|--------|------|-------|----------|--------|
| PASS | with props | 3 passed | 0.05s | ./with_props.md |
| PASS | without props | 3 passed | 0.04s | ./without_props.md |

## stdout/stderr
```
console.log: Testing button component
```
```

**Example: Group with Only Subgroups (No Direct Tests)**:
```markdown
---
group_name: src/components/App.test.js
parent_path:
status: PASS
duration: 5.2s
created: 2025-02-15T12:30:00.000Z
updated: 2025-02-15T12:31:11.000Z
---

# Results for `App.test.js`

## Summary

- Subgroups: 5
- Subgroups passed: 2
- Subgroups failed: 2
- Subgroups skipped 1

## Test case results

_No direct test cases at this level_

## Subgroups

| Status | Name | Tests | Duration | Report |
| ------ | ---- | ------ | ------ | ------- |
| PASS | Database Integration | 5 passed | 2.3s | ./database_integration.md |
| FAIL | API Integration | 3 passed, 2 failed | 3.1s | ./api_integration.md |
| PASS | Cache Integration | 4 passed | 1.8s | ./cache_integration.md |
| FAIL | Queue Integration | 1 passed, 1 failed, 1 skipped | 1.5s | ./queue_integration.md |
| SKIP | State Management | 4 skipped | 1.3s | ./state_management.md |

## stdout/stderr
```
Test suite started: App.test.js
All tests completed successfully
```
```

**Example: Group with Failed Subgroups**:
```markdown
---
group_name: src/integration.test.js
status: FAIL
duration: 8.7s
created: 2025-02-15T12:30:00.000Z
updated: 2025-02-15T12:31:11.000Z
---

# Results for `src/integration.test.js`

## Summary

- Total tests: 3
- Tests passed: 2
- Tests failed: 0
- Tests skipped: 1
- Subgroups: 4
- Subgroups passed: 2
- Subgroups failed: 2

## Test case results

- ✓ should initialize properly (0.12s)
- ✓ should cleanup resources (0.09s)
- ○ should handle edge case (skipped)

## Subgroups

| Status | Name | Tests | Duration | Report |
|--------|------|-------|----------|--------|
| PASS | Database Integration | 5 passed | 2.3s | ./database_integration.md |
| FAIL | API Integration | 3 passed, 2 failed | 3.1s | ./api_integration.md |
| PASS | Cache Integration | 4 passed | 1.8s | ./cache_integration.md |
| FAIL | Queue Integration | 1 passed, 1 failed, 1 skipped | 1.5s | ./queue_integration.md |
| SKIP | State Management | 4 skipped | 1.3s | ./state_management.md |
```

**Leaf Node Report Example (Class with Multiple Tests)**:
```markdown
---
group_name: TestMathOperations
parent_path: test_math_py
status: FAIL
duration: 0.15s
created: 2025-02-15T12:30:00.000Z
updated: 2025-02-15T12:31:11.000Z
---

# Results for `foo > bar > TestMathOperations`

## Summary

- Total tests: 6
- Tests passed: 5
- Tests failed: 1

## Test case results

- ✓ test_addition (0.01s)
- ✓ test_subtraction (0.01s)
- ✕ test_division_by_zero (0.02s)
```
AssertionError: Expected ZeroDivisionError but got None
  at line 45 in test_math.py
```
- ✓ test_multiplication (0.01s)
- ✓ test_modulo (0.01s)
- ✓ test_power (0.01s)

## stdout/stderr
```
TestMathOperations setup...
Running mathematical operations tests...
```
```


#### 2.4 Simplified Path Strategy

**Key Principles**:
- Maximum path length: 260 characters on Windows only
- Other platforms: no truncation (macOS: 1024, Linux: 4096)
- When truncating (Windows only): preserve rightmost parts
- Add hash prefix to truncated paths for uniqueness
- No group types to simplify implementation
```

#### 2.5 Practical Examples of Name Transformation

| Original Name | Sanitized Path |
|--------------|----------------|
| `github.com/user/project/pkg` | `github_com_user_project_pkg/` |
| `src/components/Button.test.js` | `src_components_button_test_js/` |
| `should render <Button /> correctly` | `should_render__lt_button__gt__correctly.md` |
| `handles "quotes" and 'apostrophes'` | `handles__dq_quotes_dq__and_apostrophes.md` |
| `works with C:\Windows\Path` | `works_with_c__windows_path.md` |
| `test with/**/glob?patterns` | `test_with__star__star__glob_q_patterns.md` |
| `CON` (Windows reserved) | `con_reserved.md` |
| `Very Long Path/That/Exceeds/260/Characters/...` | `[8char-hash]_...characters_test.md` (truncated) |


#### 2.7 Update Report Templates
```markdown
---
detected_runner: go test
group_structure: hierarchical
---

# Test Run Report

## Group Hierarchy

github.com/user/project
├── github.com/user/project/math
│   ├── TestAddition (PASS, 0.02s)
│   └── TestSubtraction (PASS, 0.01s)
└── github.com/user/project/string
    └── TestConcatenation (FAIL, 0.03s)
```

#### 2.3 Implement Incremental Updates for Nested Groups
- Update parent groups when child completes
- Aggregate statistics up the hierarchy
- Handle partial results in interrupted runs

### Phase 3: Console Output Formatter (Week 3)
**Goal**: Display group-aware progress matching existing format

#### 3.1 New Console Output Format
```
Greetings! I will now execute the test command: `npm test`

Full report: .3pio/runs/20250912T125741-funky-gestahl/test-run.md

Beginning test execution now...

RUNNING  ./src/components/Button.test.js
RUNNING  ./src/components/Button.test.js > Button rendering
RUNNING  ./src/components/Button.test.js > Button rendering > with props
PASS     ./src/components/Button.test.js > Button rendering > with props (0.05s)
PASS     ./src/components/Button.test.js > Button rendering > without props (0.04s)
PASS     ./src/components/Button.test.js > Button rendering (0.08s)
RUNNING  ./src/components/Button.test.js > Button events
FAIL     ./src/components/Button.test.js > Button events (0.13s)
  See .3pio/runs/20250912T125741-funky-gestahl/reports/src_components_button_test_js/button_events/index.md
PASS     ./src/components/Button.test.js (0.21s)

Results:     14 passed, 2 failed, 16 total
Total time:  2.345s
```

#### 3.2 Dynamic Group Display
- Show hierarchical path with `>` separator
- Display timing for each group completion
- Link to failed group reports
- Aggregate statistics at the end

### Phase 4: Test Runner Adaptations (Week 3-4)
**Goal**: Update each runner to emit group events

#### 4.1 Go Test Runner (Native)
```go
// internal/runner/definitions/gotest.go
func (g *GoTestDefinition) ProcessJSONOutput(line []byte) Event {
    // Map package to group (no type needed)
    // TestButton → group name: "TestButton" (as-is)
    // t.Run("Component") → subgroup name: "Component" (as-is)
    // Handle subtests as nested groups
    // Flat hierarchy if no nesting detected
}
```

#### 4.2 Jest Adapter
```javascript
// internal/adapters/jest.js
class ThreePioJestReporter {
  onTestStart(test) {
    // Send file as primary group
    this.sendGroupStart(test.path);

    // Extract describe blocks as subgroups - use names as-is
    // describe("Button Component") → group name: "Button Component"
    test.ancestorTitles.forEach((title, depth) => {
      this.sendGroupStart(title, depth);
    });
  }
}
```

#### 4.3 Vitest Adapter
```javascript
// internal/adapters/vitest.js
// Similar to Jest, with suite hierarchy extraction
```

#### 4.4 pytest Adapter
```python
# internal/adapters/pytest_adapter.py
def pytest_runtest_protocol(item, nextitem):
    # Send module as group
    # class TestButton → group name: "TestButton" (as-is)
    # def test_something → test case name: "test_something" (as-is)
    # Flat structure if no classes
```

### Phase 5: Testing & Deployment (Week 4-5)
**Goal**: Comprehensive testing and deployment of new system

#### 5.1 Deployment Strategy
1. Full switchover to new group model
2. All adapters updated simultaneously
3. New event schema becomes the standard
4. Breaking change accepted - early stage project

#### 5.2 Test Coverage
- Unit tests for group hierarchy management
- Integration tests for each runner's group
- E2E tests with real projects

#### 5.3 Test Scenarios
```go
// tests/integration_go/group_test.go
func TestNestedGroupHierarchy(t *testing.T) {
    // Test 5-level deep nesting
    // Verify parent-child relationships
    // Check aggregate statistics
}

func TestMixedGroupTypes(t *testing.T) {
    // Test package → file → describe → test
    // Verify type preservation
}
```

### Phase 6: Documentation & Rollout (Week 5-6)
**Goal**: Update documentation and deploy

#### 6.1 Documentation Updates
- Architecture docs with new model
- User-facing changelog
- API documentation for new events

#### 6.2 Rollout Plan
1. **Alpha**: Internal testing with test fixtures
2. **Beta**: Selected open-source projects
3. **RC**: Wider testing with new model
4. **GA**: Full release with new group model as standard

## Output Capture Strategy

### Test Case Level Output
With the new model, stdout/stderr can be captured at multiple levels:

1. **Test Case Level** (most granular):
   - Captured by test framework adapters when available
   - Stored in `TestCase.Stdout` and `TestCase.Stderr` fields
   - Displayed in reports next to individual test results

2. **Group Level** (intermediate):
   - Output not associated with specific test cases
   - Stored at group level for setup/teardown output
   - Aggregated from child groups

3. **Run Level** (global):
   - Overall process stdout/stderr in `output.log`
   - Fallback for runners without adapter support

### Adapter Implementation
Each adapter should attempt to capture output at the most granular level possible:

```javascript
// Jest/Vitest adapters
onTestCaseResult(test, result) {
  sendEvent({
    eventType: 'testCase',
    payload: {
      // ... other fields
      stdout: capturedStdout,  // if available
      stderr: capturedStderr   // if available
    }
  });
}
```

## Known Limitations

### IPC Concurrency
- Multiple adapters write to single IPC file without explicit locking
- Relies on OS atomic append guarantees (4KB on Linux/macOS, ~4KB on Windows)
- JSON Lines format with small messages (~100-1000 bytes) stays well under atomic limits
- Theoretical risk of interleaved writes if messages exceed 4KB (unlikely in practice)
- Acceptable trade-off for simplicity vs adding locking complexity

## Technical Considerations

### Cross-Platform Filesystem Compatibility

#### Maximum Path Length Limits
- **Windows**: 260 characters (full path)
- **macOS**: 1024 characters (component), 1024 UTF-8 bytes (full path)
- **Linux**: 255 bytes (component), 4096 bytes (full path)

**Strategy**: Keep individual components under 100 chars, full paths under 200 chars

#### Reserved Names (Windows)
```go
var windowsReservedNames = map[string]bool{
    "CON": true, "PRN": true, "AUX": true, "NUL": true,
    "COM1": true, "COM2": true, "COM3": true, "COM4": true,
    "LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true,
}

func IsReservedName(name string) bool {
    upper := strings.ToUpper(name)
    return windowsReservedNames[upper] ||
           windowsReservedNames[strings.TrimSuffix(upper, path.Ext(upper))]
}
```

#### Unicode and Special Characters
```go
func NormalizeUnicode(s string) string {
    // Normalize to NFC form for consistent representation
    return norm.NFC.String(s)
}

func HandleSpecialTestNames(name string) string {
    // Common test naming patterns that need special handling
    patterns := map[string]string{
        "should work with $prop": "should-work-with-prop",
        "handles @mentions": "handles-mentions",
        "#hashtag support": "hashtag-support",
        "works with 日本語": "works-with-japanese",
        "supports émojis 🎉": "supports-emojis",
    }

    // Apply pattern-based replacements
    for pattern, replacement := range patterns {
        if strings.Contains(name, pattern) {
            return replacement
        }
    }

    return name
}
```

### Memory Management
- Lazy loading for large hierarchies
- Bounded depth to prevent stack overflow
- Efficient tree traversal algorithms

### Performance
- Event batching for deeply nested structures
- Incremental report updates
- Parallel processing of independent branches

### Error Handling
- Require group info for all events
- Clear error messages for malformed events
- Strict validation of event schema

## Success Metrics

1. **Compatibility**: All existing test runners continue working
2. **Accuracy**: Nested structures correctly represented
3. **Performance**: No regression in processing time
4. **Adoption**: New runners can be added easily
5. **User Satisfaction**: Clearer test organization

## Risk Mitigation

### Risk: Path Length Issues (Windows)
**Mitigation**: 260 character limit on Windows only, with left-side truncation and hash prefix

### Risk: Performance Degradation
**Mitigation**: Benchmark tests, optimization before release

### Risk: Flat Hierarchy Runners
**Mitigation**: Graceful fallback to single-level groups for runners without hierarchy support
