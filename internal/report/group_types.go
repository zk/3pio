package report

import (
	"time"
)

// TestStatus represents the status of a test or group
type TestStatus string

const (
	TestStatusPending TestStatus = "PENDING"
	TestStatusRunning TestStatus = "RUNNING"
	TestStatusPass    TestStatus = "PASS"
	TestStatusFail    TestStatus = "FAIL"
	TestStatusSkip    TestStatus = "SKIP"
)

// TestGroup represents a hierarchical group of tests (file, describe block, class, etc.)
type TestGroup struct {
	// Identification
	ID          string // SHA256-based ID from full path
	Name        string // Name of this group (e.g., "math.test.js", "Calculator", "addition tests")
	ParentID    string // ID of parent group (empty for root groups)
	ParentNames []string // Full hierarchy from root (excludes this group's name)
	Depth       int      // Depth in hierarchy (0 for root)

	// Status and timing
	Status    TestStatus
	Duration  time.Duration
	StartTime time.Time
	EndTime   time.Time
	Created   time.Time
	Updated   time.Time

	// Test data
	TestCases []TestCase                // Direct test cases in this group
	Subgroups map[string]*TestGroup     // Child groups (key is group ID)

	// Statistics
	Stats TestGroupStats

	// Output
	Stdout string // Accumulated stdout for this group
	Stderr string // Accumulated stderr for this group
}

// TestGroupStats holds aggregated statistics for a test group
type TestGroupStats struct {
	TotalTests   int
	PassedTests  int
	FailedTests  int
	SkippedTests int
	
	// Recursive counts (includes subgroups)
	TotalTestsRecursive   int
	PassedTestsRecursive  int
	FailedTestsRecursive  int
	SkippedTestsRecursive int
}

// TestCase represents an individual test
type TestCase struct {
	// Identification
	ID      string   // Unique ID for this test case
	GroupID string   // ID of parent group
	Name    string   // Test name (e.g., "should add two numbers")

	// Status and timing
	Status    TestStatus
	Duration  time.Duration
	StartTime time.Time
	EndTime   time.Time

	// Error information
	Error *TestError

	// Output
	Stdout string // stdout captured during this test
	Stderr string // stderr captured during this test
}

// TestError represents error information for a failed test
type TestError struct {
	Message    string   // Error message
	Stack      string   // Stack trace
	Expected   string   // Expected value (for assertions)
	Actual     string   // Actual value (for assertions)
	Location   string   // File:line where error occurred
	ErrorType  string   // Type of error (e.g., "AssertionError")
}

// IsComplete returns true if the group has finished executing
func (g *TestGroup) IsComplete() bool {
	return g.Status == TestStatusPass || 
	       g.Status == TestStatusFail || 
	       g.Status == TestStatusSkip
}

// HasFailures returns true if the group or any of its children have failures
func (g *TestGroup) HasFailures() bool {
	if g.Status == TestStatusFail {
		return true
	}
	
	for _, tc := range g.TestCases {
		if tc.Status == TestStatusFail {
			return true
		}
	}
	
	for _, sg := range g.Subgroups {
		if sg.HasFailures() {
			return true
		}
	}
	
	return false
}

// UpdateStats recalculates statistics for this group and all subgroups
func (g *TestGroup) UpdateStats() {
	// Reset stats
	g.Stats = TestGroupStats{}
	
	// Count direct test cases
	for _, tc := range g.TestCases {
		g.Stats.TotalTests++
		g.Stats.TotalTestsRecursive++
		
		switch tc.Status {
		case TestStatusPass:
			g.Stats.PassedTests++
			g.Stats.PassedTestsRecursive++
		case TestStatusFail:
			g.Stats.FailedTests++
			g.Stats.FailedTestsRecursive++
		case TestStatusSkip:
			g.Stats.SkippedTests++
			g.Stats.SkippedTestsRecursive++
		}
	}
	
	// Add subgroup stats
	for _, sg := range g.Subgroups {
		sg.UpdateStats()
		g.Stats.TotalTestsRecursive += sg.Stats.TotalTestsRecursive
		g.Stats.PassedTestsRecursive += sg.Stats.PassedTestsRecursive
		g.Stats.FailedTestsRecursive += sg.Stats.FailedTestsRecursive
		g.Stats.SkippedTestsRecursive += sg.Stats.SkippedTestsRecursive
	}
	
	// Update group status based on children
	g.updateStatusFromChildren()
}

// updateStatusFromChildren updates the group's status based on its children
func (g *TestGroup) updateStatusFromChildren() {
	if g.Status == TestStatusPending || g.Status == TestStatusRunning {
		allComplete := true
		hasFailures := false
		hasSkipped := false
		hasTests := false
		
		// Check test cases
		for _, tc := range g.TestCases {
			hasTests = true
			if tc.Status == TestStatusPending || tc.Status == TestStatusRunning {
				allComplete = false
				break
			}
			if tc.Status == TestStatusFail {
				hasFailures = true
			}
			if tc.Status == TestStatusSkip {
				hasSkipped = true
			}
		}
		
		// Check subgroups
		for _, sg := range g.Subgroups {
			hasTests = true
			if !sg.IsComplete() {
				allComplete = false
				break
			}
			if sg.Status == TestStatusFail {
				hasFailures = true
			}
			if sg.Status == TestStatusSkip {
				hasSkipped = true
			}
		}
		
		// Update status if all children are complete
		if allComplete && hasTests {
			if hasFailures {
				g.Status = TestStatusFail
			} else if hasSkipped && !hasFailures {
				// Only mark as skip if ALL tests were skipped
				allSkipped := true
				for _, tc := range g.TestCases {
					if tc.Status != TestStatusSkip {
						allSkipped = false
						break
					}
				}
				if allSkipped {
					for _, sg := range g.Subgroups {
						if sg.Status != TestStatusSkip {
							allSkipped = false
							break
						}
					}
				}
				if allSkipped {
					g.Status = TestStatusSkip
				} else {
					g.Status = TestStatusPass
				}
			} else {
				g.Status = TestStatusPass
			}
			
			if g.EndTime.IsZero() {
				g.EndTime = time.Now()
				g.Duration = g.EndTime.Sub(g.StartTime)
			}
		}
	}
}

// GetFullPath returns the full hierarchical path of this group
func (g *TestGroup) GetFullPath() []string {
	return append(g.ParentNames, g.Name)
}

// HasTestCases returns true if the group or any of its subgroups have test cases
func (g *TestGroup) HasTestCases() bool {
	if len(g.TestCases) > 0 {
		return true
	}

	for _, sg := range g.Subgroups {
		if sg.HasTestCases() {
			return true
		}
	}

	return false
}

// FindSubgroup finds a subgroup by name (direct children only)
func (g *TestGroup) FindSubgroup(name string) *TestGroup {
	for _, sg := range g.Subgroups {
		if sg.Name == name {
			return sg
		}
	}
	return nil
}