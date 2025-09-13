package report

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSanitizeGroupName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: "_empty_",
		},
		{
			name:     "Path separators",
			input:    "src/components/test.js",
			expected: "src_components_test.js",
		},
		{
			name:     "Windows path",
			input:    "src\\components\\test.js",
			expected: "src_components_test.js",
		},
		{
			name:     "Invalid chars",
			input:    "test<>:\"|?*.js",
			expected: "test_.js", // Multiple invalid chars collapse to single underscore
		},
		{
			name:     "Multiple spaces",
			input:    "test   with    spaces",
			expected: "test_with_spaces",
		},
		{
			name:     "Leading/trailing dots",
			input:    "...test...",
			expected: "test",
		},
		{
			name:     "Windows reserved name",
			input:    "CON",
			expected: "_con_",
		},
		{
			name:     "Windows reserved case insensitive",
			input:    "con",
			expected: "_con_",
		},
		{
			name:     "Very long name",
			input:    strings.Repeat("a", 150),
			expected: strings.Repeat("a", 91) + "_" + "7595af82", // Hash suffix
		},
		{
			name:     "Control characters",
			input:    "test\x00\x1f\x07",
			expected: "test_", // Multiple control chars collapse to single underscore
		},
		{
			name:     "All invalid becomes empty",
			input:    "...",
			expected: "_empty_",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeGroupName(tt.input)
			if got != tt.expected {
				t.Errorf("SanitizeGroupName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
			
			// Ensure result is not too long
			if len(got) > MaxComponentLength {
				t.Errorf("Result too long: %d > %d", len(got), MaxComponentLength)
			}
		})
	}
}

func TestGenerateGroupPath(t *testing.T) {
	runDir := "/tmp/run"
	
	tests := []struct {
		name     string
		group    *TestGroup
		contains []string // Path components that should be present
	}{
		{
			name:     "Nil group",
			group:    nil,
			contains: []string{"reports"},
		},
		{
			name: "Root group",
			group: &TestGroup{
				Name:        "test.js",
				ParentNames: nil,
			},
			contains: []string{"reports", "test.js"},
		},
		{
			name: "Nested group",
			group: &TestGroup{
				Name:        "should add",
				ParentNames: []string{"math.test.js", "Calculator"},
			},
			contains: []string{"reports", "math_test.js", "calculator", "should_add"},
		},
		{
			name: "Group with invalid chars",
			group: &TestGroup{
				Name:        "test:with*invalid|chars",
				ParentNames: []string{"src/components"},
			},
			contains: []string{"src", "components", "test_with_invalid_chars"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateGroupPath(tt.group, runDir)
			
			if tt.group == nil {
				want := filepath.Join(runDir, "reports")
				if got != want {
					t.Errorf("Nil group should return runDir/reports: got %s, want %s", got, want)
				}
				return
			}
			
			// Check that path contains expected components
			for _, component := range tt.contains {
				if !strings.Contains(got, component) {
					t.Errorf("Path %s should contain %s", got, component)
				}
			}
			
			// Verify it's a valid path
			if !IsValidFilePath(got) {
				t.Errorf("Generated invalid path: %s", got)
			}
		})
	}
}

func TestGenerateGroupPath_DepthLimit(t *testing.T) {
	runDir := "/tmp/run"
	
	// Create a very deep hierarchy
	parentNames := make([]string, 30)
	for i := range parentNames {
		parentNames[i] = "level" + string(rune('0'+i%10))
	}
	
	group := &TestGroup{
		Name:        "deeptest",
		ParentNames: parentNames,
	}
	
	path := GenerateGroupPath(group, runDir)
	
	// Count path separators to check depth
	separators := strings.Count(path, string(filepath.Separator))
	// Subtract for initial runDir separators and 'reports' directory
	depth := separators - strings.Count(runDir, string(filepath.Separator)) - 1

	if depth > MaxDepth+1 { // +1 for the group name itself
		t.Errorf("Path depth %d exceeds MaxDepth %d", depth, MaxDepth)
	}
	
	// Should contain collapsed indicator
	if !strings.Contains(path, "_collapsed_") {
		t.Error("Deep hierarchy should be collapsed")
	}
}

func TestGenerateGroupPath_WindowsPathLimit(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows path limit test only runs on Windows")
	}
	
	runDir := "C:\\test"
	
	// Create a path that would exceed Windows limit
	longName := strings.Repeat("a", 100)
	group := &TestGroup{
		Name:        longName,
		ParentNames: []string{longName, longName, longName},
	}
	
	path := GenerateGroupPath(group, runDir)
	
	if len(path) > MaxWindowsPathLength {
		t.Errorf("Path length %d exceeds Windows limit %d", len(path), MaxWindowsPathLength)
	}
}

func TestCollapseHierarchy(t *testing.T) {
	tests := []struct {
		name         string
		hierarchy    []string
		expectedLen  int
		hasCollapsed bool
	}{
		{
			name:         "Short hierarchy",
			hierarchy:    []string{"a", "b", "c"},
			expectedLen:  3,
			hasCollapsed: false,
		},
		{
			name:         "Exactly at limit",
			hierarchy:    make([]string, MaxDepth),
			expectedLen:  MaxDepth,
			hasCollapsed: false,
		},
		{
			name:         "Over limit",
			hierarchy:    make([]string, MaxDepth+10),
			expectedLen:  MaxDepth+1, // Includes collapsed element
			hasCollapsed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize test data
			for i := range tt.hierarchy {
				tt.hierarchy[i] = "level" + string(rune('0'+i%10))
			}
			
			result := collapseHierarchy(tt.hierarchy)
			
			if len(result) != tt.expectedLen {
				t.Errorf("Result length = %d, want %d", len(result), tt.expectedLen)
			}
			
			hasCollapsed := false
			for _, item := range result {
				if strings.Contains(item, "_collapsed_") {
					hasCollapsed = true
					break
				}
			}
			
			if hasCollapsed != tt.hasCollapsed {
				t.Errorf("Has collapsed indicator = %v, want %v", hasCollapsed, tt.hasCollapsed)
			}
		})
	}
}

func TestGetReportFilePath(t *testing.T) {
	runDir := "/tmp/run"
	group := &TestGroup{
		Name:        "test.js",
		ParentNames: []string{"src"},
	}
	
	path := GetReportFilePath(group, runDir)
	
	if !strings.HasSuffix(path, "index.md") {
		t.Errorf("Report file should end with index.md: %s", path)
	}
	
	if !strings.Contains(path, "src") {
		t.Errorf("Path should contain parent directory: %s", path)
	}
}

func TestGetTestLogFilePath(t *testing.T) {
	runDir := "/tmp/run"
	group := &TestGroup{
		Name:        "Calculator",
		ParentNames: []string{"math.test.js"},
	}
	testName := "should add numbers"
	
	path := GetTestLogFilePath(group, testName, runDir)
	
	if !strings.HasSuffix(path, ".log") {
		t.Errorf("Log file should end with .log: %s", path)
	}
	
	if !strings.Contains(path, "logs") {
		t.Errorf("Path should contain logs directory: %s", path)
	}
	
	if !strings.Contains(path, "should_add_numbers") {
		t.Errorf("Path should contain sanitized test name: %s", path)
	}
}

func TestIsValidFilePath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		valid    bool
		skipOS   string // Skip on this OS
	}{
		{
			name:  "Valid Unix path",
			path:  "/tmp/test/file.txt",
			valid: true,
		},
		{
			name:  "Valid Windows path",
			path:  "C:\\test\\file.txt",
			valid: true,
		},
		{
			name:  "Path with null byte",
			path:  "/tmp/test\x00/file",
			valid: false,
		},
		{
			name:   "Windows reserved name",
			path:   "C:\\test\\CON\\file.txt",
			valid:  false,
			skipOS: "darwin", // Only check on Windows
		},
		{
			name:   "Windows path too long",
			path:   "C:\\" + strings.Repeat("a", 300),
			valid:  false,
			skipOS: "darwin", // Only check on Windows
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipOS != "" && runtime.GOOS == tt.skipOS {
				t.Skip("Test not applicable on this OS")
			}
			
			got := IsValidFilePath(tt.path)
			// On non-Windows, some Windows-specific checks won't apply
			if runtime.GOOS != "windows" && strings.Contains(tt.name, "Windows") {
				return
			}
			
			if got != tt.valid {
				t.Errorf("IsValidFilePath(%s) = %v, want %v", tt.path, got, tt.valid)
			}
		})
	}
}

func TestNormalizeFilePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Already clean",
			input:    "src/test/file.js",
			expected: "src/test/file.js",
		},
		{
			name:     "Trailing slash",
			input:    "src/test/",
			expected: "src/test",
		},
		{
			name:     "Double slashes",
			input:    "src//test//file.js",
			expected: "src/test/file.js",
		},
		{
			name:     "Dot segments",
			input:    "src/./test/../file.js",
			expected: "src/file.js",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeFilePath(tt.input)
			if got != tt.expected {
				t.Errorf("NormalizeFilePath(%s) = %s, want %s", tt.input, got, tt.expected)
			}
		})
	}
}

func TestGetRelativeReportPath(t *testing.T) {
	runDir := filepath.Join("tmp", "run")
	group := &TestGroup{
		Name:        "test.js",
		ParentNames: []string{"src", "components"},
	}
	
	relPath := GetRelativeReportPath(group, runDir)
	
	// Should be relative
	if filepath.IsAbs(relPath) {
		t.Errorf("Path should be relative: %s", relPath)
	}
	
	// Should contain the group hierarchy
	if !strings.Contains(relPath, "src") || !strings.Contains(relPath, "components") {
		t.Errorf("Relative path should contain hierarchy: %s", relPath)
	}
	
	// Should end with index.md
	if !strings.HasSuffix(relPath, "index.md") {
		t.Errorf("Should end with index.md: %s", relPath)
	}
}