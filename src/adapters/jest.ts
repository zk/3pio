import { IPCManager } from '../ipc';
import { Logger } from '../utils/logger';
import type { 
  Reporter, 
  Test, 
  TestResult, 
  AggregatedResult,
  TestContext
} from '@jest/reporters';

const packageJson = require('../../package.json');

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
      `3pio Jest Adapter v${packageJson.version}`,
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

  onTestCaseStart(test: Test, testCaseStartInfo: any): void {
    // Send test case start event
    if (testCaseStartInfo?.ancestorTitles && testCaseStartInfo?.title) {
      const suiteName = testCaseStartInfo.ancestorTitles.join(' › ');
      const testName = testCaseStartInfo.title;
      
      this.logger.testFlow('Test case starting', testName, { suite: suiteName });
      
      IPCManager.sendEvent({
        eventType: 'testCase',
        payload: {
          filePath: test.path,
          testName,
          suiteName: suiteName || undefined,
          status: 'RUNNING'
        }
      }).catch(error => {
        this.logger.error('Failed to send testCase start', error);
      });
    }
  }

  onTestCaseResult(test: Test, testCaseResult: any): void {
    // Send test case result event
    if (testCaseResult) {
      const suiteName = testCaseResult.ancestorTitles?.join(' › ');
      const testName = testCaseResult.title;
      let status: 'PASS' | 'FAIL' | 'SKIP' = 'PASS';
      
      if (testCaseResult.status === 'failed') {
        status = 'FAIL';
      } else if (testCaseResult.status === 'skipped' || testCaseResult.status === 'pending') {
        status = 'SKIP';
      }
      
      const error = testCaseResult.failureMessages?.join('\n\n');
      
      this.logger.testFlow('Test case completed', testName, { 
        suite: suiteName,
        status,
        duration: testCaseResult.duration
      });
      
      IPCManager.sendEvent({
        eventType: 'testCase',
        payload: {
          filePath: test.path,
          testName,
          suiteName: suiteName || undefined,
          status,
          duration: testCaseResult.duration,
          error
        }
      }).catch(error => {
        this.logger.error('Failed to send testCase result', error);
      });
    }
  }

  onTestStart(test: Test): void {
    this.logger.testFlow('Starting test file', test.path);
    this.currentTestFile = test.path;
    
    // Send testFileStart event
    this.logger.ipc('send', 'testFileStart', { filePath: test.path });
    IPCManager.sendEvent({
      eventType: 'testFileStart',
      payload: {
        filePath: test.path
      }
    }).catch(error => {
      this.logger.error('Failed to send testFileStart', error);
    });
    
    this.startCapture();
  }

  onTestResult(
    test: Test,
    testResult: TestResult,
    aggregatedResult: AggregatedResult
  ): void {
    this.stopCapture();

    // Check if console output is available (it should be, but it's always undefined)
    if (testResult.console && testResult.console.length > 0) {
      this.logger.info('Console output found in testResult!', { 
        consoleLength: testResult.console.length 
      });
      // Send console output via IPC
      for (const log of testResult.console) {
        const chunk = `${log.message}\n`;
        IPCManager.sendEvent({
          eventType: log.type === 'error' ? 'stderrChunk' : 'stdoutChunk',
          payload: {
            filePath: test.path,
            chunk
          }
        }).catch(() => {});
      }
    }

    // Send test file result
    const status = testResult.numFailingTests > 0 ? 'FAIL' : 
                   testResult.skipped ? 'SKIP' : 'PASS';
    
    // Send individual test case results if not already sent
    if (testResult.testResults) {
      for (const testCase of testResult.testResults) {
        const suiteName = testCase.ancestorTitles?.join(' › ');
        const testName = testCase.title;
        let testStatus: 'PASS' | 'FAIL' | 'SKIP' = 'PASS';
        
        if (testCase.status === 'failed') {
          testStatus = 'FAIL';
        } else if (testCase.status === 'skipped' || testCase.status === 'pending') {
          testStatus = 'SKIP';
        }
        
        const error = testCase.failureMessages?.join('\n\n');
        
        // Send test case event
        IPCManager.sendEvent({
          eventType: 'testCase',
          payload: {
            filePath: test.path,
            testName,
            suiteName: suiteName || undefined,
            status: testStatus,
            duration: testCase.duration,
            error
          }
        }).catch(() => {});
      }
    }
    
    // Collect failed test details for backward compatibility
    const failedTests: Array<{ name: string; duration?: number }> = [];
    if (testResult.testResults && status === 'FAIL') {
      for (const suite of testResult.testResults) {
        if (suite.status !== 'passed') {
          const fullName = suite.ancestorTitles.length > 0 
            ? `${suite.ancestorTitles.join(' › ')} › ${suite.title}`
            : suite.title;
          failedTests.push({
            name: fullName,
            duration: suite.duration
          });
        }
      }
    }
    
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
        status,
        failedTests: failedTests.length > 0 ? failedTests : undefined
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
    
    // Don't output summary here - the CLI will handle it
    
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
    
    // Patch stdout to capture test output (silent - no passthrough)
    process.stdout.write = (chunk: string | Uint8Array, ...args: any[]): boolean => {
      const chunkStr = chunk.toString();
      
      // Capture for IPC if we have a current test file
      if (this.currentTestFile) {
        IPCManager.sendEvent({
          eventType: 'stdoutChunk',
          payload: {
            filePath: this.currentTestFile,
            chunk: chunkStr
          }
        }).catch(() => {});
      }
      
      // Return true to indicate success, but don't actually write anything
      return true;
    };
    
    // Patch stderr to capture test output (silent - no passthrough)
    process.stderr.write = (chunk: string | Uint8Array, ...args: any[]): boolean => {
      const chunkStr = chunk.toString();
      
      // Capture for IPC if we have a current test file
      if (this.currentTestFile) {
        IPCManager.sendEvent({
          eventType: 'stderrChunk',
          payload: {
            filePath: this.currentTestFile,
            chunk: chunkStr
          }
        }).catch(() => {});
      }
      
      // Return true to indicate success, but don't actually write anything
      return true;
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