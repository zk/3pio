import * as fs from 'fs';
import * as path from 'path';
import { IPCEvent } from './types/events';

/**
 * IPC sender for adapters - no external dependencies
 */
export class IPCSender {
  /**
   * Send an event to the IPC file (used by adapters)
   */
  static sendEvent(event: IPCEvent): Promise<void> {
    return Promise.resolve(this.sendEventSync(event));
  }
  
  /**
   * Synchronous version of sendEvent
   */
  static sendEventSync(event: IPCEvent): void {
    const ipcPath = process.env.THREEPIO_IPC_PATH;
    if (!ipcPath) {
      console.error('THREEPIO_IPC_PATH not set');
      return;
    }

    try {
      // Ensure directory exists
      const dir = path.dirname(ipcPath);
      if (!fs.existsSync(dir)) {
        fs.mkdirSync(dir, { recursive: true });
      }

      // Append event to file
      const line = JSON.stringify(event) + '\n';
      fs.appendFileSync(ipcPath, line);
    } catch (error) {
      // Silently fail to avoid disrupting test execution
      // Adapters should never throw errors
    }
  }
}