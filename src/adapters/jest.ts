import { IPCManager } from '../ipc';
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

  constructor() {
    this.originalStdoutWrite = process.stdout.write.bind(process.stdout);
    this.originalStderrWrite = process.stderr.write.bind(process.stderr);
  }

  onRunStart(): void {
    // Initialize IPC if needed
    const ipcPath = process.env.THREEPIO_IPC_PATH;
    if (!ipcPath) {
      console.error('3pio: THREEPIO_IPC_PATH not set');
    }
  }

  onTestStart(test: Test): void {
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

    IPCManager.sendEvent({
      eventType: 'testFileResult',
      payload: {
        filePath: test.path,
        status
      }
    }).catch(error => {
      // Silent operation - don't log errors
    });

    this.currentTestFile = null;
  }

  onRunComplete(
    testContexts: Set<TestContext>,
    results: AggregatedResult
  ): void {
    // Ensure capture is stopped
    this.stopCapture();
  }

  private startCapture(): void {
    if (this.captureEnabled) return;
    this.captureEnabled = true;

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

    // Restore original functions
    process.stdout.write = this.originalStdoutWrite;
    process.stderr.write = this.originalStderrWrite;
  }

  getLastError(): void {
    // Required by Reporter interface
  }
}