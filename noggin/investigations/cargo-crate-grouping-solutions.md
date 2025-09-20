# Three Solutions for Cargo Crate Grouping Issue

## Solution 1: Include Binary Hash in Group Name (Quick Fix)
**Implementation Complexity: Low**

Modify the regex extraction to include the unique binary hash in the group identifier.

### Current Code (cargo.go:291-296):
```go
if matches := runningIntegrationTestsRegex.FindStringSubmatch(line); matches != nil {
    testName := matches[1]  // Just "test_client"
    c.currentCrate = testName
```

### Proposed Change:
```go
// Update regex to capture both name and hash
var runningIntegrationTestsRegex = regexp.MustCompile(`Running tests/.* \(target/.*/deps/(.*?)-([a-f0-9]+)\)`)

if matches := runningIntegrationTestsRegex.FindStringSubmatch(line); matches != nil {
    testName := matches[1]     // "test_client"
    testHash := matches[2]      // "ad5dbdac46a0463a"
    groupName := fmt.Sprintf("%s-%s", testName, testHash[:8])  // "test_client-ad5dbdac"
    c.currentCrate = groupName
```

**Pros:**
- Simple, minimal code change
- Guaranteed uniqueness
- No additional cargo commands needed

**Cons:**
- Group names become less readable (test_client-ad5dbdac)
- Hash changes on recompilation
- Doesn't show actual crate ownership

## Solution 2: Track Crate Context from Unit Test Execution (Smart State Tracking)
**Implementation Complexity: Medium**

Leverage the fact that cargo always runs unit tests before integration tests for each crate.

### Implementation:
```go
type CargoTestDefinition struct {
    // Add new field
    lastUnitTestCrate string  // Track most recent unit test crate
}

// When processing unit tests:
if matches := runningUnittestsRegex.FindStringSubmatch(line); matches != nil {
    crateName := matches[1]  // "actix_http" or "awc"
    c.mu.Lock()
    c.currentCrate = crateName
    c.lastUnitTestCrate = crateName  // Store for integration tests
    c.mu.Unlock()
}

// When processing integration tests:
if matches := runningIntegrationTestsRegex.FindStringSubmatch(line); matches != nil {
    testName := matches[1]  // "test_client"
    c.mu.Lock()
    // Use the crate from the last unit test run
    if c.lastUnitTestCrate != "" {
        groupName := fmt.Sprintf("%s::%s", c.lastUnitTestCrate, testName)
        c.currentCrate = groupName  // "actix_http::test_client"
    } else {
        c.currentCrate = testName  // Fallback
    }
    c.mu.Unlock()
}
```

**Pros:**
- Clean, readable group names (actix_http::test_client)
- Shows actual crate ownership
- No extra cargo commands

**Cons:**
- Relies on cargo's execution order (could break if cargo changes)
- May fail for crates with no unit tests
- Requires careful state management

## Solution 3: Pre-Parse Workspace Structure with Cargo Metadata (Most Robust)
**Implementation Complexity: High**

Query cargo for workspace structure before test execution.

### Implementation:
```go
type WorkspaceInfo struct {
    Packages []PackageInfo `json:"packages"`
}

type PackageInfo struct {
    Name        string   `json:"name"`
    ManifestPath string   `json:"manifest_path"`
    Targets     []Target `json:"targets"`
}

type Target struct {
    Name string `json:"name"`
    Kind []string `json:"kind"`
    SrcPath string `json:"src_path"`
}

func (c *CargoTestDefinition) loadWorkspaceStructure() error {
    cmd := exec.Command("cargo", "metadata", "--format-version", "1")
    output, err := cmd.Output()
    if err != nil {
        return err
    }

    var workspace WorkspaceInfo
    json.Unmarshal(output, &workspace)

    // Build map of test file to crate
    c.testFileToCrate = make(map[string]string)
    for _, pkg := range workspace.Packages {
        for _, target := range pkg.Targets {
            if contains(target.Kind, "test") {
                // Map test_client -> actix_http
                c.testFileToCrate[target.Name] = pkg.Name
            }
        }
    }
    return nil
}

// When processing tests:
if matches := runningIntegrationTestsRegex.FindStringSubmatch(line); matches != nil {
    testName := matches[1]
    if crateName, ok := c.testFileToCrate[testName]; ok {
        groupName := fmt.Sprintf("%s::%s", crateName, testName)
        c.currentCrate = groupName
    }
}
```

**Pros:**
- Most accurate - uses cargo's own metadata
- Works for all workspace configurations
- Handles edge cases properly

**Cons:**
- Requires extra cargo command execution
- More complex implementation
- Slight performance overhead

## Recommendation

**For immediate fix:** Use Solution 1 (binary hash) - it's simple and works today.

**For long-term:** Implement Solution 3 (cargo metadata) - it's the most correct and maintainable approach that aligns with how cargo actually understands the workspace structure.

Solution 2 could work as an intermediate step but has more fragility due to assumptions about execution order.