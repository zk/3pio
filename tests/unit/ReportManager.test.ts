import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { ReportManager } from '../../src/ReportManager';
import { promises as fs } from 'fs';
import path from 'path';
import os from 'os';

describe('ReportManager', () => {
  let tempDir: string;
  let runId: string;
  let reportManager: ReportManager;
  let originalCwd: () => string;

  beforeEach(async () => {
    // Save original cwd function
    originalCwd = process.cwd;
    
    // Create a temporary directory for testing
    tempDir = await fs.mkdtemp(path.join(os.tmpdir(), '3pio-test-'));
    
    // Mock process.cwd to return our temp directory
    vi.spyOn(process, 'cwd').mockReturnValue(tempDir);
    
    runId = '20240101T120000Z';
    reportManager = new ReportManager(runId, 'npm test');
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
      
      const logsDirStats = await fs.stat(logsDir);
      expect(logsDirStats.isDirectory()).toBe(true);
      
      // Check initial report was created
      const reportContent = await fs.readFile(reportPath, 'utf8');
      expect(reportContent).toContain('# 3pio Test Run Summary');
      expect(reportContent).toContain('npm test');
      expect(reportContent).toContain('src/test1.spec.js');
      expect(reportContent).toContain('src/test2.spec.js');
      expect(reportContent).toContain('PENDING');
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
      
      const logPath = path.join(tempDir, '.3pio', 'runs', runId, 'logs', 'test.js.log');
      const logContent = await fs.readFile(logPath, 'utf8');
      
      expect(logContent).toContain('File: test.js');
      expect(logContent).toContain('Test output line 1');
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
      
      const logPath = path.join(tempDir, '.3pio', 'runs', runId, 'logs', 'test.js.log');
      const logContent = await fs.readFile(logPath, 'utf8');
      
      expect(logContent).toContain('Error output line 1');
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
      
      expect(reportContent).toContain('✅ PASS');
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
      
      expect(reportContent).toContain('❌ FAIL');
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
      
      expect(reportContent).toContain('⏭️ SKIP');
      expect(reportContent).toContain('Files Skipped: 1');
    });
  });

  describe('appendToLogFile', () => {
    beforeEach(async () => {
      await reportManager.initialize(['test.js']);
    });

    it('should create log file with header on first write', async () => {
      await (reportManager as any).appendToLogFile('test.js', 'First chunk\n');
      
      const logPath = path.join(tempDir, '.3pio', 'runs', runId, 'logs', 'test.js.log');
      const content = await fs.readFile(logPath, 'utf8');
      
      expect(content).toContain('File: test.js');
      expect(content).toContain('Timestamp:');
      expect(content).toContain('This file represents output from a test run');
      expect(content).toContain('First chunk');
    });

    it('should append subsequent chunks without header', async () => {
      await (reportManager as any).appendToLogFile('test.js', 'First chunk\n');
      await (reportManager as any).appendToLogFile('test.js', 'Second chunk\n');
      
      const logPath = path.join(tempDir, '.3pio', 'runs', runId, 'logs', 'test.js.log');
      const content = await fs.readFile(logPath, 'utf8');
      
      expect(content).toContain('First chunk');
      expect(content).toContain('Second chunk');
      
      // Should only have one header
      const headerCount = (content.match(/File: test.js/g) || []).length;
      expect(headerCount).toBe(1);
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

    it('should close all open file handles', async () => {
      // Write to create file handles
      await (reportManager as any).appendToLogFile('test1.js', 'chunk1');
      await (reportManager as any).appendToLogFile('test2.js', 'chunk2');
      
      // Verify handles exist
      const handles = (reportManager as any).logFileHandles;
      expect(handles.size).toBe(2);
      
      await reportManager.finalize(0);
      
      // Handles should be cleared
      expect(handles.size).toBe(0);
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
});