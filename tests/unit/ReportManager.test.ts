import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { ReportManager } from '../../src/ReportManager';
import { OutputParser } from '../../src/runners/base/OutputParser';
import { promises as fs } from 'fs';
import { existsSync } from 'fs';
import path from 'path';
import os from 'os';

describe('ReportManager', () => {
  let tempDir: string;
  let runId: string;
  let reportManager: ReportManager;
  let originalCwd: () => string;
  let mockParser: OutputParser;

  beforeEach(async () => {
    // Save original cwd function
    originalCwd = process.cwd;
    
    // Create a temporary directory for testing
    tempDir = await fs.mkdtemp(path.join(os.tmpdir(), '3pio-test-'));
    
    // Mock process.cwd to return our temp directory
    vi.spyOn(process, 'cwd').mockReturnValue(tempDir);
    
    runId = '20240101T120000Z';
    
    // Create a mock output parser
    mockParser = {
      parseOutputIntoTestLogs: vi.fn().mockReturnValue(new Map([
        ['test.js', ['# Test output', 'console.log output', '# Test passed']]
      ])),
      extractTestFileFromLine: vi.fn(),
      isEndOfTestOutput: vi.fn(),
      formatTestHeading: vi.fn()
    };
    
    reportManager = new ReportManager(runId, 'npm test', mockParser);
  });

  afterEach(async () => {
    // Finalize report manager
    if (reportManager) {
      await reportManager.finalize(0);
    }
    
    // Restore mocks
    vi.restoreAllMocks();
    
    // Clean up temp directory
    await fs.rm(tempDir, { recursive: true, force: true });
  });

  describe('initialize', () => {
    it('should create directory structure and initial report', async () => {
      const testFiles = ['src/test1.spec.js', 'src/test2.spec.js'];
      
      await reportManager.initialize(testFiles);
      
      // Check directories were created
      const runDir = path.join(tempDir, '.3pio', 'runs', runId);
      const logsDir = path.join(runDir, 'logs');
      const reportPath = path.join(runDir, 'test-run.md');
      
      const runDirStats = await fs.stat(runDir);
      expect(runDirStats.isDirectory()).toBe(true);
      
      // Logs directory is now created during initialize()
      const logsDirStats = await fs.stat(logsDir);
      expect(logsDirStats.isDirectory()).toBe(true);
      
      // Check initial report was created
      const reportContent = await fs.readFile(reportPath, 'utf8');
      expect(reportContent).toContain('# 3pio Test Run');
      expect(reportContent).toContain('npm test');
      expect(reportContent).toContain('src/test1.spec.js');
      expect(reportContent).toContain('src/test2.spec.js');
      expect(reportContent).toContain('PENDING');
    });

    it('should NOT create log files immediately for known test files', async () => {
      const testFiles = ['src/test1.spec.js', 'src/test2.spec.js'];
      
      await reportManager.initialize(testFiles);
      
      // Check that log files were NOT created yet
      const logsDir = path.join(tempDir, '.3pio', 'runs', runId, 'logs');
      const log1Path = path.join(logsDir, 'src_test1.spec.js.log');
      const log2Path = path.join(logsDir, 'src_test2.spec.js.log');
      
      // Files should not exist yet
      expect(existsSync(log1Path)).toBe(false);
      expect(existsSync(log2Path)).toBe(false);
    });

    it('should create log files when test file starts', async () => {
      const testFiles = ['src/test1.spec.js'];
      
      await reportManager.initialize(testFiles);
      
      // Simulate test file starting
      await reportManager.handleEvent({
        eventType: 'testFileStart',
        payload: {
          filePath: 'src/test1.spec.js'
        }
      });
      
      // Now the log file should exist
      const logsDir = path.join(tempDir, '.3pio', 'runs', runId, 'logs');
      const log1Path = path.join(logsDir, 'src_test1.spec.js.log');
      
      const log1Stats = await fs.stat(log1Path);
      expect(log1Stats.isFile()).toBe(true);
      
      // Check that headers were written
      const log1Content = await fs.readFile(log1Path, 'utf8');
      expect(log1Content).toContain('# File: src/test1.spec.js');
      expect(log1Content).toContain('# This file contains all stdout/stderr output');
    });

    it('should sanitize file paths for log files', async () => {
      const testFiles = ['src/path/with spaces/test.js', 'src\\windows\\path\\test.js'];
      
      await reportManager.initialize(testFiles);
      
      const reportPath = path.join(tempDir, '.3pio', 'runs', runId, 'test-run.md');
      const reportContent = await fs.readFile(reportPath, 'utf8');
      
      // Check that log file paths are sanitized
      expect(reportContent).toContain('src_path_with_spaces_test.js.log');
      expect(reportContent).toContain('src_windows_path_test.js.log');
    });
  });

  describe('handleEvent', () => {
    beforeEach(async () => {
      await reportManager.initialize(['test.js']);
    });

    it('should handle stdoutChunk event', async () => {
      const event = {
        eventType: 'stdoutChunk' as const,
        payload: {
          filePath: 'test.js',
          chunk: 'Test output line 1\n'
        }
      };
      
      await reportManager.handleEvent(event);
      
      // Check that content is written to output.log
      const outputLogPath = path.join(tempDir, '.3pio', 'runs', runId, 'output.log');
      const outputContent = await fs.readFile(outputLogPath, 'utf8');
      
      expect(outputContent).toContain('Test output line 1');
    });

    it('should handle stderrChunk event', async () => {
      const event = {
        eventType: 'stderrChunk' as const,
        payload: {
          filePath: 'test.js',
          chunk: 'Error output line 1\n'
        }
      };
      
      await reportManager.handleEvent(event);
      
      // Check that content is written to output.log
      const outputLogPath = path.join(tempDir, '.3pio', 'runs', runId, 'output.log');
      const outputContent = await fs.readFile(outputLogPath, 'utf8');
      
      expect(outputContent).toContain('Error output line 1');
    });

    it('should handle testFileResult event with PASS status', async () => {
      const event = {
        eventType: 'testFileResult' as const,
        payload: {
          filePath: 'test.js',
          status: 'PASS' as const
        }
      };
      
      await reportManager.handleEvent(event);
      
      // Wait for debounced write
      await new Promise(resolve => setTimeout(resolve, 300));
      
      const reportPath = path.join(tempDir, '.3pio', 'runs', runId, 'test-run.md');
      const reportContent = await fs.readFile(reportPath, 'utf8');
      
      expect(reportContent).toContain('Status: **PASS**');
      expect(reportContent).toContain('Files Passed: 1');
      expect(reportContent).toContain('Files Completed: 1');
    });

    it('should handle testFileResult event with FAIL status', async () => {
      const event = {
        eventType: 'testFileResult' as const,
        payload: {
          filePath: 'test.js',
          status: 'FAIL' as const
        }
      };
      
      await reportManager.handleEvent(event);
      
      // Wait for debounced write
      await new Promise(resolve => setTimeout(resolve, 300));
      
      const reportPath = path.join(tempDir, '.3pio', 'runs', runId, 'test-run.md');
      const reportContent = await fs.readFile(reportPath, 'utf8');
      
      expect(reportContent).toContain('Status: **FAIL**');
      expect(reportContent).toContain('Files Failed: 1');
    });

    it('should handle testFileResult event with SKIP status', async () => {
      const event = {
        eventType: 'testFileResult' as const,
        payload: {
          filePath: 'test.js',
          status: 'SKIP' as const
        }
      };
      
      await reportManager.handleEvent(event);
      
      // Wait for debounced write
      await new Promise(resolve => setTimeout(resolve, 300));
      
      const reportPath = path.join(tempDir, '.3pio', 'runs', runId, 'test-run.md');
      const reportContent = await fs.readFile(reportPath, 'utf8');
      
      expect(reportContent).toContain('Status: **SKIP**');
      expect(reportContent).toContain('Files Skipped: 1');
    });
  });

  describe('appendToLogFile', () => {
    beforeEach(async () => {
      await reportManager.initialize(['test.js']);
    });

    it('should append to output.log file', async () => {
      await (reportManager as any).appendToLogFile('test.js', 'First chunk\n');
      
      const outputLogPath = path.join(tempDir, '.3pio', 'runs', runId, 'output.log');
      const content = await fs.readFile(outputLogPath, 'utf8');
      
      expect(content).toContain('First chunk');
    });

    it('should append subsequent chunks to output.log', async () => {
      await (reportManager as any).appendToLogFile('test.js', 'First chunk\n');
      await (reportManager as any).appendToLogFile('test.js', 'Second chunk\n');
      
      const outputLogPath = path.join(tempDir, '.3pio', 'runs', runId, 'output.log');
      const content = await fs.readFile(outputLogPath, 'utf8');
      
      expect(content).toContain('First chunk');
      expect(content).toContain('Second chunk');
    });

    it('should buffer output for incremental writing to test logs', async () => {
      // First start the test file to create the log
      await reportManager.handleEvent({
        eventType: 'testFileStart',
        payload: { filePath: 'test.js' }
      });
      
      await (reportManager as any).appendToLogFile('test.js', 'First chunk\n');
      await (reportManager as any).appendToLogFile('test.js', 'Second chunk\n');
      
      // Check that data is in buffer
      const buffer = (reportManager as any).testFileBuffers.get('test.js');
      expect(buffer).toBeDefined();
      expect(buffer.length).toBeGreaterThan(0);
    });
  });

  describe('finalize', () => {
    beforeEach(async () => {
      await reportManager.initialize(['test1.js', 'test2.js']);
    });

    it('should set status to COMPLETE for exit code 0', async () => {
      await reportManager.finalize(0);
      
      const reportPath = path.join(tempDir, '.3pio', 'runs', runId, 'test-run.md');
      const content = await fs.readFile(reportPath, 'utf8');
      
      expect(content).toContain('Status: COMPLETE');
    });

    it('should set status to COMPLETE even for non-zero exit code (test failures)', async () => {
      await reportManager.finalize(1);
      
      const reportPath = path.join(tempDir, '.3pio', 'runs', runId, 'test-run.md');
      const content = await fs.readFile(reportPath, 'utf8');
      
      expect(content).toContain('Status: COMPLETE');
    });

    it('should close output log handle', async () => {
      // Write to output log
      await (reportManager as any).appendToLogFile('test1.js', 'chunk1');
      await (reportManager as any).appendToLogFile('test2.js', 'chunk2');
      
      // Verify handle exists
      const handle = (reportManager as any).outputLogHandle;
      expect(handle).toBeTruthy();
      
      await reportManager.finalize(0);
      
      // Handle should be closed
      expect((reportManager as any).outputLogHandle).toBeNull();
    });

    it('should flush all buffers and close all file handles', async () => {
      // Start the test files first
      await reportManager.handleEvent({
        eventType: 'testFileStart',
        payload: { filePath: 'test1.js' }
      });
      await reportManager.handleEvent({
        eventType: 'testFileStart',
        payload: { filePath: 'test2.js' }
      });
      
      // Add some data to buffers
      await (reportManager as any).appendToLogFile('test1.js', 'chunk1');
      await (reportManager as any).appendToLogFile('test2.js', 'chunk2');
      
      // Verify handles and buffers exist
      const handles = (reportManager as any).testFileHandles;
      const buffers = (reportManager as any).testFileBuffers;
      expect(handles.size).toBeGreaterThan(0);
      expect(buffers.size).toBeGreaterThan(0);
      
      await reportManager.finalize(0);
      
      // All handles and buffers should be cleared
      expect(handles.size).toBe(0);
      expect(buffers.size).toBe(0);
      
      // Check that data was written to log files
      const logsDir = path.join(tempDir, '.3pio', 'runs', runId, 'logs');
      const log1Path = path.join(logsDir, 'test1.js.log');
      const log1Content = await fs.readFile(log1Path, 'utf8');
      expect(log1Content).toContain('chunk1');
    });
  });

  describe('sanitizeFilePath', () => {
    it('should replace special characters with underscores', async () => {
      // Initialize to create directories
      await reportManager.initialize(['dummy.js']);
      
      const inputs = [
        'path/with/slashes.js',
        'path\\with\\backslashes.js',
        'path with spaces.js',
        'path:with:colons.js',
        'path*with?special<chars>.js',
        'path|with"pipes".js'
      ];
      
      const expected = [
        'path_with_slashes.js',
        'path_with_backslashes.js',
        'path_with_spaces.js',
        'path_with_colons.js',
        'path_with_special_chars_.js',
        'path_with_pipes_.js'
      ];
      
      inputs.forEach((input, index) => {
        const result = (reportManager as any).sanitizeFilePath(input);
        expect(result).toBe(expected[index]);
      });
    });
  });

  describe('getReportPath', () => {
    it('should return relative path from cwd', async () => {
      // Initialize to create directories
      await reportManager.initialize(['dummy.js']);
      
      const reportPath = reportManager.getReportPath();
      expect(reportPath).toBe(path.join('.3pio', 'runs', runId, 'test-run.md'));
    });
  });

  describe('Incremental Writing', () => {
    it('should write output incrementally with debouncing', async () => {
      vi.useFakeTimers();
      
      await reportManager.initialize(['test.js']);
      
      // First need to start the test file to create the log
      await reportManager.handleEvent({
        eventType: 'testFileStart',
        payload: { filePath: 'test.js' }
      });
      
      // Add some output
      await (reportManager as any).appendToLogFile('test.js', 'Line 1\n');
      await (reportManager as any).appendToLogFile('test.js', 'Line 2\n');
      
      // Check buffer has data
      const buffer = (reportManager as any).testFileBuffers.get('test.js');
      expect(buffer.length).toBeGreaterThan(0);
      
      // Advance timers to trigger debounced write
      vi.advanceTimersByTime(150);
      await vi.runAllTimersAsync();
      
      // Check that data was written to file
      const logsDir = path.join(tempDir, '.3pio', 'runs', runId, 'logs');
      const logPath = path.join(logsDir, 'test.js.log');
      const content = await fs.readFile(logPath, 'utf8');
      
      expect(content).toContain('Line 1');
      expect(content).toContain('Line 2');
      
      // Buffer should be cleared after write
      expect(buffer.length).toBe(0);
      
      vi.useRealTimers();
    });

    it('should handle file handle errors gracefully', async () => {
      await reportManager.initialize(['test.js']);
      
      // Start the test file first
      await reportManager.handleEvent({
        eventType: 'testFileStart',
        payload: { filePath: 'test.js' }
      });
      
      // Simulate file handle error by closing the handle
      const handle = (reportManager as any).testFileHandles.get('test.js');
      if (handle) {
        await handle.close();
      }
      
      // This should not throw
      await (reportManager as any).appendToLogFile('test.js', 'Test output\n');
      
      // Trigger flush
      await (reportManager as any).flushFileBuffer('test.js');
      
      // Should have logged error but not thrown
      expect(true).toBe(true);
    });

    it('should add test case boundaries in output', async () => {
      vi.useFakeTimers();
      
      await reportManager.initialize(['test.js']);
      
      // Start the test file first
      await reportManager.handleEvent({
        eventType: 'testFileStart',
        payload: { filePath: 'test.js' }
      });
      
      // Simulate test case starting
      await reportManager.handleEvent({
        eventType: 'testCase',
        payload: {
          filePath: 'test.js',
          testName: 'should work',
          status: 'RUNNING'
        }
      });
      
      // Add output during test
      await (reportManager as any).appendToLogFile('test.js', 'Test output\n');
      
      // Complete the test
      await reportManager.handleEvent({
        eventType: 'testCase',
        payload: {
          filePath: 'test.js',
          testName: 'should work',
          status: 'PASS',
          duration: 50
        }
      });
      
      // Advance timers to trigger write
      vi.advanceTimersByTime(150);
      await vi.runAllTimersAsync();
      
      // Check log file contains test boundaries
      const logsDir = path.join(tempDir, '.3pio', 'runs', runId, 'logs');
      const logPath = path.join(logsDir, 'test.js.log');
      const content = await fs.readFile(logPath, 'utf8');
      
      expect(content).toContain('### should work');
      expect(content).toContain('Test ✓ should work');
      expect(content).toContain('Duration: 50ms');
      expect(content).toContain('Test output');
      
      vi.useRealTimers();
    });
  });
});