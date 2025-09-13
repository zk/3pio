# Universal Test Abstractions

This document describes the common abstractions across different test runners and languages, and how 3pio can better adapt to various testing models.

## The Universal Hierarchy

All test runners, regardless of language, share this fundamental hierarchy:

```
Test Run (entire execution)
  └── Grouping(s) (collection of related tests)
       └── Subgrouping(s) (optional nested collections)
            └── Test Case(s) (individual executable test)
```

Note: Groupings can be nested to arbitrary depth, though most languages use 1-3 levels.

## Core Universal Abstractions

### 1. Test Case (Atomic Unit)
The most universal abstraction - every test runner has individual tests with:
- **Name**: Unique identifier for the test
- **Result**: Pass/Fail/Skip/Error
- **Duration**: Execution time
- **Output**: stdout/stderr/logs captured during execution
- **Error details**: Stack trace or assertion failure if failed

### 2. Test Run (Execution)
The complete test execution session with:
- **Start/end time**
- **Total duration**
- **Aggregate statistics** (passed/failed/skipped counts)
- **Exit code**
- **Overall status**

### 3. Grouping (Collection with Nesting)
A collection of related test cases or other groupings. Groupings can be nested to create hierarchies. The specific unit varies by language:

**Primary groupings:**
- **Files** (JavaScript, Python, Ruby, PHP)
- **Packages** (Go)
- **Classes** (Java, C#, C++)
- **Modules** (Rust, Python submodules)

**Common subgroupings:**
- **Suites/Describes** (Jest, Mocha, Jasmine - can nest multiple levels)
- **Test classes within packages** (Go subtests, Java inner classes)
- **Contexts** (RSpec, can nest deeply)
- **Submodules** (Rust, Python)
- **Test fixtures** (pytest, can create hierarchies)

## Language-Specific Implementations

| Language | Primary Grouping | Secondary Grouping | Test Case |
|----------|-----------------|-------------------|-----------|
| **JavaScript/TypeScript** | File | Suite/Describe | Test/It |
| **Python** | File/Module | Class (optional) | Function |
| **Go** | Package | File (not exposed) | Function |
| **Java** | Class | Method groups (@Category) | Method |
| **Rust** | Module | Sub-module | Function |
| **C++ (Google Test)** | Test Suite | Test Case | Assertion |
| **PHP (PHPUnit)** | Class | Method groups | Method |
| **Ruby (RSpec)** | File | Describe/Context | It/Example |
| **C# (NUnit/xUnit)** | Class | Category/Trait | Method |
| **Swift (XCTest)** | Class | Method groups | Method |

## Examples Across Languages

### JavaScript (Jest/Vitest) - Nested Groupings
```javascript
// math.test.js (Primary Grouping: File)
describe('Math operations', () => {       // Subgrouping level 1
  describe('Addition', () => {           // Subgrouping level 2
    describe('Positive numbers', () => { // Subgrouping level 3
      it('should add two positives', () => {  // Test Case
        expect(2 + 2).toBe(4);
      });
    });
  });
});
```

### Python (pytest)
```python
# test_math.py (Grouping: File)
def test_addition():  # Test Case
    assert 2 + 2 == 4
```

### Go - Subtests as Nested Groupings
```go
// math_test.go (Part of package math - Primary Grouping: Package)
func TestMathOperations(t *testing.T) {  // Test Case with subtests
    t.Run("Addition", func(t *testing.T) {  // Subgrouping level 1
        t.Run("Positive numbers", func(t *testing.T) {  // Subgrouping level 2
            if 2+2 != 4 {
                t.Fail()
            }
        })
    })
}
```

### Java (JUnit)
```java
// MathTest.java (Grouping: Class)
public class MathTest {
    @Test
    public void testAddition() {  // Test Case
        assertEquals(4, 2 + 2);
    }
}
```

### Rust
```rust
// lib.rs or math.rs (Grouping: Module)
#[cfg(test)]
mod tests {  // Module grouping
    #[test]
    fn test_addition() {  // Test Case
        assert_eq!(2 + 2, 4);
    }
}
```

## The Challenge for 3pio

3pio currently uses **file** as its primary abstraction, which creates mismatches:

### Works Well For:
- ✅ JavaScript/TypeScript (Jest, Vitest, Mocha)
- ✅ Python (pytest, unittest)
- ✅ Ruby (RSpec)
- ✅ PHP (PHPUnit) - though class-based, typically one class per file

### Doesn't Map Well To:
- ❌ Go (package-based, multiple files form one test unit)
- ❌ Rust (module-based, may not align with files)
- ❌ Java (class-based, multiple classes possible per file)
- ❌ C++ (varies widely by framework)

## Recommended Solution: Grouping Abstraction

Instead of forcing all test runners into a file-based model, 3pio should:

### 1. Use "Grouping" as the Primary Abstraction
Replace file-centric reporting with grouping-centric reporting.

### 2. Add Grouping Type Metadata
```yaml
---
detected_runner: go test
grouping_type: package  # or: file, class, module
grouping_name: github.com/user/project/math
---
```

### 3. Adapt Console Output
```
# Current (file-centric, problematic for Go):
PASS     ./tests/integration_go/error_heading_test.go

# Proposed (grouping-aware):
PASS     [package] github.com/zk/3pio/tests/integration_go
PASS     [file] ./src/math.test.js
PASS     [class] com.example.MathTest
```

### 4. Update Report Structure
Instead of `.3pio/runs/*/reports/[filename].md`, use:
- `.3pio/runs/*/reports/[sanitized-grouping-name].md`
- Include grouping type in report metadata

### 5. IPC Event Schema Update
```json
{
  "eventType": "testGroupingStart",
  "payload": {
    "groupingType": "package|file|class|module|suite|describe|context",
    "groupingName": "...",
    "groupingPath": "...",  // filesystem path if applicable
    "parentGroupingId": "...",  // For nested groupings
    "depth": 0  // Nesting level (0 = top-level)
  }
}
```

This schema supports arbitrarily nested groupings, allowing accurate representation of complex test hierarchies like:
- Jest's nested describes
- Go's subtests
- RSpec's nested contexts
- JUnit 5's nested test classes

## Benefits of Grouping Abstraction

1. **Language Agnostic**: Works for all test runners
2. **Accurate Reporting**: Respects each language's natural organization
3. **Consistent Model**: Same data structure, different labels
4. **Future Proof**: Easy to add new test runners
5. **Clear Communication**: Users understand what unit is being tested

## Implementation Priority

1. **Phase 1**: Add grouping_type field, keep file-based for backward compatibility
2. **Phase 2**: Update Go runner to report packages properly
3. **Phase 3**: Extend to Java (JUnit), Rust (cargo test)
4. **Phase 4**: Update console output and report formats
5. **Phase 5**: Deprecate file-centric assumptions

## Conclusion

By adopting "grouping" as the primary abstraction with type-specific labels, 3pio can maintain a consistent model while respecting each language's testing conventions. This makes the tool truly universal rather than JavaScript-centric.