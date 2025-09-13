package report

import (
	"testing"
	"time"
)

func TestTestGroup_IsComplete(t *testing.T) {
	tests := []struct {
		name   string
		status TestStatus
		want   bool
	}{
		{"Pass status", TestStatusPass, true},
		{"Fail status", TestStatusFail, true},
		{"Skip status", TestStatusSkip, true},
		{"Pending status", TestStatusPending, false},
		{"Running status", TestStatusRunning, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &TestGroup{Status: tt.status}
			if got := g.IsComplete(); got != tt.want {
				t.Errorf("IsComplete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTestGroup_HasFailures(t *testing.T) {
	tests := []struct {
		name  string
		group *TestGroup
		want  bool
	}{
		{
			name: "Group with fail status",
			group: &TestGroup{
				Status: TestStatusFail,
			},
			want: true,
		},
		{
			name: "Group with passing test cases",
			group: &TestGroup{
				Status: TestStatusPass,
				TestCases: []TestCase{
					{Status: TestStatusPass},
					{Status: TestStatusPass},
				},
			},
			want: false,
		},
		{
			name: "Group with one failing test case",
			group: &TestGroup{
				Status: TestStatusPass,
				TestCases: []TestCase{
					{Status: TestStatusPass},
					{Status: TestStatusFail},
					{Status: TestStatusPass},
				},
			},
			want: true,
		},
		{
			name: "Group with failing subgroup",
			group: &TestGroup{
				Status: TestStatusPass,
				Subgroups: map[string]*TestGroup{
					"sub1": {Status: TestStatusPass},
					"sub2": {Status: TestStatusFail},
				},
			},
			want: true,
		},
		{
			name: "Group with nested failing test",
			group: &TestGroup{
				Status: TestStatusPass,
				Subgroups: map[string]*TestGroup{
					"sub1": {
						Status: TestStatusPass,
						TestCases: []TestCase{
							{Status: TestStatusFail},
						},
					},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.group.HasFailures(); got != tt.want {
				t.Errorf("HasFailures() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTestGroup_UpdateStats(t *testing.T) {
	group := &TestGroup{
		Status: TestStatusRunning,
		TestCases: []TestCase{
			{Status: TestStatusPass},
			{Status: TestStatusPass},
			{Status: TestStatusFail},
			{Status: TestStatusSkip},
		},
		Subgroups: map[string]*TestGroup{
			"sub1": {
				Status: TestStatusRunning,
				TestCases: []TestCase{
					{Status: TestStatusPass},
					{Status: TestStatusFail},
				},
			},
			"sub2": {
				Status: TestStatusRunning,
				TestCases: []TestCase{
					{Status: TestStatusSkip},
					{Status: TestStatusSkip},
				},
			},
		},
	}

	group.UpdateStats()

	// Check direct stats
	if group.Stats.TotalTests != 4 {
		t.Errorf("TotalTests = %d, want 4", group.Stats.TotalTests)
	}
	if group.Stats.PassedTests != 2 {
		t.Errorf("PassedTests = %d, want 2", group.Stats.PassedTests)
	}
	if group.Stats.FailedTests != 1 {
		t.Errorf("FailedTests = %d, want 1", group.Stats.FailedTests)
	}
	if group.Stats.SkippedTests != 1 {
		t.Errorf("SkippedTests = %d, want 1", group.Stats.SkippedTests)
	}

	// Check recursive stats
	if group.Stats.TotalTestsRecursive != 8 {
		t.Errorf("TotalTestsRecursive = %d, want 8", group.Stats.TotalTestsRecursive)
	}
	if group.Stats.PassedTestsRecursive != 3 {
		t.Errorf("PassedTestsRecursive = %d, want 3", group.Stats.PassedTestsRecursive)
	}
	if group.Stats.FailedTestsRecursive != 2 {
		t.Errorf("FailedTestsRecursive = %d, want 2", group.Stats.FailedTestsRecursive)
	}
	if group.Stats.SkippedTestsRecursive != 3 {
		t.Errorf("SkippedTestsRecursive = %d, want 3", group.Stats.SkippedTestsRecursive)
	}
}

func TestTestGroup_UpdateStatusFromChildren(t *testing.T) {
	tests := []struct {
		name         string
		setupGroup   func() *TestGroup
		expectedStatus TestStatus
	}{
		{
			name: "All tests pass",
			setupGroup: func() *TestGroup {
				return &TestGroup{
					Status: TestStatusRunning,
					StartTime: time.Now(),
					TestCases: []TestCase{
						{Status: TestStatusPass},
						{Status: TestStatusPass},
					},
				}
			},
			expectedStatus: TestStatusPass,
		},
		{
			name: "One test fails",
			setupGroup: func() *TestGroup {
				return &TestGroup{
					Status: TestStatusRunning,
					StartTime: time.Now(),
					TestCases: []TestCase{
						{Status: TestStatusPass},
						{Status: TestStatusFail},
					},
				}
			},
			expectedStatus: TestStatusFail,
		},
		{
			name: "All tests skipped",
			setupGroup: func() *TestGroup {
				return &TestGroup{
					Status: TestStatusRunning,
					StartTime: time.Now(),
					TestCases: []TestCase{
						{Status: TestStatusSkip},
						{Status: TestStatusSkip},
					},
				}
			},
			expectedStatus: TestStatusSkip,
		},
		{
			name: "Mixed pass and skip",
			setupGroup: func() *TestGroup {
				return &TestGroup{
					Status: TestStatusRunning,
					StartTime: time.Now(),
					TestCases: []TestCase{
						{Status: TestStatusPass},
						{Status: TestStatusSkip},
					},
				}
			},
			expectedStatus: TestStatusPass,
		},
		{
			name: "Subgroup fails",
			setupGroup: func() *TestGroup {
				return &TestGroup{
					Status: TestStatusRunning,
					StartTime: time.Now(),
					Subgroups: map[string]*TestGroup{
						"sub1": {Status: TestStatusPass},
						"sub2": {Status: TestStatusFail},
					},
				}
			},
			expectedStatus: TestStatusFail,
		},
		{
			name: "Tests still running",
			setupGroup: func() *TestGroup {
				return &TestGroup{
					Status: TestStatusRunning,
					TestCases: []TestCase{
						{Status: TestStatusPass},
						{Status: TestStatusRunning},
					},
				}
			},
			expectedStatus: TestStatusRunning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			group := tt.setupGroup()
			group.updateStatusFromChildren()
			
			if group.Status != tt.expectedStatus {
				t.Errorf("Status = %v, want %v", group.Status, tt.expectedStatus)
			}
			
			// Check that end time is set when complete
			if tt.expectedStatus != TestStatusRunning && group.EndTime.IsZero() {
				t.Error("EndTime should be set when group is complete")
			}
		})
	}
}

func TestTestGroup_GetFullPath(t *testing.T) {
	group := &TestGroup{
		Name:        "test.js",
		ParentNames: []string{"src", "components"},
	}

	expected := []string{"src", "components", "test.js"}
	got := group.GetFullPath()

	if len(got) != len(expected) {
		t.Fatalf("GetFullPath() length = %d, want %d", len(got), len(expected))
	}

	for i, v := range got {
		if v != expected[i] {
			t.Errorf("GetFullPath()[%d] = %s, want %s", i, v, expected[i])
		}
	}
}

func TestTestGroup_FindSubgroup(t *testing.T) {
	sub1 := &TestGroup{Name: "sub1"}
	sub2 := &TestGroup{Name: "sub2"}
	
	group := &TestGroup{
		Name: "parent",
		Subgroups: map[string]*TestGroup{
			"id1": sub1,
			"id2": sub2,
		},
	}

	tests := []struct {
		name      string
		findName  string
		wantNil   bool
	}{
		{"Find existing sub1", "sub1", false},
		{"Find existing sub2", "sub2", false},
		{"Find non-existing", "sub3", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := group.FindSubgroup(tt.findName)
			if (got == nil) != tt.wantNil {
				t.Errorf("FindSubgroup(%s) = %v, wantNil %v", tt.findName, got, tt.wantNil)
			}
		})
	}
}