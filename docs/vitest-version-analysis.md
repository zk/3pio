# Vitest Version Market Share Analysis

## Summary
Based on research conducted on 2025-09-20, there is significant usage of both Vitest 2.x and 3.x in the ecosystem, indicating backward compatibility for Vitest 2.x would be valuable.

## NPM Statistics (2024)
- **Vitest 3.2.4**: 12.7 million weekly downloads (latest)
- **Total projects using Vitest**: 1,258 on npm
- **Growth**: ~10% developer adoption, rapidly increasing

## Major Projects Version Distribution

### Vitest 3.x Users (Latest)
- **Vue.js Core** (47k+ stars): v3.2.4
- **TanStack Query** (42k+ stars): v3.1.3
- **VueUse** (verified): v3.x

### Vitest 2.x Users
- **Redux** (61k+ stars): v2.1.9
- **Svelte** (79k+ stars): v2.1.9

### Vitest 1.x Users
- While Vitest 1.x was widely used in early 2024, most major projects have migrated to 2.x or 3.x
- Vitest 1.0 released: December 4, 2023
- Vitest 1.6 released: May 2024 (final 1.x version)

## Key Findings

1. **Split Adoption**: Major frameworks are split between v2.x and v3.x
2. **v2.x Still Prevalent**: High-profile projects like Redux and Svelte still use v2.x
3. **Migration Timeline**: Vitest 3.0 is relatively recent, many projects haven't migrated

## Recommendation

**3pio should prioritize backward compatibility for Vitest 2.x** and consider v1.x support:

### Priority 1: Vitest 2.x Support
- **Significant v2.x Usage**: Major frameworks (Redux, Svelte) with 140k+ stars still on v2.x
- **Recent Version**: v2.x is relatively recent (2024), will be in use for months
- **High Impact**: Enables immediate testing of high-profile projects

### Priority 2: Vitest 1.x Support (Optional)
- **Lower Priority**: Most major projects have migrated away from v1.x
- **EOL**: v1.6 was the final v1.x release (May 2024)
- **Limited Benefit**: Few major projects still on v1.x as of late 2024

## Version Differences

### Vitest 2.x → 3.x Changes
- Different reporter API methods
- Hook execution order changes
- Default pool configuration changes
- Bundled chai 5 as ESM

### Implementation Approach
- Detect Vitest version at runtime
- Use appropriate reporter APIs based on version
- Maintain separate adapter code paths for v2.x and v3.x