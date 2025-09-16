package definitions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/zk/3pio/internal/logger"
)

// runningTestBinaryRegex matches "Running /path/to/target/debug/deps/crate_name-hash"
var runningTestBinaryRegex = regexp.MustCompile(`Running .*/target/.*/deps/(.*?)-[a-f0-9]+`)

// Keep these for backwards compatibility if needed
var runningUnittestsRegex = regexp.MustCompile(`Running unittests .* \(target/.*/deps/(.*?)-[a-f0-9]+\)`)
var runningIntegrationTestsRegex = regexp.MustCompile(`Running tests/.* \(target/.*/deps/(.*?)-[a-f0-9]+\)`)

// docTestsRegex matches "Doc-tests crate_name" with optional leading whitespace
var docTestsRegex = regexp.MustCompile(`^\s*Doc-tests\s+(.+)$`)

// CargoTestDefinition implements support for Rust's cargo test runner
type CargoTestDefinition struct {
	logger    *logger.FileLogger
	mu        sync.RWMutex
	ipcWriter *IPCWriter

	// Workspace and crate tracking
	workspaceName    string                     // Name of workspace if detected
	currentCrate     string                     // Currently executing crate
	crateTestCounts  map[string]int             // Expected test count per crate from suite events
	crateTestsSeen   map[string]int             // Number of tests seen so far per crate
	crateGroups      map[string]*CrateGroupInfo // Map of crate name to group info
	crateMetadata    map[string]*CrateMetadata  // Map of crate name to metadata from Cargo.toml
	discoveredGroups map[string]bool            // Track discovered groups to avoid duplicates
	groupStarts      map[string]bool            // Track started groups
	testStates       map[string]*CargoTestState // Track test state
}

// CrateMetadata stores metadata from Cargo.toml
type CrateMetadata struct {
	Name        string
	Description string
	Version     string
}

// CrateGroupInfo tracks information for a crate group
type CrateGroupInfo struct {
	Name      string
	StartTime time.Time
	Tests     []CargoTestInfo
	Status    string
	Finalized bool // Whether the group result has been sent
}

// CargoTestInfo tracks individual test information
type CargoTestInfo struct {
	Name     string
	Status   string
	Duration float64
}

// CargoTestState tracks the state of a running test
type CargoTestState struct {
	Name      string
	Crate     string
	StartTime time.Time
	Output    []string
}

// CargoTestEvent represents a single event from cargo test --format json output
type CargoTestEvent struct {
	Type      string  `json:"type"`  // "suite" or "test"
	Event     string  `json:"event"` // "started", "ok", "failed", "ignored"
	Name      string  `json:"name,omitempty"`
	TestCount int     `json:"test_count,omitempty"`
	Passed    int     `json:"passed,omitempty"`
	Failed    int     `json:"failed,omitempty"`
	Ignored   int     `json:"ignored,omitempty"`
	ExecTime  float64 `json:"exec_time,omitempty"`
	Stdout    string  `json:"stdout,omitempty"`
	Stderr    string  `json:"stderr,omitempty"`
}

// NewCargoTestDefinition creates a new cargo test runner definition
func NewCargoTestDefinition(logger *logger.FileLogger) *CargoTestDefinition {
	return &CargoTestDefinition{
		logger:           logger,
		crateTestCounts:  make(map[string]int),
		crateTestsSeen:   make(map[string]int),
		crateGroups:      make(map[string]*CrateGroupInfo),
		crateMetadata:    make(map[string]*CrateMetadata),
		discoveredGroups: make(map[string]bool),
		groupStarts:      make(map[string]bool),
		testStates:       make(map[string]*CargoTestState),
	}
}

// Name returns the name of this test runner
func (c *CargoTestDefinition) Name() string {
	return "cargo"
}

// Detect checks if the command is for cargo test
func (c *CargoTestDefinition) Detect(args []string) bool {
	if len(args) < 2 {
		return false
	}

	// Check for "cargo test" command
	if args[0] == "cargo" && args[1] == "test" {
		return true
	}

	// Check for "cargo +<toolchain> test" command
	if args[0] == "cargo" && len(args) > 2 && strings.HasPrefix(args[1], "+") && args[2] == "test" {
		return true
	}

	// Check for full path to cargo binary
	if strings.HasSuffix(args[0], "/cargo") && len(args) > 1 && args[1] == "test" {
		return true
	}

	// Check for full path with toolchain
	if strings.HasSuffix(args[0], "/cargo") && len(args) > 2 && strings.HasPrefix(args[1], "+") && args[2] == "test" {
		return true
	}

	return false
}

// ModifyCommand adds JSON output flags to cargo test command
func (c *CargoTestDefinition) ModifyCommand(cmd []string, ipcPath, runID string) []string {
	result := make([]string, 0, len(cmd)+6)

	// Copy original command
	result = append(result, cmd...)

	// Check if -- separator already exists
	hasSeparator := false
	for _, arg := range cmd {
		if arg == "--" {
			hasSeparator = true
			break
		}
	}

	// Add separator if not present
	if !hasSeparator {
		result = append(result, "--")
	}

	// Add JSON output flags (RUSTC_BOOTSTRAP=1 is set in orchestrator to enable on stable)
	result = append(result, "-Z", "unstable-options", "--format", "json", "--report-time")

	return result
}

// GetTestFiles returns empty array for dynamic discovery
func (c *CargoTestDefinition) GetTestFiles(args []string) ([]string, error) {
	// Cargo tests are discovered dynamically as they run
	return []string{}, nil
}

// RequiresAdapter returns false as cargo test doesn't need an external adapter
func (c *CargoTestDefinition) RequiresAdapter() bool {
	return false
}

// ProcessOutput reads combined cargo test output (stderr + stdout) and converts to IPC events
// The combined stream has "Running unittests" stderr lines immediately before JSON stdout
func (c *CargoTestDefinition) ProcessOutput(combinedOutput io.Reader, ipcPath string) error {
	// Initialize IPC writer
	var err error
	c.ipcWriter, err = NewIPCWriter(ipcPath)
	if err != nil {
		return fmt.Errorf("failed to create IPC writer: %w", err)
	}
	defer func() {
		if err := c.ipcWriter.Close(); err != nil {
			c.logger.Debug("Failed to close IPC writer: %v", err)
		}
	}()

	// Load cargo metadata for crate descriptions
	c.loadCargoMetadata()

	lineCount := 0
	jsonEventCount := 0
	totalBytes := 0

	// Chunk reading with partial line handling for real-time processing
	// Use LARGER chunks for faster reading to keep up with output
	buffer := make([]byte, 256*1024) // 256KB chunks for aggressive reading
	var partial []byte

	for {
		n, err := combinedOutput.Read(buffer)
		if n > 0 {
			totalBytes += n
			// Append partial line from previous iteration
			data := append(partial, buffer[:n]...)

			// Split into lines
			lines := bytes.Split(data, []byte{'\n'})

			// Process all complete lines (all except the last one)
			for i := 0; i < len(lines)-1; i++ {
				line := string(lines[i])
				if len(line) > 0 { // Process non-empty lines
					lineCount++
					c.processLineData(line, &jsonEventCount)
				}
			}

			// Save the last piece as partial for next iteration
			// (it might be incomplete if we're in the middle of a line)
			partial = lines[len(lines)-1]
		}

		if err != nil {
			if err == io.EOF {
				// Process final partial line if it exists
				if len(partial) > 0 {
					lineCount++
					line := string(partial)
					c.processLineData(line, &jsonEventCount)
				}
				c.logger.Debug("Reached EOF after reading %d bytes", totalBytes)
				break
			} else if strings.Contains(err.Error(), "file already closed") {
				// Process final partial line even on pipe closure
				if len(partial) > 0 {
					lineCount++
					line := string(partial)
					c.processLineData(line, &jsonEventCount)
				}
				c.logger.Debug("Pipe closed after reading %d bytes: %v", totalBytes, err)
				break
			} else {
				// Log other errors but continue reading
				c.logger.Debug("Read error at byte %d: %v, continuing", totalBytes, err)
				// Try to continue reading even on errors
				continue
			}
		}
	}

	// Log processing summary
	c.logger.Debug("ProcessOutput completed: %d total lines, %d JSON events processed", lineCount, jsonEventCount)

	// Send final events for any remaining groups
	c.finalizePendingGroups()

	// Send runComplete event to signal processing is done
	runCompleteEvent := map[string]interface{}{
		"eventType": "runComplete",
		"payload":   map[string]interface{}{},
	}
	if err := c.ipcWriter.WriteEvent(runCompleteEvent); err != nil {
		c.logger.Debug("Failed to send runComplete event: %v", err)
	}

	return nil
}

// loadCargoMetadata loads crate metadata from cargo metadata command
func (c *CargoTestDefinition) loadCargoMetadata() {
	// TODO: In a full implementation, we would run `cargo metadata --format-version 1`
	// and parse the JSON output to get crate descriptions. For now, we'll use
	// a simplified approach that could be extended later.
	c.logger.Debug("Loading cargo metadata (placeholder for now)")
}

// processLineData processes a single line of cargo test output
func (c *CargoTestDefinition) processLineData(line string, jsonEventCount *int) {
	// Check if this is a "Running unittests" line from stderr
	if matches := runningUnittestsRegex.FindStringSubmatch(line); matches != nil {
		crateName := matches[1]
		c.mu.Lock()
		c.currentCrate = crateName
		c.logger.Debug("Set current crate to: %s (unit tests)", crateName)
		c.mu.Unlock()
		return
	}

	// Check if this is a "Running tests/..." line for integration tests
	if matches := runningIntegrationTestsRegex.FindStringSubmatch(line); matches != nil {
		testName := matches[1]
		c.mu.Lock()
		c.currentCrate = testName
		c.logger.Debug("Set current crate to: %s (integration tests)", testName)
		c.mu.Unlock()
		return
	}

	// Check if this is a "Doc-tests" line
	if strings.Contains(line, "Doc-tests") {
		c.logger.Debug("Found Doc-tests line: %s", line)
		if matches := docTestsRegex.FindStringSubmatch(line); matches != nil {
			crateName := matches[1]
			docCrateName := "doc:" + crateName
			c.mu.Lock()
			c.currentCrate = docCrateName
			c.logger.Debug("Set current crate to: %s (doc tests)", docCrateName)
			c.mu.Unlock()
			return
		} else {
			c.logger.Debug("Doc-tests line did not match regex: %s", line)
		}
	}

	// Try to parse as JSON event
	var event CargoTestEvent
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		// Not JSON, might be compilation output or other messages
		return
	}

	*jsonEventCount++
	// Process the JSON event with current crate context
	if err := c.processEvent(&event); err != nil {
		c.logger.Debug("Error processing cargo test event: %v", err)
	}
}

// processEvent processes a single cargo test JSON event
func (c *CargoTestDefinition) processEvent(event *CargoTestEvent) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch event.Type {
	case "suite":
		return c.processSuiteEvent(event)
	case "test":
		return c.processTestEvent(event)
	default:
		c.logger.Debug("Unknown cargo test event type: %s", event.Type)
	}

	return nil
}

// processSuiteEvent handles suite-level events (test run start/end)
func (c *CargoTestDefinition) processSuiteEvent(event *CargoTestEvent) error {
	c.logger.Debug("Processing suite event - Event: %s, TestCount: %d, Passed: %d, Failed: %d",
		event.Event, event.TestCount, event.Passed, event.Failed)

	switch event.Event {
	case "started":
		// A suite started event tells us how many tests to expect
		// The currentCrate should already be set by the previous stderr "Running" line
		if c.currentCrate != "" {
			// Record expected test count for this crate
			c.crateTestCounts[c.currentCrate] = event.TestCount
			c.crateTestsSeen[c.currentCrate] = 0

			// Create the crate group if it doesn't exist
			if c.crateGroups[c.currentCrate] == nil {
				c.crateGroups[c.currentCrate] = &CrateGroupInfo{
					Name:      c.currentCrate,
					StartTime: time.Now(),
					Tests:     []CargoTestInfo{},
					Status:    "RUNNING",
				}
			}

			c.logger.Debug("Suite started for crate %s with %d tests", c.currentCrate, event.TestCount)
		} else {
			c.logger.Debug("Suite started but no current crate set (test count: %d)", event.TestCount)
		}

		// Send collection start event
		c.logger.Debug("Sending collectionStart with test count: %d", event.TestCount)
		c.sendCollectionStart(event.TestCount)
	case "ok", "failed":
		// Suite finished - this means all tests for the current crate are done
		if c.currentCrate != "" {
			crateName := c.currentCrate
			c.logger.Debug("Suite finished for crate %s (passed: %d, failed: %d)",
				crateName, event.Passed, event.Failed)

			// For suites with 0 tests, we need to create and complete the group now
			// since no test events will be generated
			totalTests := event.Passed + event.Failed + event.Ignored
			if totalTests == 0 {
				// Check if this is a doc-test
				var displayCrateName string
				if strings.HasPrefix(crateName, "doc:") {
					// Doc-test: display as "Doc-tests <crate>"
					actualCrateName := strings.TrimPrefix(crateName, "doc:")
					// Convert underscores to hyphens for consistency
					actualCrateName = strings.ReplaceAll(actualCrateName, "_", "-")
					displayCrateName = "Doc-tests " + actualCrateName
				} else {
					// Regular test: convert underscores to hyphens for display
					displayCrateName = strings.ReplaceAll(crateName, "_", "-")
				}

				// Send group discovered
				if !c.discoveredGroups[crateName] {
					c.sendGroupDiscovered(displayCrateName, nil)
					c.discoveredGroups[crateName] = true
				}

				// Send group start
				if !c.groupStarts[crateName] {
					c.sendGroupStart(displayCrateName, nil)
					c.groupStarts[crateName] = true
				}

				// Send group result with 0 tests and duration from exec_time
				durationMs := event.ExecTime * 1000
				// Groups with 0 tests should have NO_TESTS status
				c.sendGroupResult(displayCrateName, nil, "NO_TESTS", durationMs, 0, 0, 0)

				// Mark this group as finalized
				if group, ok := c.crateGroups[crateName]; ok {
					group.Finalized = true
					group.Status = "NO_TESTS"
				}

				c.logger.Debug("Sent group events for empty suite: %s", displayCrateName)
			}

			// Check if we've seen all expected tests
			if expected, ok := c.crateTestCounts[crateName]; ok {
				seen := c.crateTestsSeen[crateName]
				actualTotal := event.Passed + event.Failed + event.Ignored
				if seen != expected && seen != actualTotal {
					c.logger.Debug("Warning: Expected %d tests for crate %s but saw %d (suite reports %d)",
						expected, crateName, seen, actualTotal)
				}
			}
		}

		// Determine the actual test count to send
		// When tests fail, cargo sends TestCount: 0, but we want the actual count
		testCountToSend := event.TestCount
		if testCountToSend == 0 && c.currentCrate != "" {
			// Use the test count we tracked from the suite start event
			if count, ok := c.crateTestCounts[c.currentCrate]; ok {
				testCountToSend = count
			} else {
				// Fallback to the sum of passed/failed/ignored
				testCountToSend = event.Passed + event.Failed + event.Ignored
			}
		}

		// Clear current crate since this suite is done
		if c.currentCrate != "" {
			c.currentCrate = ""
		}

		// Send collection finish event with the correct test count
		c.logger.Debug("Sending collectionFinish with test count: %d", testCountToSend)
		c.sendCollectionFinish(testCountToSend)
	}
	return nil
}

// processTestEvent handles individual test events
func (c *CargoTestDefinition) processTestEvent(event *CargoTestEvent) error {
	if event.Name == "" {
		return nil
	}

	// Use the current crate context
	crateName := c.currentCrate
	if crateName == "" {
		// No current crate - this shouldn't happen in normal flow
		c.logger.Debug("No current crate context for test: %s", event.Name)
		return nil
	}

	// Parse test name to extract module hierarchy (without crate prefix)
	// The JSON test names don't include the crate name, just module::test
	parts := strings.Split(event.Name, "::")
	if len(parts) == 0 {
		return nil
	}

	// Check if we're in a workspace
	if c.workspaceName == "" {
		// Try to detect workspace from cargo metadata (simplified for now)
		// In a real implementation, we'd run `cargo metadata` to get workspace info
		// For now, assume single crate
		c.workspaceName = ""
	}

	// Build parent hierarchy
	var parentNames []string
	if c.workspaceName != "" {
		parentNames = append(parentNames, c.workspaceName)
	}

	// Get crate metadata if available
	var crateDesc string
	if metadata, ok := c.crateMetadata[crateName]; ok {
		crateDesc = metadata.Description
	}

	// Convert underscores to hyphens for display
	var displayCrateName string
	var enhancedCrateName string

	// Check if this is a doc-test crate
	if strings.HasPrefix(crateName, "doc:") {
		// Doc-test: display as "Doc-tests <crate>"
		actualCrateName := strings.TrimPrefix(crateName, "doc:")
		// Convert underscores to hyphens for consistency
		actualCrateName = strings.ReplaceAll(actualCrateName, "_", "-")
		displayCrateName = "Doc-tests " + actualCrateName
		enhancedCrateName = displayCrateName
	} else {
		// Regular crate: convert underscores to hyphens
		displayCrateName = strings.ReplaceAll(crateName, "_", "-")
		// Build enhanced crate name with description
		enhancedCrateName = displayCrateName
		if crateDesc != "" {
			enhancedCrateName = fmt.Sprintf("%s (%s)", displayCrateName, crateDesc)
		}
	}

	// Ensure crate group exists
	if !c.discoveredGroups[crateName] {
		c.sendGroupDiscovered(enhancedCrateName, parentNames)
		c.discoveredGroups[crateName] = true
	}

	// Send group start for crate if not started
	if !c.groupStarts[crateName] {
		c.sendGroupStart(enhancedCrateName, parentNames)
		c.groupStarts[crateName] = true
		c.crateGroups[crateName] = &CrateGroupInfo{
			Name:      crateName,
			StartTime: time.Now(),
			Tests:     []CargoTestInfo{},
			Status:    "RUNNING",
		}
	}

	// Build full parent hierarchy for the test
	// Start with base hierarchy: [workspace?] + crate (use display name)
	// For doc-tests, use the full "Doc-tests <crate>" format
	testParents := append(parentNames, displayCrateName)

	// Handle nested modules as groups
	// For test "grid::storage::tests::indexing", parts = ["grid", "storage", "tests", "indexing"]
	// We need to create groups for "grid", "storage", and "tests" (all but the last part)
	if len(parts) > 1 {
		// We have module hierarchy between crate and test
		for i := 0; i < len(parts)-1; i++ {
			moduleName := parts[i]

			// Build the full parent hierarchy for this module
			// For i=0 ("grid"): moduleParents = [crate]
			// For i=1 ("storage"): moduleParents = [crate, "grid"]
			// For i=2 ("tests"): moduleParents = [crate, "grid", "storage"]
			moduleParents := make([]string, len(testParents))
			copy(moduleParents, testParents)

			// Create unique key for this module within the crate
			moduleKey := fmt.Sprintf("%s::%s", crateName, strings.Join(parts[:i+1], "::"))

			// Discover and start module group if needed
			if !c.discoveredGroups[moduleKey] {
				c.sendGroupDiscovered(moduleName, moduleParents)
				c.discoveredGroups[moduleKey] = true
			}
			if !c.groupStarts[moduleKey] {
				c.sendGroupStart(moduleName, moduleParents)
				c.groupStarts[moduleKey] = true
				// Track module group for later finalization
				if c.crateGroups[moduleKey] == nil {
					c.crateGroups[moduleKey] = &CrateGroupInfo{
						Name:      moduleName,
						StartTime: time.Now(),
						Tests:     []CargoTestInfo{},
						Status:    "RUNNING",
					}
				}
			}

			// Add this module to the parent hierarchy for the next iteration
			testParents = append(testParents, moduleName)
		}
	}

	// Extract test name (last part)
	testName := parts[len(parts)-1]

	switch event.Event {
	case "started":
		// Track test start
		c.testStates[event.Name] = &CargoTestState{
			Name:      testName,
			Crate:     crateName,
			StartTime: time.Now(),
		}

	case "ok", "failed", "ignored":
		// Track that we've seen a test for this crate (only count completed tests)
		c.crateTestsSeen[crateName]++

		// Log progress for debugging
		if expected, ok := c.crateTestCounts[crateName]; ok {
			seen := c.crateTestsSeen[crateName]
			c.logger.Debug("Crate %s: test %d/%d - %s", crateName, seen, expected, event.Name)
		}

		// Map status
		status := "PASS"
		if event.Event == "failed" {
			status = "FAIL"
		} else if event.Event == "ignored" {
			status = "SKIP"
		}

		// Send test case event (convert duration from seconds to milliseconds)
		durationMs := event.ExecTime * 1000
		c.sendTestCase(testName, testParents, status, durationMs, event.Stdout, event.Stderr)

		// Create test info
		testInfo := CargoTestInfo{
			Name:     testName,
			Status:   status,
			Duration: durationMs,
		}

		// Track test in crate group
		if group, ok := c.crateGroups[crateName]; ok {
			group.Tests = append(group.Tests, testInfo)

			// Update group status if test failed
			if status == "FAIL" && group.Status != "FAIL" {
				group.Status = "FAIL"
			}
		}

		// Track test in its immediate parent module group
		// For test "grid::storage::tests::indexing", track in "alacritty-terminal::grid::storage::tests"
		if len(parts) > 1 {
			moduleGroupKey := fmt.Sprintf("%s::%s", crateName, strings.Join(parts[:len(parts)-1], "::"))
			if group, ok := c.crateGroups[moduleGroupKey]; ok {
				group.Tests = append(group.Tests, testInfo)

				// Update group status if test failed
				if status == "FAIL" && group.Status != "FAIL" {
					group.Status = "FAIL"
				}
			}
		}

		// Clean up test state
		delete(c.testStates, event.Name)
	}

	return nil
}

// finalizePendingGroups sends result events for any groups that haven't been finalized
func (c *CargoTestDefinition) finalizePendingGroups() {
	c.logger.Debug("finalizePendingGroups called with %d groups", len(c.crateGroups))

	// Sort keys to ensure we finalize child groups before parent groups
	var groupKeys []string
	for key := range c.crateGroups {
		groupKeys = append(groupKeys, key)
	}
	// Sort by depth (more :: means deeper in hierarchy, should be finalized first)
	sort.Slice(groupKeys, func(i, j int) bool {
		return strings.Count(groupKeys[i], "::") > strings.Count(groupKeys[j], "::")
	})

	for _, groupKey := range groupKeys {
		group := c.crateGroups[groupKey]
		c.logger.Debug("Group %s has status: %s, finalized: %v", groupKey, group.Status, group.Finalized)
		// Finalize groups that haven't been finalized yet
		if !group.Finalized {
			c.logger.Debug("Finalizing group: %s", groupKey)

			// Calculate totals
			passed := 0
			failed := 0
			skipped := 0
			totalDuration := 0.0

			for _, test := range group.Tests {
				switch test.Status {
				case "PASS":
					passed++
				case "FAIL":
					failed++
				case "SKIP":
					skipped++
				}
				totalDuration += test.Duration
			}

			// Determine final status
			finalStatus := "PASS"
			total := passed + failed + skipped
			if total == 0 {
				// No tests at all - mark as NO_TESTS
				finalStatus = "NO_TESTS"
			} else if failed > 0 {
				finalStatus = "FAIL"
			} else if passed == 0 && skipped > 0 {
				finalStatus = "SKIP"
			}

			// Build parent hierarchy based on the group key
			var parentNames []string
			if c.workspaceName != "" {
				parentNames = append(parentNames, c.workspaceName)
			}

			// Parse the group key to determine if it's a module or crate
			if strings.Contains(groupKey, "::") {
				// This is a module group (e.g., "alacritty-terminal::grid::storage")
				parts := strings.Split(groupKey, "::")
				crateName := parts[0]
				modulePath := parts[1:]

				// Parent hierarchy starts with crate (use display name)
				var displayCrateName string
				if strings.HasPrefix(crateName, "doc:") {
					// Doc-test: display as "Doc-tests <crate>"
					actualCrateName := strings.TrimPrefix(crateName, "doc:")
					actualCrateName = strings.ReplaceAll(actualCrateName, "_", "-")
					displayCrateName = "Doc-tests " + actualCrateName
				} else {
					displayCrateName = strings.ReplaceAll(crateName, "_", "-")
				}
				parentNames = append(parentNames, displayCrateName)

				// Add parent modules (all but the last one)
				for i := 0; i < len(modulePath)-1; i++ {
					parentNames = append(parentNames, modulePath[i])
				}

				// The group name is the last part
				groupName := modulePath[len(modulePath)-1]
				c.sendGroupResult(groupName, parentNames, finalStatus, totalDuration, passed, failed, skipped)
			} else {
				// This is a crate group
				var displayName string
				if strings.HasPrefix(groupKey, "doc:") {
					// Doc-test group: display as "Doc-tests <crate>"
					actualCrateName := strings.TrimPrefix(groupKey, "doc:")
					// Convert underscores to hyphens in the crate name
					actualCrateName = strings.ReplaceAll(actualCrateName, "_", "-")
					displayName = "Doc-tests " + actualCrateName
				} else {
					// Regular crate group - convert underscores to hyphens for display
					displayName = strings.ReplaceAll(group.Name, "_", "-")
				}
				c.sendGroupResult(displayName, parentNames, finalStatus, totalDuration, passed, failed, skipped)
			}

			// Mark this group as finalized
			group.Status = finalStatus
			group.Finalized = true
		}
	}
}

// IPC event sending methods

func (c *CargoTestDefinition) sendCollectionStart(testCount int) {
	event := map[string]interface{}{
		"eventType": "collectionStart",
		"payload": map[string]interface{}{
			"collected": testCount,
		},
	}
	c.sendIPCEvent(event)
}

func (c *CargoTestDefinition) sendCollectionFinish(testCount int) {
	event := map[string]interface{}{
		"eventType": "collectionFinish",
		"payload": map[string]interface{}{
			"collected": testCount,
		},
	}
	c.sendIPCEvent(event)
}

func (c *CargoTestDefinition) sendGroupDiscovered(groupName string, parentNames []string) {
	event := map[string]interface{}{
		"eventType": "testGroupDiscovered",
		"payload": map[string]interface{}{
			"groupName":   groupName,
			"parentNames": parentNames,
		},
	}
	c.sendIPCEvent(event)
}

func (c *CargoTestDefinition) sendGroupStart(groupName string, parentNames []string) {
	event := map[string]interface{}{
		"eventType": "testGroupStart",
		"payload": map[string]interface{}{
			"groupName":   groupName,
			"parentNames": parentNames,
		},
	}
	c.sendIPCEvent(event)
}

func (c *CargoTestDefinition) sendTestCase(testName string, parentNames []string, status string, duration float64, stdout, stderr string) {
	payload := map[string]interface{}{
		"testName":    testName,
		"parentNames": parentNames,
		"status":      status,
		"duration":    duration,
	}

	// Only include stdout/stderr if non-empty
	if stdout != "" {
		payload["stdout"] = stdout
	}
	if stderr != "" {
		payload["stderr"] = stderr
	}

	// Include error message for failed tests
	if status == "FAIL" && stderr != "" {
		payload["error"] = map[string]interface{}{
			"message": stderr,
		}
	}

	event := map[string]interface{}{
		"eventType": "testCase",
		"payload":   payload,
	}
	c.sendIPCEvent(event)
}

func (c *CargoTestDefinition) sendGroupResult(groupName string, parentNames []string, status string, duration float64, passed, failed, skipped int) {
	event := map[string]interface{}{
		"eventType": "testGroupResult",
		"payload": map[string]interface{}{
			"groupName":   groupName,
			"parentNames": parentNames,
			"status":      status,
			"duration":    duration,
			"totals": map[string]interface{}{
				"passed":  passed,
				"failed":  failed,
				"skipped": skipped,
			},
		},
	}
	c.sendIPCEvent(event)
}

func (c *CargoTestDefinition) sendIPCEvent(event map[string]interface{}) {
	if c.ipcWriter == nil {
		c.logger.Debug("IPC writer not initialized, skipping event: %v", event)
		return
	}

	if err := c.ipcWriter.WriteEvent(event); err != nil {
		c.logger.Debug("Failed to write IPC event: %v", err)
	}
}

// SetEnvironment sets RUSTC_BOOTSTRAP=1 for unstable JSON format
func (c *CargoTestDefinition) SetEnvironment() []string {
	return []string{"RUSTC_BOOTSTRAP=1"}
}
