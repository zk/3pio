import { promises as fs } from 'fs';
import path from 'path';
import debounce from 'lodash.debounce';
import { IPCEvent, TestRunState } from './types/events';

export class ReportManager {
  private runDirectory: string;
  private outputLogPath: string;
  private logsDirectory: string;
  private testRunPath: string;
  private state: TestRunState;
  private outputLogHandle: fs.FileHandle | null = null;
  private debouncedWrite: () => void;

  constructor(runId: string, testCommand: string) {
    this.runDirectory = path.join(process.cwd(), '.3pio', 'runs', runId);
    this.outputLogPath = path.join(this.runDirectory, 'output.log');
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

    // Create output.log with header
    const header = [
      `3pio Test Output Log`,
      `Timestamp: ${new Date().toISOString()}`,
      `Command: ${this.state.arguments}`,
      `This file contains all stdout/stderr output from the test run.`,
      '=' .repeat(80),
      ''
    ].join('\n');

    this.outputLogHandle = await fs.open(this.outputLogPath, 'w');
    await this.outputLogHandle.writeFile(header);

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
   * Append output chunk to the unified output log
   */
  private async appendToLogFile(
    filePath: string,
    chunk: string,
    isStderr: boolean = false
  ): Promise<void> {
    // Append to the single output.log file
    if (this.outputLogHandle) {
      await this.outputLogHandle.appendFile(chunk);
    }
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
      const relativePath = filePath.startsWith(process.cwd())
        ? path.relative(process.cwd(), filePath)
        : filePath;

      testFile = {
        file: filePath,
        logFile: `./logs/${this.sanitizeFilePath(relativePath)}.log`,
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
      '',
      '',
      `Full test output: [output.log](./output.log)`
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
    // Close the output log handle
    if (this.outputLogHandle) {
      await this.outputLogHandle.close();
      this.outputLogHandle = null;
    }

    // Parse output.log into individual test file logs
    await this.parseOutputIntoTestLogs();

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
   * Parse output.log into individual test file logs
   */
  private async parseOutputIntoTestLogs(): Promise<void> {
    try {
      // Create logs directory
      await fs.mkdir(this.logsDirectory, { recursive: true });

      // Read the output.log file
      const outputContent = await fs.readFile(this.outputLogPath, 'utf8');

      // Split by lines
      const lines = outputContent.split('\n');

      // Track current file being processed
      const fileOutputs = new Map<string, string[]>();
      let currentFile: string | null = null;
      let inHeader = true;

      for (const line of lines) {
        // Skip header lines (first 5 lines)
        if (inHeader) {
          if (line.startsWith('='.repeat(80))) {
            inHeader = false;
          }
          continue;
        }

        // Check if this line indicates output from a specific test file
        // Vitest format: "stdout | math.test.js > ..."
        // Jest format might be different
        const vitestMatch = line.match(/^(stdout|stderr) \| ([^>]+\.(?:test|spec)\.[jt]sx?) > /);
        if (vitestMatch) {
          currentFile = vitestMatch[2];
          if (!fileOutputs.has(currentFile)) {
            fileOutputs.set(currentFile, []);
          }
          // Don't push the header line itself, just extract content after the ">"
          const content = line.split(' > ').slice(1).join(' > ');
          if (content.trim()) {
            fileOutputs.get(currentFile)!.push(content);
          }
        } else if (currentFile && line.trim()) {
          // Continue adding lines to current file until we see a new file marker
          // Check if this is a summary line (starts with test runner symbols)
          if (line.match(/^\s*(✓|✔|×|✗|↓|⚠|❯|\[PASS\]|\[FAIL\]|\[SKIP\])/)) {
            currentFile = null; // Reset when we hit summary lines
          } else {
            fileOutputs.get(currentFile)!.push(line);
          }
        }
      }

      // Write individual log files
      for (const [fileName, outputLines] of fileOutputs) {
        const sanitizedName = this.sanitizeFilePath(fileName);
        const logPath = path.join(this.logsDirectory, `${sanitizedName}.log`);

        const header = [
          `File: ${fileName}`,
          `Timestamp: ${new Date().toISOString()}`,
          `This file contains all stdout/stderr output from the test file execution.`,
          '---',
          ''
        ].join('\n');

        const content = header + outputLines.join('\n') + '\n';
        await fs.writeFile(logPath, content);
      }
    } catch (error) {
      // If parsing fails, it's not critical - we still have output.log
      console.error('Warning: Failed to parse output into individual logs:', error);
    }
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
