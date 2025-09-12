# Alternative Methods for Passing IPC Path to Test Adapters

## Current Problem
The Vitest adapter relies on `process.env.THREEPIO_IPC_PATH` (line 107 in vitest.js), but this environment variable doesn't reliably propagate to child processes in monorepo setups.

## Alternative Approaches

### 1. **File-Based Discovery** ✅ (Recommended)
Instead of passing the path, have adapters discover it:
```javascript
// Adapter looks for IPC file in .3pio/ipc/ directory
const findIPCPath = () => {
  const ipcDir = path.join(process.cwd(), '.3pio', 'ipc');
  // Find the most recent .jsonl file
  const files = fs.readdirSync(ipcDir)
    .filter(f => f.endsWith('.jsonl'))
    .sort().reverse();
  return files[0] ? path.join(ipcDir, files[0]) : null;
};
```
**Pros**: No coordination needed, works across process boundaries
**Cons**: Requires timestamp-based disambiguation if multiple runs overlap

### 2. **CLI Argument Injection**
Pass IPC path as a CLI argument to the test runner:
```javascript
// Modified command construction in Go
cmd := `vitest run --reporter ${adapterPath} --reporter-options ipcPath=${ipcPath}`
```
**Pros**: Explicit, no environment variable issues
**Cons**: Different syntax for each test runner (Jest, Vitest, pytest)

### 3. **Configuration File**
Write a temporary config file that adapters read:
```javascript
// 3pio writes: .3pio/current-run.json
{
  "ipcPath": "/path/to/.3pio/ipc/20250911T085108.jsonl",
  "runId": "20250911T085108-feisty-han-solo"
}
```
**Pros**: Works across all process boundaries, extensible
**Cons**: Potential race conditions with concurrent runs

### 4. **Process Title Hack**
Encode IPC path in process title:
```javascript
// In Go before spawning
process.title = `3pio:${ipcPath}`

// In adapter
const ipcPath = process.title.match(/3pio:(.+)/)?.[1]
```
**Pros**: Survives process spawning
**Cons**: Hacky, limited length, may be overwritten

### 5. **Global Node Module**
Use Node's module system to share state:
```javascript
// Create a global module that both 3pio and adapters can access
global.__3PIO_IPC_PATH__ = ipcPath;
```
**Pros**: Simple for Node.js environments
**Cons**: Doesn't work across process boundaries in monorepos

### 6. **Named Pipe/Socket**
Use a well-known named pipe location:
```javascript
// Always use the same socket path
const SOCKET_PATH = process.platform === 'win32' 
  ? '\\\\.\\pipe\\3pio-ipc'
  : '/tmp/3pio-ipc.sock';
```
**Pros**: No path discovery needed
**Cons**: More complex, platform-specific, cleanup issues

### 7. **Reporter Configuration via Test Framework**
Leverage each framework's reporter configuration:
```javascript
// For Vitest - use reporter options
export default defineConfig({
  test: {
    reporters: [
      ['./3pio-adapter.js', { ipcPath: process.env.THREEPIO_IPC_PATH }]
    ]
  }
})
```
**Pros**: Framework-native approach
**Cons**: Requires dynamic config generation

## Recommended Solution: Hybrid Approach

Combine multiple methods for robustness:

1. **Primary**: Write `.3pio/current-run.json` with run metadata
2. **Fallback**: Check environment variable
3. **Last Resort**: Scan `.3pio/ipc/` for most recent file

```javascript
// In adapter
const getIPCPath = () => {
  // Method 1: Check current run file
  try {
    const configPath = path.join(process.cwd(), '.3pio', 'current-run.json');
    if (fs.existsSync(configPath)) {
      const config = JSON.parse(fs.readFileSync(configPath, 'utf8'));
      if (config.ipcPath && fs.existsSync(config.ipcPath)) {
        return config.ipcPath;
      }
    }
  } catch {}

  // Method 2: Check environment variable
  if (process.env.THREEPIO_IPC_PATH) {
    return process.env.THREEPIO_IPC_PATH;
  }

  // Method 3: Find most recent IPC file
  try {
    const ipcDir = path.join(process.cwd(), '.3pio', 'ipc');
    const files = fs.readdirSync(ipcDir)
      .filter(f => f.endsWith('.jsonl'))
      .map(f => ({
        name: f,
        path: path.join(ipcDir, f),
        mtime: fs.statSync(path.join(ipcDir, f)).mtime
      }))
      .sort((a, b) => b.mtime - a.mtime);
    
    if (files.length > 0) {
      return files[0].path;
    }
  } catch {}

  return null;
};
```

## Implementation Plan

1. **Phase 1**: Add `.3pio/current-run.json` creation in Go code
2. **Phase 2**: Update adapters to use hybrid discovery
3. **Phase 3**: Add integration tests for monorepo scenarios
4. **Phase 4**: Document the discovery mechanism

This approach ensures adapters can find the IPC path even when environment variables don't propagate through complex process hierarchies in monorepos.