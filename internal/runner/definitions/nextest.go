package definitions

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/zk/3pio/internal/logger"
)

// NextestDefinition implements support for cargo-nextest runner
type NextestDefinition struct {
	logger     *logger.FileLogger
	mu         sync.RWMutex
	ipcWriter  *IPCWriter

	// Workspace and package tracking
	workspaceName    string                             // Name of workspace if detected
	packageGroups    map[string]*NextestPackageGroupInfo // Map of package name to group info
	discoveredGroups map[string]bool                    // Track discovered groups to avoid duplicates
	groupStarts      map[string]bool              // Track started groups
	testStates       map[string]*NextestTestState // Track test state
}

// NextestPackageGroupInfo tracks information for a package group
type NextestPackageGroupInfo struct {
	Name      string
	StartTime time.Time
	Tests     []NextestTestInfo
	Status    string
}

// NextestTestInfo tracks individual test information
type NextestTestInfo struct {
	Name     string
	Status   string
	Duration float64
}

// NextestTestState tracks the state of a running test
type NextestTestState struct {
	Name      string
	Package   string
	StartTime time.Time
	Output    []string
}

// NextestEvent represents a single event from cargo nextest --message-format libtest-json output
type NextestEvent struct {
	Type     string  `json:"type"`     // "test" or "suite"
	Event    string  `json:"event"`    // "started", "ok", "failed", "ignored", "finished"
	Name     string  `json:"name,omitempty"`
	Passed   int     `json:"passed,omitempty"`
	Failed   int     `json:"failed,omitempty"`
	Ignored  int     `json:"ignored,omitempty"`
	ExecTime float64 `json:"exec_time,omitempty"`
	Stdout   string  `json:"stdout,omitempty"`
	Stderr   string  `json:"stderr,omitempty"`
}

// NewNextestDefinition creates a new cargo-nextest runner definition
func NewNextestDefinition(logger *logger.FileLogger) *NextestDefinition {
	return &NextestDefinition{
		logger:           logger,
		packageGroups:    make(map[string]*NextestPackageGroupInfo),
		discoveredGroups: make(map[string]bool),
		groupStarts:      make(map[string]bool),
		testStates:       make(map[string]*NextestTestState),
	}
}

// Name returns the name of this test runner
func (n *NextestDefinition) Name() string {
	return "nextest"
}

// Detect checks if the command is for cargo nextest
func (n *NextestDefinition) Detect(args []string) bool {
	if len(args) < 2 {
		return false
	}

	// Check for "cargo nextest" command (with or without "run")
	if args[0] == "cargo" && args[1] == "nextest" {
		return true
	}

	// Check for "cargo +<toolchain> nextest" command
	if args[0] == "cargo" && len(args) > 2 && strings.HasPrefix(args[1], "+") && args[2] == "nextest" {
		return true
	}

	// Check for full path to cargo binary with nextest
	if strings.HasSuffix(args[0], "/cargo") && len(args) > 1 && args[1] == "nextest" {
		return true
	}

	// Check for full path with toolchain
	if strings.HasSuffix(args[0], "/cargo") && len(args) > 2 && strings.HasPrefix(args[1], "+") && args[2] == "nextest" {
		return true
	}

	return false
}

// ModifyCommand adds JSON output flags to cargo nextest command
func (n *NextestDefinition) ModifyCommand(cmd []string, ipcPath, runID string) []string {
	result := make([]string, 0, len(cmd)+4)

	// Copy original command
	result = append(result, cmd...)

	// If "run" is not present, add it
	hasRun := false
	for _, arg := range cmd {
		if arg == "run" {
			hasRun = true
			break
		}
	}
	if !hasRun {
		result = append(result, "run")
	}

	// Add JSON output format
	result = append(result, "--message-format", "libtest-json")

	return result
}

// GetTestFiles returns empty array for dynamic discovery
func (n *NextestDefinition) GetTestFiles(args []string) ([]string, error) {
	// Nextest tests are discovered dynamically as they run
	return []string{}, nil
}

// RequiresAdapter returns false as nextest doesn't need an external adapter
func (n *NextestDefinition) RequiresAdapter() bool {
	return false
}

// ProcessOutput reads nextest JSON output and converts to IPC events
func (n *NextestDefinition) ProcessOutput(stdout io.Reader, ipcPath string) error {
	// Initialize IPC writer
	var err error
	n.ipcWriter, err = NewIPCWriter(ipcPath)
	if err != nil {
		return fmt.Errorf("failed to create IPC writer: %w", err)
	}
	defer func() {
		if err := n.ipcWriter.Close(); err != nil {
			n.logger.Debug("Failed to close IPC writer: %v", err)
		}
	}()

	scanner := bufio.NewScanner(stdout)
	testCount := 0

	for scanner.Scan() {
		line := scanner.Bytes()

		// Try to parse as JSON
		var event NextestEvent
		if err := json.Unmarshal(line, &event); err != nil {
			// Not JSON, might be compilation output or other messages
			n.logger.Debug("Non-JSON output from nextest: %s", string(line))
			continue
		}

		// Process the event
		if err := n.processEvent(&event, &testCount); err != nil {
			n.logger.Debug("Error processing nextest event: %v", err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading nextest output: %w", err)
	}

	// Send final events for any pending groups
	n.finalizePendingGroups()

	return nil
}

// processEvent processes a single nextest JSON event
func (n *NextestDefinition) processEvent(event *NextestEvent, testCount *int) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	switch event.Type {
	case "suite":
		return n.processSuiteEvent(event, testCount)
	case "test":
		return n.processTestEvent(event, testCount)
	default:
		n.logger.Debug("Unknown nextest event type: %s", event.Type)
	}

	return nil
}

// processSuiteEvent handles suite-level events (test run start/end)
func (n *NextestDefinition) processSuiteEvent(event *NextestEvent, testCount *int) error {
	switch event.Event {
	case "started":
		// Nextest doesn't provide test count at start, track as we go
		n.sendCollectionStart(0)
	case "finished":
		// Send collection finish event
		n.sendCollectionFinish(*testCount)
		// Finalize any remaining groups
		n.finalizePendingGroups()
	}
	return nil
}

// processTestEvent handles individual test events
func (n *NextestDefinition) processTestEvent(event *NextestEvent, testCount *int) error {
	if event.Name == "" {
		return nil
	}

	// Parse test name to extract package and module hierarchy
	// Nextest format often includes package name: package_name::module::test_name
	parts := strings.Split(event.Name, "::")
	if len(parts) == 0 {
		return nil
	}

	// Determine package name (first part, often with hyphens converted to underscores)
	packageName := parts[0]

	// Check if we're in a workspace
	if n.workspaceName == "" {
		// Try to detect workspace from cargo metadata (simplified for now)
		// In a real implementation, we'd run `cargo metadata` to get workspace info
		// For now, check if package name looks like it's from a workspace
		if strings.Contains(packageName, "_") {
			// Might be a workspace package, but we'll treat it as standalone for now
		}
	}

	// Build parent hierarchy
	var parentNames []string
	if n.workspaceName != "" {
		parentNames = append(parentNames, n.workspaceName)
	}

	// Ensure package group exists
	if !n.discoveredGroups[packageName] {
		n.sendGroupDiscovered(packageName, parentNames)
		n.discoveredGroups[packageName] = true
	}

	// Send group start for package if not started
	if !n.groupStarts[packageName] {
		n.sendGroupStart(packageName, parentNames)
		n.groupStarts[packageName] = true
		n.packageGroups[packageName] = &NextestPackageGroupInfo{
			Name:      packageName,
			StartTime: time.Now(),
			Tests:     []NextestTestInfo{},
			Status:    "RUNNING",
		}
	}

	// Build full parent hierarchy for the test
	testParents := append(parentNames, packageName)

	// Handle nested modules as groups
	if len(parts) > 2 {
		// We have module hierarchy between package and test
		for i := 1; i < len(parts)-1; i++ {
			moduleName := parts[i]
			moduleParents := append(parentNames, parts[:i]...)

			// Discover and start module group if needed
			moduleKey := strings.Join(parts[:i+1], "::")
			if !n.discoveredGroups[moduleKey] {
				n.sendGroupDiscovered(moduleName, moduleParents)
				n.discoveredGroups[moduleKey] = true
			}
			if !n.groupStarts[moduleKey] {
				n.sendGroupStart(moduleName, moduleParents)
				n.groupStarts[moduleKey] = true
			}

			testParents = append(testParents, moduleName)
		}
	}

	// Extract test name (last part)
	testName := parts[len(parts)-1]

	switch event.Event {
	case "started":
		// Track test start
		n.testStates[event.Name] = &NextestTestState{
			Name:      testName,
			Package:   packageName,
			StartTime: time.Now(),
		}
		(*testCount)++

	case "ok", "failed", "ignored":
		// Map status
		status := "PASS"
		if event.Event == "failed" {
			status = "FAIL"
		} else if event.Event == "ignored" {
			status = "SKIP"
		}

		// Send test case event
		n.sendTestCase(testName, testParents, status, event.ExecTime, event.Stdout, event.Stderr)

		// Track test in package group
		if group, ok := n.packageGroups[packageName]; ok {
			group.Tests = append(group.Tests, NextestTestInfo{
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
		delete(n.testStates, event.Name)
	}

	return nil
}

// finalizePendingGroups sends result events for any groups that haven't been finalized
func (n *NextestDefinition) finalizePendingGroups() {
	for packageName, group := range n.packageGroups {
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
			if n.workspaceName != "" {
				parentNames = append(parentNames, n.workspaceName)
			}

			n.sendGroupResult(packageName, parentNames, finalStatus, totalDuration, passed, failed, skipped)
			group.Status = finalStatus
		}
	}
}

// IPC event sending methods

func (n *NextestDefinition) sendCollectionStart(testCount int) {
	event := map[string]interface{}{
		"eventType": "collectionStart",
		"payload": map[string]interface{}{
			"collected": testCount,
		},
	}
	n.sendIPCEvent(event)
}

func (n *NextestDefinition) sendCollectionFinish(testCount int) {
	event := map[string]interface{}{
		"eventType": "collectionFinish",
		"payload": map[string]interface{}{
			"collected": testCount,
		},
	}
	n.sendIPCEvent(event)
}

func (n *NextestDefinition) sendGroupDiscovered(groupName string, parentNames []string) {
	event := map[string]interface{}{
		"eventType": "testGroupDiscovered",
		"payload": map[string]interface{}{
			"groupName":   groupName,
			"parentNames": parentNames,
		},
	}
	n.sendIPCEvent(event)
}

func (n *NextestDefinition) sendGroupStart(groupName string, parentNames []string) {
	event := map[string]interface{}{
		"eventType": "testGroupStart",
		"payload": map[string]interface{}{
			"groupName":   groupName,
			"parentNames": parentNames,
		},
	}
	n.sendIPCEvent(event)
}

func (n *NextestDefinition) sendTestCase(testName string, parentNames []string, status string, duration float64, stdout, stderr string) {
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
	n.sendIPCEvent(event)
}

func (n *NextestDefinition) sendGroupResult(groupName string, parentNames []string, status string, duration float64, passed, failed, skipped int) {
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
	n.sendIPCEvent(event)
}

func (n *NextestDefinition) sendIPCEvent(event map[string]interface{}) {
	if n.ipcWriter == nil {
		n.logger.Debug("IPC writer not initialized, skipping event: %v", event)
		return
	}

	if err := n.ipcWriter.WriteEvent(event); err != nil {
		n.logger.Debug("Failed to write IPC event: %v", err)
	}
}

// SetEnvironment returns the required environment variable for nextest JSON output
func (n *NextestDefinition) SetEnvironment() []string {
	return []string{"NEXTEST_EXPERIMENTAL_LIBTEST_JSON=1"}
}