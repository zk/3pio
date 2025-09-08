import { IPCManager } from '../ipc';
import type { File, Reporter, Vitest } from 'vitest';

export default class ThreePioVitestReporter implements Reporter {
  private originalStdoutWrite: typeof process.stdout.write;
  private originalStderrWrite: typeof process.stderr.write;
  private currentTestFile: string | null = null;
  private captureEnabled: boolean = false;

  constructor() {
    this.originalStdoutWrite = process.stdout.write.bind(process.stdout);
    this.originalStderrWrite = process.stderr.write.bind(process.stderr);
  }

  onInit(ctx: Vitest): void {
    // Initialize IPC if needed
    const ipcPath = process.env.THREEPIO_IPC_PATH;
    if (!ipcPath) {
      console.error('3pio: THREEPIO_IPC_PATH not set');
    }
  }

  onPathsCollected(paths: string[]): void {
    // Called when test files are discovered
  }

  onCollected(files: File[]): void {
    // Called when test files are collected
  }

  onTestFileStart(file: File): void {
    this.currentTestFile = file.filepath;
    this.startCapture();
  }

  onTestFileResult(file: File): void {
    this.stopCapture();

    // Determine status from file result
    let status: 'PASS' | 'FAIL' | 'SKIP' = 'PASS';
    
    if (file.result?.state === 'fail') {
      status = 'FAIL';
    } else if (file.result?.state === 'skip' || file.mode === 'skip') {
      status = 'SKIP';
    }

    IPCManager.sendEvent({
      eventType: 'testFileResult',
      payload: {
        filePath: file.filepath,
        status
      }
    }).catch(error => {
      // Silent operation - don't log errors
    });

    this.currentTestFile = null;
  }

  onFinished(files: File[]): void {
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
}