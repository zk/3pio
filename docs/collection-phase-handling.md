# Collection Phase Handling in Test Runners

## Overview

Many test frameworks have a distinct "collection" or "discovery" phase that occurs before test execution. During this phase, the framework:
- Discovers test files
- Loads/imports test modules
- Validates test structure
- Catches syntax and import errors

This document describes how 3pio handles collection phases across different test runners.

## Current Implementation

### pytest (Python) - ✅ Implemented

pytest has a well-defined collection phase that:
- Imports all test modules
- Discovers test functions/classes via naming conventions
- Can fail before any tests run if there are import errors

**3pio Implementation:**
- Hooks into `pytest_configure()` for early capture
- Uses `pytest_collectreport()` to catch collection errors
- Sends specialized IPC events:
  - `collectionStart` - Collection beginning
  - `collectionError` - Import/syntax errors during collection
  - `collectionFinish` - Collection complete with test count

**Key Files:**
- `internal/adapters/pytest_adapter.py` - Adapter implementation
- `internal/ipc/events.go` - Event type definitions

### Jest (JavaScript) - ⚠️ Not Needed

Jest handles test discovery as part of execution:
- Import errors are reported as test failures
- No separate collection phase in the reporter API
- Current implementation handles these cases adequately

### Vitest (JavaScript) - ⚠️ Not Needed

Similar to Jest:
- Test discovery integrated with execution
- Errors reported through normal test failure mechanisms
- Current adapter implementation sufficient

## Other Test Runners with Collection Phases

### Potential Future Implementations

#### Go test
- **Collection Equivalent**: Compilation phase
- **Failure Mode**: Build errors prevent any test execution
- **Implementation Note**: Would need to capture compiler output

#### JUnit/TestNG (Java)
- **Collection Phase**: Class loading and annotation processing
- **Failure Mode**: ClassNotFoundException, NoClassDefFoundError
- **Implementation Note**: Would need JUnit Platform Launcher API integration

#### RSpec (Ruby)
- **Collection Phase**: Spec file loading
- **Failure Mode**: LoadError, SyntaxError
- **Implementation Note**: Could use RSpec's `--dry-run` with custom formatter

#### Mocha (JavaScript)
- **Collection Phase**: Test file requiring/importing
- **Failure Mode**: Syntax errors, missing dependencies
- **Implementation Note**: Could hook into Mocha's file loading

#### PHPUnit (PHP)
- **Collection Phase**: Test class discovery via naming/annotations
- **Failure Mode**: Fatal errors, parse errors
- **Implementation Note**: Would need custom test listener

#### cargo test (Rust)
- **Collection Equivalent**: Compilation phase
- **Failure Mode**: Compilation errors
- **Implementation Note**: Would need to parse cargo output

#### .NET test runners (NUnit/xUnit/MSTest)
- **Collection Phase**: Assembly scanning via reflection
- **Failure Mode**: Assembly load errors, missing dependencies
- **Implementation Note**: Would use test discovery APIs

## Design Principles

### When to Implement Collection Handling

Implement specialized collection phase handling when:
1. The test runner has a distinct discovery phase
2. Errors in this phase prevent ALL tests from running
3. The default adapter approach would leave users with no feedback

### Implementation Pattern

For test runners with collection phases:

```python
# Pseudo-code pattern
def adapter_init():
    start_capture_immediately()  # Catch early errors
    send_event("collectionStart")
    
def on_collection_error(error):
    send_event("collectionError", {
        "filePath": error.file,
        "error": error.message
    })
    
def on_collection_complete(test_count):
    send_event("collectionFinish", {
        "collected": test_count
    })
```

### Event Structure

Collection events follow this pattern:

```json
{
  "eventType": "collectionError",
  "payload": {
    "filePath": "test/example.test.js",
    "error": "Cannot find module 'missing-dep'",
    "phase": "collection"
  }
}
```

## Benefits

Proper collection phase handling provides:
1. **Early Feedback** - Users see why tests won't run
2. **Complete Error Context** - Full stack traces for import/syntax errors  
3. **Better Debugging** - Clear indication of collection vs execution failures
4. **Consistent Experience** - Similar error reporting across frameworks

## Testing Collection Phase Handling

To test collection phase handling:

1. **Missing Dependencies**
   ```python
   # test_missing_dep.py
   import nonexistent_module  # Should trigger collection error
   ```

2. **Syntax Errors**
   ```python
   # test_syntax.py
   def test_example(:  # Invalid syntax
       pass
   ```

3. **Import Cycles**
   ```python
   # test_circular.py
   from . import test_circular  # Circular import
   ```

## Future Work

1. Consider standardizing collection event names across all adapters
2. Add collection phase support for commonly requested test runners
3. Provide configuration to disable collection phase capture if needed
4. Consider adding test count predictions during collection phase