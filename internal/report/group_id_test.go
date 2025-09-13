package report

import (
	"strings"
	"testing"
)

func TestGenerateGroupID(t *testing.T) {
	tests := []struct {
		name        string
		groupName   string
		parentNames []string
		expectLen   int
	}{
		{
			name:        "Simple group",
			groupName:   "test.js",
			parentNames: nil,
			expectLen:   32,
		},
		{
			name:        "Group with parents",
			groupName:   "Calculator",
			parentNames: []string{"src/math.test.js"},
			expectLen:   32,
		},
		{
			name:        "Deep hierarchy",
			groupName:   "should add",
			parentNames: []string{"src/math.test.js", "Calculator", "addition"},
			expectLen:   32,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := GenerateGroupID(tt.groupName, tt.parentNames)
			
			if len(id) != tt.expectLen {
				t.Errorf("ID length = %d, want %d", len(id), tt.expectLen)
			}
			
			// Verify it's hex
			for _, c := range id {
				if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
					t.Errorf("ID contains non-hex character: %c", c)
				}
			}
		})
	}
}

func TestGenerateGroupID_Deterministic(t *testing.T) {
	// Same inputs should always produce same ID
	groupName := "test.js"
	parentNames := []string{"src", "components"}
	
	id1 := GenerateGroupID(groupName, parentNames)
	id2 := GenerateGroupID(groupName, parentNames)
	
	if id1 != id2 {
		t.Errorf("IDs not deterministic: %s != %s", id1, id2)
	}
}

func TestGenerateGroupID_UniqueForDifferentPaths(t *testing.T) {
	id1 := GenerateGroupID("test.js", []string{"src"})
	id2 := GenerateGroupID("test.js", []string{"lib"})
	id3 := GenerateGroupID("other.js", []string{"src"})
	
	if id1 == id2 {
		t.Error("Same name in different parents should have different IDs")
	}
	if id1 == id3 {
		t.Error("Different names in same parent should have different IDs")
	}
}

func TestParseHierarchy(t *testing.T) {
	tests := []struct {
		name           string
		fullPath       []string
		wantParents    []string
		wantItemName   string
	}{
		{
			name:           "Empty path",
			fullPath:       []string{},
			wantParents:    nil,
			wantItemName:   "",
		},
		{
			name:           "Single item",
			fullPath:       []string{"test.js"},
			wantParents:    nil,
			wantItemName:   "test.js",
		},
		{
			name:           "Two items",
			fullPath:       []string{"src", "test.js"},
			wantParents:    []string{"src"},
			wantItemName:   "test.js",
		},
		{
			name:           "Multiple items",
			fullPath:       []string{"src", "components", "Calculator", "test.js"},
			wantParents:    []string{"src", "components", "Calculator"},
			wantItemName:   "test.js",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parents, item := ParseHierarchy(tt.fullPath)
			
			if !equalStringSlices(parents, tt.wantParents) {
				t.Errorf("Parents = %v, want %v", parents, tt.wantParents)
			}
			if item != tt.wantItemName {
				t.Errorf("Item = %s, want %s", item, tt.wantItemName)
			}
		})
	}
}

func TestBuildHierarchicalPath(t *testing.T) {
	tests := []struct {
		name     string
		group    *TestGroup
		expected string
	}{
		{
			name:     "Nil group",
			group:    nil,
			expected: "",
		},
		{
			name: "Root group",
			group: &TestGroup{
				Name:        "test.js",
				ParentNames: nil,
			},
			expected: "test.js",
		},
		{
			name: "Nested group",
			group: &TestGroup{
				Name:        "should add",
				ParentNames: []string{"math.test.js", "Calculator"},
			},
			expected: "math.test.js → Calculator → should add",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildHierarchicalPath(tt.group)
			if got != tt.expected {
				t.Errorf("BuildHierarchicalPath() = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestTruncatePathForDisplay(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		maxLength int
		expected  string
	}{
		{
			name:      "Short path",
			path:      "test.js",
			maxLength: 20,
			expected:  "test.js",
		},
		{
			name:      "Exact length",
			path:      "12345",
			maxLength: 5,
			expected:  "12345",
		},
		{
			name:      "Need truncation",
			path:      "very/long/path/to/file.js",
			maxLength: 15,
			expected:  "...to/file.js",
		},
		{
			name:      "Very short max",
			path:      "test.js",
			maxLength: 3,
			expected:  "...",
		},
		{
			name:      "Too short max",
			path:      "test.js",
			maxLength: 2,
			expected:  "...", // Can't truncate to less than 3 chars
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncatePathForDisplay(tt.path, tt.maxLength)
			if got != tt.expected {
				t.Errorf("TruncatePathForDisplay() = %s, want %s", got, tt.expected)
			}
			// Don't check length constraint for maxLength < 3 (minimum is "...")
			if tt.maxLength >= 3 && len(got) > tt.maxLength {
				t.Errorf("Result length %d exceeds max %d", len(got), tt.maxLength)
			}
		})
	}
}

func TestNormalizeGroupName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Trim spaces", "  test  ", "test"},
		{"Multiple spaces", "test  with   spaces", "test with spaces"},
		{"Tabs and newlines", "test\t\nwith\twhitespace", "test with whitespace"},
		{"Already normalized", "test", "test"},
		{"Empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeGroupName(tt.input)
			if got != tt.expected {
				t.Errorf("NormalizeGroupName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestExtractFileFromPath(t *testing.T) {
	tests := []struct {
		name     string
		path     []string
		expected string
	}{
		{
			name:     "JavaScript file",
			path:     []string{"src/test.js", "Calculator", "addition"},
			expected: "src/test.js",
		},
		{
			name:     "TypeScript file",
			path:     []string{"components/Button.tsx", "Button", "renders"},
			expected: "components/Button.tsx",
		},
		{
			name:     "Python file",
			path:     []string{"test_math.py", "TestMath", "test_add"},
			expected: "test_math.py",
		},
		{
			name:     "Go file",
			path:     []string{"math_test.go", "TestAdd"},
			expected: "math_test.go",
		},
		{
			name:     "No file extension",
			path:     []string{"Calculator", "addition"},
			expected: "",
		},
		{
			name:     "Empty path",
			path:     []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractFileFromPath(tt.path)
			if got != tt.expected {
				t.Errorf("ExtractFileFromPath() = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestCompareGroupPaths(t *testing.T) {
	tests := []struct {
		name  string
		path1 []string
		path2 []string
		equal bool
	}{
		{
			name:  "Equal paths",
			path1: []string{"src", "test.js"},
			path2: []string{"src", "test.js"},
			equal: true,
		},
		{
			name:  "Equal with normalization",
			path1: []string{"src", "  test.js  "},
			path2: []string{"src", "test.js"},
			equal: true,
		},
		{
			name:  "Different length",
			path1: []string{"src"},
			path2: []string{"src", "test.js"},
			equal: false,
		},
		{
			name:  "Different content",
			path1: []string{"src", "test1.js"},
			path2: []string{"src", "test2.js"},
			equal: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CompareGroupPaths(tt.path1, tt.path2)
			if got != tt.equal {
				t.Errorf("CompareGroupPaths() = %v, want %v", got, tt.equal)
			}
		})
	}
}

func TestIsChildPath(t *testing.T) {
	tests := []struct {
		name       string
		parentPath []string
		childPath  []string
		isChild    bool
	}{
		{
			name:       "Direct child",
			parentPath: []string{"src"},
			childPath:  []string{"src", "test.js"},
			isChild:    true,
		},
		{
			name:       "Deep child",
			parentPath: []string{"src"},
			childPath:  []string{"src", "components", "Button", "test.js"},
			isChild:    true,
		},
		{
			name:       "Not a child",
			parentPath: []string{"src"},
			childPath:  []string{"lib", "test.js"},
			isChild:    false,
		},
		{
			name:       "Same path",
			parentPath: []string{"src", "test.js"},
			childPath:  []string{"src", "test.js"},
			isChild:    false,
		},
		{
			name:       "Parent longer",
			parentPath: []string{"src", "components"},
			childPath:  []string{"src"},
			isChild:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsChildPath(tt.parentPath, tt.childPath)
			if got != tt.isChild {
				t.Errorf("IsChildPath() = %v, want %v", got, tt.isChild)
			}
		})
	}
}

func TestGetRelativePath(t *testing.T) {
	tests := []struct {
		name       string
		parentPath []string
		childPath  []string
		expected   []string
	}{
		{
			name:       "Direct child",
			parentPath: []string{"src"},
			childPath:  []string{"src", "test.js"},
			expected:   []string{"test.js"},
		},
		{
			name:       "Deep child",
			parentPath: []string{"src"},
			childPath:  []string{"src", "components", "Button"},
			expected:   []string{"components", "Button"},
		},
		{
			name:       "Not a child",
			parentPath: []string{"src"},
			childPath:  []string{"lib"},
			expected:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetRelativePath(tt.parentPath, tt.childPath)
			if !equalStringSlices(got, tt.expected) {
				t.Errorf("GetRelativePath() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetGroupIDInfo(t *testing.T) {
	info := GetGroupIDInfo("test.js", []string{"src", "components"})
	
	if info.GroupName != "test.js" {
		t.Errorf("GroupName = %s, want test.js", info.GroupName)
	}
	
	if len(info.ParentNames) != 2 {
		t.Errorf("ParentNames length = %d, want 2", len(info.ParentNames))
	}
	
	if len(info.ID) != 32 {
		t.Errorf("ID length = %d, want 32", len(info.ID))
	}
	
	expectedPathString := "src:components:test.js"
	if info.PathString != expectedPathString {
		t.Errorf("PathString = %s, want %s", info.PathString, expectedPathString)
	}
	
	// Test String() method
	str := info.String()
	if !strings.Contains(str, info.ID) {
		t.Error("String() should contain ID")
	}
	if !strings.Contains(str, "src → components → test.js") {
		t.Error("String() should contain hierarchical path")
	}
}

// Helper function
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}