package report

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// GenerateGroupID generates a unique ID for a test group based on its full path
// The ID is a truncated SHA256 hash of the hierarchical path
func GenerateGroupID(groupName string, parentNames []string) string {
	fullPath := append(parentNames, groupName)
	pathString := strings.Join(fullPath, ":")

	hash := sha256.Sum256([]byte(pathString))
	// Use first 16 bytes (32 hex chars) for ID
	return hex.EncodeToString(hash[:16])
}

// GenerateTestCaseID generates a unique ID for a test case
func GenerateTestCaseID(testName string, parentNames []string) string {
	fullPath := append(parentNames, testName)
	pathString := strings.Join(fullPath, ":")

	hash := sha256.Sum256([]byte(pathString))
	// Use first 16 bytes (32 hex chars) for ID
	return hex.EncodeToString(hash[:16])
}

// GenerateGroupIDFromPath generates an ID from a complete path slice
func GenerateGroupIDFromPath(path []string) string {
	if len(path) == 0 {
		return ""
	}

	pathString := strings.Join(path, ":")
	hash := sha256.Sum256([]byte(pathString))
	return hex.EncodeToString(hash[:16])
}

// ParseHierarchy parses a hierarchical path and returns parent names and the item name
func ParseHierarchy(fullPath []string) (parentNames []string, itemName string) {
	if len(fullPath) == 0 {
		return nil, ""
	}

	if len(fullPath) == 1 {
		return nil, fullPath[0]
	}

	return fullPath[:len(fullPath)-1], fullPath[len(fullPath)-1]
}

// BuildHierarchicalPath builds a display path with separators
func BuildHierarchicalPath(group *TestGroup) string {
	if group == nil {
		return ""
	}

	parts := append(group.ParentNames, group.Name)

	// Use → as separator for better visual hierarchy
	return strings.Join(parts, " → ")
}

// BuildHierarchicalPathFromSlice builds a display path from a slice of names
func BuildHierarchicalPathFromSlice(parts []string) string {
	if len(parts) == 0 {
		return ""
	}

	// Use → as separator for better visual hierarchy
	return strings.Join(parts, " → ")
}

// TruncatePathForDisplay truncates a path to fit within a maximum length
func TruncatePathForDisplay(path string, maxLength int) string {
	if len(path) <= maxLength {
		return path
	}

	// Reserve space for ellipsis
	if maxLength <= 3 {
		return "..."
	}

	// Try to keep the end of the path visible
	keepLength := maxLength - 3
	if keepLength > 0 {
		// Find a good breaking point (after a separator if possible)
		endPart := path[len(path)-keepLength:]
		// Try to start after a separator for cleaner truncation
		for i := 0; i < len(endPart) && i < 10; i++ {
			if endPart[i] == '/' || endPart[i] == '\\' {
				endPart = endPart[i+1:]
				break
			}
		}
		return "..." + endPart
	}

	return "..."
}

// NormalizeGroupName normalizes a group name for consistent comparison
func NormalizeGroupName(name string) string {
	// Trim whitespace
	name = strings.TrimSpace(name)

	// Replace multiple spaces with single space
	name = strings.Join(strings.Fields(name), " ")

	return name
}

// ExtractFileFromPath extracts the file component from a test hierarchy
// Returns empty string if no file component is found
func ExtractFileFromPath(path []string) string {
	if len(path) == 0 {
		return ""
	}

	// The first element is typically the file path
	first := path[0]

	// Check if it looks like a file path
	if strings.Contains(first, "/") || strings.Contains(first, "\\") ||
		strings.HasSuffix(first, ".js") || strings.HasSuffix(first, ".ts") ||
		strings.HasSuffix(first, ".jsx") || strings.HasSuffix(first, ".tsx") ||
		strings.HasSuffix(first, ".py") || strings.HasSuffix(first, ".go") {
		return first
	}

	return ""
}

// GetParentGroupID generates the ID of the parent group
func GetParentGroupID(parentNames []string) string {
	if len(parentNames) == 0 {
		return ""
	}

	if len(parentNames) == 1 {
		// Parent is root, generate ID for just the first element
		return GenerateGroupIDFromPath([]string{parentNames[0]})
	}

	// Generate ID for all parent names
	return GenerateGroupIDFromPath(parentNames)
}

// CompareGroupPaths compares two group paths for equality
func CompareGroupPaths(path1, path2 []string) bool {
	if len(path1) != len(path2) {
		return false
	}

	for i := range path1 {
		if NormalizeGroupName(path1[i]) != NormalizeGroupName(path2[i]) {
			return false
		}
	}

	return true
}

// IsChildPath returns true if childPath is a child of parentPath
func IsChildPath(parentPath, childPath []string) bool {
	if len(childPath) <= len(parentPath) {
		return false
	}

	for i := range parentPath {
		if NormalizeGroupName(parentPath[i]) != NormalizeGroupName(childPath[i]) {
			return false
		}
	}

	return true
}

// GetRelativePath returns the relative path from parent to child
func GetRelativePath(parentPath, childPath []string) []string {
	if !IsChildPath(parentPath, childPath) {
		return nil
	}

	return childPath[len(parentPath):]
}

// GroupIDInfo provides debug information about a group ID
type GroupIDInfo struct {
	ID          string
	GroupName   string
	ParentNames []string
	FullPath    []string
	PathString  string
}

// GetGroupIDInfo returns debug information about how a group ID was generated
func GetGroupIDInfo(groupName string, parentNames []string) GroupIDInfo {
	fullPath := append(parentNames, groupName)
	pathString := strings.Join(fullPath, ":")

	return GroupIDInfo{
		ID:          GenerateGroupID(groupName, parentNames),
		GroupName:   groupName,
		ParentNames: parentNames,
		FullPath:    fullPath,
		PathString:  pathString,
	}
}

// String returns a string representation of GroupIDInfo
func (info GroupIDInfo) String() string {
	return fmt.Sprintf("ID: %s\nPath: %s\nGenerated from: %s",
		info.ID,
		BuildHierarchicalPathFromSlice(info.FullPath),
		info.PathString)
}
