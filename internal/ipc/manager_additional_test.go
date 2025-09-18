package ipc

import (
    "bufio"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "strings"
    "sync"
    "testing"
    "time"
)

// testLogger implements ipc.Logger and records messages for assertions
type testLogger struct {
    mu     sync.Mutex
    debug  []string
    errors []string
}

func (l *testLogger) Debug(format string, args ...interface{}) {
    l.mu.Lock()
    defer l.mu.Unlock()
    l.debug = append(l.debug, sprintf(format, args...))
}

func (l *testLogger) Error(format string, args ...interface{}) {
    l.mu.Lock()
    defer l.mu.Unlock()
    l.errors = append(l.errors, sprintf(format, args...))
}

func sprintf(format string, args ...interface{}) string { return fmt.Sprintf(format, args...) }

func TestWatch_ReadsPartialLinesAndSkipsMalformed(t *testing.T) {
    t.Parallel()
    dir := t.TempDir()
    ipcPath := filepath.Join(dir, "ipc.jsonl")

    lg := &testLogger{}
    m, err := NewManager(ipcPath, lg)
    if err != nil {
        t.Fatalf("NewManager error: %v", err)
    }
    defer func() { _ = m.Cleanup() }()

    if err := m.WatchEvents(); err != nil {
        t.Fatalf("WatchEvents error: %v", err)
    }

    // Start a consumer
    received := make(chan Event, 10)
    done := make(chan struct{})
    go func() {
        for ev := range m.Events {
            received <- ev
        }
        close(done)
    }()

    // Open writer and write a valid event without newline first
    f, err := os.OpenFile(ipcPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
    if err != nil {
        t.Fatalf("open ipc: %v", err)
    }

    ev := NewGroupTestCaseEvent("t1", []string{"file.test.js"}, "PASS")
    b, _ := json.Marshal(ev)
    if _, err := f.Write(b); err != nil {
        t.Fatalf("write partial: %v", err)
    }

    // Ensure no event yet (no newline)
    select {
    case <-received:
        t.Fatal("should not have received event before newline")
    case <-time.After(50 * time.Millisecond):
        // ok
    }

    // Complete the line with newline and expect event
    if _, err := f.WriteString("\n"); err != nil {
        t.Fatalf("write newline: %v", err)
    }

    select {
    case <-received:
        // ok
    case <-time.After(2 * time.Second):
        t.Fatal("timed out waiting for event after newline")
    }

    // Write a malformed JSON line; should be skipped and logged at debug
    if _, err := f.WriteString("{this is not json}\n"); err != nil {
        t.Fatalf("write malformed: %v", err)
    }

    // Follow with a good event and ensure it still arrives
    ev2 := NewGroupTestCaseEvent("t2", []string{"file.test.js"}, "PASS")
    b2, _ := json.Marshal(ev2)
    if _, err := f.Write(append(b2, '\n')); err != nil {
        t.Fatalf("write good after malformed: %v", err)
    }
    _ = f.Close()

    select {
    case <-received:
        // ok
    case <-time.After(2 * time.Second):
        t.Fatal("timed out waiting for post-malformed event")
    }

    // Check we logged a debug parse failure
    foundDebug := false
    lg.mu.Lock()
    for _, d := range lg.debug {
        if strings.Contains(d, "Failed to parse event") {
            foundDebug = true
            break
        }
    }
    lg.mu.Unlock()
    if !foundDebug {
        t.Error("expected debug log for malformed JSON not found")
    }

    // Cleanup should close Events exactly once and stop goroutine
    if err := m.Cleanup(); err != nil {
        t.Fatalf("Cleanup error: %v", err)
    }
    // Second cleanup should be no-op
    if err := m.Cleanup(); err != nil {
        t.Fatalf("Cleanup 2 error: %v", err)
    }

    // Channel must be closed
    select {
    case <-done:
        // ok
    case <-time.After(500 * time.Millisecond):
        t.Fatal("watch loop did not stop after cleanup")
    }
}

func TestWatch_BurstOrderingAndCapacity(t *testing.T) {
    t.Parallel()
    dir := t.TempDir()
    ipcPath := filepath.Join(dir, "ipc.jsonl")

    m, err := NewManager(ipcPath, &noopLogger{})
    if err != nil {
        t.Fatalf("NewManager error: %v", err)
    }
    defer func() { _ = m.Cleanup() }()
    if err := m.WatchEvents(); err != nil {
        t.Fatalf("WatchEvents error: %v", err)
    }

    // Writer
    f, err := os.OpenFile(ipcPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
    if err != nil {
        t.Fatalf("open: %v", err)
    }
    w := bufio.NewWriter(f)

    const N = 2000
    for i := 0; i < N; i++ {
        ev := NewGroupTestCaseEvent(fmt.Sprintf("t-%04d", i), []string{"file.test.js"}, "PASS")
        b, _ := json.Marshal(ev)
        w.Write(b)
        w.WriteByte('\n')
    }
    w.Flush()
    _ = f.Close()

    // Read back
    count := 0
    first := ""
    last := ""
    timeout := time.After(5 * time.Second)
readloop:
    for {
        select {
        case e, ok := <-m.Events:
            if !ok {
                break readloop
            }
            if g, ok := e.(GroupTestCaseEvent); ok {
                name := g.Payload.TestName
                if count == 0 {
                    first = name
                }
                last = name
            }
            count++
            if count == N {
                break readloop
            }
        case <-timeout:
            t.Fatalf("timeout reading events: got %d", count)
        }
    }
    if count != N {
        t.Fatalf("expected %d events, got %d", N, count)
    }
    if first != "t-0000" || last != "t-1999" {
        t.Errorf("ordering mismatch: first=%s last=%s", first, last)
    }
}

func TestWatch_DoubleStartAndWatcherErrors(t *testing.T) {
    t.Parallel()
    dir := t.TempDir()
    ipcPath := filepath.Join(dir, "ipc.jsonl")

    lg := &testLogger{}
    m, err := NewManager(ipcPath, lg)
    if err != nil {
        t.Fatalf("NewManager error: %v", err)
    }
    defer func() { _ = m.Cleanup() }()

    if err := m.WatchEvents(); err != nil {
        t.Fatalf("WatchEvents error: %v", err)
    }
    // Second call should error
    if err := m.WatchEvents(); err == nil {
        t.Fatal("expected error on second WatchEvents call")
    }

    // Inject an error into the watcher's error channel; manager should log and continue
    select {
    case m.watcher.Errors <- fmt.Errorf("synthetic watcher error"):
    case <-time.After(time.Second):
        t.Fatal("failed to inject watcher error")
    }

    // Now write a valid event and ensure it still flows
    f, _ := os.OpenFile(ipcPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
    ev := NewGroupTestCaseEvent("ok", []string{"file.test.js"}, "PASS")
    b, _ := json.Marshal(ev)
    f.Write(append(b, '\n'))
    _ = f.Close()

    select {
    case <-m.Events:
        // ok
    case <-time.After(2 * time.Second):
        t.Fatal("no event received after watcher error")
    }
}
