import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { ReportManager } from '../../src/ReportManager';
import { OutputParser } from '../../src/runners/base/OutputParser';
import { promises as fs } from 'fs';
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
      const reportPath = path.join(runDir, 'test-run.md');
      
      const runDirStats = await fs.stat(runDir);
      expect(runDirStats.isDirectory()).toBe(true);
      
      // Note: logs directory is created during finalize(), not initialize()
      
      // Check initial report was created
      const reportContent = await fs.readFile(reportPath, 'utf8');
      expect(reportContent).toContain('# 3pio Test Run');
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