# Vitest Unified Adapter Analysis

## Executive Summary

After analyzing the unified Vitest adapter plan and verifying against actual Vitest versions and documentation, I've identified **critical issues** with the proposed implementation strategy that need to be addressed.

## Verified Findings

### 1. Ecosystem Version Claims ✓ CONFIRMED
The plan's ecosystem analysis is accurate:
- **Redux**: Uses Vitest 2.1.9
- **Svelte**: Uses Vitest ^2.1.9
- **Vue Core**: Uses Vitest ^3.2.4
- **TanStack Query**: Uses Vitest 3.1.3

Current 3pio adapter has a hard version check that fails on v2.x (confirmed by testing with Redux).

### 2. Breaking API Changes - CRITICAL ISSUE

The plan suggests using `onTaskUpdate` as the primary API for v1.x and v2.x, but this is **incorrect**:

#### Vitest 2.x Reporter API
```typescript
interface Reporter {
    onInit?: (ctx: Vitest) => void;
    onPathsCollected?: (paths?: string[]) => Awaitable<void>;
    onSpecsCollected?: (specs?: SerializedTestSpecification[]) => Awaitable<void>;
    onCollected?: (files?: File[]) => Awaitable<void>;
    onFinished?: (files: File[], errors: unknown[], coverage?: unknown) => Awaitable<void>;
    onTaskUpdate?: (packs: TaskResultPack[]) => Awaitable<void>;  // ← Available in v2
    // ... other methods
}
```

#### Vitest 3.x Reporter API
- **REMOVED**: `onTaskUpdate`, `onCollected`, `onSpecsCollected`, `onPathsCollected`, `onFinished`
- **ADDED**: New lifecycle methods:
  - `onTestCaseResult` - Primary method for test results
  - `onTestModuleEnd` - Module completion
  - `onTestRunStart/End` - Run lifecycle
  - `onTestSuiteResult` - Suite results
  - And more granular hooks

### 3. Implementation Strategy Issues

The plan's suggested approach has problems:

1. **Cannot use `onTaskUpdate` for v3.x** - It was completely removed in v3.0
2. **Need different primary methods**:
   - v2.x: Must use `onTaskUpdate` + `onFinished`
   - v3.x: Must use `onTestCaseResult` + `onTestModuleEnd`

3. **Version detection is correct** but implementation strategy needs revision

## Corrected Implementation Strategy

### Version-Specific Reporter Methods

#### Vitest 2.x Strategy
```javascript
class ThreePioVitestReporter {
  // V2 Methods (primary)
  onTaskUpdate(packs) {
    // Process TaskResultPack[] for test results
    // Extract test cases from task tree
  }

  onFinished(files, errors) {
    // Final cleanup and file results
  }

  onCollected(files) {
    // Handle file collection
  }
}
```

#### Vitest 3.x Strategy
```javascript
class ThreePioVitestReporter {
  // V3 Methods (primary)
  onTestCaseResult(testCase) {
    // Process individual test results
  }

  onTestModuleEnd(testModule) {
    // Handle file completion
  }

  onTestRunEnd(testModules, errors) {
    // Final cleanup
  }
}
```

### Unified Adapter Architecture

```javascript
class ThreePioVitestReporter {
  constructor() {
    this.version = this.detectVersion();
    this.useV2Api = this.version < 3;
    this.useV3Api = this.version >= 3;
  }

  // Implement ALL methods, conditionally active
  onTaskUpdate(packs) {
    if (!this.useV2Api) return;
    // V2 implementation
  }

  onTestCaseResult(testCase) {
    if (!this.useV3Api) return;
    // V3 implementation
  }

  // ... implement both API surfaces
}
```

## Key Differences from Original Plan

1. **Cannot use `onTaskUpdate` as fallback in v3** - It doesn't exist
2. **Must implement both API surfaces completely** - Not just version strategies
3. **Data extraction differs significantly**:
   - v2: Extract from `TaskResultPack[]` with nested task trees
   - v3: Direct access via `testCase.result()` and `testCase.diagnostic()`

## Risks and Mitigations

### Risk 1: Maintenance Burden
**Mitigation**: Create abstraction layer that normalizes events internally before sending to IPC.

### Risk 2: Version Detection Edge Cases
**Mitigation**: Robust version parsing with fallback to feature detection if needed.

### Risk 3: Future Vitest 4.x Changes
**Mitigation**: Design adapter with clear version boundaries and extensibility points.

## Recommendation

**Proceed with unified adapter** but with revised implementation:

1. **Phase 1**: Add v2.x support using `onTaskUpdate` (unlock Redux/Svelte)
2. **Phase 2**: Maintain v3.x support with new APIs
3. **Phase 3**: Consider v1.x only if user demand exists
4. **Critical**: Do NOT rely on `onTaskUpdate` existing in v3.x

## Testing Requirements

- Test matrix must include actual versions: 2.1.9, 3.1.3, 3.2.4
- Integration tests with Redux (v2) and Vue Core (v3)
- Verify both API surfaces work correctly
- Test version detection and fallback behavior

## Conclusion

The unified adapter plan is **feasible but requires significant corrections** to the API strategy. The removal of `onTaskUpdate` in v3.x means we need truly separate implementations for v2 and v3, not just a strategy pattern. The benefit of supporting v2.x (enabling Redux and Svelte) justifies the added complexity.