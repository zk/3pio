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

### Error Handling and Console Output

3pio distinguishes between different types of errors:

- **Critical Errors**: Problems that prevent test execution (e.g., test runner not found, failed to start process) are displayed to stderr for immediate visibility
- **Parsing Errors**: Issues parsing IPC events or test output are logged to debug.log only, not shown in console
- **Internal Errors**: Non-critical issues like malformed event data are logged as DEBUG level to avoid cluttering console output

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
