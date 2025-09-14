package definitions

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/zk/3pio/internal/logger"
)

// GoTestDefinition implements support for Go's native test runner
type GoTestDefinition struct {
	logger     *logger.FileLogger
	packageMap map[string]*PackageInfo
	testStates map[string]*TestState
	mu         sync.RWMutex
	ipcWriter  *IPCWriter

	// Package-level tracking
	packageTestFiles  map[string][]string          // Map of package to its test files
	packageStarted    map[string]bool              // Track if we've sent package group start
	packageStartTimes map[string]time.Time         // Track when package started
	packageTestCounts map[string]int               // Track number of tests per package
	packageTestsDone  map[string]int               // Track completed tests per package
	packageStatuses   map[string]string            // Track overall status per package
	packageGroups     map[string]*PackageGroupInfo // Track package-level group info
	packageResultSent map[string]bool              // Track if result has been sent for package

	// Group tracking for universal abstractions
	discoveredGroups map[string]bool           // Track discovered groups to avoid duplicates
	groupStarts      map[string]bool           // Track started groups
	subgroupStats    map[string]*SubgroupStats // Track test counts and timing for subgroups
}

// PackageInfo holds information about a Go package
type PackageInfo struct {
	ImportPath   string   `json:"ImportPath"`
	Dir          string   `json:"Dir"`
	TestGoFiles  []string `json:"TestGoFiles"`
	XTestGoFiles []string `json:"XTestGoFiles"`
	IsCached     bool
	Status       string
}

// TestState tracks the state of a running test
type TestState struct {
	Name      string
	Package   string
	StartTime time.Time
	Output    []string
	IsPaused  bool
}

// TestInfo tracks individual test information
type TestInfo struct {
	Name     string
	Status   string
	Duration float64
}

// GoTestEvent represents a single event from go test -json output
type GoTestEvent struct {
	Time    time.Time `json:"Time"`
	Action  string    `json:"Action"`
	Package string    `json:"Package"`
	Test    string    `json:"Test,omitempty"`
	Output  string    `json:"Output,omitempty"`
	Elapsed float64   `json:"Elapsed,omitempty"`
}

// PackageGroupInfo tracks information for a package group
type PackageGroupInfo struct {
	StartTime   time.Time
	Tests       []TestInfo
	NoTestFiles bool // True if package has no test files
}

// SubgroupStats tracks statistics for a subgroup (e.g., TestMain when it has subtests)
type SubgroupStats struct {
	TotalTests   int
	PassedTests  int
	FailedTests  int
	SkippedTests int
	StartTime    time.Time
	Duration     float64 // in seconds
	Status       string
}

// IPCWriter handles writing IPC events
type IPCWriter struct {
	path string
	file *os.File
	mu   sync.Mutex
}

// NewGoTestDefinition creates a new Go test runner definition
func NewGoTestDefinition(logger *logger.FileLogger) *GoTestDefinition {
	return &GoTestDefinition{
		logger:            logger,
		packageMap:        make(map[string]*PackageInfo),
		testStates:        make(map[string]*TestState),
		packageTestFiles:  make(map[string][]string),
		packageStarted:    make(map[string]bool),
		packageStartTimes: make(map[string]time.Time),
		packageTestCounts: make(map[string]int),
		packageTestsDone:  make(map[string]int),
		packageStatuses:   make(map[string]string),
		packageGroups:     make(map[string]*PackageGroupInfo),
		packageResultSent: make(map[string]bool),
		discoveredGroups:  make(map[string]bool),
		groupStarts:       make(map[string]bool),
		subgroupStats:     make(map[string]*SubgroupStats),
	}
}

// Name returns the name of this test runner
func (g *GoTestDefinition) Name() string {
	return "go"
}

// Detect checks if the command is for go test
func (g *GoTestDefinition) Detect(args []string) bool {
	if len(args) < 2 {
		return false
	}

	// Check for "go test" command
	if args[0] == "go" && args[1] == "test" {
		return true
	}

	// Check for full path to go binary
	if strings.HasSuffix(args[0], "/go") && len(args) > 1 && args[1] == "test" {
		return true
	}

	return false
}

// ModifyCommand ensures the -json flag is present in the go test command
func (g *GoTestDefinition) ModifyCommand(cmd []string, ipcPath, runID string) []string {
	result := make([]string, 0, len(cmd)+1)
	hasJSON := false

	// Check if -json flag already exists
	for _, arg := range cmd {
		if arg == "-json" {
			hasJSON = true
		}
	}

	// Add -json flag after "go test" if not present
	for i, arg := range cmd {
		result = append(result, arg)

		// After "test" command, add -json if needed
		if i > 0 && cmd[i-1] == "go" && arg == "test" && !hasJSON {
			result = append(result, "-json")
		}
	}

	return result
}

// GetTestFiles extracts test files from command arguments or uses go list
func (g *GoTestDefinition) GetTestFiles(args []string) ([]string, error) {
	// Check if specific test files are provided
	var testFiles []string
	startIdx := 2
	if len(args) < 2 {
		startIdx = len(args)
	}
	for i := startIdx; i < len(args); i++ {
		arg := args[i]
		if strings.HasSuffix(arg, "_test.go") || strings.HasSuffix(arg, ".go") {
			// Specific test file provided
			if !strings.HasPrefix(arg, "-") {
				testFiles = append(testFiles, arg)
			}
		}
	}

	// If specific test files were provided, return them
	if len(testFiles) > 0 {
		return testFiles, nil
	}

	// Build package patterns from args
	patterns := g.extractPackagePatterns(args)

	// If no patterns, use current directory
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	// Run go list to get package info
	start := time.Now()
	packageMap, err := g.runGoList(patterns)
	if err != nil {
		g.logger.Debug("Failed to run go list: %v", err)
		return []string{}, nil // Return empty for dynamic discovery
	}
	g.logger.Debug("go list completed in %v", time.Since(start))

	// Store package map for later use
	g.packageMap = packageMap

	// Note: buildTestToFileMap() removed - it was legacy code that caused performance issues
	// Go tests operate at package level, so test-to-file mapping provides no value

	// Extract all test files from packages
	var allTestFiles []string
	for _, pkg := range packageMap {
		for _, file := range pkg.TestGoFiles {
			allTestFiles = append(allTestFiles, filepath.Join(pkg.Dir, file))
		}
		for _, file := range pkg.XTestGoFiles {
			allTestFiles = append(allTestFiles, filepath.Join(pkg.Dir, file))
		}
	}

	return allTestFiles, nil
}

// RequiresAdapter returns false as Go test doesn't need an external adapter
func (g *GoTestDefinition) RequiresAdapter() bool {
	return false
}

// ProcessOutput reads go test JSON output and converts to IPC events
func (g *GoTestDefinition) ProcessOutput(stdout io.Reader, ipcPath string) error {
	// Initialize IPC writer
	var err error
	g.ipcWriter, err = NewIPCWriter(ipcPath)
	if err != nil {
		return fmt.Errorf("failed to create IPC writer: %w", err)
	}
	defer func() {
		if err := g.ipcWriter.Close(); err != nil {
			g.logger.Debug("Failed to close IPC writer: %v", err)
		}
	}()

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Bytes()

		// Try to parse as JSON
		var event GoTestEvent
		if err := json.Unmarshal(line, &event); err != nil {
			// Non-JSON line (likely build error)
			g.logger.Debug("Non-JSON output: %s", string(line))
			// Non-JSON output no longer sent as events - handled by group events
			continue
		}

		// Process the event
		if err := g.processEvent(&event); err != nil {
			g.logger.Error("Failed to process event: %v", err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading output: %w", err)
	}

	return nil
}

// processEvent handles a single go test JSON event
func (g *GoTestDefinition) processEvent(event *GoTestEvent) error {
	if event == nil {
		return nil
	}


	switch event.Action {
	case "start":
		// Package execution starting
		if event.Test == "" {
			g.handlePackageStart(event)
		}

	case "run":
		// Test starting
		if event.Test != "" {
			g.handleTestRun(event)
		}

	case "pause":
		// Test paused (parallel execution)
		if event.Test != "" {
			g.handleTestPause(event)
		}

	case "cont":
		// Test continued
		if event.Test != "" {
			g.handleTestCont(event)
		}

	case "pass", "fail", "skip":
		// Test or package result
		if event.Test != "" {
			g.handleTestResult(event)
		} else {
			g.handlePackageResult(event)
		}

	case "output":
		// Test output
		g.handleOutput(event)

	case "bench":
		// Benchmark result (not supported yet)
		g.logger.Debug("Benchmark event (not supported): %+v", event)
	}

	return nil
}

// handlePackageStart processes package start events
func (g *GoTestDefinition) handlePackageStart(event *GoTestEvent) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Track package group
	if _, exists := g.packageGroups[event.Package]; !exists {
		g.packageGroups[event.Package] = &PackageGroupInfo{
			StartTime:   event.Time,
			Tests:       []TestInfo{},
			NoTestFiles: false,
		}

		// Send discovery event for package
		g.sendGroupDiscovered(event.Package, []string{})
		g.sendGroupStart(event.Package, []string{})
		g.packageStarted[event.Package] = true
	}

	// Package mapping is ready - groups are handled at package level for Go
	if _, ok := g.packageMap[event.Package]; !ok {
		g.logger.Debug("Package %s started but not in packageMap", event.Package)
	}
}

// handleTestRun processes test run events
func (g *GoTestDefinition) handleTestRun(event *GoTestEvent) {
	g.mu.Lock()
	defer g.mu.Unlock()

	key := fmt.Sprintf("%s/%s", event.Package, event.Test)
	g.testStates[key] = &TestState{
		Name:      event.Test,
		Package:   event.Package,
		StartTime: event.Time,
		Output:    []string{},
		IsPaused:  false,
	}

	// Get the file path for this test
	filePath := g.getFilePathForTest(event.Package, event.Test)
	if filePath == "" {
		// Log but don't return - we still need to process this test
		g.logger.Debug("Could not determine file path for test %s in package %s", event.Test, event.Package)
	}

	// Check if this is the first test from this package
	if !g.packageStarted[event.Package] {
		g.packageStarted[event.Package] = true
		g.packageStartTimes[event.Package] = event.Time
		// testFileStart event removed - using group events instead

		// Discover the package as a root group and start it
		g.ensureGroupsDiscovered(event.Package, []string{})
		g.ensureGroupStarted([]string{event.Package})

		// Store package group info
		g.packageGroups[event.Package] = &PackageGroupInfo{
			StartTime: event.Time,
			Tests:     []TestInfo{},
		}

		g.logger.Debug("Package group discovered and started for %s at %v", event.Package, event.Time)
	}

	// Increment expected test count for this package
	g.packageTestCounts[event.Package]++
}

// handleTestPause processes test pause events
func (g *GoTestDefinition) handleTestPause(event *GoTestEvent) {
	g.mu.Lock()
	defer g.mu.Unlock()

	key := fmt.Sprintf("%s/%s", event.Package, event.Test)
	if state, ok := g.testStates[key]; ok {
		state.IsPaused = true
	}
}

// handleTestCont processes test continuation events
func (g *GoTestDefinition) handleTestCont(event *GoTestEvent) {
	g.mu.Lock()
	defer g.mu.Unlock()

	key := fmt.Sprintf("%s/%s", event.Package, event.Test)
	if state, ok := g.testStates[key]; ok {
		state.IsPaused = false
	}
}

// handleTestResult processes test result events
func (g *GoTestDefinition) handleTestResult(event *GoTestEvent) {
	g.mu.Lock()
	defer g.mu.Unlock()

	key := fmt.Sprintf("%s/%s", event.Package, event.Test)
	state, ok := g.testStates[key]
	if !ok {
		// Create state if it doesn't exist
		state = &TestState{
			Name:    event.Test,
			Package: event.Package,
		}
	}

	// Get file path for the test (package-based mapping for Go)
	filePath := g.getFilePathForTest(event.Package, event.Test)
	if filePath == "" {
		g.logger.Debug("Could not map test %s in package %s to file", event.Test, event.Package)
		// Don't return - still need to process the test even without file mapping
	}

	// Determine status
	status := strings.ToUpper(event.Action)

	// Parse the test hierarchy (handle subtests with "/" separator)
	suiteChain, finalTestName := g.parseTestHierarchy(event.Test)

	// Ensure all parent groups are discovered and started
	g.ensureGroupsDiscovered(event.Package, suiteChain)

	// Start all parent groups in the hierarchy
	for i := 0; i <= len(suiteChain); i++ {
		hierarchy := append([]string{event.Package}, suiteChain[:i]...)
		g.ensureGroupStarted(hierarchy)
	}

	// Build complete hierarchy for this test case using package
	parentNames := g.buildHierarchyFromPackage(event.Package, suiteChain)

	// Send test case event with group hierarchy
	outputStr := strings.Join(state.Output, "\n")
	g.sendTestCaseWithGroups(finalTestName, parentNames, status, event.Elapsed, outputStr)

	// Track subgroup statistics for parent groups
	if len(suiteChain) > 0 {
		// This is a subtest, update stats for its parent group
		for i := 0; i < len(suiteChain); i++ {
			groupHierarchy := append([]string{event.Package}, suiteChain[:i+1]...)
			groupKey := strings.Join(groupHierarchy, "/")

			if _, exists := g.subgroupStats[groupKey]; !exists {
				g.subgroupStats[groupKey] = &SubgroupStats{
					StartTime: time.Now(),
				}
			}

			stats := g.subgroupStats[groupKey]
			stats.TotalTests++
			switch status {
			case "PASS":
				stats.PassedTests++
			case "FAIL":
				stats.FailedTests++
				if stats.Status != "FAIL" {
					stats.Status = "FAIL" // Fail takes precedence
				}
			case "SKIP":
				stats.SkippedTests++
			}

			// Update status if not already failed
			if stats.Status != "FAIL" && stats.PassedTests == stats.TotalTests {
				stats.Status = "PASS"
			}
		}

		// Check if this subtest itself is a parent group (has further subtests)
		// If it is, send a group result for it
		groupKey := event.Package + "/" + event.Test
		if stats, exists := g.subgroupStats[groupKey]; exists {
			// This subtest has its own subtests, send group result for it
			stats.Duration = event.Elapsed

			// Determine final status
			if stats.Status == "" {
				if stats.FailedTests > 0 {
					stats.Status = "FAIL"
				} else if stats.PassedTests == stats.TotalTests {
					stats.Status = "PASS"
				} else {
					stats.Status = "SKIP"
				}
			}

			// Send group result for this subgroup
			totals := map[string]interface{}{
				"total":   stats.TotalTests,
				"passed":  stats.PassedTests,
				"failed":  stats.FailedTests,
				"skipped": stats.SkippedTests,
			}

			// Build parent names for this subgroup
			g.sendGroupResult(finalTestName, parentNames, stats.Status, stats.Duration, totals)

			// Clean up stats
			delete(g.subgroupStats, groupKey)
		}
	} else {
		// This is a top-level test (no subtests), check if it's a parent group
		groupKey := event.Package + "/" + event.Test
		if stats, exists := g.subgroupStats[groupKey]; exists {
			// This test has subtests, send group result for it
			stats.Duration = event.Elapsed

			// Determine final status
			if stats.Status == "" {
				if stats.FailedTests > 0 {
					stats.Status = "FAIL"
				} else if stats.PassedTests == stats.TotalTests {
					stats.Status = "PASS"
				} else {
					stats.Status = "SKIP"
				}
			}

			// Send group result for this subgroup
			totals := map[string]interface{}{
				"total":   stats.TotalTests,
				"passed":  stats.PassedTests,
				"failed":  stats.FailedTests,
				"skipped": stats.SkippedTests,
			}

			// The parent names for this group are just [package]
			g.sendGroupResult(event.Test, []string{event.Package}, stats.Status, stats.Duration, totals)

			// Clean up stats
			delete(g.subgroupStats, groupKey)
		}
	}

	// Clean up state
	delete(g.testStates, key)

	// Track test in package group (only top-level tests, not subtests)
	if len(suiteChain) == 0 {
		// This is a top-level test (no parent hierarchy)
		if pkgGroup, ok := g.packageGroups[event.Package]; ok {
			pkgGroup.Tests = append(pkgGroup.Tests, TestInfo{
				Name:     finalTestName,
				Status:   status,
				Duration: event.Elapsed,
			})
		}
	}

	// Track package completion
	g.packageTestsDone[event.Package]++

	// Update package status based on test result
	if event.Action == "fail" {
		g.packageStatuses[event.Package] = "FAIL"
	} else if g.packageStatuses[event.Package] != "FAIL" {
		// Only update to PASS/SKIP if not already failed
		if event.Action == "pass" {
			g.packageStatuses[event.Package] = "PASS"
		} else if event.Action == "skip" && g.packageStatuses[event.Package] != "PASS" {
			g.packageStatuses[event.Package] = "SKIP"
		}
	}

	// Don't send package result here - wait for the actual package result event
	// The package-level pass/fail event comes at the end after all tests complete
}

// handlePackageResult processes package result events
func (g *GoTestDefinition) handlePackageResult(event *GoTestEvent) {
	g.mu.Lock()
	defer g.mu.Unlock()


	// Update package info
	if pkg, ok := g.packageMap[event.Package]; ok {
		pkg.Status = strings.ToUpper(event.Action)

		// Check if cached
		if event.Output != "" && strings.Contains(event.Output, "(cached)") {
			pkg.IsCached = true
		}

		// testFileStart/testFileResult events removed - using group events instead
		// Cached packages are handled by group events
		if pkg.IsCached {
			g.logger.Debug("Cached package detected: %s", event.Package)
		} else if len(pkg.TestGoFiles) == 0 && len(pkg.XTestGoFiles) == 0 {
			// Package has no test files - this shouldn't normally happen
			// but handle it just in case
			g.logger.Debug("Package %s has no test files", event.Package)
		}
	}

	// Send package result if we haven't already
	if !g.packageResultSent[event.Package] {
		// Mark that we're sending the result
		g.packageResultSent[event.Package] = true

		// Make sure package is discovered and started
		if !g.packageStarted[event.Package] {
			g.ensureGroupsDiscovered(event.Package, []string{})
			g.ensureGroupStarted([]string{event.Package})
			g.packageStarted[event.Package] = true
		}

		// Send the package group result
		status := strings.ToUpper(event.Action)

		// Calculate totals from tracked tests
		totals := map[string]interface{}{
			"total":   0,
			"passed":  0,
			"failed":  0,
			"skipped": 0,
		}


		// If we have a package group with tests, use those totals
		if pkgGroup, ok := g.packageGroups[event.Package]; ok && len(pkgGroup.Tests) > 0 {
			totals["total"] = len(pkgGroup.Tests)
			for _, test := range pkgGroup.Tests {
				switch test.Status {
				case "PASS":
					totals["passed"] = totals["passed"].(int) + 1
				case "FAIL":
					totals["failed"] = totals["failed"].(int) + 1
				case "SKIP":
					totals["skipped"] = totals["skipped"].(int) + 1
				}
			}
		}

		// Check if this is a package with no test files
		if pkgGroup, ok := g.packageGroups[event.Package]; ok {
			if pkgGroup.NoTestFiles && status == "SKIP" {
				// Mark this specially so orchestrator can display ???
				status = "NOTESTS"
			}
		}

		// Send GroupResult for the package
		g.sendGroupResult(event.Package, []string{}, status, event.Elapsed, totals)

		g.logger.Debug("Sent package result for %s: status=%s, duration=%.2fs, tests=%d",
			event.Package, status, event.Elapsed, totals["total"])
	}
}

// handleOutput processes output events
func (g *GoTestDefinition) handleOutput(event *GoTestEvent) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// If output is for a specific test, buffer it
	if event.Test != "" {
		key := fmt.Sprintf("%s/%s", event.Package, event.Test)
		if state, ok := g.testStates[key]; ok {
			state.Output = append(state.Output, event.Output)
		}
	} else {
		// Check for "no test files" indicator
		if strings.Contains(event.Output, "[no test files]") {
			// Ensure package group exists
			if _, exists := g.packageGroups[event.Package]; !exists {
				g.packageGroups[event.Package] = &PackageGroupInfo{
					StartTime:   event.Time,
					Tests:       []TestInfo{},
					NoTestFiles: true,
				}
			} else {
				g.packageGroups[event.Package].NoTestFiles = true
			}
		}

		// Package-level output - now handled by group events
		filePath := g.getFilePathForPackage(event.Package)
		if filePath != "" {
			// Output capture handled by group events
			g.logger.Debug("Package output for %s: %s", event.Package, strings.TrimSpace(event.Output))
		}
	}
}

// getFilePathForTest maps a test to its source file - simplified to always use package-level mapping
func (g *GoTestDefinition) getFilePathForTest(packageName, testName string) string {
	// Go tests operate at package level, not file level
	// Always use package-based mapping since we can't reliably determine which file contains a test
	return g.getFilePathForPackage(packageName)
}

// getFilePathForPackage maps a package to its first test file
func (g *GoTestDefinition) getFilePathForPackage(packageName string) string {
	if pkg, ok := g.packageMap[packageName]; ok {
		var absolutePath string
		if len(pkg.TestGoFiles) > 0 {
			absolutePath = filepath.Join(pkg.Dir, pkg.TestGoFiles[0])
		} else if len(pkg.XTestGoFiles) > 0 {
			absolutePath = filepath.Join(pkg.Dir, pkg.XTestGoFiles[0])
		} else {
			return ""
		}

		// Convert to relative path
		cwd, err := os.Getwd()
		if err != nil {
			// If we can't get cwd, return the absolute path
			return absolutePath
		}

		relPath, err := filepath.Rel(cwd, absolutePath)
		if err != nil {
			// If we can't get relative path, return the absolute path
			return absolutePath
		}

		// Always use "./" prefix for consistency
		if !strings.HasPrefix(relPath, ".") && !strings.HasPrefix(relPath, "/") {
			relPath = "./" + relPath
		}

		return relPath
	}
	return ""
}

// buildTestToFileMap removed - legacy code from pre-universal-abstractions
// Go tests operate at the package level, not file level, so mapping individual
// tests to files provides no value and causes performance issues on large repos

// Group management helper methods

func (g *GoTestDefinition) getGroupId(hierarchy []string) string {
	return strings.Join(hierarchy, ":")
}

func (g *GoTestDefinition) parseTestHierarchy(testName string) (suiteChain []string, finalTestName string) {
	if strings.Contains(testName, "/") {
		parts := strings.Split(testName, "/")
		return parts[:len(parts)-1], parts[len(parts)-1]
	}
	return []string{}, testName
}
func (g *GoTestDefinition) buildHierarchyFromPackage(packageName string, suiteChain []string) []string {
	hierarchy := []string{packageName}
	hierarchy = append(hierarchy, suiteChain...)
	return hierarchy
}

func (g *GoTestDefinition) ensureGroupsDiscovered(packageName string, suiteChain []string) {
	// First, the package itself is a group
	packageHierarchy := []string{packageName}
	packageGroupId := g.getGroupId(packageHierarchy)
	if !g.discoveredGroups[packageGroupId] {
		g.discoveredGroups[packageGroupId] = true
		g.sendGroupDiscovered(packageName, []string{})
	}

	// Then each level of suites creates a nested group
	for i := range suiteChain {
		parentNames := append([]string{packageName}, suiteChain[:i]...)
		groupName := suiteChain[i]
		hierarchy := append(parentNames, groupName)
		groupId := g.getGroupId(hierarchy)

		if !g.discoveredGroups[groupId] {
			g.discoveredGroups[groupId] = true
			g.sendGroupDiscovered(groupName, parentNames)
		}
	}
}

func (g *GoTestDefinition) ensureGroupStarted(hierarchy []string) {
	groupId := g.getGroupId(hierarchy)
	if !g.groupStarts[groupId] {
		g.groupStarts[groupId] = true

		if len(hierarchy) == 1 {
			// File group
			g.sendGroupStart(hierarchy[0], []string{})
		} else {
			// Subtest group
			groupName := hierarchy[len(hierarchy)-1]
			parentNames := hierarchy[:len(hierarchy)-1]
			g.sendGroupStart(groupName, parentNames)
		}
	}
}

// extractPackagePatterns extracts package patterns from go test arguments
func (g *GoTestDefinition) extractPackagePatterns(args []string) []string {
	var patterns []string
	skipNext := false

	for i, arg := range args {
		if skipNext {
			skipNext = false
			continue
		}

		// Skip go and test commands
		if i < 2 {
			continue
		}

		// Skip flags
		if strings.HasPrefix(arg, "-") {
			// Check if flag takes a value
			if arg == "-run" || arg == "-bench" || arg == "-count" ||
				arg == "-cpu" || arg == "-parallel" || arg == "-timeout" ||
				arg == "-benchtime" || arg == "-blockprofile" || arg == "-coverprofile" ||
				arg == "-cpuprofile" || arg == "-memprofile" || arg == "-mutexprofile" ||
				arg == "-outputdir" || arg == "-trace" {
				skipNext = true
			}
			continue
		}

		// Skip test files (ending in _test.go or .go)
		if strings.HasSuffix(arg, ".go") {
			continue
		}

		// This is likely a package pattern
		patterns = append(patterns, arg)
	}

	return patterns
}

// runGoList executes go list -json to get package information
func (g *GoTestDefinition) runGoList(patterns []string) (map[string]*PackageInfo, error) {
	args := append([]string{"list", "-json"}, patterns...)
	cmd := exec.Command("go", args...)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list failed: %w", err)
	}

	return g.parseGoListOutput(output)
}

// parseGoListOutput parses the output of go list -json
func (g *GoTestDefinition) parseGoListOutput(output []byte) (map[string]*PackageInfo, error) {
	packages := make(map[string]*PackageInfo)
	decoder := json.NewDecoder(strings.NewReader(string(output)))

	for {
		var pkg PackageInfo
		if err := decoder.Decode(&pkg); err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("failed to parse go list output: %w", err)
		}

		// Only include packages with test files
		if len(pkg.TestGoFiles) > 0 || len(pkg.XTestGoFiles) > 0 {
			packages[pkg.ImportPath] = &pkg
		}
	}

	return packages, nil
}

// IPC event sending methods

func (g *GoTestDefinition) sendGroupDiscovered(groupName string, parentNames []string) {
	event := map[string]interface{}{
		"eventType": "testGroupDiscovered",
		"payload": map[string]interface{}{
			"groupName":   groupName,
			"parentNames": parentNames,
		},
	}
	if err := g.ipcWriter.WriteEvent(event); err != nil {
		g.logger.Error("Failed to send testGroupDiscovered: %v", err)
	}
}

func (g *GoTestDefinition) sendGroupStart(groupName string, parentNames []string) {
	event := map[string]interface{}{
		"eventType": "testGroupStart",
		"payload": map[string]interface{}{
			"groupName":   groupName,
			"parentNames": parentNames,
		},
	}
	if err := g.ipcWriter.WriteEvent(event); err != nil {
		g.logger.Error("Failed to send testGroupStart: %v", err)
	}
}

func (g *GoTestDefinition) sendGroupResult(groupName string, parentNames []string, status string, duration float64, totals map[string]interface{}) {
	event := map[string]interface{}{
		"eventType": "testGroupResult",
		"payload": map[string]interface{}{
			"groupName":   groupName,
			"parentNames": parentNames,
			"status":      status,
			"duration":    duration * 1000, // Convert seconds to milliseconds
			"totals":      totals,
		},
	}
	if err := g.ipcWriter.WriteEvent(event); err != nil {
		g.logger.Error("Failed to send testGroupResult: %v", err)
	}
}

// sendTestCaseWithGroups sends a test case event with group hierarchy
func (g *GoTestDefinition) sendTestCaseWithGroups(testName string, parentNames []string, status string, duration float64, output string) {
	event := map[string]interface{}{
		"eventType": "testCase",
		"payload": map[string]interface{}{
			"testName":    testName,
			"parentNames": parentNames,
			"status":      status,
			"duration":    int64(duration * 1000), // Convert to milliseconds
		},
	}

	// Add error details for failed tests
	if status == "FAIL" && output != "" {
		event["payload"].(map[string]interface{})["error"] = map[string]interface{}{
			"message": output,
		}
	}

	if err := g.ipcWriter.WriteEvent(event); err != nil {
		g.logger.Debug("Failed to write test case event: %v", err)
	}
}

// sendTestFileResult, sendTestFileResultWithDuration, sendStdoutChunk removed - using group events instead

// NewIPCWriter creates a new IPC writer
func NewIPCWriter(path string) (*IPCWriter, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	return &IPCWriter{
		path: path,
		file: file,
	}, nil
}

// WriteEvent writes an IPC event to the file
func (w *IPCWriter) WriteEvent(event interface{}) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	_, err = w.file.Write(append(data, '\n'))
	if err != nil {
		return fmt.Errorf("failed to write event: %w", err)
	}

	return nil
}

// Close closes the IPC writer
func (w *IPCWriter) Close() error {
	return w.file.Close()
}
