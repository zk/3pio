# Debugging 3pio

## Debug Logging

3pio provides comprehensive debug logging to help troubleshoot issues with test runner adapters and the overall system. All components write to a centralized debug log file.

### Debug Log Location

All debug logs are written to `.3pio/debug.log` in your project directory. This includes:
- CLI orchestrator logs (process management, runner detection, IPC events)
- Adapter logs (Jest, Vitest, pytest lifecycle events)
- Error messages and stack traces

### What Gets Logged

The debug log captures the complete lifecycle of test execution across all components. The CLI orchestrator logs session boundaries, test runner detection, process management, and IPC event processing. Each test adapter (Jest, Vitest, pytest) logs its lifecycle events including initialization, test file discovery, execution progress, and completion status. Any errors, missing environment variables, or IPC communication issues are also logged with full context to aid in troubleshooting.

### Logging Policy

**All logging goes to `.3pio/debug.log`** - no debug output to console:

- **Debug/Info/Warning**: Written to debug.log only
- **Errors**: Written to debug.log (may also show user-friendly message in console)
- **Critical Failures**: User-facing error message to stderr, full details in debug.log

The console is reserved for:
- Test execution progress (from test runners)
- Final test results summary
- Critical error messages that require user action

This ensures the console remains clean and focused on test results while comprehensive debugging information is available in the debug.log file.

### Log Format

The debug log uses a consistent timestamped format:
```
=== 3pio Debug Log ===
Session started: 2025-09-11T13:40:56-10:00
PID: 60381
Working directory: /Users/project/path
---

[2025-09-11 13:40:56.856] [DEBUG] Using embedded adapter: /path/to/adapter
[2025-09-11 13:40:56.860] [DEBUG] Executing command: [npx vitest run]
[2025-09-11 13:40:56.865] [INFO] Test runner detected: vitest
[vitest-adapter] Lifecycle: Test run initializing
[jest-adapter] onRunStart called
[pytest-adapter] Session started

--- Session ended: 2025-09-11T13:40:57-10:00 ---
```
