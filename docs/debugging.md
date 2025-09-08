# Debugging 3pio

## Debug Logging

3pio provides debug logging capabilities to help troubleshoot issues with test runner adapters and the overall system.

### Debug Log Location

All debug logs are written to `.3pio/debug.log` in your project directory.

### What Gets Logged

#### Jest Adapter
The Jest adapter logs the following events:
- `onRunStart` - When the test run begins
- `onTestStart` - When each test file starts
- `onTestResult` - When each test file completes (with pass/fail status)
- `onRunComplete` - When the entire test run completes
- IPC communication errors
- Environment variable issues (e.g., missing `THREEPIO_IPC_PATH`)

#### Log Format
```
2025-09-08T08:50:00.000Z [jest-adapter] onRunStart called
2025-09-08T08:50:00.100Z [jest-adapter] IPC path: /path/to/.3pio/ipc/timestamp.jsonl
2025-09-08T08:50:00.200Z [jest-adapter] onTestStart called for: /path/to/test.js
```

### Enabling Debug Mode

Debug logging is automatically enabled when running tests through 3pio. No additional configuration is needed.

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