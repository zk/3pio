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
  private currentTestCase: Map<string, string> = new Map();
  
  // File handle management for incremental writing
  private testFileHandles: Map<string, fs.FileHandle> = new Map();
  private testFileBuffers: Map<string, string[]> = new Map();
  private debouncedFileWrites: Map<string, (() => void) & { cancel: () => void }> = new Map();

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
   * Initialize the report with optional list of test files
   * If no test files provided, they will be discovered dynamically
   */
  async initialize(testFiles: string[] = []): Promise<void> {
    this.logger.lifecycle('Initializing report structure', { 
      testFiles: testFiles.length,
      mode: testFiles.length > 0 ? 'static' : 'dynamic'
    });
    
    // Create directories
    this.logger.debug('Creating report directories');
    await fs.mkdir(this.runDirectory, { recursive: true });
    
    // Create logs directory immediately for incremental writing
    this.logger.debug('Creating logs directory', { path: this.logsDirectory });
    await fs.mkdir(this.logsDirectory, { recursive: true });

    // Initialize state with pending test files (if any provided)
    if (testFiles.length > 0) {
      this.logger.info(`Initializing state with ${testFiles.length} known test files`);
    } else {
      this.logger.info('No test files provided, will discover dynamically');
    }
    this.state.totalFiles = testFiles.length;
    this.state.testFiles = [];
    
    // Process each test file (but don't create log files yet - wait until they start running)
    for (const filePath of testFiles) {
      // Normalize the file path by removing leading ./ and resolving to absolute
      const normalizedPath = path.resolve(filePath.replace(/^\.\//, ''));
      
      // Convert absolute path to relative for display
      const relativePath = path.relative(process.cwd(), normalizedPath);

      this.state.testFiles.push({
        status: 'PENDING' as const,
        file: normalizedPath,  // Store the normalized path for consistent comparison
        logFile: `./logs/${this.sanitizeFilePath(relativePath)}.log`,
        testCases: []
      });
      
      // Don't create log files here - wait until testFileStart event
    }

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
   * Ensure a test file is registered in the state
   * Called when we receive the first event for a file during dynamic discovery
   */
  private async ensureTestFileRegistered(filePath: string): Promise<void> {
    // Normalize the file path
    const normalizedPath = path.resolve(filePath.replace(/^\.\//, ''));
    
    // Check if already registered
    const exists = this.state.testFiles.some(tf => {
      const normalizedStoredPath = path.resolve(tf.file.replace(/^\.\//, ''));
      return normalizedStoredPath === normalizedPath;
    });
    
    if (!exists) {
      this.logger.info('Dynamically registering test file', { file: filePath });
      
      // Convert absolute path to relative for display
      const relativePath = path.relative(process.cwd(), normalizedPath);
      
      // Add to state
      this.state.testFiles.push({
        status: 'RUNNING',  // Start as RUNNING since we're getting events
        file: normalizedPath,
        logFile: `./logs/${this.sanitizeFilePath(relativePath)}.log`,
        testCases: []
      });
      
      // Update total count
      this.state.totalFiles = this.state.testFiles.length;
      
      // Don't create log file here - wait until testFileStart event
      
      // Trigger report update
      this.state.updatedAt = new Date().toISOString();
      this.debouncedWrite();
    }
  }

  /**
   * Handle incoming IPC events
   */
  async handleEvent(event: IPCEvent): Promise<void> {
    this.logger.debug('Handling IPC event', { type: event.eventType, file: event.payload?.filePath });
    
    switch (event.eventType) {
      case 'stdoutChunk':
      case 'stderrChunk':
        // Ensure file is registered (for dynamic discovery)
        if (event.payload.filePath) {
          await this.ensureTestFileRegistered(event.payload.filePath);
        }
        await this.appendToLogFile(
          event.payload.filePath,
          event.payload.chunk,
          event.eventType === 'stderrChunk'
        );
        break;

      case 'testFileStart':
        this.logger.testFlow('Test file starting', event.payload.filePath);
        // Ensure file is registered (for dynamic discovery)
        await this.ensureTestFileRegistered(event.payload.filePath);
        
        // Update test file status to RUNNING
        const normalizedPath = path.resolve(event.payload.filePath.replace(/^\.\//, ''));
        const relativePath = path.relative(process.cwd(), normalizedPath);
        
        let testFile = this.state.testFiles.find(tf => {
          const normalizedStoredPath = path.resolve(tf.file.replace(/^\.\//, ''));
          return normalizedStoredPath === normalizedPath;
        });
        
        if (testFile) {
          testFile.status = 'RUNNING';
          
          // Create the log file now that the test is actually starting
          await this.createLogFileForTest(relativePath);
          
          this.state.updatedAt = new Date().toISOString();
          this.debouncedWrite();
        }
        break;
      case 'testCase':
        this.logger.testFlow('Test case event', event.payload.testName, { status: event.payload.status });
        // Ensure file is registered (for dynamic discovery)
        if (event.payload.filePath) {
          await this.ensureTestFileRegistered(event.payload.filePath);
        }
        await this.handleTestCaseEvent(event.payload);
        break;

      case 'testFileResult':
        this.logger.testFlow('Test file completed', event.payload.filePath, { status: event.payload.status });
        // Ensure file is registered (for dynamic discovery)
        await this.ensureTestFileRegistered(event.payload.filePath);
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
      // Test case completed - add completion marker to buffer
      const normalizedPath = path.resolve(payload.filePath.replace(/^\.\//, ''));
      const relativePath = path.relative(process.cwd(), normalizedPath);
      const buffer = this.testFileBuffers.get(relativePath);
      
      if (buffer && this.currentTestCase.get(payload.filePath) === testFullName) {
        // Add test result marker
        const statusSymbol = payload.status === 'PASS' ? '✓' : 
                           payload.status === 'FAIL' ? '✕' :
                           payload.status === 'SKIP' ? '○' : '';
        if (statusSymbol) {
          buffer.push('');
          buffer.push(`Test ${statusSymbol} ${testFullName}`);
          if (payload.duration) {
            buffer.push(`Duration: ${payload.duration}ms`);
          }
          if (payload.error) {
            buffer.push('Error:');
            const errorLines = payload.error.split('\n');
            for (const line of errorLines) {
              buffer.push(`  ${line}`);
            }
          }
          buffer.push('---');
          
          // Trigger debounced write
          const debouncedWrite = this.debouncedFileWrites.get(relativePath);
          if (debouncedWrite) {
            debouncedWrite();
          }
        }
      }
      
      this.currentTestCase.delete(payload.filePath);
    }

    // Update timestamp and trigger debounced write
    this.state.updatedAt = new Date().toISOString();
    this.debouncedWrite();
  }

  /**
   * Append output chunk to the unified output log and write incrementally to test file logs
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
    
    // Get the relative path for the file
    const normalizedPath = path.resolve(filePath.replace(/^\.\//, ''));
    const relativePath = path.relative(process.cwd(), normalizedPath);
    
    // Add to buffer for incremental writing to individual log files
    const buffer = this.testFileBuffers.get(relativePath);
    if (buffer) {
      // Add test case boundary markers if we have a current test case
      const currentTest = this.currentTestCase.get(filePath);
      if (currentTest && !buffer.some(line => line.includes(`### ${currentTest}`))) {
        buffer.push('');
        buffer.push(`### ${currentTest}`);
        buffer.push('');
      }
      
      // Add the output chunk with optional stderr prefix
      const prefix = isStderr ? '[stderr] ' : '';
      const lines = chunk.trimEnd().split('\n');
      for (const line of lines) {
        buffer.push(prefix + line);
      }
      
      // Trigger debounced write for this file
      const debouncedWrite = this.debouncedFileWrites.get(relativePath);
      if (debouncedWrite) {
        debouncedWrite();
      }
    } else {
      this.logger.debug('No buffer found for file, output will not be incrementally written', { file: relativePath });
    }
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
    
    // Cancel all pending debounced writes for individual files
    this.logger.debug('Canceling all pending debounced file writes');
    for (const [file, debouncedWrite] of this.debouncedFileWrites) {
      debouncedWrite.cancel();
    }
    
    // Flush all remaining buffers
    this.logger.debug('Flushing all remaining buffers');
    for (const [relativePath] of this.testFileBuffers) {
      await this.flushFileBuffer(relativePath);
    }
    
    // Close all test file handles
    this.logger.debug('Closing all test file handles');
    for (const [file, handle] of this.testFileHandles) {
      try {
        await handle.close();
        this.logger.debug('Closed file handle', { file });
      } catch (error) {
        this.logger.error('Failed to close file handle', { file, error });
      }
    }
    
    // Clear all maps
    this.testFileHandles.clear();
    this.testFileBuffers.clear();
    this.debouncedFileWrites.clear();
    
    // Close the output log handle
    if (this.outputLogHandle) {
      this.logger.debug('Closing output log file handle');
      await this.outputLogHandle.close();
      this.outputLogHandle = null;
    }

    // Individual log files have been written incrementally, no need to parse output.log

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
   * Create and open a log file for a test file
   */
  private async createLogFileForTest(relativePath: string): Promise<void> {
    // Check if we already have a file handle
    if (this.testFileHandles.has(relativePath)) {
      return;
    }

    const logFileName = `${this.sanitizeFilePath(relativePath)}.log`;
    const logPath = path.join(this.logsDirectory, logFileName);
    
    try {
      this.logger.debug('Opening file handle for test log', { file: relativePath, logPath });
      const handle = await fs.open(logPath, 'w');
      this.testFileHandles.set(relativePath, handle);
      
      // Write header immediately
      const header = [
        `# File: ${relativePath}`,
        `# Timestamp: ${new Date().toISOString()}`,
        `# This file contains all stdout/stderr output from the test file execution.`,
        '# ---',
        '',
      ].join('\n');
      
      await handle.writeFile(header);
      
      // Initialize buffer for this file
      this.testFileBuffers.set(relativePath, []);
      
      // Create per-file debounced write function (100ms delay, 500ms max wait)
      const flushBuffer = async () => {
        await this.flushFileBuffer(relativePath);
      };
      
      this.debouncedFileWrites.set(
        relativePath,
        debounce(flushBuffer, 100, { maxWait: 500 }) as (() => void) & { cancel: () => void }
      );
    } catch (error) {
      this.logger.error('Failed to open file handle for test log', { file: relativePath, error });
    }
  }

  /**
   * Flush buffered content to a specific test file
   */
  private async flushFileBuffer(relativePath: string): Promise<void> {
    const buffer = this.testFileBuffers.get(relativePath);
    const handle = this.testFileHandles.get(relativePath);
    
    if (!buffer || !handle || buffer.length === 0) {
      return;
    }
    
    try {
      this.logger.debug('Flushing buffer to test log', { 
        file: relativePath, 
        lines: buffer.length 
      });
      
      // Join buffer with newlines and append to file
      const content = buffer.join('\n') + '\n';
      await handle.appendFile(content);
      
      // Clear the buffer
      buffer.length = 0;
    } catch (error: any) {
      this.logger.error('Failed to flush buffer to test log', { 
        file: relativePath, 
        error 
      });
      
      // If the directory was deleted (ENOENT), try to recreate it
      if (error.code === 'ENOENT') {
        try {
          this.logger.info('Attempting to recreate logs directory and file handle');
          await fs.mkdir(this.logsDirectory, { recursive: true });
          
          // Try to reopen the file handle
          const logFileName = `${this.sanitizeFilePath(relativePath)}.log`;
          const logPath = path.join(this.logsDirectory, logFileName);
          const newHandle = await fs.open(logPath, 'w');
          
          // Close old handle and replace with new one
          try {
            await handle.close();
          } catch (closeError) {
            // Ignore close errors on invalid handle
          }
          
          this.testFileHandles.set(relativePath, newHandle);
          
          // Write header and buffered content
          const header = [
            `# File: ${relativePath}`,
            `# Timestamp: ${new Date().toISOString()}`,
            `# This file contains all stdout/stderr output from the test file execution.`,
            '# ---',
            '',
          ].join('\n');
          
          await newHandle.writeFile(header);
          
          // Try to write the buffer again if it has content
          if (buffer.length > 0) {
            const content = buffer.join('\n') + '\n';
            await newHandle.appendFile(content);
            buffer.length = 0;
          }
        } catch (recoveryError) {
          this.logger.error('Failed to recover from ENOENT error', { 
            file: relativePath, 
            error: recoveryError 
          });
          // Keep data in buffer on error so we can retry
        }
      }
      // For other errors, keep data in buffer so we can retry
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
