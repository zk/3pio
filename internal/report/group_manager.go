package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/zk/3pio/internal/ipc"
	"github.com/zk/3pio/internal/logger"
)

// GroupManager manages the hierarchical test group state
type GroupManager struct {
	mu          sync.RWMutex
	groups      map[string]*TestGroup // ID -> Group
	rootGroups  []*TestGroup          // Top-level groups (typically files)
	runDir      string
	ipcPath     string
	logger      *logger.FileLogger
	
	// Debouncing for report generation
	pendingUpdates map[string]time.Time // Group ID -> last update time
	updateTimer    *time.Timer
	updateMutex    sync.Mutex
}

// NewGroupManager creates a new GroupManager instance
func NewGroupManager(runDir string, ipcPath string, logger *logger.FileLogger) *GroupManager {
	return &GroupManager{
		groups:         make(map[string]*TestGroup),
		rootGroups:     make([]*TestGroup, 0),
		runDir:         runDir,
		ipcPath:        ipcPath,
		logger:         logger,
		pendingUpdates: make(map[string]time.Time),
	}
}

// ProcessGroupDiscovered handles a group discovery event
func (gm *GroupManager) ProcessGroupDiscovered(event ipc.GroupDiscoveredEvent) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	payload := event.Payload
	groupID := GenerateGroupID(payload.GroupName, payload.ParentNames)
	
	// Check if group already exists
	if _, exists := gm.groups[groupID]; exists {
		// Group already discovered, this is idempotent
		return nil
	}
	
	// Create the new group
	group := &TestGroup{
		ID:          groupID,
		Name:        payload.GroupName,
		ParentNames: payload.ParentNames,
		Depth:       len(payload.ParentNames),
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
		gm.ensureParentHierarchy(group)
	}
	
	gm.logger.Info("Discovered group: %s (ID: %s)", 
		BuildHierarchicalPath(group), groupID)
	
	return nil
}

// ProcessGroupStart handles a group start event
func (gm *GroupManager) ProcessGroupStart(event ipc.GroupStartEvent) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	payload := event.Payload
	groupID := GenerateGroupID(payload.GroupName, payload.ParentNames)
	
	group, exists := gm.groups[groupID]
	if !exists {
		// Auto-discover the group if not already known
		gm.mu.Unlock()
		err := gm.ProcessGroupDiscovered(ipc.GroupDiscoveredEvent{
			EventType: string(ipc.EventTypeGroupDiscovered),
			Payload: ipc.GroupDiscoveredPayload{
				GroupName:   payload.GroupName,
				ParentNames: payload.ParentNames,
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
	
	gm.logger.Info("Started group: %s", BuildHierarchicalPath(group))
	
	return nil
}

// ProcessGroupResult handles a group completion event
func (gm *GroupManager) ProcessGroupResult(event ipc.GroupResultEvent) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	payload := event.Payload
	groupID := GenerateGroupID(payload.GroupName, payload.ParentNames)
	
	group, exists := gm.groups[groupID]
	if !exists {
		return fmt.Errorf("group not found: %s", groupID)
	}
	
	// Update group status
	switch payload.Status {
	case "PASS":
		group.Status = TestStatusPass
	case "FAIL":
		group.Status = TestStatusFail
	case "SKIP":
		group.Status = TestStatusSkip
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
	
	gm.logger.Info("Completed group: %s [%s]", 
		BuildHierarchicalPath(group), group.Status)
	
	return nil
}

// ProcessTestCase handles a test case event
func (gm *GroupManager) ProcessTestCase(event ipc.GroupTestCaseEvent) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	payload := event.Payload
	
	// The test's parent is the full parent hierarchy
	parentID := GenerateGroupIDFromPath(payload.ParentNames)
	
	// Find or create the parent group
	var parentGroup *TestGroup
	if len(payload.ParentNames) > 0 {
		var exists bool
		parentGroup, exists = gm.groups[parentID]
		if !exists {
			// Auto-discover parent group hierarchy
			gm.mu.Unlock()
			err := gm.ensureGroupHierarchy(payload.ParentNames)
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
		ID:        GenerateTestCaseID(payload.TestName, payload.ParentNames),
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
			Message:   payload.Error.Message,
			Stack:     payload.Error.Stack,
			Expected:  payload.Error.Expected,
			Actual:    payload.Error.Actual,
			Location:  payload.Error.Location,
			ErrorType: payload.Error.ErrorType,
		}
	}
	
	// Set output if present
	testCase.Stdout = payload.Stdout
	testCase.Stderr = payload.Stderr
	
	// Add to parent group
	parentGroup.TestCases = append(parentGroup.TestCases, testCase)
	parentGroup.Updated = time.Now()
	
	// Update parent group statistics
	parentGroup.UpdateStats()
	
	// Schedule report update
	gm.scheduleReportUpdate(parentID)
	
	gm.logger.Info("Test case: %s → %s [%s]", 
		BuildHierarchicalPathFromSlice(payload.ParentNames), 
		payload.TestName, testCase.Status)
	
	return nil
}

// ProcessStdoutChunk handles stdout output for a group
func (gm *GroupManager) ProcessStdoutChunk(groupName string, parentNames []string, chunk string) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	
	groupID := GenerateGroupID(groupName, parentNames)
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
	
	groupID := GenerateGroupID(groupName, parentNames)
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
	
	// Create each level of the hierarchy
	for i := 1; i <= len(path); i++ {
		currentPath := path[:i]
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
				gm.logger.Error("Failed to generate report for group %s: %v", 
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
	// This is a simplified version - the full implementation would match
	// the existing report format from the current manager
	var content string
	
	// Header
	content += fmt.Sprintf("# Test Report: %s\n\n", group.Name)
	
	// Metadata
	content += "---\n"
	content += fmt.Sprintf("group: %s\n", group.Name)
	content += fmt.Sprintf("path: %s\n", BuildHierarchicalPath(group))
	content += fmt.Sprintf("status: %s\n", group.Status)
	content += fmt.Sprintf("created: %s\n", group.Created.Format(time.RFC3339))
	content += fmt.Sprintf("updated: %s\n", group.Updated.Format(time.RFC3339))
	if group.Duration > 0 {
		content += fmt.Sprintf("duration: %s\n", group.Duration)
	}
	content += "---\n\n"
	
	// Statistics
	if group.Stats.TotalTests > 0 {
		content += "## Summary\n\n"
		content += fmt.Sprintf("- Total Tests: %d\n", group.Stats.TotalTests)
		content += fmt.Sprintf("- Passed: %d\n", group.Stats.PassedTests)
		content += fmt.Sprintf("- Failed: %d\n", group.Stats.FailedTests)
		content += fmt.Sprintf("- Skipped: %d\n", group.Stats.SkippedTests)
		content += "\n"
	}
	
	// Test Cases
	if len(group.TestCases) > 0 {
		content += "## Test Cases\n\n"
		for _, tc := range group.TestCases {
			icon := "✓"
			if tc.Status == TestStatusFail {
				icon = "✕"
			} else if tc.Status == TestStatusSkip {
				icon = "○"
			}
			
			content += fmt.Sprintf("- %s %s", icon, tc.Name)
			if tc.Duration > 0 {
				content += fmt.Sprintf(" (%dms)", tc.Duration.Milliseconds())
			}
			content += "\n"
			
			if tc.Error != nil {
				content += "  ```\n"
				content += fmt.Sprintf("  %s\n", tc.Error.Message)
				if tc.Error.Stack != "" {
					content += fmt.Sprintf("  %s\n", tc.Error.Stack)
				}
				content += "  ```\n"
			}
		}
		content += "\n"
	}
	
	// Subgroups
	if len(group.Subgroups) > 0 {
		content += "## Subgroups\n\n"
		for _, subgroup := range group.Subgroups {
			relPath := GetRelativeReportPath(subgroup, gm.runDir)
			icon := "✓"
			if subgroup.Status == TestStatusFail {
				icon = "✕"
			} else if subgroup.Status == TestStatusSkip {
				icon = "○"
			} else if subgroup.Status == TestStatusRunning {
				icon = "⚡"
			} else if subgroup.Status == TestStatusPending {
				icon = "⏳"
			}
			
			content += fmt.Sprintf("- %s [%s](%s)", icon, subgroup.Name, relPath)
			if subgroup.Stats.TotalTests > 0 {
				content += fmt.Sprintf(" (%d tests)", subgroup.Stats.TotalTests)
			}
			content += "\n"
		}
		content += "\n"
	}
	
	// Output
	if group.Stdout != "" || group.Stderr != "" {
		content += "## Output\n\n"
		if group.Stdout != "" {
			content += "### stdout\n```\n"
			content += group.Stdout
			content += "\n```\n\n"
		}
		if group.Stderr != "" {
			content += "### stderr\n```\n"
			content += group.Stderr
			content += "\n```\n\n"
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
			gm.logger.Error("Failed to generate final report for group %s: %v",
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
		icon := "✓"
		if group.Status == TestStatusFail {
			icon = "✕"
		} else if group.Status == TestStatusSkip {
			icon = "○"
		} else if group.Status == TestStatusRunning {
			icon = "⚡"
		} else if group.Status == TestStatusPending {
			icon = "⏳"
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