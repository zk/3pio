import { promises as fs } from 'fs';
import { watch } from 'chokidar';
import { IPCEvent } from './types/events';
import path from 'path';
import { Logger } from './utils/logger';

export class IPCManager {
  private ipcFilePath: string;
  private lastReadPosition: number = 0;
  private watcher: any = null;
  private logger: Logger;

  constructor(ipcFilePath: string) {
    this.ipcFilePath = ipcFilePath;
    this.logger = Logger.create('ipc-manager');
    this.logger.info('IPCManager initialized', { ipcFilePath });
  }

  /**
   * Writer method for adapters to send events
   */
  async writeEvent(event: IPCEvent): Promise<void> {
    const line = JSON.stringify(event) + '\n';
    this.logger.debug('Writing IPC event', { eventType: event.eventType });
    await fs.appendFile(this.ipcFilePath, line, 'utf8');
  }

  /**
   * Static helper for adapters running in test runner process
   */
  static async sendEvent(event: IPCEvent): Promise<void> {
    const ipcPath = process.env.THREEPIO_IPC_PATH;
    if (!ipcPath) {
      throw new Error('THREEPIO_IPC_PATH environment variable not set');
    }
    const line = JSON.stringify(event) + '\n';
    
    // Use synchronous file operations for reliability in test runner context
    const syncFs = require('fs');
    
    try {
      // Use synchronous write for immediate file creation
      syncFs.appendFileSync(ipcPath, line, 'utf8');
    } catch (error: any) {
      throw error;
    }
  }

  /**
   * Reader method for CLI to watch events
   */
  watchEvents(callback: (event: IPCEvent) => void): void {
    this.logger.info('Starting IPC event watcher', { path: this.ipcFilePath });
    
    // Ensure the IPC file exists
    fs.writeFile(this.ipcFilePath, '', { flag: 'a' }).catch(error => {
      this.logger.error('Failed to ensure IPC file exists', error);
      console.error(error);
    });

    this.watcher = watch(this.ipcFilePath, {
      persistent: true,
      usePolling: false,
      awaitWriteFinish: {
        stabilityThreshold: 50,
        pollInterval: 10
      }
    });

    this.watcher.on('change', async () => {
      this.logger.debug('IPC file changed, processing new events');
      try {
        const content = await fs.readFile(this.ipcFilePath, 'utf8');
        const lines = content.split('\n');
        
        // Process only new lines since last read
        const newLines = lines.slice(this.lastReadPosition);
        const newEventCount = newLines.filter(l => l.trim()).length;
        this.logger.debug(`Found ${newEventCount} new events to process`);
        this.lastReadPosition = lines.length - 1; // -1 because last line might be empty

        for (const line of newLines) {
          if (line.trim()) {
            try {
              const event = JSON.parse(line) as IPCEvent;
              this.logger.debug('Parsed IPC event', { type: event.eventType });
              callback(event);
            } catch (parseError) {
              this.logger.error('Failed to parse IPC event', parseError as Error, { line });
              console.error('Failed to parse IPC event:', parseError);
            }
          }
        }
      } catch (error) {
        this.logger.error('Error reading IPC file', error as Error);
        console.error('Error reading IPC file:', error);
      }
    });
  }

  /**
   * Stop watching the IPC file
   */
  async stopWatching(): Promise<void> {
    if (this.watcher) {
      this.logger.debug('Stopping IPC file watcher');
      await this.watcher.close();
      this.watcher = null;
    }
  }

  /**
   * Clean up the IPC file
   */
  async cleanup(): Promise<void> {
    this.logger.lifecycle('Cleaning up IPC resources');
    await this.stopWatching();
    // Don't delete the IPC file - it's useful for debugging and is in a timestamped directory anyway
    // The file will be cleaned up when the entire .3pio directory is removed if needed
    this.logger.debug('IPC cleanup complete');
  }

  /**
   * Create IPC directory if it doesn't exist
   */
  static async ensureIPCDirectory(): Promise<string> {
    const ipcDir = path.join(process.cwd(), '.3pio', 'ipc');
    const logger = Logger.create('ipc-manager-static');
    logger.debug('Ensuring IPC directory exists', { path: ipcDir });
    await fs.mkdir(ipcDir, { recursive: true });
    logger.debug('IPC directory ready');
    return ipcDir;
  }
}