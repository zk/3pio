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

// CargoTestDefinition implements support for Rust's cargo test runner
type CargoTestDefinition struct {
	logger     *logger.FileLogger
	mu         sync.RWMutex
	ipcWriter  *IPCWriter

	// Workspace and crate tracking
	workspaceName    string                    // Name of workspace if detected
	crateGroups      map[string]*CrateGroupInfo // Map of crate name to group info
	discoveredGroups map[string]bool            // Track discovered groups to avoid duplicates
	groupStarts      map[string]bool            // Track started groups
	testStates       map[string]*CargoTestState // Track test state
}

// CrateGroupInfo tracks information for a crate group
type CrateGroupInfo struct {
	Name      string
	StartTime time.Time
	Tests     []CargoTestInfo
	Status    string
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
	Type      string  `json:"type"`      // "suite" or "test"
	Event     string  `json:"event"`     // "started", "ok", "failed", "ignored"
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
		crateGroups:      make(map[string]*CrateGroupInfo),
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

	// Check for full path to cargo binary
	if strings.HasSuffix(args[0], "/cargo") && len(args) > 1 && args[1] == "test" {
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

	// Add JSON output flags
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

// ProcessOutput reads cargo test JSON output and converts to IPC events
func (c *CargoTestDefinition) ProcessOutput(stdout io.Reader, ipcPath string) error {
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

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Bytes()

		// Try to parse as JSON
		var event CargoTestEvent
		if err := json.Unmarshal(line, &event); err != nil {
			// Not JSON, might be compilation output or other messages
			c.logger.Debug("Non-JSON output from cargo test: %s", string(line))
			continue
		}

		// Process the event
		if err := c.processEvent(&event); err != nil {
			c.logger.Debug("Error processing cargo test event: %v", err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading cargo test output: %w", err)
	}

	// Send final events for any pending groups
	c.finalizePendingGroups()

	return nil
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
	switch event.Event {
	case "started":
		// Send collection start event
		c.sendCollectionStart(event.TestCount)
	case "ok", "failed":
		// Send collection finish event
		c.sendCollectionFinish(event.TestCount)
		// Finalize any remaining groups
		c.finalizePendingGroups()
	}
	return nil
}

// processTestEvent handles individual test events
func (c *CargoTestDefinition) processTestEvent(event *CargoTestEvent) error {
	if event.Name == "" {
		return nil
	}

	// Parse test name to extract crate and module hierarchy
	// Format: crate::module::submodule::test_name
	parts := strings.Split(event.Name, "::")
	if len(parts) == 0 {
		return nil
	}

	// Determine crate name (first part)
	crateName := parts[0]

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

	// Ensure crate group exists
	if !c.discoveredGroups[crateName] {
		c.sendGroupDiscovered(crateName, parentNames)
		c.discoveredGroups[crateName] = true
	}

	// Send group start for crate if not started
	if !c.groupStarts[crateName] {
		c.sendGroupStart(crateName, parentNames)
		c.groupStarts[crateName] = true
		c.crateGroups[crateName] = &CrateGroupInfo{
			Name:      crateName,
			StartTime: time.Now(),
			Tests:     []CargoTestInfo{},
			Status:    "RUNNING",
		}
	}

	// Build full parent hierarchy for the test
	testParents := append(parentNames, crateName)

	// Handle nested modules as groups
	if len(parts) > 2 {
		// We have module hierarchy between crate and test
		for i := 1; i < len(parts)-1; i++ {
			moduleName := parts[i]
			moduleParents := append(parentNames, parts[:i]...)

			// Discover and start module group if needed
			moduleKey := strings.Join(parts[:i+1], "::")
			if !c.discoveredGroups[moduleKey] {
				c.sendGroupDiscovered(moduleName, moduleParents)
				c.discoveredGroups[moduleKey] = true
			}
			if !c.groupStarts[moduleKey] {
				c.sendGroupStart(moduleName, moduleParents)
				c.groupStarts[moduleKey] = true
			}

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
		// Map status
		status := "PASS"
		if event.Event == "failed" {
			status = "FAIL"
		} else if event.Event == "ignored" {
			status = "SKIP"
		}

		// Send test case event
		c.sendTestCase(testName, testParents, status, event.ExecTime, event.Stdout, event.Stderr)

		// Track test in crate group
		if group, ok := c.crateGroups[crateName]; ok {
			group.Tests = append(group.Tests, CargoTestInfo{
				Name:     testName,
				Status:   status,
				Duration: event.ExecTime,
			})

			// Update group status if test failed
			if status == "FAIL" && group.Status != "FAIL" {
				group.Status = "FAIL"
			}
		}

		// Clean up test state
		delete(c.testStates, event.Name)
	}

	return nil
}

// finalizePendingGroups sends result events for any groups that haven't been finalized
func (c *CargoTestDefinition) finalizePendingGroups() {
	for crateName, group := range c.crateGroups {
		if group.Status == "RUNNING" {
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
			if failed > 0 {
				finalStatus = "FAIL"
			} else if passed == 0 && skipped > 0 {
				finalStatus = "SKIP"
			}

			// Send group result
			var parentNames []string
			if c.workspaceName != "" {
				parentNames = append(parentNames, c.workspaceName)
			}

			c.sendGroupResult(crateName, parentNames, finalStatus, totalDuration, passed, failed, skipped)
			group.Status = finalStatus
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

	data, err := json.Marshal(event)
	if err != nil {
		c.logger.Debug("Failed to marshal IPC event: %v", err)
		return
	}

	if _, err := c.ipcWriter.Write(data); err != nil {
		c.logger.Debug("Failed to write IPC event: %v", err)
	}
}

// SetEnvironment sets RUSTC_BOOTSTRAP=1 for unstable JSON format
func (c *CargoTestDefinition) SetEnvironment() []string {
	return []string{"RUSTC_BOOTSTRAP=1"}
}