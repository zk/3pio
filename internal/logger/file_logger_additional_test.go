package logger

import (
    "bufio"
    "bytes"
    "io"
    "os"
    "path/filepath"
    "regexp"
    "runtime"
    "strings"
    "sync"
    "testing"
)

func withTempCwd(t *testing.T, fn func()) {
    t.Helper()
    old, _ := os.Getwd()
    dir := t.TempDir()
    if err := os.Chdir(dir); err != nil { t.Fatalf("chdir: %v", err) }
    defer func() { _ = os.Chdir(old) }()
    fn()
}

func readLog(t *testing.T) string {
    t.Helper()
    b, err := os.ReadFile(filepath.Join(".3pio", "debug.log"))
    if err != nil { t.Fatalf("read log: %v", err) }
    return string(b)
}

func captureStderr(t *testing.T, fn func()) string {
    t.Helper()
    old := os.Stderr
    r, w, _ := os.Pipe()
    os.Stderr = w
    fn()
    _ = w.Close()
    os.Stderr = old
    var buf bytes.Buffer
    _, _ = io.Copy(&buf, r)
    return buf.String()
}

func TestFileLogger_HeaderFooterAndLevels(t *testing.T) {
    withTempCwd(t, func() {
        // Set DEBUG to allow all logs
        t.Setenv("THREEPIO_LOG_LEVEL", "DEBUG")
        l, err := NewFileLogger()
        if err != nil { t.Fatalf("NewFileLogger: %v", err) }

        // Log at all levels
        l.Debug("dbg %d", 1)
        l.Info("inf %s", "x")
        l.Warn("wrn")
        stderr := captureStderr(t, func() { l.Error("err %v", 123) })
        if !strings.Contains(stderr, "[ERROR] err 123") {
            t.Errorf("stderr missing error line: %q", stderr)
        }
        if err := l.Close(); err != nil { t.Fatalf("Close: %v", err) }

        content := readLog(t)
        if !strings.Contains(content, "=== 3pio Debug Log ===") { t.Error("missing header") }
        if !strings.Contains(content, "--- Session ended:") { t.Error("missing footer") }
        // Check lines exist for each level
        for _, s := range []string{"[DEBUG] dbg 1", "[INFO] inf x", "[WARN] wrn", "[ERROR] err 123"} {
            if !strings.Contains(content, s) { t.Errorf("missing log line: %s", s) }
        }

        // Basic timestamp format check
        re := regexp.MustCompile(`\n\[\d{4}-\d{2}-\d{2} .*] \[(DEBUG|INFO|WARN|ERROR)] `)
        if !re.MatchString(content) {
            t.Error("missing timestamped entries")
        }
    })
}

func TestFileLogger_LevelFiltering(t *testing.T) {
    withTempCwd(t, func() {
        t.Setenv("THREEPIO_LOG_LEVEL", "WARN")
        l, err := NewFileLogger()
        if err != nil { t.Fatalf("NewFileLogger: %v", err) }
        l.Debug("hidden")
        l.Info("hidden")
        l.Warn("visible")
        l.Error("vis")
        _ = l.Close()

        content := readLog(t)
        if strings.Contains(content, "hidden") {
            t.Error("DEBUG/INFO should be filtered at WARN level")
        }
        if !strings.Contains(content, "visible") || !strings.Contains(content, "vis") {
            t.Error("WARN/ERROR should be present at WARN level")
        }
    })
}

func TestFileLogger_Concurrency_NoRaceAndCount(t *testing.T) {
    withTempCwd(t, func() {
        t.Setenv("THREEPIO_LOG_LEVEL", "INFO")
        l, err := NewFileLogger()
        if err != nil { t.Fatalf("NewFileLogger: %v", err) }

        const goroutines = 10
        const perG = 200
        wg := sync.WaitGroup{}
        wg.Add(goroutines)
        for g := 0; g < goroutines; g++ {
            go func(id int) {
                defer wg.Done()
                for i := 0; i < perG; i++ {
                    l.Info("g%02d-%03d", id, i)
                }
            }(g)
        }
        wg.Wait()
        _ = l.Close()

        f, err := os.Open(filepath.Join(".3pio", "debug.log"))
        if err != nil { t.Fatalf("open log: %v", err) }
        defer f.Close()
        r := bufio.NewScanner(f)
        count := 0
        for r.Scan() {
            if strings.Contains(r.Text(), "] [INFO] g") {
                count++
            }
        }
        expected := goroutines * perG
        if count < expected {
            t.Fatalf("expected at least %d info lines, got %d", expected, count)
        }
    })
}

func TestFileLogger_UnwritableDir(t *testing.T) {
    withTempCwd(t, func() {
        // Ensure no preexisting .3pio directory
        _ = os.RemoveAll(".3pio")
        // Create a file named .3pio to force MkdirAll failure
        if err := os.WriteFile(".3pio", []byte("x"), 0644); err != nil {
            t.Fatalf("prep .3pio file: %v", err)
        }
        _, err := NewFileLogger()
        if err == nil { t.Fatal("expected error for unwritable .3pio setup") }
        if runtime.GOOS == "windows" {
            // Message text can vary; just ensure it's non-nil
            return
        }
        if !strings.Contains(err.Error(), "failed to create .3pio directory") {
            t.Errorf("unexpected error: %v", err)
        }
    })
}
