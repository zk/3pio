package report

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

const (
	// MaxWindowsPathLength is the maximum path length on Windows
	MaxWindowsPathLength = 260

	// MaxComponentLength is the maximum length for a single path component
	MaxComponentLength = 100

	// MaxDepth is the maximum nesting depth to prevent excessive directory nesting
	MaxDepth = 20
)

var (
	// Pattern to match invalid filesystem characters
	invalidCharsPattern = regexp.MustCompile(`[<>:"|?*\x00-\x1f]`)

	// Pattern to match multiple spaces or underscores
	multiSpacePattern = regexp.MustCompile(`[\s_]+`)

	// Pattern to match leading/trailing dots and spaces (Windows restriction)
	trimPattern = regexp.MustCompile(`^[\s.]+|[\s.]+$`)

	// Reserved Windows filenames
	windowsReservedNames = map[string]bool{
		"CON": true, "PRN": true, "AUX": true, "NUL": true,
		"COM1": true, "COM2": true, "COM3": true, "COM4": true,
		"COM5": true, "COM6": true, "COM7": true, "COM8": true,
		"COM9": true, "LPT1": true, "LPT2": true, "LPT3": true,
		"LPT4": true, "LPT5": true, "LPT6": true, "LPT7": true,
		"LPT8": true, "LPT9": true,
	}
)

// SanitizeGroupName sanitizes a group name for use as a filesystem path component
func SanitizeGroupName(name string) string {
	if name == "" {
		return "_empty_"
	}

	// Step 1: Convert to lowercase (as per migration plan)
	name = strings.ToLower(name)

	// Step 2: Remove leading/trailing dots and spaces FIRST
	name = trimPattern.ReplaceAllString(name, "")

	// Step 3: Replace path separators with underscores
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")

	// Preserve file extension if present (only for common extensions)
	lastDot := strings.LastIndex(name, ".")
	var ext string
	if lastDot > 0 && lastDot < len(name)-1 {
		possibleExt := name[lastDot:]
		// Only preserve if it looks like a file extension (1-4 alphanumeric chars)
		if len(possibleExt) >= 2 && len(possibleExt) <= 5 &&
			strings.IndexFunc(possibleExt[1:], func(r rune) bool {
				return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9')
			}) == -1 {
			ext = possibleExt
			name = name[:lastDot]
		}
	}

	// Replace remaining dots with underscores
	name = strings.ReplaceAll(name, ".", "_")

	// Add extension back if it was preserved
	if ext != "" {
		name = name + ext
	}

	// Step 4: Replace invalid filesystem characters
	name = invalidCharsPattern.ReplaceAllString(name, "_")

	// Step 5: Collapse multiple spaces/underscores to single underscore
	name = multiSpacePattern.ReplaceAllString(name, "_")

	// Step 6: Handle Windows reserved names
	upperName := strings.ToUpper(name)
	if windowsReservedNames[upperName] {
		name = "_" + name + "_"
	}

	// Step 7: Ensure name is not empty after sanitization
	if name == "" {
		name = "_empty_"
	}

	// Step 8: Truncate if too long
	if len(name) > MaxComponentLength {
		// Keep first 90 chars and add hash suffix
		hash := sha256.Sum256([]byte(name))
		hashStr := hex.EncodeToString(hash[:4]) // 8 chars
		name = name[:MaxComponentLength-9] + "_" + hashStr
	}

	return name
}

// GenerateGroupPath generates a filesystem path for a test group
func GenerateGroupPath(group *TestGroup, runDir string) string {
	if group == nil {
		return filepath.Join(runDir, "reports")
	}

	// Build the full hierarchy path
	hierarchy := append(group.ParentNames, group.Name)

	// Limit depth to prevent excessive nesting
	if len(hierarchy) > MaxDepth {
		// Collapse intermediate levels
		hierarchy = collapseHierarchy(hierarchy)
	}

	// Sanitize each component
	components := make([]string, 0, len(hierarchy)+2)
	components = append(components, runDir, "reports")

	for _, part := range hierarchy {
		// Always sanitize the entire group name as a single unit
		// This ensures Go package names like "github.com/zk/3pio" become "github_com_zk_3pio"
		// and file paths like "./src/test.js" become "_src_test_js"
		sanitized := SanitizeGroupName(part)
		if sanitized != "" {
			components = append(components, sanitized)
		}
	}

	// Build the path
	path := filepath.Join(components...)

	// Check Windows path length limit
	if runtime.GOOS == "windows" && len(path) > MaxWindowsPathLength {
		path = shortenPath(path, runDir, hierarchy)
	}

	return path
}

// GenerateGroupPathFromHierarchy generates a filesystem path from a hierarchy slice
func GenerateGroupPathFromHierarchy(hierarchy []string, runDir string) string {
	if len(hierarchy) == 0 {
		return filepath.Join(runDir, "reports")
	}

	// Limit depth to prevent excessive nesting
	if len(hierarchy) > MaxDepth {
		hierarchy = collapseHierarchy(hierarchy)
	}

	// Sanitize each component
	components := make([]string, 0, len(hierarchy)+2)
	components = append(components, runDir, "reports")

	for _, part := range hierarchy {
		// Always sanitize the entire group name as a single unit
		// This ensures Go package names like "github.com/zk/3pio" become "github_com_zk_3pio"
		// and file paths like "./src/test.js" become "_src_test_js"
		sanitized := SanitizeGroupName(part)
		if sanitized != "" {
			components = append(components, sanitized)
		}
	}

	// Build the path
	path := filepath.Join(components...)

	// Check Windows path length limit
	if runtime.GOOS == "windows" && len(path) > MaxWindowsPathLength {
		path = shortenPath(path, runDir, hierarchy)
	}

	return path
}

// collapseHierarchy reduces hierarchy depth by combining intermediate levels
func collapseHierarchy(hierarchy []string) []string {
	if len(hierarchy) <= MaxDepth {
		return hierarchy
	}

	// Keep first few and last few levels, combine middle ones
	keepStart := MaxDepth / 2
	keepEnd := MaxDepth / 2

	result := make([]string, 0, MaxDepth)

	// Add first levels
	result = append(result, hierarchy[:keepStart]...)

	// Combine middle levels into one
	middleStart := keepStart
	middleEnd := len(hierarchy) - keepEnd
	if middleEnd > middleStart {
		combined := strings.Join(hierarchy[middleStart:middleEnd], "_")
		// Hash the combined part to keep it short
		hash := sha256.Sum256([]byte(combined))
		hashStr := hex.EncodeToString(hash[:4])
		result = append(result, fmt.Sprintf("_collapsed_%s_", hashStr))
	}

	// Add last levels
	result = append(result, hierarchy[len(hierarchy)-keepEnd:]...)

	return result
}

// shortenPath shortens a path that exceeds Windows path length limit
func shortenPath(longPath string, runDir string, hierarchy []string) string {
	// Strategy: Use hash-based short names for intermediate directories

	if len(hierarchy) == 0 {
		return runDir
	}

	// Always keep the last component readable
	lastComponent := SanitizeGroupName(hierarchy[len(hierarchy)-1])

	// Calculate available space
	baseLen := len(runDir) + len(lastComponent) + 10 // Extra space for separators
	availableSpace := MaxWindowsPathLength - baseLen

	if availableSpace <= 0 {
		// Even the last component is too long, hash everything
		hash := sha256.Sum256([]byte(strings.Join(hierarchy, ":")))
		hashStr := hex.EncodeToString(hash[:8])
		return filepath.Join(runDir, hashStr)
	}

	// Build shortened path
	components := []string{runDir}

	if len(hierarchy) > 1 {
		// Hash all intermediate components
		intermediate := hierarchy[:len(hierarchy)-1]
		hash := sha256.Sum256([]byte(strings.Join(intermediate, ":")))
		hashStr := hex.EncodeToString(hash[:8])
		components = append(components, hashStr)
	}

	components = append(components, lastComponent)

	return filepath.Join(components...)
}

// GetReportFilePath returns the path to the report file for a group
func GetReportFilePath(group *TestGroup, runDir string) string {
	groupPath := GenerateGroupPath(group, runDir)
	return filepath.Join(groupPath, "index.md")
}

// GetTestLogFilePath returns the path to the log file for a specific test
func GetTestLogFilePath(group *TestGroup, testName string, runDir string) string {
	groupPath := GenerateGroupPath(group, runDir)
	sanitizedTestName := SanitizeGroupName(testName)
	return filepath.Join(groupPath, "logs", sanitizedTestName+".log")
}

// GetGroupOutputFilePath returns the path to the output file for a group
func GetGroupOutputFilePath(group *TestGroup, runDir string) string {
	groupPath := GenerateGroupPath(group, runDir)
	return filepath.Join(groupPath, "output.log")
}

// IsValidFilePath checks if a path is valid for the current OS
func IsValidFilePath(path string) bool {
	// Check length
	if runtime.GOOS == "windows" && len(path) > MaxWindowsPathLength {
		return false
	}

	// Check for invalid characters (basic check)
	if strings.ContainsAny(path, "\x00") {
		return false
	}

	// Check for reserved names on Windows
	if runtime.GOOS == "windows" {
		components := strings.Split(path, string(filepath.Separator))
		for _, comp := range components {
			upperComp := strings.ToUpper(comp)
			// Remove extension for checking
			if idx := strings.LastIndex(upperComp, "."); idx > 0 {
				upperComp = upperComp[:idx]
			}
			if windowsReservedNames[upperComp] {
				return false
			}
		}
	}

	return true
}

// NormalizeFilePath normalizes a file path for consistent comparison
func NormalizeFilePath(path string) string {
	// Clean the path
	path = filepath.Clean(path)

	// Convert to forward slashes for consistency
	path = filepath.ToSlash(path)

	// Remove trailing slash
	path = strings.TrimSuffix(path, "/")

	return path
}

// GetRelativeReportPath gets the relative path from run directory to a group's report
func GetRelativeReportPath(group *TestGroup, runDir string) string {
	fullPath := GetReportFilePath(group, runDir)
	rel, err := filepath.Rel(runDir, fullPath)
	if err != nil {
		return fullPath
	}
	return rel
}
