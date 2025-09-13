# Universal Abstractions Implementation Plan

## Overview
This document provides a detailed, phased implementation plan for migrating 3pio from a file-centric to a group-centric universal abstraction model. This is a breaking change with no backward compatibility.

## Phase 1: Core Data Structures & IPC Schema
**Duration**: 3-4 days  
**Goal**: Establish foundation without breaking existing functionality

### 1.1 Define New Types (Day 1)
**File**: `internal/report/group_types.go`
```go
type TestGroup struct {
    ID           string
    Name         string
    ParentID     string
    ParentNames  []string  // Full hierarchy from root
    Depth        int
    Status       TestStatus
    Duration     time.Duration
    TestCases    []TestCase
    Subgroups    map[string]*TestGroup
    StartTime    time.Time
    EndTime      time.Time
    Created      time.Time
    Updated      time.Time
}

type TestCase struct {
    ID           string
    GroupID      string
    Name         string
    Status       TestStatus
    Duration     time.Duration
    Error        *TestError
    Stdout       string
    Stderr       string
    StartTime    time.Time
    EndTime      time.Time
}
```

### 1.2 IPC Event Types (Day 1)
**File**: `internal/ipc/group_events.go`
```go
type GroupDiscoveredEvent struct {
    GroupName    string   `json:"groupName"`
    ParentNames  []string `json:"parentNames"`
}

type GroupStartEvent struct {
    GroupName    string   `json:"groupName"`
    ParentNames  []string `json:"parentNames"`
    Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

type GroupResultEvent struct {
    GroupName    string   `json:"groupName"`
    ParentNames  []string `json:"parentNames"`
    Status       string   `json:"status"`
    Duration     float64  `json:"duration"`
    Totals       struct {
        Passed   int `json:"passed"`
        Failed   int `json:"failed"`
        Skipped  int `json:"skipped"`
    } `json:"totals"`
}

type TestCaseEvent struct {
    TestName     string   `json:"testName"`
    ParentNames  []string `json:"parentNames"`
    Status       string   `json:"status"`
    Duration     float64  `json:"duration,omitempty"`
    Error        *string  `json:"error,omitempty"`
    Stdout       string   `json:"stdout,omitempty"`
    Stderr       string   `json:"stderr,omitempty"`
}
```

### 1.3 ID Generation (Day 2)
**File**: `internal/report/group_id.go`
```go
func GenerateGroupID(groupName string, parentNames []string) string {
    fullPath := append(parentNames, groupName)
    pathString := strings.Join(fullPath, ":")
    hash := sha256.Sum256([]byte(pathString))
    return hex.EncodeToString(hash[:16])
}
```

### 1.4 Filesystem Path Generation (Day 2)
**File**: `internal/report/group_path.go`
```go
func SanitizeGroupName(name string) string {
    // Implementation from migration plan
}

func GenerateGroupPath(group *TestGroup) string {
    // Build hierarchical path with sanitized names
    // Handle Windows 260 char limit
}
```

### 1.5 Tests (Day 3)
- Unit tests for ID generation
- Unit tests for path sanitization
- Unit tests for Windows path limits
- Test event serialization/deserialization

**Deliverables**:
- [ ] New type definitions
- [ ] Event schemas
- [ ] ID generation logic
- [ ] Path generation logic
- [ ] Comprehensive unit tests

---

## Phase 2: Report Manager Refactor
**Duration**: 4-5 days  
**Goal**: Update Report Manager to handle groups while maintaining file support temporarily

### 2.1 Group State Management (Day 1-2)
**File**: `internal/report/group_manager.go`
```go
type GroupManager struct {
    mu          sync.RWMutex
    groups      map[string]*TestGroup  // ID -> Group
    rootGroups  []*TestGroup
    runDir      string
    ipcPath     string
}

func (gm *GroupManager) ProcessGroupDiscovered(event GroupDiscoveredEvent) {
    // Create group if not exists
    // Create parent groups on-demand
    // Handle idempotently
}

func (gm *GroupManager) ProcessTestCase(event TestCaseEvent) {
    // Find or create parent group
    // Add test case to group
    // Update group statistics
}
```

### 2.2 Report Generation (Day 3-4)
**File**: `internal/report/group_report.go`
```go
func (gm *GroupManager) GenerateGroupReport(group *TestGroup) error {
    // Create directory structure
    // Generate index.md with test cases
    // Include subgroup links
    // Write stdout/stderr section
}

func (gm *GroupManager) RegenerateAncestors(group *TestGroup) {
    // Check if group is complete
    // If parent has all children complete, mark parent complete
    // Recursively regenerate parent reports
}
```

### 2.3 Incremental Updates (Day 4-5)
- Implement debounced writes
- Handle partial results
- Support interrupted runs

### 2.4 Tests (Day 5)
- Integration tests with mock events
- Test hierarchy building
- Test report generation
- Test concurrent event processing

**Deliverables**:
- [ ] Group state management
- [ ] Hierarchical report generation
- [ ] Incremental update logic
- [ ] Integration tests

---

## Phase 3: Adapter Updates - Jest
**Duration**: 3 days  
**Goal**: Update Jest adapter to emit group events

### 3.1 Event Emission (Day 1)
**File**: `internal/adapters/jest.js`
```javascript
class ThreePioJestReporter {
  onTestStart(test) {
    // Extract hierarchy from ancestorTitles
    const filePath = test.path;
    
    // Send file group discovery
    this.sendEvent({
      eventType: 'testGroupDiscovered',
      payload: {
        groupName: filePath,
        parentNames: []
      }
    });
    
    // Send describe block discoveries
    for (let i = 0; i < test.ancestorTitles.length; i++) {
      this.sendEvent({
        eventType: 'testGroupDiscovered',
        payload: {
          groupName: test.ancestorTitles[i],
          parentNames: [filePath, ...test.ancestorTitles.slice(0, i)]
        }
      });
    }
  }
  
  onTestCaseResult(test, testCaseResult) {
    // Send test case with hierarchy
    const parentNames = [test.path, ...test.ancestorTitles];
    this.sendEvent({
      eventType: 'testCase',
      payload: {
        testName: testCaseResult.title,
        parentNames: parentNames,
        status: testCaseResult.status,
        duration: testCaseResult.duration,
        error: testCaseResult.failureMessages?.[0]
      }
    });
  }
}
```

### 3.2 Backward Compatibility Bridge (Day 2)
- Temporarily emit both old and new events
- Add feature flag for new events

### 3.3 Testing (Day 3)
- Test with fixture projects
- Verify hierarchy extraction
- Test parallel execution

**Deliverables**:
- [ ] Updated Jest adapter
- [ ] Hierarchy extraction
- [ ] Event emission
- [ ] Integration tests

---

## Phase 4: Adapter Updates - Vitest
**Duration**: 3 days  
**Goal**: Update Vitest adapter with V3 hooks

### 4.1 Event Emission (Day 1-2)
**File**: `internal/adapters/vitest.js`
```javascript
class ThreePioVitestReporter {
  onTestModuleCollected(module) {
    // Extract hierarchy from module
    this.sendGroupHierarchy(module);
  }
  
  onTestModuleEnd(module) {
    // Send test results with hierarchy
    this.sendTestCasesFromModule(module);
  }
  
  sendGroupHierarchy(task) {
    // Recursively extract and send group discoveries
    const hierarchy = this.extractHierarchy(task);
    // Send testGroupDiscovered events
  }
}
```

### 4.2 Testing (Day 3)
- Test with fixture projects
- Verify V3 hook compatibility
- Test parallel execution

**Deliverables**:
- [ ] Updated Vitest adapter
- [ ] V3 hook integration
- [ ] Hierarchy extraction
- [ ] Integration tests

---

## Phase 5: Adapter Updates - pytest & Go
**Duration**: 3 days  
**Goal**: Update remaining adapters

### 5.1 pytest Adapter (Day 1)
**File**: `internal/adapters/pytest_adapter.py`
```python
def pytest_runtest_protocol(item, nextitem):
    # Parse node ID for hierarchy
    parts = item.nodeid.split("::")
    file_path = parts[0]
    
    # Send group discoveries
    send_event({
        "eventType": "testGroupDiscovered",
        "payload": {
            "groupName": file_path,
            "parentNames": []
        }
    })
    
    if len(parts) > 2:  # Has class
        send_event({
            "eventType": "testGroupDiscovered",
            "payload": {
                "groupName": parts[1],
                "parentNames": [file_path]
            }
        })
```

### 5.2 Go Test Handler (Day 2)
**File**: `internal/runner/definitions/gotest.go`
```go
func (g *GoTestDefinition) ProcessJSONOutput(line []byte) {
    // Parse package and test hierarchy
    // Generate group events from JSON
    // Handle subtests with "/" separator
}
```

### 5.3 Testing (Day 3)
- Test all adapters
- Cross-runner validation

**Deliverables**:
- [ ] Updated pytest adapter
- [ ] Updated Go test handler
- [ ] Integration tests

---

## Phase 6: Console Output Formatter
**Duration**: 2 days  
**Goal**: Update console display for hierarchical groups

### 6.1 Hierarchical Display (Day 1)
**File**: `internal/output/console_formatter.go`
```go
func (cf *ConsoleFormatter) FormatGroupRunning(group *TestGroup) string {
    // Build path with → separator
    path := buildHierarchicalPath(group)
    return fmt.Sprintf("RUNNING  %s", path)
}

func (cf *ConsoleFormatter) FormatGroupComplete(group *TestGroup) string {
    // Include timing and status
    // Add report link for failures
}
```

### 6.2 Testing (Day 2)
- Test formatting
- Test with deep hierarchies

**Deliverables**:
- [ ] Updated console formatter
- [ ] Hierarchical path display
- [ ] Unit tests

---

## Phase 7: Integration & Cutover
**Duration**: 3 days  
**Goal**: Complete integration and remove old code

### 7.1 Full Integration Testing (Day 1)
- Run all fixture projects
- Test with real projects
- Performance benchmarks

### 7.2 Remove Old Code (Day 2)
- Remove file-centric code paths
- Remove old IPC events
- Clean up unused functions

### 7.3 Documentation Updates (Day 3)
- Update architecture docs
- Update adapter writing guide
- Update README
- Create migration guide

**Deliverables**:
- [ ] All tests passing
- [ ] Old code removed
- [ ] Documentation updated
- [ ] Migration guide

---

## Phase 8: Validation & Release
**Duration**: 2 days  
**Goal**: Final validation and release

### 8.1 End-to-End Testing (Day 1)
- Test with large projects
- Test with monorepos
- Test parallel execution
- Test interrupted runs

### 8.2 Release Preparation (Day 2)
- Update version
- Write changelog
- Create release notes
- Tag release

**Deliverables**:
- [ ] E2E tests complete
- [ ] Performance validated
- [ ] Release prepared
- [ ] Version tagged

---

## Risk Mitigation

### Rollback Plan
- Keep old code in separate branch
- Feature flag for gradual rollout
- Clear version marking

### Testing Strategy
1. Unit tests for each component
2. Integration tests with fixtures
3. E2E tests with real projects
4. Performance benchmarks

### Known Challenges
1. **Parallel execution**: Ensure idempotent event handling
2. **Memory usage**: Monitor with large test suites
3. **Path limits**: Thorough Windows testing
4. **Report generation**: Optimize for large hierarchies

---

## Success Criteria

1. All existing test runners work with new model
2. Hierarchical structure correctly represented
3. No performance regression
4. Reports generate correctly
5. Console output shows hierarchy
6. All tests pass

---

## Timeline Summary

- **Phase 1**: Core Data Structures (3-4 days)
- **Phase 2**: Report Manager (4-5 days)
- **Phase 3**: Jest Adapter (3 days)
- **Phase 4**: Vitest Adapter (3 days)
- **Phase 5**: pytest & Go (3 days)
- **Phase 6**: Console Output (2 days)
- **Phase 7**: Integration (3 days)
- **Phase 8**: Validation (2 days)

**Total Duration**: ~23-25 days

---

## Next Steps

1. Review and approve plan
2. Create feature branch `feature/universal-abstractions`
3. Begin Phase 1 implementation
4. Daily progress updates
5. Phase gate reviews