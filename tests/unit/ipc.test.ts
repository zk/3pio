import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { IPCManager } from '../../src/ipc';
import { promises as fs } from 'fs';
import path from 'path';
import os from 'os';

describe('IPCManager', () => {
  let tempDir: string;
  let ipcFilePath: string;
  let ipcManager: IPCManager;

  beforeEach(async () => {
    // Create a temporary directory for testing
    tempDir = await fs.mkdtemp(path.join(os.tmpdir(), '3pio-test-'));
    ipcFilePath = path.join(tempDir, 'test.ipc');
    ipcManager = new IPCManager(ipcFilePath);
  });

  afterEach(async () => {
    // Clean up
    if (ipcManager) {
      await ipcManager.cleanup();
    }
    await fs.rm(tempDir, { recursive: true, force: true });
  });

  describe('writeEvent', () => {
    it('should write event to IPC file', async () => {
      const event = {
        eventType: 'testFileResult' as const,
        payload: {
          filePath: 'test.js',
          status: 'PASS' as const
        }
      };

      await ipcManager.writeEvent(event);

      const content = await fs.readFile(ipcFilePath, 'utf8');
      const lines = content.split('\n').filter(line => line.trim());
      expect(lines).toHaveLength(1);
      
      const parsedEvent = JSON.parse(lines[0]);
      expect(parsedEvent).toEqual(event);
    });

    it('should append multiple events', async () => {
      const event1 = {
        eventType: 'stdoutChunk' as const,
        payload: {
          filePath: 'test.js',
          chunk: 'output 1'
        }
      };

      const event2 = {
        eventType: 'stderrChunk' as const,
        payload: {
          filePath: 'test.js',
          chunk: 'error 1'
        }
      };

      await ipcManager.writeEvent(event1);
      await ipcManager.writeEvent(event2);

      const content = await fs.readFile(ipcFilePath, 'utf8');
      const lines = content.split('\n').filter(line => line.trim());
      expect(lines).toHaveLength(2);
      
      expect(JSON.parse(lines[0])).toEqual(event1);
      expect(JSON.parse(lines[1])).toEqual(event2);
    });
  });

  describe('sendEvent (static)', () => {
    it('should throw error if THREEPIO_IPC_PATH not set', async () => {
      const originalEnv = process.env.THREEPIO_IPC_PATH;
      delete process.env.THREEPIO_IPC_PATH;

      const event = {
        eventType: 'testFileResult' as const,
        payload: {
          filePath: 'test.js',
          status: 'PASS' as const
        }
      };

      await expect(IPCManager.sendEvent(event)).rejects.toThrow(
        'THREEPIO_IPC_PATH environment variable not set'
      );

      process.env.THREEPIO_IPC_PATH = originalEnv;
    });

    it('should write to IPC path from environment', async () => {
      process.env.THREEPIO_IPC_PATH = ipcFilePath;

      const event = {
        eventType: 'testFileResult' as const,
        payload: {
          filePath: 'test.js',
          status: 'FAIL' as const
        }
      };

      await IPCManager.sendEvent(event);

      const content = await fs.readFile(ipcFilePath, 'utf8');
      const lines = content.split('\n').filter(line => line.trim());
      expect(lines).toHaveLength(1);
      
      const parsedEvent = JSON.parse(lines[0]);
      expect(parsedEvent).toEqual(event);
    });
  });

  describe('watchEvents', () => {
    it('should trigger callback for new events', async () => {
      const events: any[] = [];
      const callback = vi.fn((event) => {
        events.push(event);
      });

      ipcManager.watchEvents(callback);

      // Give watcher time to initialize
      await new Promise(resolve => setTimeout(resolve, 100));

      const testEvent = {
        eventType: 'testFileResult' as const,
        payload: {
          filePath: 'test.js',
          status: 'PASS' as const
        }
      };

      await ipcManager.writeEvent(testEvent);

      // Give watcher time to detect change
      await new Promise(resolve => setTimeout(resolve, 200));

      expect(callback).toHaveBeenCalled();
      expect(events).toHaveLength(1);
      expect(events[0]).toEqual(testEvent);
    });

    it('should handle multiple events in sequence', async () => {
      const events: any[] = [];
      const callback = vi.fn((event) => {
        events.push(event);
      });

      ipcManager.watchEvents(callback);

      // Give watcher time to initialize
      await new Promise(resolve => setTimeout(resolve, 100));

      const event1 = {
        eventType: 'stdoutChunk' as const,
        payload: { filePath: 'test1.js', chunk: 'output 1' }
      };

      const event2 = {
        eventType: 'stderrChunk' as const,
        payload: { filePath: 'test2.js', chunk: 'error 1' }
      };

      await ipcManager.writeEvent(event1);
      await new Promise(resolve => setTimeout(resolve, 100));
      
      await ipcManager.writeEvent(event2);
      await new Promise(resolve => setTimeout(resolve, 100));

      expect(events).toHaveLength(2);
      expect(events[0]).toEqual(event1);
      expect(events[1]).toEqual(event2);
    });

    it('should ignore malformed JSON lines', async () => {
      const events: any[] = [];
      const callback = vi.fn((event) => {
        events.push(event);
      });

      const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

      ipcManager.watchEvents(callback);

      // Give watcher time to initialize
      await new Promise(resolve => setTimeout(resolve, 100));

      // Write invalid JSON
      await fs.appendFile(ipcFilePath, 'invalid json\n', 'utf8');
      
      // Write valid event
      const validEvent = {
        eventType: 'testFileResult' as const,
        payload: { filePath: 'test.js', status: 'PASS' as const }
      };
      await ipcManager.writeEvent(validEvent);

      // Give watcher time to process
      await new Promise(resolve => setTimeout(resolve, 200));

      expect(events).toHaveLength(1);
      expect(events[0]).toEqual(validEvent);
      expect(consoleErrorSpy).toHaveBeenCalledWith(
        'Failed to parse IPC event:',
        expect.any(Error)
      );

      consoleErrorSpy.mockRestore();
    });
  });

  describe('ensureIPCDirectory', () => {
    it('should create IPC directory if it does not exist', async () => {
      const cwd = process.cwd();
      const expectedDir = path.join(cwd, '.3pio', 'ipc');
      
      // Mock process.cwd to use our temp directory
      const cwdSpy = vi.spyOn(process, 'cwd').mockReturnValue(tempDir);
      
      const ipcDir = await IPCManager.ensureIPCDirectory();
      
      expect(ipcDir).toBe(path.join(tempDir, '.3pio', 'ipc'));
      
      // Check directory exists
      const stats = await fs.stat(ipcDir);
      expect(stats.isDirectory()).toBe(true);
      
      cwdSpy.mockRestore();
    });

    it('should not throw if directory already exists', async () => {
      const cwdSpy = vi.spyOn(process, 'cwd').mockReturnValue(tempDir);
      
      // Create directory first
      const ipcDir = path.join(tempDir, '.3pio', 'ipc');
      await fs.mkdir(ipcDir, { recursive: true });
      
      // Should not throw
      const result = await IPCManager.ensureIPCDirectory();
      expect(result).toBe(ipcDir);
      
      cwdSpy.mockRestore();
    });
  });

  describe('cleanup', () => {
    it('should stop watching and delete IPC file', async () => {
      // Create the file first
      await fs.writeFile(ipcFilePath, '', 'utf8');
      
      // Start watching
      ipcManager.watchEvents(() => {});
      
      // Give watcher time to initialize
      await new Promise(resolve => setTimeout(resolve, 100));
      
      await ipcManager.cleanup();
      
      // File should be deleted
      await expect(fs.access(ipcFilePath)).rejects.toThrow();
    });

    it('should not throw if file does not exist', async () => {
      // Don't create the file
      await expect(ipcManager.cleanup()).resolves.not.toThrow();
    });
  });
});