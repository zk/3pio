package ipc

// Additional event types for group-based architecture
const (
	EventTypeGroupDiscovered EventType = "testGroupDiscovered"
	EventTypeGroupStart      EventType = "testGroupStart"
	EventTypeGroupResult     EventType = "testGroupResult"
	EventTypeGroupTestCase   EventType = "testCase" // Reuse existing testCase type with group hierarchy
	EventTypeGroupStdout     EventType = "groupStdout"
	EventTypeGroupStderr     EventType = "groupStderr"
)

// GroupDiscoveredEvent indicates a test group has been discovered (during collection phase)
type GroupDiscoveredEvent struct {
	EventType string                 `json:"eventType"`
	Payload   GroupDiscoveredPayload `json:"payload"`
}

func (e GroupDiscoveredEvent) Type() EventType { return EventTypeGroupDiscovered }

type GroupDiscoveredPayload struct {
	GroupName   string                 `json:"groupName"`             // Name of this group
	ParentNames []string               `json:"parentNames,omitempty"` // Full hierarchy from root (excluding this group)
	Metadata    map[string]interface{} `json:"metadata,omitempty"`    // Additional metadata (file path, line numbers, etc.)
}

// GroupStartEvent indicates a test group has started executing
type GroupStartEvent struct {
	EventType string            `json:"eventType"`
	Payload   GroupStartPayload `json:"payload"`
}

func (e GroupStartEvent) Type() EventType { return EventTypeGroupStart }

type GroupStartPayload struct {
	GroupName   string                 `json:"groupName"`
	ParentNames []string               `json:"parentNames,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Timestamp   int64                  `json:"timestamp,omitempty"` // Unix timestamp in milliseconds
}

// GroupResultEvent indicates a test group has finished executing
type GroupResultEvent struct {
	EventType string             `json:"eventType"`
	Payload   GroupResultPayload `json:"payload"`
}

func (e GroupResultEvent) Type() EventType { return EventTypeGroupResult }

type GroupResultPayload struct {
	GroupName   string                 `json:"groupName"`
	ParentNames []string               `json:"parentNames,omitempty"`
	Status      string                 `json:"status"`             // "PASS", "FAIL", "SKIP"
	Duration    float64                `json:"duration,omitempty"` // Duration in milliseconds
	Totals      GroupTotals            `json:"totals,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Timestamp   int64                  `json:"timestamp,omitempty"`
}

// GroupTotals holds test count statistics for a group
type GroupTotals struct {
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
	Total   int `json:"total,omitempty"`
}

// GroupTestCaseEvent represents an individual test case result with group hierarchy
type GroupTestCaseEvent struct {
	EventType string          `json:"eventType"`
	Payload   TestCasePayload `json:"payload"`
}

func (e GroupTestCaseEvent) Type() EventType { return EventTypeGroupTestCase }

type TestCasePayload struct {
	TestName    string                 `json:"testName"`
	ParentNames []string               `json:"parentNames,omitempty"` // Full hierarchy including file and describe blocks
	Status      string                 `json:"status"`                // "PASS", "FAIL", "SKIP", "PENDING"
	Duration    float64                `json:"duration,omitempty"`    // Duration in milliseconds
	Error       *TestError             `json:"error,omitempty"`
	Stdout      string                 `json:"stdout,omitempty"`
	Stderr      string                 `json:"stderr,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Timestamp   int64                  `json:"timestamp,omitempty"`
}

// TestError contains error information for failed tests
type TestError struct {
	Message   string `json:"message"`
	Stack     string `json:"stack,omitempty"`
	Expected  string `json:"expected,omitempty"`
	Actual    string `json:"actual,omitempty"`
	Location  string `json:"location,omitempty"`  // File:line
	ErrorType string `json:"errorType,omitempty"` // e.g., "AssertionError"
}

// GroupStdoutChunkEvent represents stdout output from a test group
type GroupStdoutChunkEvent struct {
	EventType string             `json:"eventType"`
	Payload   OutputChunkPayload `json:"payload"`
}

func (e GroupStdoutChunkEvent) Type() EventType { return EventTypeGroupStdout }

// GroupStderrChunkEvent represents stderr output from a test group
type GroupStderrChunkEvent struct {
	EventType string             `json:"eventType"`
	Payload   OutputChunkPayload `json:"payload"`
}

func (e GroupStderrChunkEvent) Type() EventType { return EventTypeGroupStderr }

type OutputChunkPayload struct {
	GroupName   string   `json:"groupName,omitempty"`   // The group this output belongs to
	ParentNames []string `json:"parentNames,omitempty"` // Full hierarchy
	Chunk       string   `json:"chunk"`                 // The output chunk
	Timestamp   int64    `json:"timestamp,omitempty"`

	// Legacy field for backward compatibility (will be removed)
	FilePath string `json:"filePath,omitempty"`
}

// GenericEvent is used for parsing unknown event types
type GenericEvent struct {
	EventType string                 `json:"eventType"`
	Payload   map[string]interface{} `json:"payload"`
}

// Helper functions to create events

// NewGroupDiscoveredEvent creates a new group discovered event
func NewGroupDiscoveredEvent(groupName string, parentNames []string) GroupDiscoveredEvent {
	return GroupDiscoveredEvent{
		EventType: string(EventTypeGroupDiscovered),
		Payload: GroupDiscoveredPayload{
			GroupName:   groupName,
			ParentNames: parentNames,
		},
	}
}

// NewGroupStartEvent creates a new group start event
func NewGroupStartEvent(groupName string, parentNames []string) GroupStartEvent {
	return GroupStartEvent{
		EventType: string(EventTypeGroupStart),
		Payload: GroupStartPayload{
			GroupName:   groupName,
			ParentNames: parentNames,
		},
	}
}

// NewGroupResultEvent creates a new group result event
func NewGroupResultEvent(groupName string, parentNames []string, status string, duration float64) GroupResultEvent {
	return GroupResultEvent{
		EventType: string(EventTypeGroupResult),
		Payload: GroupResultPayload{
			GroupName:   groupName,
			ParentNames: parentNames,
			Status:      status,
			Duration:    duration,
		},
	}
}

// NewGroupTestCaseEvent creates a new test case event with group hierarchy
func NewGroupTestCaseEvent(testName string, parentNames []string, status string) GroupTestCaseEvent {
	return GroupTestCaseEvent{
		EventType: string(EventTypeTestCase),
		Payload: TestCasePayload{
			TestName:    testName,
			ParentNames: parentNames,
			Status:      status,
		},
	}
}

// IsGroupEvent returns true if the event type is a group-related event
func IsGroupEvent(eventType string) bool {
	switch EventType(eventType) {
	case EventTypeGroupDiscovered, EventTypeGroupStart, EventTypeGroupResult:
		return true
	default:
		return false
	}
}

// GetHierarchyFromEvent extracts the full hierarchy path from an event
func GetHierarchyFromEvent(event GenericEvent) []string {
	payload := event.Payload

	var hierarchy []string

	// Extract parent names
	if parentNames, ok := payload["parentNames"].([]interface{}); ok {
		for _, name := range parentNames {
			if str, ok := name.(string); ok {
				hierarchy = append(hierarchy, str)
			}
		}
	}

	// Add the group/test name
	if groupName, ok := payload["groupName"].(string); ok {
		hierarchy = append(hierarchy, groupName)
	} else if testName, ok := payload["testName"].(string); ok {
		hierarchy = append(hierarchy, testName)
	}

	return hierarchy
}
