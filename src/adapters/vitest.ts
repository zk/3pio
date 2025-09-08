import { IPCManager } from '../ipc';
import { Logger } from '../utils/logger';
import type { File, Reporter, Vitest } from 'vitest';

const packageJson = require('../../package.json');

export default class ThreePioVitestReporter implements Reporter {
  private originalStdoutWrite: typeof process.stdout.write;
  private originalStderrWrite: typeof process.stderr.write;
  private currentTestFile: string | null = null;
  private captureEnabled: boolean = false;
  private logger: Logger;

  constructor() {
    this.originalStdoutWrite = process.stdout.write.bind(process.stdout);
    this.originalStderrWrite = process.stderr.write.bind(process.stderr);
    this.logger = Logger.create('vitest-adapter');
    
    // Log startup preamble
    this.logger.startupPreamble([
      '==================================',
      `3pio Vitest Adapter v${packageJson.version}`,
      'Configuration:',
      `  - IPC Path: ${process.env.THREEPIO_IPC_PATH || 'not set'}`,
      `  - Process ID: ${process.pid}`,
      '=================================='
    ]);
  }

  onInit(ctx: Vitest): void {
    this.logger.lifecycle('Test run initializing');
    // Initialize IPC if needed
    const ipcPath = process.env.THREEPIO_IPC_PATH;
    if (!ipcPath) {
      this.logger.error('THREEPIO_IPC_PATH not set - adapter cannot function');
      // Silent operation - adapters must not output to console
    } else {
      this.logger.info('IPC communication channel ready', { path: ipcPath });
    }
    
    this.logger.initComplete({ ipcPath });
    
    // Start capturing stdout/stderr immediately
    // This ensures we capture output even when onTestFileStart is not called
    this.logger.debug('Starting global capture for test output');
    this.startCapture();
  }

  onPathsCollected(paths: string[]): void {
    // Called when test files are discovered
    this.logger.info('Test paths collected', { count: paths.length });
  }

  onCollected(files: File[]): void {
    // Called when test files are collected
    this.logger.info('Test files collected', { count: files.length });
  }

  onTestFileStart(file: File): void {
    this.logger.testFlow('Starting test file', file.filepath);
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
    
    const testStats = file.result ? {
      tests: file.result.tests?.length || 0,
      duration: file.result.duration || 0,
      state: file.result.state
    } : {};
    
    this.logger.testFlow('Test file completed', file.filepath, { status, ...testStats });

    this.logger.ipc('send', 'testFileResult', { filePath: file.filepath, status });
    IPCManager.sendEvent({
      eventType: 'testFileResult',
      payload: {
        filePath: file.filepath,
        status
      }
    }).catch(error => {
      this.logger.error('Failed to send testFileResult', error);
    });

    this.currentTestFile = null;
  }

  async onFinished(files?: File[], errors?: unknown[]): Promise<void> {
    this.logger.lifecycle('Test run finishing', { 
      files: files?.length || 0,
      errors: errors?.length || 0 
    });
    
    // Ensure capture is stopped
    this.stopCapture();
    
    // For Vitest 3.x, we need to send results here if they weren't sent via onTestFileResult
    if (files && files.length > 0) {
      this.logger.info('Processing files in onFinished (fallback mode)', { count: files.length });
      
      for (const file of files) {
        let status: 'PASS' | 'FAIL' | 'SKIP' = 'PASS';
        
        if (file.result?.state === 'fail') {
          status = 'FAIL';
        } else if (file.result?.state === 'skip' || file.mode === 'skip') {
          status = 'SKIP';
        }
        
        this.logger.debug('Sending deferred test result', { file: file.filepath, status });
        
        try {
          this.logger.ipc('send', 'testFileResult', { filePath: file.filepath, status });
          await IPCManager.sendEvent({
            eventType: 'testFileResult',
            payload: {
              filePath: file.filepath,
              status
            }
          });
        } catch (error: any) {
          this.logger.error('Failed to send deferred test result', error, { file: file.filepath });
        }
      }
    }
    
    this.logger.lifecycle('Vitest adapter shutdown complete');
  }

  private startCapture(): void {
    if (this.captureEnabled) return;
    this.captureEnabled = true;
    this.logger.debug('Starting stdout/stderr capture', { currentFile: this.currentTestFile });

    // Patch stdout (silent - no passthrough)
    process.stdout.write = (chunk: string | Uint8Array, ...args: any[]): boolean => {
      if (chunk) {
        const chunkStr = chunk.toString();
        // Use currentTestFile if available, otherwise use 'global' for suite-level output
        const filePath = this.currentTestFile || 'global';
        IPCManager.sendEvent({
          eventType: 'stdoutChunk',
          payload: {
            filePath,
            chunk: chunkStr
          }
        }).catch(() => {});
      }
      // Return true to indicate success, but don't actually write anything
      return true;
    };

    // Patch stderr (silent - no passthrough)
    process.stderr.write = (chunk: string | Uint8Array, ...args: any[]): boolean => {
      if (chunk) {
        const chunkStr = chunk.toString();
        // Use currentTestFile if available, otherwise use 'global' for suite-level output
        const filePath = this.currentTestFile || 'global';
        IPCManager.sendEvent({
          eventType: 'stderrChunk',
          payload: {
            filePath,
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
}