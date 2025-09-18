package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/zk/3pio/internal/ipc"
)

// Logger interface for debug logging
type Logger interface {
	Debug(format string, args ...interface{})
	Error(format string, args ...interface{})
	Info(format string, args ...interface{})
}

// GroupManager manages the hierarchical test group state
type GroupManager struct {
	mu         sync.RWMutex
	groups     map[string]*TestGroup // ID -> Group
	rootGroups []*TestGroup          // Top-level groups (typically files)
	runDir     string
	ipcPath    string
	logger     Logger

	// Debouncing for report generation
	pendingUpdates map[string]time.Time // Group ID -> last update time
	updateTimer    *time.Timer
	updateMutex    sync.Mutex
}

// NewGroupManager creates a new GroupManager instance
func NewGroupManager(runDir string, ipcPath string, logger Logger) *GroupManager {
	return &GroupManager{
		groups:         make(map[string]*TestGroup),
		rootGroups:     make([]*TestGroup, 0),
		runDir:         runDir,
		ipcPath:        ipcPath,
		logger:         logger,
		pendingUpdates: make(map[string]time.Time),
	}
}

// logDebug logs a debug message if logger is available
func (gm *GroupManager) logDebug(format string, args ...interface{}) {
	if gm.logger != nil {
		gm.logger.Debug(format, args...)
	}
}

// logInfo logs an info message if logger is available
func (gm *GroupManager) logInfo(format string, args ...interface{}) {
	if gm.logger != nil {
		gm.logger.Info(format, args...)
	}
}

// logError logs an error message if logger is available
func (gm *GroupManager) logError(format string, args ...interface{}) {
	if gm.logger != nil {
		gm.logger.Error(format, args...)
	}
}

// normalizeToAbsolutePath converts any path to an absolute path for consistent storage
func (gm *GroupManager) normalizeToAbsolutePath(name string) string {
	// If it's not a file path (e.g., test names, suite names), return as-is
	if !strings.HasPrefix(name, "/") && !strings.HasPrefix(name, "./") && !strings.Contains(name, "/") {
		return name
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(name)
	if err != nil {
		// If we can't get absolute path, return original
		return name
	}

	// Always attempt to resolve symlinks for absolute paths
	// This is crucial for macOS where /tmp is a symlink to /private/tmp
	resolved, err := filepath.EvalSymlinks(absPath)
	if err == nil {
		return resolved
	}

	// If symlink resolution fails, it might be because:
	// 1. The file doesn't exist yet (which is ok for test group names)
	// 2. There's no symlink to resolve
	// In either case, return the absolute path
	return absPath
}

// makeRelativePath converts absolute paths to relative for display purposes only
// nolint:unused // relative path helper retained for future path normalization
func (gm *GroupManager) makeRelativePath(name string) string {
	// Only convert if it looks like an absolute file path
	if !strings.HasPrefix(name, "/") && !strings.HasPrefix(name, "./") {
		// Not a path, return as-is (e.g., test names, suite names)
		return name
	}

	// Try to make relative to current working directory
	if cwd, err := os.Getwd(); err == nil {
		if relPath, err := filepath.Rel(cwd, name); err == nil {
			// Ensure relative paths start with ./
			if !strings.HasPrefix(relPath, ".") && !strings.HasPrefix(relPath, "/") {
				relPath = "./" + relPath
			}
			return relPath
		}
	}

	// If we can't make it relative, return as-is
	return name
}

// ProcessGroupDiscovered handles a group discovery event
func (gm *GroupManager) ProcessGroupDiscovered(event ipc.GroupDiscoveredEvent) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	payload := event.Payload

	// Normalize paths to absolute for consistent storage
	groupName := gm.normalizeToAbsolutePath(payload.GroupName)
	parentNames := make([]string, len(payload.ParentNames))
	for i, name := range payload.ParentNames {
		parentNames[i] = gm.normalizeToAbsolutePath(name)
	}

	groupID := GenerateGroupID(groupName, parentNames)

	// Check if group already exists
	if _, exists := gm.groups[groupID]; exists {
		// Group already discovered, this is idempotent
		return nil
	}

	// Create the new group
	group := &TestGroup{
		ID:          groupID,
		Name:        groupName,
		ParentNames: parentNames,
		Depth:       len(parentNames),
		Status:      TestStatusPending,
		Created:     time.Now(),
		Updated:     time.Now(),
		Subgroups:   make(map[string]*TestGroup),
		TestCases:   make([]TestCase, 0),
	}

	// Store in groups map
	gm.groups[groupID] = group

	// Handle parent relationship
	if len(payload.ParentNames) == 0 {
		// This is a root group
		gm.rootGroups = append(gm.rootGroups, group)
	} else {
		// Find or create parent groups
		if err := gm.ensureParentHierarchy(group); err != nil {
			gm.logDebug("Failed to ensure parent hierarchy: %v", err)
		}
	}

	gm.logInfo("Discovered group: %s (ID: %s)",
		BuildHierarchicalPath(group), groupID)

	return nil
}

// ProcessGroupStart handles a group start event
func (gm *GroupManager) ProcessGroupStart(event ipc.GroupStartEvent) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	payload := event.Payload

	// Normalize paths to absolute for consistent storage
	groupName := gm.normalizeToAbsolutePath(payload.GroupName)
	parentNames := make([]string, len(payload.ParentNames))
	for i, name := range payload.ParentNames {
		parentNames[i] = gm.normalizeToAbsolutePath(name)
	}

	groupID := GenerateGroupID(groupName, parentNames)

	group, exists := gm.groups[groupID]
	if !exists {
		// Auto-discover the group if not already known
		gm.mu.Unlock()
		err := gm.ProcessGroupDiscovered(ipc.GroupDiscoveredEvent{
			EventType: string(ipc.EventTypeGroupDiscovered),
			Payload: ipc.GroupDiscoveredPayload{
				GroupName:   groupName,
				ParentNames: parentNames,
			},
		})
		gm.mu.Lock()
		if err != nil {
			return err
		}
		group = gm.groups[groupID]
	}

	group.Status = TestStatusRunning
	group.StartTime = time.Now()
	group.Updated = time.Now()

	// Schedule report update
	gm.scheduleReportUpdate(groupID)

	gm.logInfo("Started group: %s", BuildHierarchicalPath(group))

	return nil
}

// ProcessGroupResult handles a group completion event
func (gm *GroupManager) ProcessGroupResult(event ipc.GroupResultEvent) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	payload := event.Payload

	// Normalize paths to absolute for consistent storage
	groupName := gm.normalizeToAbsolutePath(payload.GroupName)
	parentNames := make([]string, len(payload.ParentNames))
	for i, name := range payload.ParentNames {
		parentNames[i] = gm.normalizeToAbsolutePath(name)
	}

	groupID := GenerateGroupID(groupName, parentNames)

	group, exists := gm.groups[groupID]
	if !exists {
		// Auto-discover the group if not already known
		gm.mu.Unlock()
		err := gm.ProcessGroupDiscovered(ipc.GroupDiscoveredEvent{
			EventType: string(ipc.EventTypeGroupDiscovered),
			Payload: ipc.GroupDiscoveredPayload{
				GroupName:   groupName,
				ParentNames: parentNames,
			},
		})
		gm.mu.Lock()
		if err != nil {
			return err
		}
		group = gm.groups[groupID]
	}

	// Update group status
	switch payload.Status {
	case "PASS":
		group.Status = TestStatusPass
	case "FAIL":
		group.Status = TestStatusFail
	case "SKIP":
		group.Status = TestStatusSkip
	case "NO_TESTS":
		group.Status = TestStatusNoTests
	default:
		group.Status = TestStatusPending
	}

	group.EndTime = time.Now()
	// Use provided duration directly if available
	if payload.Duration > 0 {
		group.Duration = time.Duration(payload.Duration) * time.Millisecond
	} else if !group.StartTime.IsZero() {
		group.Duration = group.EndTime.Sub(group.StartTime)
	}
	group.Updated = time.Now()

	// Update statistics if provided
	if payload.Totals.Total > 0 || payload.Totals.Passed > 0 ||
		payload.Totals.Failed > 0 || payload.Totals.Skipped > 0 {
		group.Stats.PassedTests = payload.Totals.Passed
		group.Stats.FailedTests = payload.Totals.Failed
		group.Stats.SkippedTests = payload.Totals.Skipped
		group.Stats.TotalTests = payload.Totals.Total
		if group.Stats.TotalTests == 0 {
			group.Stats.TotalTests = payload.Totals.Passed + payload.Totals.Failed + payload.Totals.Skipped
		}
	}

	// Propagate completion to ancestors
	gm.propagateCompletion(group)

	// Schedule report update
	gm.scheduleReportUpdate(groupID)

	gm.logInfo("Completed group: %s [%s]",
		BuildHierarchicalPath(group), group.Status)

	return nil
}

// ProcessGroupError handles a group error event
func (gm *GroupManager) ProcessGroupError(event ipc.GroupErrorEvent) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	payload := event.Payload

	// Normalize paths to absolute for consistent storage
	groupName := gm.normalizeToAbsolutePath(payload.GroupName)
	parentNames := make([]string, len(payload.ParentNames))
	for i, name := range payload.ParentNames {
		parentNames[i] = gm.normalizeToAbsolutePath(name)
	}

	groupID := GenerateGroupID(groupName, parentNames)

	group, exists := gm.groups[groupID]
	if !exists {
		// Auto-discover the group if it doesn't exist
		err := gm.ensureGroupHierarchy(append(parentNames, groupName))
		if err != nil {
			return err
		}
		group = gm.groups[groupID]
	}

	// Set error status
	group.Status = TestStatusError
	group.EndTime = time.Now()

	// Use provided duration if available
	if payload.Duration > 0 {
		group.Duration = time.Duration(payload.Duration) * time.Millisecond
	} else if !group.StartTime.IsZero() {
		group.Duration = group.EndTime.Sub(group.StartTime)
	}
	group.Updated = time.Now()

	// Store error information
	if payload.Error != nil {
		group.ErrorInfo = &TestError{
			Message: payload.Error.Message,
			Type:    payload.ErrorType,
		}
	}

	// Set totals to indicate setup failure for group results display
	group.Stats.SetupFailed = true

	// Propagate completion to ancestors
	gm.propagateCompletion(group)

	// Schedule report update
	gm.scheduleReportUpdate(groupID)

	errorMessage := "unknown error"
	if payload.Error != nil {
		errorMessage = payload.Error.Message
	}
	gm.logInfo("Group error: %s [%s] - %s",
		BuildHierarchicalPath(group), payload.ErrorType, errorMessage)

	return nil
}

// ProcessTestCase handles a test case event
func (gm *GroupManager) ProcessTestCase(event ipc.GroupTestCaseEvent) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	payload := event.Payload

	// Normalize paths to absolute for consistent storage
	parentNames := make([]string, len(payload.ParentNames))
	for i, name := range payload.ParentNames {
		parentNames[i] = gm.normalizeToAbsolutePath(name)
	}

	// The test's parent is the full parent hierarchy
	parentID := GenerateGroupIDFromPath(parentNames)

	// Find or create the parent group
	var parentGroup *TestGroup
	if len(parentNames) > 0 {
		var exists bool
		parentGroup, exists = gm.groups[parentID]
		if !exists {
			// Auto-discover parent group hierarchy
			gm.mu.Unlock()
			err := gm.ensureGroupHierarchy(parentNames)
			gm.mu.Lock()
			if err != nil {
				return err
			}
			parentGroup = gm.groups[parentID]
		}
	}

	if parentGroup == nil {
		return fmt.Errorf("unable to find or create parent group for test: %s", payload.TestName)
	}

	// Create the test case
	testCase := TestCase{
		ID:        GenerateTestCaseID(payload.TestName, parentNames),
		GroupID:   parentID,
		Name:      payload.TestName,
		StartTime: time.Now(),
	}

	// Set status
	switch payload.Status {
	case "PASS":
		testCase.Status = TestStatusPass
	case "FAIL":
		testCase.Status = TestStatusFail
	case "SKIP":
		testCase.Status = TestStatusSkip
	default:
		testCase.Status = TestStatusPending
	}

	// Set duration
	if payload.Duration > 0 {
		testCase.Duration = time.Duration(payload.Duration) * time.Millisecond
	}
	testCase.EndTime = time.Now()

	// Set error if present
	if payload.Error != nil {
		testCase.Error = &TestError{
			Message:  payload.Error.Message,
			Stack:    payload.Error.Stack,
			Expected: payload.Error.Expected,
			Actual:   payload.Error.Actual,
			Location: payload.Error.Location,
			Type:     payload.Error.ErrorType,
		}
	}

	// Set output if present
	testCase.Stdout = payload.Stdout
	testCase.Stderr = payload.Stderr

	// Check if test case already exists (deduplication)
	testExists := false
	for i, existingTest := range parentGroup.TestCases {
		if existingTest.ID == testCase.ID {
			// Update existing test case instead of adding duplicate
			parentGroup.TestCases[i] = testCase
			testExists = true
			break
		}
	}

	// Add to parent group only if it doesn't exist
	if !testExists {
		parentGroup.TestCases = append(parentGroup.TestCases, testCase)
	}
	parentGroup.Updated = time.Now()

	// Update parent group statistics
	parentGroup.UpdateStats()

	// Schedule report update
	gm.scheduleReportUpdate(parentID)

	gm.logInfo("Test case: %s → %s [%s]",
		BuildHierarchicalPathFromSlice(payload.ParentNames),
		payload.TestName, testCase.Status)

	return nil
}

// ProcessStdoutChunk handles stdout output for a group
func (gm *GroupManager) ProcessStdoutChunk(groupName string, parentNames []string, chunk string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	// Normalize paths to absolute for consistent storage
	normalizedGroupName := gm.normalizeToAbsolutePath(groupName)
	normalizedParentNames := make([]string, len(parentNames))
	for i, name := range parentNames {
		normalizedParentNames[i] = gm.normalizeToAbsolutePath(name)
	}

	groupID := GenerateGroupID(normalizedGroupName, normalizedParentNames)
	group, exists := gm.groups[groupID]
	if !exists {
		// Ignore output for unknown groups
		return nil
	}

	group.Stdout += chunk
	group.Updated = time.Now()

	// Schedule debounced report update
	gm.scheduleReportUpdate(groupID)

	return nil
}

// ProcessStderrChunk handles stderr output for a group
func (gm *GroupManager) ProcessStderrChunk(groupName string, parentNames []string, chunk string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	// Normalize paths to absolute for consistent storage
	normalizedGroupName := gm.normalizeToAbsolutePath(groupName)
	normalizedParentNames := make([]string, len(parentNames))
	for i, name := range parentNames {
		normalizedParentNames[i] = gm.normalizeToAbsolutePath(name)
	}

	groupID := GenerateGroupID(normalizedGroupName, normalizedParentNames)
	group, exists := gm.groups[groupID]
	if !exists {
		// Ignore output for unknown groups
		return nil
	}

	group.Stderr += chunk
	group.Updated = time.Now()

	// Schedule debounced report update
	gm.scheduleReportUpdate(groupID)

	return nil
}

// ensureParentHierarchy ensures all parent groups exist for a given group
func (gm *GroupManager) ensureParentHierarchy(group *TestGroup) error {
	if len(group.ParentNames) == 0 {
		return nil
	}

	// Build parent hierarchy from root to immediate parent
	for i := 1; i <= len(group.ParentNames); i++ {
		parentPath := group.ParentNames[:i]
		parentName := parentPath[len(parentPath)-1]
		var grandparentNames []string
		if len(parentPath) > 1 {
			grandparentNames = parentPath[:len(parentPath)-1]
		}

		parentID := GenerateGroupIDFromPath(parentPath)

		if _, exists := gm.groups[parentID]; !exists {
			// Create parent group
			parent := &TestGroup{
				ID:          parentID,
				Name:        parentName,
				ParentNames: grandparentNames,
				Depth:       len(grandparentNames),
				Status:      TestStatusPending,
				Created:     time.Now(),
				Updated:     time.Now(),
				Subgroups:   make(map[string]*TestGroup),
				TestCases:   make([]TestCase, 0),
			}

			gm.groups[parentID] = parent

			// Add to root groups if this is a root
			if len(grandparentNames) == 0 {
				gm.rootGroups = append(gm.rootGroups, parent)
			}
		}
	}

	// Link the group to its immediate parent
	parentID := GetParentGroupID(group.ParentNames)
	if parent, exists := gm.groups[parentID]; exists {
		parent.Subgroups[group.ID] = group
		group.ParentID = parentID
	}

	return nil
}

// ensureGroupHierarchy ensures a full hierarchy exists given a path
// IMPORTANT: Must be called with gm.mu lock held
func (gm *GroupManager) ensureGroupHierarchy(path []string) error {
	if len(path) == 0 {
		return nil
	}

	// Normalize the entire path first
	normalizedPath := make([]string, len(path))
	for j, p := range path {
		normalizedPath[j] = gm.normalizeToAbsolutePath(p)
	}

	// Create each level of the hierarchy
	for i := 1; i <= len(normalizedPath); i++ {
		currentPath := normalizedPath[:i]
		groupName := currentPath[len(currentPath)-1]
		var parentNames []string
		if len(currentPath) > 1 {
			parentNames = currentPath[:len(currentPath)-1]
		}

		groupID := GenerateGroupIDFromPath(currentPath)

		if _, exists := gm.groups[groupID]; !exists {
			// Create the group directly without calling ProcessGroupDiscovered
			// since we already have the lock
			group := &TestGroup{
				ID:          groupID,
				Name:        groupName,
				ParentNames: parentNames,
				Depth:       len(parentNames),
				Status:      TestStatusPending,
				Created:     time.Now(),
				Updated:     time.Now(),
				Subgroups:   make(map[string]*TestGroup),
				TestCases:   make([]TestCase, 0),
			}

			gm.groups[groupID] = group

			// Handle parent relationship
			if len(parentNames) == 0 {
				// This is a root group
				gm.rootGroups = append(gm.rootGroups, group)
			} else {
				// Link to parent
				parentID := GetParentGroupID(parentNames)
				if parent, exists := gm.groups[parentID]; exists {
					parent.Subgroups[groupID] = group
					group.ParentID = parentID
				}
			}
		}
	}

	return nil
}

// propagateCompletion propagates completion status up the hierarchy
func (gm *GroupManager) propagateCompletion(group *TestGroup) {
	if group.ParentID == "" {
		return
	}

	parent, exists := gm.groups[group.ParentID]
	if !exists {
		return
	}

	// Check if all children are complete
	allComplete := true
	for _, subgroup := range parent.Subgroups {
		if !subgroup.IsComplete() {
			allComplete = false
			break
		}
	}

	if allComplete {
		// Check test cases in parent
		for _, tc := range parent.TestCases {
			if tc.Status == TestStatusPending || tc.Status == TestStatusRunning {
				allComplete = false
				break
			}
		}
	}

	if allComplete && parent.Status != TestStatusPass &&
		parent.Status != TestStatusFail && parent.Status != TestStatusSkip {
		// Update parent status based on children
		parent.UpdateStats()

		// Recursively propagate to grandparent
		gm.propagateCompletion(parent)

		// Schedule report update for parent
		gm.scheduleReportUpdate(parent.ID)
	}
}

// scheduleReportUpdate schedules a debounced report update for a group
func (gm *GroupManager) scheduleReportUpdate(groupID string) {
	gm.updateMutex.Lock()
	defer gm.updateMutex.Unlock()

	gm.pendingUpdates[groupID] = time.Now()

	// Cancel existing timer
	if gm.updateTimer != nil {
		gm.updateTimer.Stop()
	}

	// Schedule new update after 100ms of inactivity
	gm.updateTimer = time.AfterFunc(100*time.Millisecond, func() {
		gm.flushPendingUpdates()
	})
}

// flushPendingUpdates generates reports for all pending updates
func (gm *GroupManager) flushPendingUpdates() {
	gm.updateMutex.Lock()
	updates := make(map[string]time.Time)
	for k, v := range gm.pendingUpdates {
		updates[k] = v
	}
	gm.pendingUpdates = make(map[string]time.Time)
	gm.updateMutex.Unlock()

	gm.mu.RLock()
	defer gm.mu.RUnlock()

	for groupID := range updates {
		if group, exists := gm.groups[groupID]; exists {
			if err := gm.generateGroupReport(group); err != nil {
				gm.logError("Failed to generate report for group %s: %v",
					groupID, err)
			}
		}
	}
}

// generateGroupReport generates a report file for a group
func (gm *GroupManager) generateGroupReport(group *TestGroup) error {
	reportPath := GetReportFilePath(group, gm.runDir)

	// Ensure directory exists
	reportDir := filepath.Dir(reportPath)
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		return fmt.Errorf("failed to create report directory: %w", err)
	}

	// Generate report content
	content := gm.formatGroupReport(group)

	// Write report file
	if err := os.WriteFile(reportPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write report file: %w", err)
	}

	return nil
}

// formatGroupReport formats a group's data as a markdown report
func (gm *GroupManager) formatGroupReport(group *TestGroup) string {
	var content string

	// Use the ParentNames field which already contains the hierarchy
	parentPath := group.ParentNames

	// Metadata (YAML frontmatter) - MUST come first
	content += "---\n"
	content += fmt.Sprintf("group_name: %s\n", group.Name)

	// Parent path as slash-separated list for frontmatter
	if len(parentPath) > 0 {
		content += fmt.Sprintf("parent_path: %s\n", strings.Join(parentPath, "/"))
	} else {
		content += "parent_path:\n"
	}

	content += fmt.Sprintf("status: %s\n", group.Status)

	// Format duration - use seconds for all groups consistently
	if group.Duration > 0 {
		seconds := group.Duration.Seconds()
		content += fmt.Sprintf("duration: %.2fs\n", seconds)
	}

	content += fmt.Sprintf("created: %s\n", group.Created.Format(time.RFC3339))
	content += fmt.Sprintf("updated: %s\n", group.Updated.Format(time.RFC3339))
	content += "---\n\n"

	// Header - use consistent "Test Report:" format for all groups
	if len(parentPath) == 0 {
		// Root group (file)
		content += fmt.Sprintf("# Test Report: %s\n\n", group.Name)
	} else {
		// All non-root groups show full hierarchical path
		fullPath := strings.Join(append(parentPath, group.Name), " > ")
		content += fmt.Sprintf("# Test Report: %s\n\n", fullPath)
	}

	// Summary section - show direct tests OR subgroups, not both aggregated counts
	content += "## Summary\n\n"

	// Only show direct test statistics if there are direct test cases
	if len(group.TestCases) > 0 {
		content += fmt.Sprintf("- Group tests: %d\n", group.Stats.TotalTests)
		if group.Stats.PassedTests > 0 {
			content += fmt.Sprintf("- Group tests passed: %d\n", group.Stats.PassedTests)
		}
		if group.Stats.FailedTests > 0 {
			content += fmt.Sprintf("- Group tests failed: %d\n", group.Stats.FailedTests)
		}
		if group.Stats.SkippedTests > 0 {
			content += fmt.Sprintf("- Group tests skipped: %d\n", group.Stats.SkippedTests)
		}

		// Also show subgroup counts if we have both direct tests and subgroups
		if len(group.Subgroups) > 0 {
			passedSubgroups := 0
			failedSubgroups := 0
			skippedSubgroups := 0
			for _, sg := range group.Subgroups {
				switch sg.Status {
				case TestStatusPass:
					passedSubgroups++
				case TestStatusFail:
					failedSubgroups++
				case TestStatusSkip:
					skippedSubgroups++
				}
			}

			content += fmt.Sprintf("- Subgroups: %d\n", len(group.Subgroups))
			if passedSubgroups > 0 {
				content += fmt.Sprintf("- Subgroups passed: %d\n", passedSubgroups)
			}
			if failedSubgroups > 0 {
				content += fmt.Sprintf("- Subgroups failed: %d\n", failedSubgroups)
			}
			if skippedSubgroups > 0 {
				content += fmt.Sprintf("- Subgroups skipped: %d\n", skippedSubgroups)
			}
		}
	} else if len(group.Subgroups) > 0 {
		// Only show subgroup statistics when there are no direct tests
		passedSubgroups := 0
		failedSubgroups := 0
		skippedSubgroups := 0
		for _, sg := range group.Subgroups {
			switch sg.Status {
			case TestStatusPass:
				passedSubgroups++
			case TestStatusFail:
				failedSubgroups++
			case TestStatusSkip:
				skippedSubgroups++
			}
		}

		content += fmt.Sprintf("- Subgroups: %d\n", len(group.Subgroups))
		if passedSubgroups > 0 {
			content += fmt.Sprintf("- Subgroups passed: %d\n", passedSubgroups)
		}
		if failedSubgroups > 0 {
			content += fmt.Sprintf("- Subgroups failed: %d\n", failedSubgroups)
		}
		if skippedSubgroups > 0 {
			content += fmt.Sprintf("- Subgroups skipped: %d\n", skippedSubgroups)
		}
	}
	content += "\n"

	// Test case results section - only show if there are test cases
	if len(group.TestCases) > 0 {
		content += "## Test case results\n\n"
		for _, tc := range group.TestCases {
			var icon string
			switch tc.Status {
			case TestStatusFail:
				icon = "✕"
			case TestStatusSkip:
				icon = "○"
			default:
				icon = "✓"
			}

			content += fmt.Sprintf("- %s %s", icon, tc.Name)
			if tc.Duration > 0 {
				content += fmt.Sprintf(" (%.2fs)", tc.Duration.Seconds())
			}
			content += "\n"

			// Error details indented under the test
			if tc.Error != nil && tc.Status == TestStatusFail {
				content += "```\n"
				content += tc.Error.Message
				if tc.Error.Stack != "" {
					content += "\n" + tc.Error.Stack
				}
				content += "\n```\n"
			}
		}
		content += "\n"
	}

	// Subgroups
	if len(group.Subgroups) > 0 {
		content += "## Subgroups\n\n"
		content += "| Status | Name | Tests | Duration | Report |\n"
		content += "|--------|------|-------|----------|--------|\n"

		for _, subgroup := range group.Subgroups {
			relPath := GetRelativeReportPath(subgroup, gm.runDir)

			// Status column
			statusStr := string(subgroup.Status)

			// Name column
			nameStr := subgroup.Name

			// Tests column - show breakdown of test results (using recursive counts)
			var testsStr string
			if subgroup.Stats.TotalTestsRecursive > 0 {
				parts := []string{}
				if subgroup.Stats.PassedTestsRecursive > 0 {
					parts = append(parts, fmt.Sprintf("%d passed", subgroup.Stats.PassedTestsRecursive))
				}
				if subgroup.Stats.FailedTestsRecursive > 0 {
					parts = append(parts, fmt.Sprintf("%d failed", subgroup.Stats.FailedTestsRecursive))
				}
				if subgroup.Stats.SkippedTestsRecursive > 0 {
					parts = append(parts, fmt.Sprintf("%d skipped", subgroup.Stats.SkippedTestsRecursive))
				}
				testsStr = strings.Join(parts, ", ")
			} else {
				testsStr = "0 tests"
			}

			// Duration column
			durationStr := "-"
			if subgroup.Duration > 0 {
				durationStr = fmt.Sprintf("%.1fs", subgroup.Duration.Seconds())
			}

			// Report link column
			reportStr := fmt.Sprintf("./%s", relPath)

			content += fmt.Sprintf("| %s | %s | %s | %s | %s |\n",
				statusStr, nameStr, testsStr, durationStr, reportStr)
		}
		content += "\n"
	}

	// stdout/stderr section
	if group.Stdout != "" || group.Stderr != "" {
		content += "## stdout/stderr\n"

		// Combined output in single code block as per migration plan
		if group.Stdout != "" || group.Stderr != "" {
			content += "```\n"
			if group.Stdout != "" {
				content += group.Stdout
				if !strings.HasSuffix(group.Stdout, "\n") {
					content += "\n"
				}
			}
			if group.Stderr != "" {
				content += group.Stderr
				if !strings.HasSuffix(group.Stderr, "\n") {
					content += "\n"
				}
			}
			content += "```\n"
		}
	}

	return content
}

// GetRootGroups returns all root-level groups
func (gm *GroupManager) GetRootGroups() []*TestGroup {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	result := make([]*TestGroup, len(gm.rootGroups))
	copy(result, gm.rootGroups)
	return result
}

// GetGroup returns a group by ID
func (gm *GroupManager) GetGroup(groupID string) (*TestGroup, bool) {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	group, exists := gm.groups[groupID]
	return group, exists
}

// GetAllGroups returns all groups
func (gm *GroupManager) GetAllGroups() map[string]*TestGroup {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	result := make(map[string]*TestGroup)
	for k, v := range gm.groups {
		result[k] = v
	}
	return result
}

// GenerateFinalReport generates the final summary report
func (gm *GroupManager) GenerateFinalReport() error {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	// Generate reports for all groups
	for _, group := range gm.groups {
		if err := gm.generateGroupReport(group); err != nil {
			gm.logError("Failed to generate final report for group %s: %v",
				group.ID, err)
		}
	}

	// Generate root summary
	summaryPath := filepath.Join(gm.runDir, "test-run.md")
	summaryContent := gm.generateSummaryReport()

	if err := os.WriteFile(summaryPath, []byte(summaryContent), 0644); err != nil {
		return fmt.Errorf("failed to write summary report: %w", err)
	}

	return nil
}

// generateSummaryReport generates the overall summary report
func (gm *GroupManager) generateSummaryReport() string {
	var content string

	content += "# Test Run Summary\n\n"
	content += fmt.Sprintf("Generated: %s\n\n", time.Now().Format(time.RFC3339))

	// Calculate totals
	var totalTests, passedTests, failedTests, skippedTests int
	for _, group := range gm.rootGroups {
		group.UpdateStats()
		totalTests += group.Stats.TotalTestsRecursive
		passedTests += group.Stats.PassedTestsRecursive
		failedTests += group.Stats.FailedTestsRecursive
		skippedTests += group.Stats.SkippedTestsRecursive
	}

	// Overall statistics
	content += "## Overall Statistics\n\n"
	content += fmt.Sprintf("- Total Tests: %d\n", totalTests)
	content += fmt.Sprintf("- Passed: %d (%.1f%%)\n", passedTests,
		float64(passedTests)*100/float64(max(totalTests, 1)))
	content += fmt.Sprintf("- Failed: %d (%.1f%%)\n", failedTests,
		float64(failedTests)*100/float64(max(totalTests, 1)))
	content += fmt.Sprintf("- Skipped: %d (%.1f%%)\n", skippedTests,
		float64(skippedTests)*100/float64(max(totalTests, 1)))
	content += "\n"

	// Root groups
	content += "## Test Groups\n\n"
	for _, group := range gm.rootGroups {
		var icon string
		switch group.Status {
		case TestStatusFail:
			icon = "✕"
		case TestStatusSkip:
			icon = "○"
		case TestStatusRunning:
			icon = "⚡"
		case TestStatusPending:
			icon = "⏳"
		default:
			icon = "✓"
		}

		relPath := GetRelativeReportPath(group, gm.runDir)
		content += fmt.Sprintf("- %s [%s](%s)", icon, group.Name, relPath)
		if group.Stats.TotalTestsRecursive > 0 {
			content += fmt.Sprintf(" (%d tests: %d passed, %d failed, %d skipped)",
				group.Stats.TotalTestsRecursive,
				group.Stats.PassedTestsRecursive,
				group.Stats.FailedTestsRecursive,
				group.Stats.SkippedTestsRecursive)
		}
		content += "\n"
	}

	return content
}

// Cleanup performs cleanup tasks
func (gm *GroupManager) Cleanup() {
	gm.updateMutex.Lock()
	if gm.updateTimer != nil {
		gm.updateTimer.Stop()
		gm.updateTimer = nil
	}
	gm.updateMutex.Unlock()

	// Flush any pending updates
	gm.flushPendingUpdates()
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// MarshalJSON implements json.Marshaler for GroupManager
func (gm *GroupManager) MarshalJSON() ([]byte, error) {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	return json.Marshal(struct {
		Groups     map[string]*TestGroup `json:"groups"`
		RootGroups []string              `json:"rootGroups"`
	}{
		Groups:     gm.groups,
		RootGroups: gm.getRootGroupIDs(),
	})
}

// getRootGroupIDs returns the IDs of all root groups
func (gm *GroupManager) getRootGroupIDs() []string {
	ids := make([]string, len(gm.rootGroups))
	for i, group := range gm.rootGroups {
		ids[i] = group.ID
	}
	return ids
}

// ProcessGroupStdout processes stdout event from IPC
func (gm *GroupManager) ProcessGroupStdout(event ipc.GroupStdoutChunkEvent) error {
	return gm.ProcessStdoutChunk(event.Payload.GroupName, event.Payload.ParentNames, event.Payload.Chunk)
}

// ProcessGroupStderr processes stderr event from IPC
func (gm *GroupManager) ProcessGroupStderr(event ipc.GroupStderrChunkEvent) error {
	return gm.ProcessStderrChunk(event.Payload.GroupName, event.Payload.ParentNames, event.Payload.Chunk)
}

// ProcessRunComplete processes run complete event
func (gm *GroupManager) ProcessRunComplete(event ipc.RunCompleteEvent) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	// Trigger final report generation
	gm.scheduleReportUpdate("")
	return nil
}

// Flush immediately writes all pending group reports
func (gm *GroupManager) Flush() {
	// Cancel any pending timer
	gm.updateMutex.Lock()
	if gm.updateTimer != nil {
		gm.updateTimer.Stop()
		gm.updateTimer = nil
	}
	gm.updateMutex.Unlock()

	// Flush all pending updates immediately
	gm.flushPendingUpdates()

	// Also generate reports for all groups to ensure nothing is missed
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	for _, group := range gm.groups {
		if err := gm.generateGroupReport(group); err != nil {
			gm.logError("Failed to generate report for group %s: %v",
				group.ID, err)
		}
	}
}
