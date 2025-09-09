import { promises as fs } from 'fs';
import path from 'path';
import debounce from 'lodash.debounce';
import { IPCEvent, TestRunState, TestCase } from './types/events';
import { OutputParser } from './runners/base/OutputParser';
import { Logger } from './utils/logger';

export class ReportManager {
  private runDirectory: string;
  private outputLogPath: string;
  private logsDirectory: string;
  private testRunPath: string;
  private state: TestRunState;
  private outputLogHandle: fs.FileHandle | null = null;
  private debouncedWrite: (() => void) & { cancel: () => void };
  private outputParser: OutputParser;
  private logger: Logger;
  private testFileOutputs: Map<string, string[]> = new Map();
  private testCaseOutputs: Map<string, Map<string, string[]>> = new Map();
  private currentTestCase: Map<string, string> = new Map();

  constructor(runId: string, testCommand: string, outputParser: OutputParser) {
    this.logger = Logger.create('report-manager');
    this.runDirectory = path.join(process.cwd(), '.3pio', 'runs', runId);
    this.outputLogPath = path.join(this.runDirectory, 'output.log');
    this.logsDirectory = path.join(this.runDirectory, 'logs');
    this.testRunPath = path.join(this.runDirectory, 'test-run.md');
    this.outputParser = outputParser;
    
    this.logger.info('ReportManager initialized', {
      runId,
      runDirectory: this.runDirectory,
      outputLogPath: this.outputLogPath,
      testRunPath: this.testRunPath
    });

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
      this.logger.debug('Debounced write triggered');
      this.writeTestRunReport().catch(error => {
        this.logger.error('Failed to write test run report', error);
        console.error(error);
      });
    }, 250, { maxWait: 1000 }) as (() => void) & { cancel: () => void };
  }

  /**
   * Initialize the report with list of test files
   */
  async initialize(testFiles: string[]): Promise<void> {
    this.logger.lifecycle('Initializing report structure', { testFiles: testFiles.length });
    
    // Create directories
    this.logger.debug('Creating report directories');
    await fs.mkdir(this.runDirectory, { recursive: true });

    // Initialize state with pending test files
    this.logger.info(`Initializing state with ${testFiles.length} test files`);
    this.state.totalFiles = testFiles.length;
    this.state.testFiles = testFiles.map(filePath => {
      // Normalize the file path by removing leading ./ and resolving to absolute
      const normalizedPath = path.resolve(filePath.replace(/^\.\//, ''));
      
      // Convert absolute path to relative for display
      const relativePath = path.relative(process.cwd(), normalizedPath);

      return {
        status: 'PENDING' as const,
        file: normalizedPath,  // Store the normalized path for consistent comparison
        logFile: `./logs/${this.sanitizeFilePath(relativePath)}.log`,
        testCases: []
      };
    });

    // Create output.log with header
    const header = [
      `# 3pio Test Output Log`,
      `# Timestamp: ${new Date().toISOString()}`,
      `# Command: ${this.state.arguments}`,
      `# This file contains all stdout/stderr output from the test run.`,
      '# ---',
      ''
    ].join('\n');

    this.logger.debug('Creating output.log file with header');
    this.outputLogHandle = await fs.open(this.outputLogPath, 'w');
    await this.outputLogHandle.writeFile(header);

    // Write initial report
    this.logger.info('Writing initial test run report');
    await this.writeTestRunReport();
    
    this.logger.lifecycle('Report initialization complete');
  }

  /**
   * Handle incoming IPC events
   */
  async handleEvent(event: IPCEvent): Promise<void> {
    this.logger.debug('Handling IPC event', { type: event.eventType, file: event.payload?.filePath });
    
    switch (event.eventType) {
      case 'stdoutChunk':
      case 'stderrChunk':
        await this.appendToLogFile(
          event.payload.filePath,
          event.payload.chunk,
          event.eventType === 'stderrChunk'
        );
        break;

      case 'testFileStart':
        this.logger.testFlow('Test file starting', event.payload.filePath);
        // Update test file status to RUNNING
        const normalizedPath = path.resolve(event.payload.filePath.replace(/^\.\//, ''));
        let testFile = this.state.testFiles.find(tf => {
          const normalizedStoredPath = path.resolve(tf.file.replace(/^\.\//, ''));
          return normalizedStoredPath === normalizedPath;
        });
        
        if (testFile) {
          testFile.status = 'RUNNING';
          this.state.updatedAt = new Date().toISOString();
          this.debouncedWrite();
        }
        break;
      case 'testCase':
        this.logger.testFlow('Test case event', event.payload.testName, { status: event.payload.status });
        await this.handleTestCaseEvent(event.payload);
        break;

      case 'testFileResult':
        this.logger.testFlow('Test file completed', event.payload.filePath, { status: event.payload.status });
        await this.updateTestFileStatus(
          event.payload.filePath,
          event.payload.status
        );
        break;
        
      default:
        this.logger.debug('Ignoring unknown event type', { type: (event as any).eventType });
    }
  }

  /**
   * Append output directly to output.log without creating individual log files
   * Used for non-test-specific output like startup messages, warnings, etc.
   */
  async appendToOutputLog(chunk: string): Promise<void> {
    if (this.outputLogHandle) {
      await this.outputLogHandle.appendFile(chunk);
    } else {
      this.logger.warn('Output log handle not available, dropping chunk');
    }
  }

  /**
   * Handle test case events
   */
  private async handleTestCaseEvent(payload: {
    filePath: string;
    testName: string;
    suiteName?: string;
    status: 'PASS' | 'FAIL' | 'SKIP' | 'PENDING' | 'RUNNING';
    duration?: number;
    error?: string;
  }): Promise<void> {
    const normalizedPath = path.resolve(payload.filePath.replace(/^\.\//, ''));
    
    let testFile = this.state.testFiles.find(tf => {
      const normalizedStoredPath = path.resolve(tf.file.replace(/^\.\//, ''));
      return normalizedStoredPath === normalizedPath;
    });

    if (!testFile) {
      // Add file dynamically if not found
      const relativePath = path.relative(process.cwd(), normalizedPath);
      testFile = {
        file: normalizedPath,
        logFile: `./logs/${this.sanitizeFilePath(relativePath)}.log`,
        status: 'PENDING',
        testCases: []
      };
      this.state.testFiles.push(testFile);
      this.state.totalFiles++;
    }

    if (!testFile.testCases) {
      testFile.testCases = [];
    }

    // Find or create test case
    const testFullName = payload.suiteName 
      ? `${payload.suiteName} › ${payload.testName}`
      : payload.testName;
    
    let testCase = testFile.testCases.find(tc => tc.name === testFullName);
    
    if (!testCase) {
      testCase = {
        name: testFullName,
        suite: payload.suiteName,
        status: payload.status,
        duration: payload.duration,
        error: payload.error
      };
      testFile.testCases.push(testCase);
    } else {
      // Update existing test case
      testCase.status = payload.status;
      if (payload.duration !== undefined) testCase.duration = payload.duration;
      if (payload.error !== undefined) testCase.error = payload.error;
    }

    // Track current test case for output demarcation
    if (payload.status === 'RUNNING') {
      this.currentTestCase.set(payload.filePath, testFullName);
    } else if (payload.status !== 'PENDING') {
      this.currentTestCase.delete(payload.filePath);
    }

    // Update timestamp and trigger debounced write
    this.state.updatedAt = new Date().toISOString();
    this.debouncedWrite();
  }

  /**
   * Append output chunk to the unified output log and collect per-file output
   */
  private async appendToLogFile(
    filePath: string,
    chunk: string,
    isStderr: boolean = false
  ): Promise<void> {
    // Append to the single output.log file
    if (this.outputLogHandle) {
      const chunkSize = chunk.length;
      this.logger.debug('Appending output chunk', { 
        file: filePath, 
        isStderr, 
        size: chunkSize 
      });
      await this.outputLogHandle.appendFile(chunk);
    } else {
      this.logger.warn('Output log handle not available, dropping chunk', { file: filePath });
    }
    
    // Collect per-file output for individual log files
    const fileName = path.basename(filePath);
    if (!this.testFileOutputs.has(fileName)) {
      this.testFileOutputs.set(fileName, []);
    }
    
    // If we have a current test case, also track output per test case
    const currentTest = this.currentTestCase.get(filePath);
    if (currentTest) {
      if (!this.testCaseOutputs.has(fileName)) {
        this.testCaseOutputs.set(fileName, new Map());
      }
      const testOutputs = this.testCaseOutputs.get(fileName)!;
      if (!testOutputs.has(currentTest)) {
        testOutputs.set(currentTest, []);
      }
      const prefix = isStderr ? '[stderr] ' : '';
      testOutputs.get(currentTest)!.push(prefix + chunk.trimEnd());
    }
    
    // Add chunk to the file's output buffer
    const prefix = isStderr ? '[stderr] ' : '';
    this.testFileOutputs.get(fileName)!.push(prefix + chunk.trimEnd());
  }

  /**
   * Update test file status and trigger debounced report write
   */
  private async updateTestFileStatus(
    filePath: string,
    status: 'PASS' | 'FAIL' | 'SKIP'
  ): Promise<void> {
    // Normalize the file path for comparison
    // Remove leading ./ and resolve to absolute path for consistent comparison
    const normalizedPath = path.resolve(filePath.replace(/^\.\//, ''));
    
    let testFile = this.state.testFiles.find(tf => {
      // Normalize the stored path the same way for comparison
      const normalizedStoredPath = path.resolve(tf.file.replace(/^\.\//, ''));
      return normalizedStoredPath === normalizedPath;
    });

    // If file wasn't in the initial list (e.g., Vitest couldn't do dry run),
    // add it dynamically
    if (!testFile) {
      this.logger.info('Adding dynamically discovered test file', { file: filePath });
      const relativePath = filePath.startsWith(process.cwd())
        ? path.relative(process.cwd(), filePath)
        : filePath;

      testFile = {
        file: filePath,
        logFile: `./logs/${this.sanitizeFilePath(relativePath)}.log`,
        status: 'PENDING',
        testCases: []
      };
      this.state.testFiles.push(testFile);
      this.state.totalFiles++;
    }

    // Update status
    const previousStatus = testFile.status;
    testFile.status = status;
    this.logger.debug('Test file status updated', { 
      file: filePath, 
      previousStatus, 
      newStatus: status 
    });

    // Only update counters if this is a new completion (not already in a terminal state)
    const wasCompleted = previousStatus === 'PASS' || previousStatus === 'FAIL' || previousStatus === 'SKIP';
    const isNowCompleted = status === 'PASS' || status === 'FAIL' || status === 'SKIP';
    
    if (!wasCompleted && isNowCompleted) {
      // This is a new completion
      this.state.filesCompleted++;
      
      if (status === 'PASS') this.state.filesPassed++;
      else if (status === 'FAIL') this.state.filesFailed++;
      else if (status === 'SKIP') this.state.filesSkipped++;
    } else if (wasCompleted && isNowCompleted && previousStatus !== status) {
      // Status changed from one terminal state to another (e.g., FAIL -> PASS)
      // Adjust the counters
      if (previousStatus === 'PASS') this.state.filesPassed--;
      else if (previousStatus === 'FAIL') this.state.filesFailed--;
      else if (previousStatus === 'SKIP') this.state.filesSkipped--;
      
      if (status === 'PASS') this.state.filesPassed++;
      else if (status === 'FAIL') this.state.filesFailed++;
      else if (status === 'SKIP') this.state.filesSkipped++;
    }
    
    this.logger.debug('Test run progress', {
      completed: this.state.filesCompleted,
      total: this.state.totalFiles,
      passed: this.state.filesPassed,
      failed: this.state.filesFailed,
      skipped: this.state.filesSkipped
    });

    // Update timestamp
    this.state.updatedAt = new Date().toISOString();

    // Trigger debounced write
    this.debouncedWrite();
  }

  /**
   * Write the test-run.md report file
   */
  private async writeTestRunReport(): Promise<void> {
    this.logger.debug('Writing test run report to', { path: this.testRunPath });

    // Convert absolute paths to relative paths for display
    const getRelativePath = (filePath: string): string => {
      const cwd = process.cwd();
      if (filePath.startsWith(cwd)) {
        return path.relative(cwd, filePath);
      }
      return filePath;
    };

    const formatStatus = (status: string): string => {
      const statusMap: Record<string, string> = {
        'PASS': '✓',
        'FAIL': '✕',
        'SKIP': '○',
        'PENDING': '⋯',
        'RUNNING': '▶'
      };
      return statusMap[status] || status;
    };

    const formatDuration = (ms?: number): string => {
      if (!ms) return '';
      if (ms < 1000) return `(${ms} ms)`;
      return `(${(ms / 1000).toFixed(2)} s)`;
    };

    // Build test results sections
    const testFileSections: string[] = [];
    
    for (const tf of this.state.testFiles) {
      const relativePath = getRelativePath(tf.file);
      const lines: string[] = [];
      
      lines.push(`## ${relativePath}`);
      lines.push(`Status: **${tf.status}** | [Log](${tf.logFile})`);
      lines.push('');
      
      if (tf.testCases && tf.testCases.length > 0) {
        // Group test cases by suite
        const suites = new Map<string | undefined, TestCase[]>();
        for (const tc of tf.testCases) {
          const suite = tc.suite || '';
          if (!suites.has(suite)) {
            suites.set(suite, []);
          }
          suites.get(suite)!.push(tc);
        }
        
        // Render test cases
        for (const [suite, tests] of suites) {
          if (suite) {
            lines.push(`### ${suite}`);
          }
          
          for (const test of tests) {
            const status = formatStatus(test.status);
            const duration = formatDuration(test.duration);
            const testName = suite ? test.name.replace(`${suite} › `, '') : test.name;
            lines.push(`  ${status} ${testName} ${duration}`);
            
            if (test.error) {
              // Indent error message
              const errorLines = test.error.split('\n');
              for (const errorLine of errorLines) {
                lines.push(`      ${errorLine}`);
              }
            }
          }
          lines.push('');
        }
      } else if (tf.status === 'RUNNING') {
        lines.push('  ▶ Running...');
        lines.push('');
      } else if (tf.status === 'PENDING') {
        lines.push('  ⋯ Pending');
        lines.push('');
      }
      
      testFileSections.push(lines.join('\n'));
    }

    const markdown = [
      '# 3pio Test Run',
      '',
      `- Timestamp: ${this.state.timestamp}`,
      `- Status: ${this.state.status} (updated ${this.state.updatedAt})`,
      `- Arguments: \`${this.state.arguments}\``,
      '',
      `## Summary`,
      `- Total Files: ${this.state.totalFiles}`,
      `- Files Completed: ${this.state.filesCompleted}`,
      `- Files Passed: ${this.state.filesPassed}`,
      `- Files Failed: ${this.state.filesFailed}`,
      `- Files Skipped: ${this.state.filesSkipped}`,
      '',
      '---',
      '',
      ...testFileSections,
      '---',
      '',
      `Full test output: [output.log](./output.log)`
    ].join('\n');

    await fs.writeFile(this.testRunPath, markdown);
    this.logger.debug('Test run report written successfully', { size: markdown.length });
  }

  /**
   * Get the current summary statistics
   */
  getSummary(): { 
    totalFiles: number; 
    filesCompleted: number;
    filesPassed: number;
    filesFailed: number;
    filesSkipped: number;
  } {
    const summary = {
      totalFiles: this.state.totalFiles,
      filesCompleted: this.state.filesCompleted,
      filesPassed: this.state.filesPassed,
      filesFailed: this.state.filesFailed,
      filesSkipped: this.state.filesSkipped
    };
    this.logger.debug('Summary requested', summary);
    return summary;
  }

  /**
   * Finalize the report when test run completes
   */
  async finalize(exitCode: number): Promise<void> {
    this.logger.lifecycle('Finalizing report', { exitCode });
    
    // Close the output log handle
    if (this.outputLogHandle) {
      this.logger.debug('Closing output log file handle');
      await this.outputLogHandle.close();
      this.outputLogHandle = null;
    }

    // Parse output.log into individual test file logs
    this.logger.info('Parsing output into individual test logs');
    await this.parseOutputIntoTestLogs();

    // Update state
    // Only set ERROR if the test runner itself had an error (exit codes like 127, etc.)
    // Normal test failures should still be COMPLETE status
    // Common exit codes: 0 = success, 1 = test failures, 127 = command not found, etc.
    this.state.status = 'COMPLETE';
    this.state.updatedAt = new Date().toISOString();
    
    this.logger.info('Test run completed', {
      exitCode,
      status: this.state.status,
      totalFiles: this.state.totalFiles,
      filesCompleted: this.state.filesCompleted,
      filesPassed: this.state.filesPassed,
      filesFailed: this.state.filesFailed,
      filesSkipped: this.state.filesSkipped
    });

    // Cancel any pending debounced writes and do final write
    this.logger.debug('Canceling pending debounced writes and writing final report');
    this.debouncedWrite.cancel();
    await this.writeTestRunReport();
    
    this.logger.lifecycle('Report finalization complete');
  }

  /**
   * Parse output.log into individual test file logs using the pluggable parser
   */
  private async parseOutputIntoTestLogs(): Promise<void> {
    try {
      this.logger.debug('Creating logs directory', { path: this.logsDirectory });
      // Create logs directory
      await fs.mkdir(this.logsDirectory, { recursive: true });

      // Use collected output from IPC events instead of parsing
      this.logger.debug('Writing individual log files from collected output');
      this.logger.info(`Collected output for ${this.testFileOutputs.size} test files`);

      // Create a set of files that have been written
      const writtenFiles = new Set<string>();

      // Write individual log files with test case demarcation
      for (const [fileName, outputLines] of this.testFileOutputs) {
        const sanitizedName = this.sanitizeFilePath(fileName);
        const logPath = path.join(this.logsDirectory, `${sanitizedName}.log`);
        this.logger.debug('Writing test log file', { fileName, logPath, lines: outputLines.length });

        const lines: string[] = [];
        lines.push(`# File: ${fileName}`);
        lines.push(`# Timestamp: ${new Date().toISOString()}`);
        lines.push(`# This file contains all stdout/stderr output from the test file execution.`);
        lines.push('# ---');
        lines.push('');

        // If we have test case outputs, organize by test case
        const testCaseOutputs = this.testCaseOutputs.get(fileName);
        if (testCaseOutputs && testCaseOutputs.size > 0) {
          lines.push('## Test Case Output');
          lines.push('');
          
          for (const [testName, testOutput] of testCaseOutputs) {
            lines.push(`### ${testName}`);
            lines.push('');
            if (testOutput.length > 0) {
              lines.push(...testOutput);
            } else {
              lines.push('(no output)');
            }
            lines.push('');
            lines.push('---');
            lines.push('');
          }
          
          // Add any remaining output not attributed to specific tests
          const remainingOutput = outputLines.filter(line => {
            // Check if this line was already included in test-specific output
            for (const testOutput of testCaseOutputs.values()) {
              if (testOutput.includes(line)) return false;
            }
            return true;
          });
          
          if (remainingOutput.length > 0) {
            lines.push('## Other Output');
            lines.push('');
            lines.push(...remainingOutput);
            lines.push('');
          }
        } else {
          // No test case demarcation, write all output
          if (outputLines.length > 0) {
            lines.push(...outputLines);
          } else {
            lines.push('# No output captured for this test file.');
          }
          lines.push('');
        }

        const content = lines.join('\n');
        await fs.writeFile(logPath, content);
        writtenFiles.add(fileName);
      }

      // Ensure log files exist for all test files, even if no output was captured
      // This is important for Jest which runs tests in worker processes where
      // output isn't captured by the reporter
      for (const testFile of this.state.testFiles) {
        const fileName = path.basename(testFile.file);
        if (!writtenFiles.has(fileName)) {
          const sanitizedName = this.sanitizeFilePath(fileName);
          const logPath = path.join(this.logsDirectory, `${sanitizedName}.log`);
          this.logger.debug('Creating empty log file for test without captured output', { fileName, logPath });

          const header = [
            `# File: ${fileName}`,
            `# Timestamp: ${new Date().toISOString()}`,
            `# This file contains all stdout/stderr output from the test file execution.`,
            '# ---',
            '',
            '# No output captured for this test file.',
            ''
          ].join('\n');

          await fs.writeFile(logPath, header);
        }
      }
      this.logger.debug('All test log files written successfully');
    } catch (error) {
      // If parsing fails, it's not critical - we still have output.log
      this.logger.error('Failed to parse output into individual logs', error as Error);
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
