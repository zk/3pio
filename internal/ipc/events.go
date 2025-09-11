package ipc

import "time"

// EventType represents the type of IPC event
type EventType string

const (
	EventTypeStdoutChunk    EventType = "stdoutChunk"
	EventTypeStderrChunk    EventType = "stderrChunk"
	EventTypeTestFileStart  EventType = "testFileStart"
	EventTypeTestCase       EventType = "testCase"
	EventTypeTestFileResult EventType = "testFileResult"
	EventTypeRunComplete    EventType = "runComplete"
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

// StdoutChunkEvent represents stdout output from a test file
type StdoutChunkEvent struct {
	EventType EventType `json:"eventType"`
	Payload   struct {
		FilePath string `json:"filePath"`
		Chunk    string `json:"chunk"`
	} `json:"payload"`
}

func (e StdoutChunkEvent) Type() EventType { return EventTypeStdoutChunk }

// StderrChunkEvent represents stderr output from a test file
type StderrChunkEvent struct {
	EventType EventType `json:"eventType"`
	Payload   struct {
		FilePath string `json:"filePath"`
		Chunk    string `json:"chunk"`
	} `json:"payload"`
}

func (e StderrChunkEvent) Type() EventType { return EventTypeStderrChunk }

// TestFileStartEvent indicates a test file has started running
type TestFileStartEvent struct {
	EventType EventType `json:"eventType"`
	Payload   struct {
		FilePath string `json:"filePath"`
	} `json:"payload"`
}

func (e TestFileStartEvent) Type() EventType { return EventTypeTestFileStart }

// TestCaseEvent represents an individual test case result
type TestCaseEvent struct {
	EventType EventType `json:"eventType"`
	Payload   struct {
		FilePath  string     `json:"filePath"`
		TestName  string     `json:"testName"`
		SuiteName string     `json:"suiteName,omitempty"`
		Status    TestStatus `json:"status"`
		Duration  float64    `json:"duration,omitempty"`
		Error     string     `json:"error,omitempty"`
	} `json:"payload"`
}

func (e TestCaseEvent) Type() EventType { return EventTypeTestCase }

// TestFileResultEvent represents the completion of a test file
type TestFileResultEvent struct {
	EventType EventType `json:"eventType"`
	Payload   struct {
		FilePath    string     `json:"filePath"`
		Status      TestStatus `json:"status"`
		FailedTests []struct {
			Name     string `json:"name"`
			Duration float64 `json:"duration,omitempty"`
		} `json:"failedTests,omitempty"`
	} `json:"payload"`
}

func (e TestFileResultEvent) Type() EventType { return EventTypeTestFileResult }

// RunCompleteEvent indicates that the test runner has completed
type RunCompleteEvent struct {
	EventType EventType `json:"eventType"`
	Payload   struct{}  `json:"payload"`
}

func (e RunCompleteEvent) Type() EventType { return EventTypeRunComplete }

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
	Status    TestStatus `json:"status"`
	File      string     `json:"file"`
	LogFile   string     `json:"logFile,omitempty"`
	TestCases []TestCase `json:"testCases,omitempty"`
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
