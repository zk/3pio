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
    
    // Debug: Log that reporter was instantiated
    require('fs').appendFileSync('/tmp/vitest-debug.log', `${new Date().toISOString()} - ThreePioVitestReporter instantiated, IPC=${process.env.THREEPIO_IPC_PATH}\n`);
  }

  onInit(ctx: Vitest): void {
    // Initialize IPC if needed
    const ipcPath = process.env.THREEPIO_IPC_PATH;
    require('fs').appendFileSync('/tmp/vitest-debug.log', `${new Date().toISOString()} - onInit called, IPC=${ipcPath}\n`);
    if (!ipcPath) {
      // Silent operation - adapters must not output to console
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
    
    // Debug: Log file start
    require('fs').appendFileSync('/tmp/vitest-debug.log', `${new Date().toISOString()} - onTestFileStart: ${file.filepath}\n`);
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

    // Debug: Write to a known location to verify reporter is working
    const debugPath = '/tmp/vitest-debug.log';
    require('fs').appendFileSync(debugPath, `${new Date().toISOString()} - Sending event for ${file.filepath}: ${status}, IPC=${process.env.THREEPIO_IPC_PATH}\n`);

    IPCManager.sendEvent({
      eventType: 'testFileResult',
      payload: {
        filePath: file.filepath,
        status
      }
    }).catch(error => {
      // Debug: Log error to debug file
      require('fs').appendFileSync(debugPath, `${new Date().toISOString()} - Error: ${error.message}\n`);
    });

    this.currentTestFile = null;
  }

  async onFinished(files?: File[], errors?: unknown[]): Promise<void> {
    // Ensure capture is stopped
    this.stopCapture();
    
    // Debug: Log that onFinished was called
    require('fs').appendFileSync('/tmp/vitest-debug.log', `${new Date().toISOString()} - onFinished called with ${files?.length || 0} files\n`);
    
    // For Vitest 3.x, we need to send results here if they weren't sent via onTestFileResult
    if (files && files.length > 0) {
      for (const file of files) {
        let status: 'PASS' | 'FAIL' | 'SKIP' = 'PASS';
        
        if (file.result?.state === 'fail') {
          status = 'FAIL';
        } else if (file.result?.state === 'skip' || file.mode === 'skip') {
          status = 'SKIP';
        }
        
        require('fs').appendFileSync('/tmp/vitest-debug.log', `${new Date().toISOString()} - File ${file.filepath}: ${status}\n`);
        
        try {
          await IPCManager.sendEvent({
            eventType: 'testFileResult',
            payload: {
              filePath: file.filepath,
              status
            }
          });
          require('fs').appendFileSync('/tmp/vitest-debug.log', `${new Date().toISOString()} - Successfully sent event for ${file.filepath}\n`);
        } catch (error: any) {
          require('fs').appendFileSync('/tmp/vitest-debug.log', `${new Date().toISOString()} - Error sending: ${error.message}\n`);
        }
      }
    }
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