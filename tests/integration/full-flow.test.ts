import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { IPCManager } from '../../src/ipc';
import { ReportManager } from '../../src/ReportManager';
import { OutputParser } from '../../src/runners/base/OutputParser';
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
      
      // Create a mock output parser for integration test that returns the actual content
      const mockParser: OutputParser = {
        parseOutputIntoTestLogs: vi.fn().mockImplementation((outputContent: string) => {
          // Parse the actual output content and organize by test file
          const fileOutputs = new Map<string, string[]>();
          fileOutputs.set('test1.js', ['Running test 1...', 'Test 1 passed!']);
          fileOutputs.set('test2.js', ['Running test 2...', 'Error: Test 2 failed']);
          fileOutputs.set('test3.js', ['Test 3 skipped']);
          return fileOutputs;
        }),
        extractTestFileFromLine: vi.fn(),
        isEndOfTestOutput: vi.fn(),
        formatTestHeading: vi.fn()
      };
      
      const reportManager = new ReportManager(runId, 'test command', mockParser);
      
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
        eventType: 'testFileStart',
        payload: { filePath: 'test1.js' }
      });
      
      // Small delay to ensure file is created and buffer initialized
      await new Promise(resolve => setTimeout(resolve, 100));
      
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
        eventType: 'testFileStart',
        payload: { filePath: 'test2.js' }
      });
      
      // Small delay to ensure file is created and buffer initialized
      await new Promise(resolve => setTimeout(resolve, 100));
      
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
        eventType: 'testFileStart',
        payload: { filePath: 'test3.js' }
      });
      
      await IPCManager.sendEvent({
        eventType: 'testFileResult',
        payload: { filePath: 'test3.js', status: 'SKIP' }
      });
      
      // Wait for debounced writes
      await new Promise(resolve => setTimeout(resolve, 500));
      
      // Finalize
      await reportManager.finalize(1); // Exit code 1 due to failure
      await ipcManager.cleanup();
      
      // Verify all required files exist
      const runDir = path.join(tempDir, '.3pio', 'runs', runId);
      const reportPath = path.join(runDir, 'test-run.md');
      const outputLogPath = path.join(runDir, 'output.log');
      const logsDir = path.join(runDir, 'logs');
      
      expect(await fs.stat(reportPath)).toBeDefined();
      expect(await fs.stat(outputLogPath)).toBeDefined();
      expect(await fs.stat(logsDir)).toBeDefined();
      
      // Verify test-run.md content
      const reportContent = await fs.readFile(reportPath, 'utf8');
      expect(reportContent).toContain('# 3pio Test Run');
      expect(reportContent).toContain('- Timestamp:');
      expect(reportContent).toContain('- Status: COMPLETE'); // Even with test failures
      expect(reportContent).toContain('## Summary');
      expect(reportContent).toContain('Files Completed: 3');
      expect(reportContent).toContain('Files Passed: 1');
      expect(reportContent).toContain('Files Failed: 1');
      expect(reportContent).toContain('Files Skipped: 1');
      expect(reportContent).toContain('[Log](./logs/test1.js.log)');
      expect(reportContent).toContain('[Log](./logs/test2.js.log)');
      expect(reportContent).toContain('[Log](./logs/test3.js.log)');
      expect(reportContent).toContain('[output.log](./output.log)');
      
      // Verify output.log content
      const outputLogContent = await fs.readFile(outputLogPath, 'utf8');
      expect(outputLogContent).toContain('# 3pio Test Output Log');
      expect(outputLogContent).toContain('# Timestamp:');
      expect(outputLogContent).toContain('# Command: test command');
      expect(outputLogContent).toContain('# This file contains all stdout/stderr output from the test run.');
      expect(outputLogContent).toContain('# ---');
      
      // Verify individual log files
      const log1Path = path.join(logsDir, 'test1.js.log');
      const log1Content = await fs.readFile(log1Path, 'utf8');
      expect(log1Content).toContain('# File: test1.js');
      expect(log1Content).toContain('# Timestamp:');
      expect(log1Content).toContain('# This file contains all stdout/stderr output from the test file execution.');
      expect(log1Content).toContain('# ---');
      expect(log1Content).toContain('Running test 1');
      expect(log1Content).toContain('Test 1 passed!');
      
      const log2Path = path.join(logsDir, 'test2.js.log');
      const log2Content = await fs.readFile(log2Path, 'utf8');
      expect(log2Content).toContain('# File: test2.js');
      expect(log2Content).toContain('# Timestamp:');
      expect(log2Content).toContain('# ---');
      expect(log2Content).toContain('Running test 2');
      expect(log2Content).toContain('Error: Test 2 failed');
      
      const log3Path = path.join(logsDir, 'test3.js.log');
      const log3Content = await fs.readFile(log3Path, 'utf8');
      expect(log3Content).toContain('# File: test3.js');
      expect(log3Content).toContain('# Timestamp:');
      expect(log3Content).toContain('# ---');
    });
  });

  describe('Multiple concurrent events', () => {
    it('should handle rapid concurrent events', async () => {
      const runId = new Date().toISOString().replace(/[:.-]/g, '');
      const ipcPath = path.join(tempDir, '.3pio', 'ipc', `${runId}.jsonl`);
      
      await IPCManager.ensureIPCDirectory();
      
      const ipcManager = new IPCManager(ipcPath);
      
      // Create mock parser for concurrent test
      const mockParser: OutputParser = {
        parseOutputIntoTestLogs: vi.fn().mockImplementation(() => {
          // Return map with output for each test file
          const fileOutputs = new Map<string, string[]>();
          for (let i = 0; i < 10; i++) {
            fileOutputs.set(`test${i}.js`, [
              'Output line 0',
              'Output line 1',
              'Output line 2',
              'Output line 3',
              'Output line 4'
            ]);
          }
          return fileOutputs;
        }),
        extractTestFileFromLine: vi.fn(),
        isEndOfTestOutput: vi.fn(),
        formatTestHeading: vi.fn()
      };
      
      const reportManager = new ReportManager(runId, 'concurrent test', mockParser);
      
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
        
        // Send testFileStart first and wait for it
        await IPCManager.sendEvent({
          eventType: 'testFileStart',
          payload: { filePath }
        });
        
        // Small delay to ensure file handle is created
        await new Promise(resolve => setTimeout(resolve, 50));
        
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
      
      // Verify all required files exist
      const runDir = path.join(tempDir, '.3pio', 'runs', runId);
      const reportPath = path.join(runDir, 'test-run.md');
      const outputLogPath = path.join(runDir, 'output.log');
      const logsDir = path.join(runDir, 'logs');
      
      expect(await fs.stat(reportPath)).toBeDefined();
      expect(await fs.stat(outputLogPath)).toBeDefined();
      expect(await fs.stat(logsDir)).toBeDefined();
      
      // Verify test-run.md content
      const reportContent = await fs.readFile(reportPath, 'utf8');
      expect(reportContent).toContain('# 3pio Test Run');
      expect(reportContent).toContain('- Timestamp:');
      expect(reportContent).toContain('## Summary');
      expect(reportContent).toContain('Files Completed: 10');
      
      // Verify output.log exists and has header
      const outputLogContent = await fs.readFile(outputLogPath, 'utf8');
      expect(outputLogContent).toContain('# 3pio Test Output Log');
      expect(outputLogContent).toContain('# Timestamp:');
      expect(outputLogContent).toContain('# Command: concurrent test');
      expect(outputLogContent).toContain('# ---');
      
      // Check individual log files exist and have proper headers
      for (let i = 0; i < 10; i++) {
        const logPath = path.join(logsDir, `test${i}.js.log`);
        const logContent = await fs.readFile(logPath, 'utf8');
        expect(logContent).toContain(`# File: test${i}.js`);
        expect(logContent).toContain('# Timestamp:');
        expect(logContent).toContain('# This file contains all stdout/stderr output from the test file execution.');
        expect(logContent).toContain('# ---');
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
      
      // Create mock parser for error test
      const mockParser: OutputParser = {
        parseOutputIntoTestLogs: vi.fn().mockReturnValue(new Map([
          ['test.js', ['Test output']]
        ])),
        extractTestFileFromLine: vi.fn(),
        isEndOfTestOutput: vi.fn(),
        formatTestHeading: vi.fn()
      };
      
      const reportManager = new ReportManager(runId, 'error test', mockParser);
      
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
        eventType: 'testFileStart',
        payload: { filePath: 'test.js' }
      });
      
      await IPCManager.sendEvent({
        eventType: 'testFileResult',
        payload: { filePath: 'test.js', status: 'PASS' }
      });
      
      await new Promise(resolve => setTimeout(resolve, 200));
      
      // Should have processed only the valid events
      expect(eventCount).toBe(2);
      
      await reportManager.finalize(0);
      await ipcManager.cleanup();
      
      // Verify all required files exist despite errors
      const runDir = path.join(tempDir, '.3pio', 'runs', runId);
      const reportPath = path.join(runDir, 'test-run.md');
      const outputLogPath = path.join(runDir, 'output.log');
      const logsDir = path.join(runDir, 'logs');
      
      expect(await fs.stat(reportPath)).toBeDefined();
      expect(await fs.stat(outputLogPath)).toBeDefined();
      expect(await fs.stat(logsDir)).toBeDefined();
      
      // Verify test-run.md content
      const reportContent = await fs.readFile(reportPath, 'utf8');
      expect(reportContent).toContain('# 3pio Test Run');
      expect(reportContent).toContain('## Summary');
      expect(reportContent).toContain('Files Passed: 1');
      
      // Verify output.log has proper header
      const outputLogContent = await fs.readFile(outputLogPath, 'utf8');
      expect(outputLogContent).toContain('# 3pio Test Output Log');
      expect(outputLogContent).toContain('# ---');
      
      // Verify log file exists
      const logPath = path.join(logsDir, 'test.js.log');
      const logContent = await fs.readFile(logPath, 'utf8');
      expect(logContent).toContain('# File: test.js');
      expect(logContent).toContain('# ---');
    });
  });
});