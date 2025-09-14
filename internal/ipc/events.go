package ipc

import "time"

// EventType represents the type of IPC event
type EventType string

const (
	EventTypeTestCase         EventType = "testCase"
	EventTypeRunComplete      EventType = "runComplete"
	EventTypeCollectionStart  EventType = "collectionStart"
	EventTypeCollectionError  EventType = "collectionError"
	EventTypeCollectionFinish EventType = "collectionFinish"
)

// TestStatus represents the status of a test
type TestStatus string

const (
	TestStatusPass    TestStatus = "PASS"
	TestStatusFail    TestStatus = "FAIL"
	TestStatusSkip    TestStatus = "SKIP"
	TestStatusPending TestStatus = "PENDING"
	TestStatusRunning TestStatus = "RUNNING"
)

// Event is the base interface for all IPC events
type Event interface {
	Type() EventType
}

// RunCompleteEvent indicates that the test runner has completed
type RunCompleteEvent struct {
	EventType EventType `json:"eventType"`
	Payload   struct{}  `json:"payload"`
}

func (e RunCompleteEvent) Type() EventType { return EventTypeRunComplete }

// CollectionStartEvent indicates test collection is starting (pytest specific)
type CollectionStartEvent struct {
	EventType EventType `json:"eventType"`
	Payload   struct {
		Phase string `json:"phase"`
	} `json:"payload"`
}

func (e CollectionStartEvent) Type() EventType { return EventTypeCollectionStart }

// CollectionErrorEvent represents an error during test collection (pytest specific)
type CollectionErrorEvent struct {
	EventType EventType `json:"eventType"`
	Payload   struct {
		FilePath string `json:"filePath"`
		Error    string `json:"error"`
		Phase    string `json:"phase"`
	} `json:"payload"`
}

func (e CollectionErrorEvent) Type() EventType { return EventTypeCollectionError }

// CollectionFinishEvent indicates test collection is complete (pytest specific)
type CollectionFinishEvent struct {
	EventType EventType `json:"eventType"`
	Payload   struct {
		Collected int `json:"collected"`
	} `json:"payload"`
}

func (e CollectionFinishEvent) Type() EventType { return EventTypeCollectionFinish }

// TestCase represents a test case in the test run state
type TestCase struct {
	Name     string     `json:"name"`
	Suite    string     `json:"suite,omitempty"`
	Status   TestStatus `json:"status"`
	Duration float64    `json:"duration,omitempty"`
	Error    string     `json:"error,omitempty"`
}

// TestFile represents a test file in the test run state
type TestFile struct {
	Status         TestStatus `json:"status"`
	File           string     `json:"file"`
	LogFile        string     `json:"logFile,omitempty"`
	TestCases      []TestCase `json:"testCases,omitempty"`
	Created        time.Time  `json:"created"`
	Updated        time.Time  `json:"updated"`
	ExecutionError string     `json:"executionError,omitempty"`
}

// TestRunState represents the complete state of a test run
type TestRunState struct {
	Timestamp      time.Time  `json:"timestamp"`
	Status         string     `json:"status"` // RUNNING, COMPLETE, ERROR
	UpdatedAt      time.Time  `json:"updatedAt"`
	Arguments      string     `json:"arguments"`
	TotalFiles     int        `json:"totalFiles"`
	FilesCompleted int        `json:"filesCompleted"`
	FilesPassed    int        `json:"filesPassed"`
	FilesFailed    int        `json:"filesFailed"`
	FilesSkipped   int        `json:"filesSkipped"`
	TestFiles      []TestFile `json:"testFiles"`
	ErrorDetails   string     `json:"errorDetails,omitempty"` // Error details when status is ERROR
}
