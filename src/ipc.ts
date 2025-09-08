import { promises as fs } from 'fs';
import { watch } from 'chokidar';
import { IPCEvent } from './types/events';
import path from 'path';

export class IPCManager {
  private ipcFilePath: string;
  private lastReadPosition: number = 0;
  private watcher: any = null;

  constructor(ipcFilePath: string) {
    this.ipcFilePath = ipcFilePath;
  }

  /**
   * Writer method for adapters to send events
   */
  async writeEvent(event: IPCEvent): Promise<void> {
    const line = JSON.stringify(event) + '\n';
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
    await fs.appendFile(ipcPath, line, 'utf8');
  }

  /**
   * Reader method for CLI to watch events
   */
  watchEvents(callback: (event: IPCEvent) => void): void {
    // Ensure the IPC file exists
    fs.writeFile(this.ipcFilePath, '', { flag: 'a' }).catch(console.error);

    this.watcher = watch(this.ipcFilePath, {
      persistent: true,
      usePolling: false,
      awaitWriteFinish: {
        stabilityThreshold: 50,
        pollInterval: 10
      }
    });

    this.watcher.on('change', async () => {
      try {
        const content = await fs.readFile(this.ipcFilePath, 'utf8');
        const lines = content.split('\n');
        
        // Process only new lines since last read
        const newLines = lines.slice(this.lastReadPosition);
        this.lastReadPosition = lines.length - 1; // -1 because last line might be empty

        for (const line of newLines) {
          if (line.trim()) {
            try {
              const event = JSON.parse(line) as IPCEvent;
              callback(event);
            } catch (parseError) {
              console.error('Failed to parse IPC event:', parseError);
            }
          }
        }
      } catch (error) {
        console.error('Error reading IPC file:', error);
      }
    });
  }

  /**
   * Stop watching the IPC file
   */
  async stopWatching(): Promise<void> {
    if (this.watcher) {
      await this.watcher.close();
      this.watcher = null;
    }
  }

  /**
   * Clean up the IPC file
   */
  async cleanup(): Promise<void> {
    await this.stopWatching();
    try {
      await fs.unlink(this.ipcFilePath);
    } catch (error) {
      // Ignore error if file doesn't exist
    }
  }

  /**
   * Create IPC directory if it doesn't exist
   */
  static async ensureIPCDirectory(): Promise<string> {
    const ipcDir = path.join(process.cwd(), '.3pio', 'ipc');
    await fs.mkdir(ipcDir, { recursive: true });
    return ipcDir;
  }
}