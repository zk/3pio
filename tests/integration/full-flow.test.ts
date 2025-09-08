import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { IPCManager } from '../../src/ipc';
import { ReportManager } from '../../src/ReportManager';
import { promises as fs } from 'fs';
import path from 'path';
import os from 'os';
import { spawn } from 'child_process';

describe('Full Integration Flow', () => {
  let tempDir: string;

  beforeEach(async () => {
    tempDir = await fs.mkdtemp(path.join(os.tmpdir(), '3pio-integration-'));
    // Mock process.cwd to return temp directory
    vi.spyOn(process, 'cwd').mockReturnValue(tempDir);
  });

  afterEach(async () => {
    // Restore mocks
    vi.restoreAllMocks();
    await fs.rm(tempDir, { recursive: true, force: true });
  });

  describe('IPC and ReportManager Integration', () => {
    it('should handle complete test flow', async () => {
      const runId = new Date().toISOString().replace(/[:.-]/g, '');
      const ipcPath = path.join(tempDir, '.3pio', 'ipc', `${runId}.jsonl`);
      
      // Create IPC directory
      await IPCManager.ensureIPCDirectory();
      
      // Initialize components
      const ipcManager = new IPCManager(ipcPath);
      const reportManager = new ReportManager(runId, 'test command');
      
      const testFiles = ['test1.js', 'test2.js', 'test3.js'];
      await reportManager.initialize(testFiles);
      
      // Set up IPC event handling
      ipcManager.watchEvents(async (event) => {
        await reportManager.handleEvent(event);
      });
      
      // Wait for watcher to initialize
      await new Promise(resolve => setTimeout(resolve, 100));
      
      // Simulate test execution
      process.env.THREEPIO_IPC_PATH = ipcPath;
      
      // Test 1: Output and pass
      await IPCManager.sendEvent({
        eventType: 'stdoutChunk',
        payload: { filePath: 'test1.js', chunk: 'Running test 1...\n' }
      });
      
      // Delay between chunks
      await new Promise(resolve => setTimeout(resolve, 100));
      
      await IPCManager.sendEvent({
        eventType: 'stdoutChunk',
        payload: { filePath: 'test1.js', chunk: 'Test 1 passed!\n' }
      });
      
      // Small delay to ensure event is processed
      await new Promise(resolve => setTimeout(resolve, 100));
      
      await IPCManager.sendEvent({
        eventType: 'testFileResult',
        payload: { filePath: 'test1.js', status: 'PASS' }
      });
      
      // Test 2: Error output and fail
      await IPCManager.sendEvent({
        eventType: 'stdoutChunk',
        payload: { filePath: 'test2.js', chunk: 'Running test 2...\n' }
      });
      
      // Delay between chunks
      await new Promise(resolve => setTimeout(resolve, 100));
      
      await IPCManager.sendEvent({
        eventType: 'stderrChunk',
        payload: { filePath: 'test2.js', chunk: 'Error: Test 2 failed\n' }
      });
      
      // Small delay to ensure event is processed
      await new Promise(resolve => setTimeout(resolve, 100));
      
      await IPCManager.sendEvent({
        eventType: 'testFileResult',
        payload: { filePath: 'test2.js', status: 'FAIL' }
      });
      
      // Test 3: Skip
      await IPCManager.sendEvent({
        eventType: 'testFileResult',
        payload: { filePath: 'test3.js', status: 'SKIP' }
      });
      
      // Wait for debounced writes
      await new Promise(resolve => setTimeout(resolve, 500));
      
      // Finalize
      await reportManager.finalize(1); // Exit code 1 due to failure
      await ipcManager.cleanup();
      
      // Verify final report
      const reportPath = path.join(tempDir, '.3pio', 'runs', runId, 'test-run.md');
      const reportContent = await fs.readFile(reportPath, 'utf8');
      
      expect(reportContent).toContain('Status:** ERROR'); // Due to exit code 1
      expect(reportContent).toContain('Files Completed:** 3');
      expect(reportContent).toContain('Files Passed:** 1');
      expect(reportContent).toContain('Files Failed:** 1');
      expect(reportContent).toContain('Files Skipped:** 1');
      
      // Verify log files
      const log1Path = path.join(tempDir, '.3pio', 'runs', runId, 'logs', 'test1.js.log');
      const log1Content = await fs.readFile(log1Path, 'utf8');
      expect(log1Content).toContain('Running test 1');
      expect(log1Content).toContain('Test 1 passed!');
      
      const log2Path = path.join(tempDir, '.3pio', 'runs', runId, 'logs', 'test2.js.log');
      const log2Content = await fs.readFile(log2Path, 'utf8');
      expect(log2Content).toContain('Running test 2');
      expect(log2Content).toContain('Error: Test 2 failed');
    });
  });

  describe('Multiple concurrent events', () => {
    it('should handle rapid concurrent events', async () => {
      const runId = new Date().toISOString().replace(/[:.-]/g, '');
      const ipcPath = path.join(tempDir, '.3pio', 'ipc', `${runId}.jsonl`);
      
      await IPCManager.ensureIPCDirectory();
      
      const ipcManager = new IPCManager(ipcPath);
      const reportManager = new ReportManager(runId, 'concurrent test');
      
      const testFiles = Array.from({ length: 10 }, (_, i) => `test${i}.js`);
      await reportManager.initialize(testFiles);
      
      ipcManager.watchEvents(async (event) => {
        await reportManager.handleEvent(event);
      });
      
      // Wait for watcher
      await new Promise(resolve => setTimeout(resolve, 100));
      
      process.env.THREEPIO_IPC_PATH = ipcPath;
      
      // Send many events concurrently
      const promises = [];
      
      for (let i = 0; i < 10; i++) {
        const filePath = `test${i}.js`;
        
        // Send multiple chunks per file
        for (let j = 0; j < 5; j++) {
          promises.push(
            IPCManager.sendEvent({
              eventType: 'stdoutChunk',
              payload: { filePath, chunk: `Output line ${j}\n` }
            })
          );
        }
        
        // Send result
        const status = i % 3 === 0 ? 'FAIL' : i % 3 === 1 ? 'SKIP' : 'PASS';
        promises.push(
          IPCManager.sendEvent({
            eventType: 'testFileResult',
            payload: { filePath, status: status as any }
          })
        );
      }
      
      await Promise.all(promises);
      
      // Wait for processing
      await new Promise(resolve => setTimeout(resolve, 1000));
      
      await reportManager.finalize(0);
      await ipcManager.cleanup();
      
      // Verify all events were processed
      const reportPath = path.join(tempDir, '.3pio', 'runs', runId, 'test-run.md');
      const reportContent = await fs.readFile(reportPath, 'utf8');
      
      expect(reportContent).toContain('Files Completed:** 10');
      
      // Check log files exist and have content
      for (let i = 0; i < 10; i++) {
        const logPath = path.join(tempDir, '.3pio', 'runs', runId, 'logs', `test${i}.js.log`);
        const logContent = await fs.readFile(logPath, 'utf8');
        expect(logContent).toContain('Output line');
      }
    });
  });

  describe('Error recovery', () => {
    it('should recover from malformed events', async () => {
      const runId = new Date().toISOString().replace(/[:.-]/g, '');
      const ipcPath = path.join(tempDir, '.3pio', 'ipc', `${runId}.jsonl`);
      
      await IPCManager.ensureIPCDirectory();
      
      const ipcManager = new IPCManager(ipcPath);
      const reportManager = new ReportManager(runId, 'error test');
      
      await reportManager.initialize(['test.js']);
      
      let eventCount = 0;
      ipcManager.watchEvents(async (event) => {
        eventCount++;
        await reportManager.handleEvent(event);
      });
      
      await new Promise(resolve => setTimeout(resolve, 100));
      
      // Write malformed JSON directly
      await fs.appendFile(ipcPath, 'not valid json\n', 'utf8');
      
      // Write valid event after malformed one
      process.env.THREEPIO_IPC_PATH = ipcPath;
      await IPCManager.sendEvent({
        eventType: 'testFileResult',
        payload: { filePath: 'test.js', status: 'PASS' }
      });
      
      await new Promise(resolve => setTimeout(resolve, 200));
      
      // Should have processed only the valid event
      expect(eventCount).toBe(1);
      
      await reportManager.finalize(0);
      await ipcManager.cleanup();
      
      const reportPath = path.join(tempDir, '.3pio', 'runs', runId, 'test-run.md');
      const reportContent = await fs.readFile(reportPath, 'utf8');
      expect(reportContent).toContain('Files Passed:** 1');
    });
  });
});