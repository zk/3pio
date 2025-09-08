import { promises as fs } from 'fs';
import path from 'path';
import debounce from 'lodash.debounce';
import { IPCEvent, TestRunState } from './types/events';

export class ReportManager {
  private runDirectory: string;
  private logsDirectory: string;
  private testRunPath: string;
  private state: TestRunState;
  private logFileHandles: Map<string, fs.FileHandle> = new Map();
  private debouncedWrite: () => void;

  constructor(runId: string, testCommand: string) {
    this.runDirectory = path.join(process.cwd(), '.3pio', 'runs', runId);
    this.logsDirectory = path.join(this.runDirectory, 'logs');
    this.testRunPath = path.join(this.runDirectory, 'test-run.md');

    this.state = {
      timestamp: runId,
      status: 'RUNNING',
      updatedAt: new Date().toISOString(),
      arguments: testCommand,
      totalFiles: 0,
      filesCompleted: 0,
      filesPassed: 0,
      filesFailed: 0,
      filesSkipped: 0,
      testFiles: []
    };

    // Debounced write function - batches updates every 250ms
    this.debouncedWrite = debounce(() => {
      this.writeTestRunReport().catch(console.error);
    }, 250, { maxWait: 1000 });
  }

  /**
   * Initialize the report with list of test files
   */
  async initialize(testFiles: string[]): Promise<void> {
    // Create directories
    await fs.mkdir(this.runDirectory, { recursive: true });
    await fs.mkdir(this.logsDirectory, { recursive: true });

    // Initialize state with pending test files
    this.state.totalFiles = testFiles.length;
    this.state.testFiles = testFiles.map(filePath => {
      // Convert absolute path to relative for display
      const relativePath = filePath.startsWith(process.cwd()) 
        ? path.relative(process.cwd(), filePath) 
        : filePath;
      
      return {
        status: 'PENDING' as const,
        file: filePath,
        logFile: `./logs/${this.sanitizeFilePath(relativePath)}.log`
      };
    });

    // Write initial report
    await this.writeTestRunReport();
  }

  /**
   * Handle incoming IPC events
   */
  async handleEvent(event: IPCEvent): Promise<void> {
    switch (event.eventType) {
      case 'stdoutChunk':
      case 'stderrChunk':
        await this.appendToLogFile(
          event.payload.filePath,
          event.payload.chunk,
          event.eventType === 'stderrChunk'
        );
        break;

      case 'testFileResult':
        await this.updateTestFileStatus(
          event.payload.filePath,
          event.payload.status
        );
        break;
    }
  }

  /**
   * Append output chunk to log file
   */
  private async appendToLogFile(
    filePath: string,
    chunk: string,
    isStderr: boolean = false
  ): Promise<void> {
    // Convert absolute path to relative for sanitization
    const relativePath = filePath.startsWith(process.cwd()) 
      ? path.relative(process.cwd(), filePath) 
      : filePath;
    
    const logPath = path.join(this.logsDirectory, `${this.sanitizeFilePath(relativePath)}.log`);
    
    // Create file with header if it doesn't exist
    if (!this.logFileHandles.has(filePath)) {
      // Convert absolute path to relative for display
      const relativePath = filePath.startsWith(process.cwd()) 
        ? path.relative(process.cwd(), filePath) 
        : filePath;
      
      const header = [
        `File: ${relativePath}`,
        `Timestamp: ${new Date().toISOString()}`,
        `This file represents output from a test run for the listed test file. See \`../test-run.md\`.`,
        '---',
        ''
      ].join('\n');

      const handle = await fs.open(logPath, 'w');
      await handle.writeFile(header);
      this.logFileHandles.set(filePath, handle);
    }

    // Append chunk to file
    const handle = this.logFileHandles.get(filePath)!;
    await handle.appendFile(chunk);
  }

  /**
   * Update test file status and trigger debounced report write
   */
  private async updateTestFileStatus(
    filePath: string,
    status: 'PASS' | 'FAIL' | 'SKIP'
  ): Promise<void> {
    let testFile = this.state.testFiles.find(tf => tf.file === filePath);
    
    // If file wasn't in the initial list (e.g., Vitest couldn't do dry run),
    // add it dynamically
    if (!testFile) {
      // Convert absolute path to relative for sanitization
      const relativePath = filePath.startsWith(process.cwd()) 
        ? path.relative(process.cwd(), filePath) 
        : filePath;
      
      const sanitizedPath = this.sanitizeFilePath(relativePath);
      testFile = {
        file: filePath,
        logFile: `./logs/${sanitizedPath}.log`,
        status: 'PENDING'
      };
      this.state.testFiles.push(testFile);
      this.state.totalFiles++;
    }

    // Update status
    testFile.status = status;
    
    // Update counters
    this.state.filesCompleted++;
    if (status === 'PASS') this.state.filesPassed++;
    else if (status === 'FAIL') this.state.filesFailed++;
    else if (status === 'SKIP') this.state.filesSkipped++;

    // Close log file handle
    const handle = this.logFileHandles.get(filePath);
    if (handle) {
      await handle.close();
      this.logFileHandles.delete(filePath);
    }

    // Update timestamp
    this.state.updatedAt = new Date().toISOString();

    // Trigger debounced write
    this.debouncedWrite();
  }

  /**
   * Write the test-run.md report file
   */
  private async writeTestRunReport(): Promise<void> {
    const getStatusEmoji = (status: string) => {
      switch (status) {
        case 'PASS': return '';
        case 'FAIL': return '';
        case 'SKIP': return '';
        case 'RUNNING': return '';
        default: return '';
      }
    };

    // Convert absolute paths to relative paths for display
    const getRelativePath = (filePath: string): string => {
      const cwd = process.cwd();
      if (filePath.startsWith(cwd)) {
        return path.relative(cwd, filePath);
      }
      return filePath;
    };

    const markdown = [
      '# 3pio Test Run Summary',
      '',
      `- Timestamp: ${this.state.timestamp}`,
      `- Status: ${this.state.status} (updated ${this.state.updatedAt})`,
      `- Arguments: \`${this.state.arguments}\``,
      '',
      `## Summary (updated ${this.state.updatedAt})`,
      `- Total Files: ${this.state.totalFiles}`,
      `- Files Completed: ${this.state.filesCompleted}`,
      `- Files Passed: ${this.state.filesPassed}`,
      `- Files Failed: ${this.state.filesFailed}`,
      `- Files Skipped: ${this.state.filesSkipped}`,
      '',
      '## Test Files',
      '| Status | File | Log File |',
      '| --- | --- | --- |',
      ...this.state.testFiles.map(tf => 
        `| ${tf.status} | \`${getRelativePath(tf.file)}\` | [details](${tf.logFile}) |`
      ),
      ''
    ].join('\n');

    await fs.writeFile(this.testRunPath, markdown);
  }

  /**
   * Get the current summary statistics
   */
  getSummary(): { totalFiles: number; filesCompleted: number } {
    return {
      totalFiles: this.state.totalFiles,
      filesCompleted: this.state.filesCompleted
    };
  }

  /**
   * Finalize the report when test run completes
   */
  async finalize(exitCode: number): Promise<void> {
    // Close any remaining file handles
    for (const [filePath, handle] of this.logFileHandles) {
      await handle.close();
    }
    this.logFileHandles.clear();

    // Update state
    // Only set ERROR if the test runner itself had an error (exit codes like 127, etc.)
    // Normal test failures should still be COMPLETE status
    // Common exit codes: 0 = success, 1 = test failures, 127 = command not found, etc.
    this.state.status = 'COMPLETE';
    this.state.updatedAt = new Date().toISOString();

    // Cancel any pending debounced writes and do final write
    this.debouncedWrite.cancel();
    await this.writeTestRunReport();
  }

  /**
   * Sanitize file path for use as filename
   */
  private sanitizeFilePath(filePath: string): string {
    return filePath
      .replace(/\//g, '_')
      .replace(/\\/g, '_')
      .replace(/:/g, '_')
      .replace(/\*/g, '_')
      .replace(/\?/g, '_')
      .replace(/"/g, '_')
      .replace(/</g, '_')
      .replace(/>/g, '_')
      .replace(/\|/g, '_')
      .replace(/\s+/g, '_');
  }

  /**
   * Get the report path for the preamble
   */
  getReportPath(): string {
    return path.relative(process.cwd(), this.testRunPath);
  }
}