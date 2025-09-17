package definitions

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/zk/3pio/internal/logger"
)

// GoTestDefinition implements support for Go's native test runner
type GoTestDefinition struct {
	logger *logger.FileLogger
	// packageMap removed - no longer using go list
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
	packageErrors     map[string][]string          // Buffer package-level error output

	// Group tracking for universal abstractions
	discoveredGroups map[string]bool           // Track discovered groups to avoid duplicates
	groupStarts      map[string]bool           // Track started groups
	subgroupStats    map[string]*SubgroupStats // Track test counts and timing for subgroups
}

// PackageInfo removed - no longer using go list for package metadata

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
		logger: logger,
		// packageMap removed - using dynamic discovery
		testStates:        make(map[string]*TestState),
		packageTestFiles:  make(map[string][]string),
		packageStarted:    make(map[string]bool),
		packageStartTimes: make(map[string]time.Time),
		packageTestCounts: make(map[string]int),
		packageTestsDone:  make(map[string]int),
		packageStatuses:   make(map[string]string),
		packageGroups:     make(map[string]*PackageGroupInfo),
		packageResultSent: make(map[string]bool),
		packageErrors:     make(map[string][]string),
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

	// Go list removed for performance - causes 200-500ms startup latency
	// Tests are discovered dynamically from JSON output instead
	// This means we can't provide upfront file discovery, but test execution works fine
	g.logger.Debug("Skipping go list for performance - using dynamic discovery")

	// Return empty list to trigger dynamic discovery from test output
	return []string{}, nil
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
	// Configure larger buffer for long JSON lines (especially with embedded output)
	// Default is 64KB which can be exceeded by test output
	const maxScanTokenSize = 10 * 1024 * 1024 // 10MB max line size
	buf := make([]byte, 0, 1024*1024)         // 1MB initial buffer
	scanner.Buffer(buf, maxScanTokenSize)

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

	// Check for scanner error but don't fail immediately - we need to finalize groups
	scanErr := scanner.Err()
	if scanErr != nil {
		g.logger.Debug("Scanner encountered error (will finalize groups anyway): %v", scanErr)
	}

	// Finalize any pending groups even if scanner had an error
	g.logger.Debug("Finalizing pending groups...")
	g.finalizePendingGroups()

	if scanErr != nil {
		// Log the error but don't fail - we've already processed the events
		g.logger.Debug("Scanner error (may be due to pipe closing): %v", scanErr)
		// Only return error if it's not a closed pipe error
		if !strings.Contains(scanErr.Error(), "file already closed") && !strings.Contains(scanErr.Error(), "closed pipe") {
			return fmt.Errorf("error reading output: %w", scanErr)
		}
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
		// Use ensureGroupStarted to prevent duplicate start events
		g.ensureGroupStarted([]string{event.Package})
		g.packageStarted[event.Package] = true
	}

	// Package mapping removed - discovered dynamically from test output
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

	// Check if cached (without packageMap)
	var isCached bool
	if event.Output != "" && strings.Contains(event.Output, "(cached)") {
		isCached = true
	}

	// testFileStart/testFileResult events removed - using group events instead
	// Cached packages are handled by group events
	if isCached {
		g.logger.Debug("Cached package detected: %s", event.Package)
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

		// Detect setup failures and send testGroupError event
		if event.Action == "fail" && totals["total"].(int) == 0 {
			// This is a setup failure - construct error message
			errorMessage := g.constructErrorMessage(event.Package)

			// Send testGroupError event
			g.sendGroupError(event.Package, []string{}, "SETUP_FAILURE", event.Elapsed, errorMessage)

			// Mark setupFailed in testGroupResult totals
			totals["setupFailed"] = true
		}

		// Send GroupResult for the package
		g.sendGroupResult(event.Package, []string{}, status, event.Elapsed, totals)

		// Clear the started flag since we've sent the result
		delete(g.packageStarted, event.Package)

		// Clean up buffered errors to prevent memory leaks
		g.cleanupPackageErrors(event.Package)

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
		// Package-level output processing

		// Filter and capture relevant error lines
		output := strings.TrimSpace(event.Output)
		if g.isErrorOutput(output) {
			// Initialize package error buffer if needed
			if g.packageErrors == nil {
				g.packageErrors = make(map[string][]string)
			}
			g.packageErrors[event.Package] = append(g.packageErrors[event.Package], output)
		}

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
			g.logger.Debug("Package output for %s: %s", event.Package, output)
		}
	}
}

// getFilePathForTest maps a test to its source file - simplified to always use package-level mapping
func (g *GoTestDefinition) getFilePathForTest(packageName, testName string) string {
	// Go tests operate at package level, not file level
	// Always use package-based mapping since we can't reliably determine which file contains a test
	return g.getFilePathForPackage(packageName)
}

// getFilePathForPackage simplified - no longer maps to files without go list
func (g *GoTestDefinition) getFilePathForPackage(packageName string) string {
	// Without go list, we can't map packages to files
	// Return empty string to indicate no file mapping available
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

// runGoList removed - no longer using go list

// parseGoListOutput removed - no longer using go list

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

// finalizePendingGroups sends group results for any groups that haven't been finalized
func (g *GoTestDefinition) finalizePendingGroups() {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Check if there are any package groups that haven't been finalized
	for pkgName, pkgGroup := range g.packageGroups {
		// Check if this package has been started but not completed
		if _, started := g.packageStarted[pkgName]; started {
			// Check if we already sent a result for this package
			// (packageStarted is set when we send the group start, and should be cleared when we send result)
			g.logger.Debug("Checking if package %s needs finalization", pkgName)

			// Calculate totals from tracked tests
			totals := map[string]interface{}{
				"total":   0,
				"passed":  0,
				"failed":  0,
				"skipped": 0,
			}

			status := "FAIL" // Default to FAIL if incomplete

			if len(pkgGroup.Tests) > 0 {
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

				// Determine status based on test results
				if totals["failed"].(int) > 0 {
					status = "FAIL"
				} else if totals["passed"].(int) > 0 {
					status = "PASS"
				} else if totals["skipped"].(int) > 0 {
					status = "SKIP"
				}
			}

			g.logger.Debug("Finalizing incomplete package %s with status %s (passed=%d, failed=%d, skipped=%d)",
				pkgName, status, totals["passed"], totals["failed"], totals["skipped"])
			g.sendGroupResult(pkgName, []string{}, status, 0, totals)

			// Clear the started flag so we don't finalize again
			delete(g.packageStarted, pkgName)
		}
	}
}

// isErrorOutput determines if a line of output should be captured as an error
func (g *GoTestDefinition) isErrorOutput(output string) bool {
	// Skip empty lines and standard go test output
	if output == "" {
		return false
	}

	// Skip standard success/info lines
	skipPatterns := []string{
		"?   \t",    // No test files indicator
		"ok  \t",    // Package passed
		"FAIL\t",    // Package failed (redundant with status)
		"coverage:", // Coverage information
		"=== RUN",   // Test start (should be captured by test-specific logic)
		"--- PASS",  // Test pass (should be captured by test-specific logic)
		"--- FAIL",  // Test fail (should be captured by test-specific logic)
		"--- SKIP",  // Test skip (should be captured by test-specific logic)
	}

	for _, pattern := range skipPatterns {
		if strings.Contains(output, pattern) {
			return false
		}
	}

	// Capture everything else as potential error output
	return true
}

// constructErrorMessage builds an error message from buffered package output
func (g *GoTestDefinition) constructErrorMessage(packageName string) string {
	// Get buffered error output for this package
	errorLines, hasErrors := g.packageErrors[packageName]

	if !hasErrors || len(errorLines) == 0 {
		// Fallback message if no specific error captured
		return "Package failed during setup or compilation"
	}

	// Join error lines with newlines for readability
	message := strings.Join(errorLines, "\n")

	// Clean up common noise
	message = g.cleanErrorMessage(message)

	return message
}

// cleanErrorMessage removes noise from error messages
func (g *GoTestDefinition) cleanErrorMessage(message string) string {
	// Remove leading/trailing whitespace
	message = strings.TrimSpace(message)

	// Remove redundant "FAIL" lines since status is already FAIL
	lines := strings.Split(message, "\n")
	var cleanLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip redundant FAIL lines like "FAIL\tpackage.name\t1.23s"
		if strings.HasPrefix(line, "FAIL\t") {
			continue
		}

		cleanLines = append(cleanLines, line)
	}

	return strings.Join(cleanLines, "\n")
}

// sendGroupError sends a testGroupError event
func (g *GoTestDefinition) sendGroupError(groupName string, parentNames []string, errorType string, duration float64, message string) {
	event := map[string]interface{}{
		"eventType": "testGroupError",
		"payload": map[string]interface{}{
			"groupName":   groupName,
			"parentNames": parentNames,
			"errorType":   errorType,
			"duration":    duration * 1000, // Convert seconds to milliseconds
			"error": map[string]interface{}{
				"message": message,
				"phase":   "setup",
			},
		},
	}
	if err := g.ipcWriter.WriteEvent(event); err != nil {
		g.logger.Error("Failed to send testGroupError: %v", err)
	}
}

// cleanupPackageErrors removes buffered errors to prevent memory leaks
func (g *GoTestDefinition) cleanupPackageErrors(packageName string) {
	delete(g.packageErrors, packageName)
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
