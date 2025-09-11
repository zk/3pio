# Debugging 3pio

## Debug Logging

3pio provides comprehensive debug logging to help troubleshoot issues with test runner adapters and the overall system. All components write to a centralized debug log file.

### Debug Log Location

All debug logs are written to `.3pio/debug.log` in your project directory. This includes:
- CLI orchestrator logs (process management, runner detection, IPC events)
- Adapter logs (Jest, Vitest, pytest lifecycle events)
- Error messages and stack traces

### What Gets Logged

#### CLI Orchestrator
- Session start/end with PID and working directory
- Test runner detection and command building
- Process spawning and management
- IPC event processing
- Report generation
- Error conditions

#### Jest Adapter
- `onRunStart` - When the test run begins
- `onTestStart` - When each test file starts
- `onTestResult` - When each test file completes (with pass/fail status)
- `onRunComplete` - When the entire test run completes
- IPC communication errors
- Environment variable issues (e.g., missing `THREEPIO_IPC_PATH`)

#### Vitest Adapter
- Initialization and configuration
- Test file discovery
- Test execution lifecycle events
- IPC communication status
- Adapter shutdown

#### pytest Adapter
- Plugin initialization
- Test collection phase
- Test execution hooks
- Test results and outcomes
- Session completion

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

### Enabling Debug Mode

Debug logging is automatically enabled for all 3pio runs. The file logger writes to `.3pio/debug.log` without any console output (unless errors occur).

### Viewing Debug Logs

To view the debug log in real-time:
```bash
tail -f .3pio/debug.log
```

To view debug logs for a specific run:
```bash
grep "2025-09-08T08:50" .3pio/debug.log
```

### Troubleshooting Common Issues

#### Missing Test Results
If test results are not being captured:
1. Check `.3pio/debug.log` for adapter lifecycle events
2. Look for "onTestResult" entries to see if tests are completing
3. Check for IPC errors in the log

#### Race Conditions
If tests complete but results are incomplete:
1. Look for "onRunComplete" in the debug log
2. Check if it appears before all "onTestResult" events
3. The adapter adds a "runComplete" marker event to help detect this

#### Silent Failures
Since test adapters must be silent (no console output), all errors are logged to `.3pio/debug.log` instead of the console.

### Cleaning Up Debug Logs

Debug logs can grow large over time. To clean them:
```bash
rm .3pio/debug.log
```

Or to archive them:
```bash
mv .3pio/debug.log .3pio/debug-$(date +%Y%m%d).log
```