package ipc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// Manager handles IPC communication via file-based JSONL
type Manager struct {
	IPCPath   string
	watcher   *fsnotify.Watcher
	Events    chan Event
	stopChan  chan struct{}
	stopped   chan struct{} // Signals when watchLoop has stopped
	mu        sync.RWMutex
	closeOnce sync.Once
	logger    Logger
	file      *os.File
	reader    *bufio.Reader
	readerMu  sync.Mutex // Protects concurrent access to reader
}

// Logger interface for debug logging
type Logger interface {
	Debug(format string, args ...interface{})
	Error(format string, args ...interface{})
}

// NewManager creates a new IPC manager for reading events
func NewManager(ipcPath string, logger Logger) (*Manager, error) {
	if logger == nil {
		logger = &noopLogger{}
	}

	// Ensure IPC directory exists
	ipcDir := filepath.Dir(ipcPath)
	if err := os.MkdirAll(ipcDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create IPC directory: %w", err)
	}

	// Create IPC file if it doesn't exist
	file, err := os.OpenFile(ipcPath, os.O_CREATE|os.O_RDONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open IPC file: %w", err)
	}

	return &Manager{
		IPCPath:  ipcPath,
		Events:   make(chan Event, 10000), // Large buffer for handling burst of events
		stopChan: make(chan struct{}),
		stopped:  make(chan struct{}),
		logger:   logger,
		file:     file,
		reader:   bufio.NewReader(file),
	}, nil
}

// WatchEvents starts watching the IPC file for new events
func (m *Manager) WatchEvents() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}
	m.watcher = watcher

	// Add the IPC file to the watcher
	if err := watcher.Add(m.IPCPath); err != nil {
		return fmt.Errorf("failed to watch IPC file: %w", err)
	}

	// Start the single watch loop that handles both existing and new events
	go m.watchLoop()

	// Trigger initial read of any existing content
	go m.readEvents()

	return nil
}

// readEvents reads events from the current position in the file
func (m *Manager) readEvents() {
	m.readerMu.Lock()
	defer m.readerMu.Unlock()

	for {
		line, err := m.reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				m.logger.Error("Error reading events: %v", err)
			}
			break
		}

		if len(line) > 0 {
			m.parseAndSendEvent(line)
		}
	}
}

// watchLoop watches for file changes and triggers reads
func (m *Manager) watchLoop() {
	defer close(m.stopped)

	for {
		select {
		case event, ok := <-m.watcher.Events:
			if !ok {
				return
			}

			if event.Op&fsnotify.Write == fsnotify.Write {
				m.logger.Debug("IPC file modified: %s", event.Name)
				m.readEvents()
			}

		case err, ok := <-m.watcher.Errors:
			if !ok {
				return
			}
			// Log the error but don't block on channel send
			m.logger.Error("Watcher error: %v", err)

		case <-m.stopChan:
			return
		}
	}
}

// parseAndSendEvent parses a JSON line and sends it as an event
func (m *Manager) parseAndSendEvent(line []byte) {
	// First, decode to determine event type
	var rawEvent map[string]interface{}
	if err := json.Unmarshal(line, &rawEvent); err != nil {
		m.logger.Debug("Failed to parse event: %v", err)
		return
	}

	eventType, ok := rawEvent["eventType"].(string)
	if !ok {
		m.logger.Error("Event missing eventType field")
		return
	}

	// Parse based on event type
	var event Event
	switch EventType(eventType) {
	case EventTypeTestCase:
		// Only new group-based testCase events are supported
		var e GroupTestCaseEvent
		if err := json.Unmarshal(line, &e); err != nil {
			m.logger.Debug("Failed to parse group test case event: %v", err)
			return
		}
		event = e

	case EventTypeRunComplete:
		var e RunCompleteEvent
		if err := json.Unmarshal(line, &e); err != nil {
			m.logger.Debug("Failed to parse run complete event: %v", err)
			return
		}
		event = e

	case EventTypeCollectionStart:
		var e CollectionStartEvent
		if err := json.Unmarshal(line, &e); err != nil {
			m.logger.Debug("Failed to parse collection start event: %v", err)
			return
		}
		event = e

	case EventTypeCollectionError:
		var e CollectionErrorEvent
		if err := json.Unmarshal(line, &e); err != nil {
			m.logger.Debug("Failed to parse collection error event: %v", err)
			return
		}
		event = e

	case EventTypeCollectionFinish:
		var e CollectionFinishEvent
		if err := json.Unmarshal(line, &e); err != nil {
			m.logger.Debug("Failed to parse collection finish event: %v", err)
			return
		}
		event = e

	case EventTypeGroupDiscovered:
		var e GroupDiscoveredEvent
		if err := json.Unmarshal(line, &e); err != nil {
			m.logger.Debug("Failed to parse group discovered event: %v", err)
			return
		}
		event = e

	case EventTypeGroupStart:
		var e GroupStartEvent
		if err := json.Unmarshal(line, &e); err != nil {
			m.logger.Debug("Failed to parse group start event: %v", err)
			return
		}
		event = e

	case EventTypeGroupResult:
		var e GroupResultEvent
		if err := json.Unmarshal(line, &e); err != nil {
			m.logger.Debug("Failed to parse group result event: %v", err)
			return
		}
		event = e

	case EventTypeGroupStdout:
		var e GroupStdoutChunkEvent
		if err := json.Unmarshal(line, &e); err != nil {
			m.logger.Debug("Failed to parse group stdout event: %v", err)
			return
		}
		event = e

	case EventTypeGroupStderr:
		var e GroupStderrChunkEvent
		if err := json.Unmarshal(line, &e); err != nil {
			m.logger.Debug("Failed to parse group stderr event: %v", err)
			return
		}
		event = e

	default:
		m.logger.Error("[3PIO ERROR] Unknown event type: %s", eventType)
		return
	}

	// Send event to channel (blocking send for natural backpressure)
	m.Events <- event
	m.logger.Debug("Processing IPC event: %s", eventType)
}

// Cleanup stops watching and closes resources
func (m *Manager) Cleanup() error {
	// Signal stop to goroutines (not under lock to avoid deadlock)
	select {
	case <-m.stopChan:
		// Already closed
	default:
		close(m.stopChan)
	}

	// Wait for watchLoop to finish before cleaning up resources
	<-m.stopped

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.watcher != nil {
		_ = m.watcher.Close()
		m.watcher = nil
	}

	if m.file != nil {
		_ = m.file.Close()
		m.file = nil
	}

	// Close channels only once using sync.Once
	m.closeOnce.Do(func() {
		if m.Events != nil {
			close(m.Events)
		}
	})

	return nil
}

// SendEvent writes an event to the IPC file (for adapters)
func SendEvent(event interface{}) error {
	ipcPath := os.Getenv("THREEPIO_IPC_PATH")
	if ipcPath == "" {
		return fmt.Errorf("THREEPIO_IPC_PATH not set")
	}

	// Ensure directory exists
	ipcDir := filepath.Dir(ipcPath)
	if err := os.MkdirAll(ipcDir, 0755); err != nil {
		return fmt.Errorf("failed to create IPC directory: %w", err)
	}

	// Marshal event to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Append to file
	file, err := os.OpenFile(ipcPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open IPC file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Write JSON line
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("failed to write event: %w", err)
	}
	if _, err := file.WriteString("\n"); err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}

	return nil
}

// EnsureIPCDirectory creates the .3pio/ipc directory if it doesn't exist
func EnsureIPCDirectory() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	ipcDir := filepath.Join(cwd, ".3pio", "ipc")
	if err := os.MkdirAll(ipcDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create IPC directory: %w", err)
	}

	return ipcDir, nil
}

// noopLogger is a default logger that does nothing
type noopLogger struct{}

func (n *noopLogger) Debug(format string, args ...interface{}) {}
func (n *noopLogger) Error(format string, args ...interface{}) {}
