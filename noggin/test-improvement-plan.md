IPC, Logger, Orchestrator, and Property/Fuzz Test Improvements

Scope: implement targeted tests to raise coverage and resilience.

- IPC manager tests (increase from ~33%)
  - Add watch loop and cleanup tests:
    - Verify write→read behavior for successive writes without newline boundaries (partial lines) and malformed JSON dropped with debug logs (internal/ipc/manager.go:99, internal/ipc/manager.go:164).
    - Ensure Cleanup closes channels exactly once, watcher closed, and goroutine stops (internal/ipc/manager.go:306).
    - Burst tests: write thousands of lines quickly; ensure Events channel doesn’t deadlock and ordering is preserved (buffer is 10k) (internal/ipc/manager.go:49).
  - Error/edge coverage:
    - Double WatchEvents call error path.
    - Watcher error path (fsnotify Errors chan) logs but doesn’t crash (internal/ipc/manager.go:137).

- Logger tests (increase from ~15%)
  - FileLogger end-to-end:
    - NewFileLogger writes header and respects THREEPIO_LOG_LEVEL, and Close writes footer (internal/logger/file_logger.go:53, internal/logger/file_logger.go:141).
    - Filtering: DEBUG/INFO/WARN thresholds; ensure Error always logs + writes to stderr.
    - Concurrency: multiple goroutines logging concurrently without data races (use -race; already in Makefile test target).
    - Failure handling: simulate unwritable .3pio dir (permission denied) → proper error wrapping.

- Orchestrator tests (beyond ~41%)
  - Run lifecycle without invoking real tools:
    - Inject a fake runner/manager producing a deterministic event stream through IPC; confirm end-to-end counters, exit codes, and final report status (internal/orchestrator/orchestrator_test.go:1).
    - Context cancellation: ensure SIGINT/SIGTERM equivalents (orchestration cancellation) propagate to process abstraction and tear down gracefully.
  - Parallel event processing: simulate interleaved stdout/stderr/group events, confirm stable counts and deterministic report contents.

- Fuzz + property tests
  - Fuzz SanitizeGroupName and path generators with random Unicode/long inputs to guarantee constraints (no invalid chars, length caps, stable hashing) (internal/report/group_path.go:1).
  - Invariants for runner BuildCommand:
    - “Reporter always present exactly once”
    - “Separator policy honored” (no extra “--” for yarn, correct placement for npm/pnpm/bun)
    - “Argument order preserved except for deterministic injection points”

