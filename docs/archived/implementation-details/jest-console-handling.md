# Jest Console Output Handling

## Investigation Summary

This document describes how Jest handles console output from tests and the implications for custom reporters like 3pio.

## Key Findings

### 1. Direct stdout/stderr Writes Bypass Jest

When test code uses `process.stdout.write()` or `process.stderr.write()` directly, this output:
- Appears immediately in the terminal
- Is NOT captured by Jest's reporting system
- Cannot be accessed by custom reporters through Jest's API

Example:
```javascript
process.stdout.write('Direct stdout write\n');  // Bypasses Jest
process.stderr.write('Direct stderr write\n');  // Bypasses Jest
```

### 2. Console Methods Are Intercepted by Jest

Methods like `console.log()`, `console.error()`, `console.warn()`, etc.:
- Are intercepted by Jest during test execution
- Appear with stack traces in Jest's default reporter
- Are formatted with file location and line numbers

Example:
```javascript
console.log('This is captured by Jest');    // Intercepted
console.error('This is also captured');     // Intercepted
```

### 3. testResult.console Property Is Not Populated

Despite the Jest Reporter API documentation suggesting otherwise:
- The `testResult.console` property exists on the testResult object
- It is always `undefined` in practice (tested with Jest 29.x)
- Custom reporters cannot access console output through this mechanism

Investigation code confirmed:
```javascript
// In custom reporter's onTestResult hook:
console.log('Has console?', 'console' in testResult);  // true
console.log('Console value:', testResult.console);     // undefined
```

### 4. Implications for 3pio

Because Jest's `testResult.console` is not populated, the 3pio Jest adapter must:

1. **Patch Both Console Methods AND Stream Writers**
   - Patch `console.log`, `console.error`, `console.warn`, etc.
   - Patch `process.stdout.write` and `process.stderr.write`
   - This ensures ALL output is captured, regardless of how it's generated

2. **Capture Output During Test Execution**
   - Cannot rely on Jest's testResult object for console data
   - Must actively intercept output as tests run
   - Store captured output and associate it with the current test file

3. **Restore Original Functions After Tests**
   - Critical to restore original stdout/stderr functions
   - Prevents interference with other tools or reporters

## Implementation Strategy

The current 3pio Jest adapter correctly implements this strategy:

```javascript
// In startCapture():
this.originalStdoutWrite = process.stdout.write.bind(process.stdout);
this.originalStderrWrite = process.stderr.write.bind(process.stderr);

process.stdout.write = (chunk, encoding, callback) => {
  // Capture chunk and send via IPC
  IPCManager.sendEvent({
    eventType: 'stdoutChunk',
    payload: { filePath: this.currentTestFile, chunk: String(chunk) }
  });
  // Pass through to original
  return this.originalStdoutWrite(chunk, encoding, callback);
};
```

## Testing Verification

The investigation script at `investigation/run-investigation.js` demonstrates:
- How different output methods behave in Jest
- That `testResult.console` remains undefined
- The difference between intercepted console methods and direct writes

To run the investigation:
```bash
node investigation/run-investigation.js
```

## References

- Investigation files: `investigation/jest-console-reporter.js`, `investigation/console-output.test.js`
- Jest Reporter API: https://jestjs.io/docs/configuration#reporters-arraymodulename--modulename-options
- Related issue context: Jest's console buffering behavior has changed across versions