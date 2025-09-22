# Vitest Adapter Duplicate Test Name Fix - Implementation Plan

## Problem Statement
The Vitest adapter is missing 6 tests when running against the Svelte repository. These tests have duplicate names within the same describe block, causing them to be deduplicated by 3pio's Go-side test identity system which uses `parentNames + testName` as the unique identifier.

### Specific Duplicate Tests in Svelte
1. **"schedules rerun when writing to signal before reading it"** (2 tests lost)
   - Line 511: `test(...)` - creates 2 tests (legacy + runes modes)
   - Line 562: `test.skip(...)` - creates 2 tests that should be skipped

2. **"unowned deriveds correctly update"** (2 tests lost)
   - Line 1021: `test(...)` - creates 2 tests
   - Line 1353: `test(...)` - creates 2 more tests

3. **"nested effects depend on state of upper effects"** (2 tests lost)
   - Line 1090: `test(...)` - creates 2 tests
   - Line 1119: `test(...)` - creates 2 more tests

## Root Cause
- Vitest internally uses unique IDs for each test: `[file_id]_[suite_index]_[test_index]`
- Example IDs for duplicates:
  - First occurrence: `575827767_0_68` (legacy), `575827767_0_69` (runes)
  - Second occurrence: `575827767_0_70` (legacy), `575827767_0_71` (runes)
- Our adapter doesn't use these IDs, causing legitimate duplicate tests to be merged

## Solution: Simplified Option 3 - Use Vitest ID for Duplicate Detection

### Core Strategy
1. Track test identities (parentNames + testName) as we process them
2. When we encounter a duplicate identity, append the Vitest ID's test index to make it unique
3. First occurrence keeps the original name, subsequent occurrences get `[index]` appended
4. This ensures all tests are reported uniquely while minimizing name changes

### Implementation Details

#### 1. Add State Tracking to VitestAdapter Class
```javascript
class VitestAdapter {
  constructor() {
    // ... existing constructor code ...

    // Track seen test identities for duplicate detection
    // Map: "parent1:parent2:testName" -> vitest ID of first occurrence
    this.seenTestIdentities = new Map();
  }
}
```

#### 2. Modify onTaskUpdate Method
Location: `internal/adapters/vitest.js`, lines ~1040-1120

```javascript
onTaskUpdate(packs) {
  if (!(this.vitestMajor && this.vitestMajor < 3)) return;
  if (!Array.isArray(packs)) return;

  // Initialize tracking if not present
  if (!this.seenTestIdentities) {
    this.seenTestIdentities = new Map();
  }

  const terminal = (st) => st === 'pass' || st === 'passed' || st === 'fail' || st === 'failed' || st === 'skip' || st === 'skipped' || st === 'todo';

  for (const pack of packs) {
    try {
      const id = Array.isArray(pack) ? pack[0] : pack?.id;
      const res = Array.isArray(pack) ? pack[1] : pack?.result;
      if (id == null || !res) continue;

      const meta = this.v2TaskIndex.get(id);
      if (!meta) continue;

      const state = res.state || 'unknown';
      const { filePath, suiteChain, name } = meta;

      // Build parent names for IPC event
      const parentNames = this.buildHierarchyFromFile(filePath, suiteChain);

      // Create test identity key (what Go uses for deduplication)
      const testIdentity = `${parentNames.join(':')}:${name}`;

      // Check if we've seen this test identity before
      const previousId = this.seenTestIdentities.get(testIdentity);
      let displayName = name;

      if (previousId && previousId !== id) {
        // This is a duplicate - extract index from current Vitest ID
        // Vitest ID format: "fileId_suiteIndex_testIndex"
        const testIndex = id.split('_').pop();
        displayName = `${name} [${testIndex}]`;

        this.logger.debug('Duplicate test detected, appending index', {
          originalName: name,
          displayName: displayName,
          currentId: id,
          previousId: previousId,
          testIdentity: testIdentity
        });
      } else if (!previousId) {
        // First occurrence - store the ID
        this.seenTestIdentities.set(testIdentity, id);
      }

      // Continue with existing logic but use displayName
      const contentKey = `${filePath}::${suiteChain.join('::')}::${name}`;

      // ... rest of existing onTaskUpdate logic ...

      // When sending event, use displayName instead of name
      IPCSender.sendEvent({
        eventType: 'testCase',
        payload: {
          testName: displayName, // Use modified name for duplicates
          parentNames,
          status,
          duration: res.duration,
          error: errorObj,
        }
      });

      // ... rest of method ...
    } catch (e) {
      this.logger.error('Error in onTaskUpdate', e);
    }
  }
}
```

#### 3. Modify processV2Files Method
Location: `internal/adapters/vitest.js`, lines ~795-950

```javascript
processV2Files(files) {
  if (!Array.isArray(files)) return;

  // Initialize tracking if not present
  if (!this.seenTestIdentities) {
    this.seenTestIdentities = new Map();
  }

  const toArray = (val) => (Array.isArray(val) ? val : val ? [val] : []);

  for (const file of files) {
    // ... existing file processing ...

    const visitSuite = (suite, chain) => {
      // ... existing suite visiting logic ...

      else if (t.type === 'test') {
        // ... existing test processing ...

        const parentNames = this.buildHierarchyFromFile(filePath, nextChain);

        // Create test identity for duplicate detection
        const testIdentity = `${parentNames.join(':')}:${t.name}`;

        // Check if this is a duplicate
        let displayName = t.name;
        if (t.id) {
          const previousId = this.seenTestIdentities.get(testIdentity);

          if (previousId && previousId !== t.id) {
            // Duplicate detected - append test index
            const testIndex = t.id.split('_').pop();
            displayName = `${t.name} [${testIndex}]`;

            this.logger.debug('Duplicate test in processV2Files', {
              originalName: t.name,
              displayName: displayName,
              currentId: t.id,
              previousId: previousId
            });
          } else if (!previousId) {
            // First occurrence
            this.seenTestIdentities.set(testIdentity, t.id);
          }
        }

        // ... rest of test processing ...

        IPCSender.sendEvent({
          eventType: 'testCase',
          payload: {
            testName: displayName, // Use modified name
            parentNames,
            status,
            duration: result.duration,
            error: errorObj,
          }
        });
      }
    };

    // ... rest of method ...
  }
}
```

#### 4. Handle Edge Cases

##### Reset tracking between runs (if needed)
```javascript
onCollected(files) {
  // Clear previous run's tracking
  this.seenTestIdentities = new Map();

  // ... rest of onCollected ...
}
```

##### Handle tests without IDs
```javascript
// In the duplicate detection logic
if (t.id) {
  // Use ID-based detection
} else {
  // Fall back to occurrence counting
  const count = this.testNameCounts.get(testIdentity) || 0;
  this.testNameCounts.set(testIdentity, count + 1);
  if (count > 0) {
    displayName = `${t.name} [occurrence ${count + 1}]`;
  }
}
```

## Testing Plan

### 1. Unit Testing
- Create test file with intentionally duplicate test names
- Verify each duplicate gets unique suffix
- Verify first occurrence keeps original name

### 2. Integration Testing with Svelte
- Run against `packages/svelte/tests/signals/test.ts`
- Verify all 94 tests are reported
- Verify duplicates show as:
  - `nested effects depend on state of upper effects (legacy mode)`
  - `nested effects depend on state of upper effects (legacy mode) [70]`
  - `nested effects depend on state of upper effects (runes mode)`
  - `nested effects depend on state of upper effects (runes mode) [71]`

### 3. Regression Testing
- Verify non-duplicate tests are unaffected
- Verify other test files still work correctly
- Check that test counts match baseline

## Expected Outcomes

### Before Fix
- Svelte shows 6878 total tests (missing 6)
- Duplicate tests are silently deduplicated
- Only first occurrence of each duplicate is reported

### After Fix
- Svelte shows 6884 total tests (matches baseline)
- All duplicate tests are reported with unique names
- Test output shows:
  - First occurrence: original name
  - Subsequent occurrences: name + `[testIndex]`

## Implementation Steps

1. **Backup current adapter**
   ```bash
   cp internal/adapters/vitest.js internal/adapters/vitest.js.backup
   ```

2. **Add state tracking to constructor**
   - Add `this.seenTestIdentities = new Map()`

3. **Modify onTaskUpdate**
   - Add duplicate detection logic
   - Use displayName for duplicates

4. **Modify processV2Files**
   - Mirror the duplicate detection logic
   - Ensure consistency between both methods

5. **Add debug logging**
   - Log when duplicates are detected
   - Include original name, modified name, and IDs

6. **Test with Svelte**
   ```bash
   make adapters && make build
   cd /tmp/3pio-open-source/svelte
   /path/to/3pio pnpm exec vitest run
   ```

7. **Verify results**
   - Check total count matches 6884
   - Check debug logs for duplicate detection
   - Verify test names in reports

## Rollback Plan
If issues arise:
1. Restore backup: `cp internal/adapters/vitest.js.backup internal/adapters/vitest.js`
2. Rebuild: `make adapters && make build`
3. Document any unexpected behavior for further investigation

## Notes
- This solution is entirely contained within the Vitest adapter
- No Go-side changes required
- Maintains backwards compatibility with other test runners
- Minimal impact on test name display (only affects true duplicates)
- Uses Vitest's own ID system for accurate duplicate detection