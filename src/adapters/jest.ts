import { IPCManager } from '../ipc';
import { Logger } from '../utils/logger';
import type { 
  Reporter, 
  Test, 
  TestResult, 
  AggregatedResult,
  TestContext
} from '@jest/reporters';

export default class ThreePioJestReporter implements Reporter {
  private originalStdoutWrite: typeof process.stdout.write;
  private originalStderrWrite: typeof process.stderr.write;
  private currentTestFile: string | null = null;
  private captureEnabled: boolean = false;
  private logger: Logger;

  constructor() {
    this.originalStdoutWrite = process.stdout.write.bind(process.stdout);
    this.originalStderrWrite = process.stderr.write.bind(process.stderr);
    this.logger = Logger.create('jest-adapter');
    
    // Log startup preamble
    this.logger.startupPreamble([
      '==================================',
      '3pio Jest Adapter v1.0.0',
      'Configuration:',
      `  - IPC Path: ${process.env.THREEPIO_IPC_PATH || 'not set'}`,
      `  - Process ID: ${process.pid}`,
      '=================================='
    ]);
  }

  onRunStart(): void {
    this.logger.lifecycle('Test run starting');
    // Initialize IPC if needed - no console output (adapter must be silent)
    const ipcPath = process.env.THREEPIO_IPC_PATH;
    if (!ipcPath) {
      this.logger.error('THREEPIO_IPC_PATH not set - adapter cannot function');
      return;
    }
    this.logger.info('IPC communication channel ready', { path: ipcPath });
    this.logger.initComplete({ ipcPath });
  }

  onTestStart(test: Test): void {
    this.logger.testFlow('Starting test file', test.path);
    this.currentTestFile = test.path;
    this.startCapture();
  }

  onTestResult(
    test: Test,
    testResult: TestResult,
    aggregatedResult: AggregatedResult
  ): void {
    this.stopCapture();

    // Send test file result
    const status = testResult.numFailingTests > 0 ? 'FAIL' : 
                   testResult.skipped ? 'SKIP' : 'PASS';
    
    this.logger.testFlow('Test file completed', test.path, { 
      status, 
      failures: testResult.numFailingTests,
      tests: testResult.numPassedTests + testResult.numFailingTests,
      passed: testResult.numPassedTests
    });

    this.logger.ipc('send', 'testFileResult', { filePath: test.path, status });
    IPCManager.sendEvent({
      eventType: 'testFileResult',
      payload: {
        filePath: test.path,
        status
      }
    }).catch(error => {
      this.logger.error('Failed to send testFileResult', error);
    });

    this.currentTestFile = null;
  }

  onRunComplete(
    testContexts: Set<TestContext>,
    results: AggregatedResult
  ): void {
    this.logger.lifecycle('Test run completing', {
      totalSuites: results.numTotalTestSuites,
      failedSuites: results.numFailedTestSuites,
      passedSuites: results.numPassedTestSuites,
      totalTests: results.numTotalTests,
      passedTests: results.numPassedTests,
      failedTests: results.numFailedTests
    });
    
    // Ensure capture is stopped
    this.stopCapture();
    
    // Force a small delay to ensure all IPC writes complete
    // This is a workaround for Jest's rapid shutdown
    const syncFs = require('fs');
    const ipcPath = process.env.THREEPIO_IPC_PATH;
    if (ipcPath) {
      // Write a final marker event to indicate completion
      try {
        this.logger.ipc('send', 'runComplete', {});
        syncFs.appendFileSync(ipcPath, JSON.stringify({ 
          eventType: 'runComplete', 
          payload: {} 
        }) + '\n', 'utf8');
        this.logger.info('Run completion marker sent');
      } catch (error: any) {
        this.logger.error('Failed to write runComplete marker', error);
      }
    }
    
    this.logger.lifecycle('Jest adapter shutdown complete');
  }

  private startCapture(): void {
    if (this.captureEnabled) return;
    this.captureEnabled = true;
    this.logger.debug('Starting stdout/stderr capture for', { file: this.currentTestFile });

    // Patch stdout
    process.stdout.write = (chunk: string | Uint8Array, ...args: any[]): boolean => {
      if (this.currentTestFile && chunk) {
        const chunkStr = chunk.toString();
        IPCManager.sendEvent({
          eventType: 'stdoutChunk',
          payload: {
            filePath: this.currentTestFile,
            chunk: chunkStr
          }
        }).catch(() => {});
      }
      return this.originalStdoutWrite(chunk, ...args);
    };

    // Patch stderr
    process.stderr.write = (chunk: string | Uint8Array, ...args: any[]): boolean => {
      if (this.currentTestFile && chunk) {
        const chunkStr = chunk.toString();
        IPCManager.sendEvent({
          eventType: 'stderrChunk',
          payload: {
            filePath: this.currentTestFile,
            chunk: chunkStr
          }
        }).catch(() => {});
      }
      return this.originalStderrWrite(chunk, ...args);
    };
  }

  private stopCapture(): void {
    if (!this.captureEnabled) return;
    this.captureEnabled = false;
    this.logger.debug('Stopping stdout/stderr capture');

    // Restore original functions
    process.stdout.write = this.originalStdoutWrite;
    process.stderr.write = this.originalStderrWrite;
  }

  getLastError(): void {
    // Required by Reporter interface
  }
}