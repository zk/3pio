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

type testLogger struct {
    mu     sync.Mutex
    debug  []string
    errors []string
}

func (l *testLogger) Debug(format string, args ...interface{}) {
    l.mu.Lock()
    defer l.mu.Unlock()
    l.debug = append(l.debug, fmt.Sprintf(format, args...))
}

func (l *testLogger) Error(format string, args ...interface{}) {
    l.mu.Lock()
    defer l.mu.Unlock()
    l.errors = append(l.errors, fmt.Sprintf(format, args...))
}

func TestWatch_ReadsPartialLinesAndSkipsMalformed(t *testing.T) {
    dir := t.TempDir()
    ipcPath := filepath.Join(dir, "ipc.jsonl")

    lg := &testLogger{}
    m, err := NewManager(ipcPath, lg)
    if err != nil { t.Fatalf("NewManager: %v", err) }
    defer func() { _ = m.Cleanup() }()
    if err := m.WatchEvents(); err != nil { t.Fatalf("WatchEvents: %v", err) }

    received := make(chan Event, 10)
    done := make(chan struct{})
    go func() {
        for ev := range m.Events { received <- ev }
        close(done)
    }()

    f, err := os.OpenFile(ipcPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
    if err != nil { t.Fatalf("open: %v", err) }

    ev := NewGroupTestCaseEvent("t1", []string{"file.test.js"}, "PASS")
    b, _ := json.Marshal(ev)
    if _, err := f.Write(b); err != nil { t.Fatalf("write partial: %v", err) }

    select {
    case <-received:
        t.Fatal("received event before newline")
    case <-time.After(50 * time.Millisecond):
    }

    if _, err := f.WriteString("\n"); err != nil { t.Fatalf("newline: %v", err) }
    select {
    case <-received:
    case <-time.After(2 * time.Second):
        t.Fatal("timeout waiting event after newline")
    }

    if _, err := f.WriteString("{bad json}\n"); err != nil { t.Fatalf("malformed: %v", err) }

    ev2 := NewGroupTestCaseEvent("t2", []string{"file.test.js"}, "PASS")
    b2, _ := json.Marshal(ev2)
    if _, err := f.Write(append(b2, '\n')); err != nil { t.Fatalf("good after malformed: %v", err) }
    _ = f.Close()

    select {
    case <-received:
    case <-time.After(2 * time.Second):
        t.Fatal("timeout post-malformed event")
    }

    lg.mu.Lock()
    found := false
    for _, d := range lg.debug { if strings.Contains(d, "Failed to parse event") { found = true; break } }
    lg.mu.Unlock()
    if !found { t.Error("expected debug log for malformed JSON") }

    if err := m.Cleanup(); err != nil { t.Fatalf("cleanup: %v", err) }
    if err := m.Cleanup(); err != nil { t.Fatalf("cleanup 2: %v", err) }
    select {
    case <-done:
    case <-time.After(500 * time.Millisecond):
        t.Fatal("watch loop did not stop")
    }
}

func TestWatch_BurstOrderingAndCapacity(t *testing.T) {
    dir := t.TempDir()
    ipcPath := filepath.Join(dir, "ipc.jsonl")
    m, err := NewManager(ipcPath, &noopLogger{})
    if err != nil { t.Fatalf("NewManager: %v", err) }
    defer func() { _ = m.Cleanup() }()
    if err := m.WatchEvents(); err != nil { t.Fatalf("WatchEvents: %v", err) }

    f, _ := os.OpenFile(ipcPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
    w := bufio.NewWriter(f)
    const N = 2000
    for i := 0; i < N; i++ {
        ev := NewGroupTestCaseEvent(fmt.Sprintf("t-%04d", i), []string{"file.test.js"}, "PASS")
        b, _ := json.Marshal(ev)
        w.Write(b); w.WriteByte('\n')
    }
    _ = w.Flush(); _ = f.Close()

    count := 0
    first, last := "", ""
    timeout := time.After(5 * time.Second)
    for count < N {
        select {
        case e := <-m.Events:
            if g, ok := e.(GroupTestCaseEvent); ok {
                if count == 0 { first = g.Payload.TestName }
                last = g.Payload.TestName
            }
            count++
        case <-timeout:
            t.Fatalf("timeout: got %d events", count)
        }
    }
    if first != "t-0000" || last != "t-1999" {
        t.Errorf("ordering mismatch: first=%s last=%s", first, last)
    }
}

func TestWatch_DoubleStartAndWatcherErrors(t *testing.T) {
    dir := t.TempDir()
    ipcPath := filepath.Join(dir, "ipc.jsonl")
    lg := &testLogger{}
    m, err := NewManager(ipcPath, lg)
    if err != nil { t.Fatalf("NewManager: %v", err) }
    defer func() { _ = m.Cleanup() }()
    if err := m.WatchEvents(); err != nil { t.Fatalf("WatchEvents: %v", err) }
    if err := m.WatchEvents(); err == nil { t.Fatal("expected error on second WatchEvents") }

    select { case m.watcher.Errors <- fmt.Errorf("synthetic error"): case <-time.After(time.Second): t.Fatal("inject error") }

    f, _ := os.OpenFile(ipcPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
    ev := NewGroupTestCaseEvent("ok", []string{"file.test.js"}, "PASS")
    b, _ := json.Marshal(ev)
    f.Write(append(b, '\n')); _ = f.Close()
    select { case <-m.Events: case <-time.After(2 * time.Second): t.Fatal("no event after watcher error") }
}

