import * as fs from 'fs';
import * as path from 'path';

export type LogLevel = 'DEBUG' | 'INFO' | 'WARN' | 'ERROR';

export class Logger {
  private static instance: Logger | null = null;
  private logPath: string;
  private component: string;
  private isInitComplete: boolean = false;
  
  private constructor(component: string) {
    this.component = component;
    this.logPath = path.join(process.cwd(), '.3pio', 'debug.log');
    this.ensureLogDirectory();
  }
  
  static getInstance(component: string): Logger {
    if (!Logger.instance) {
      Logger.instance = new Logger(component);
    }
    return Logger.instance;
  }
  
  static create(component: string): Logger {
    return new Logger(component);
  }
  
  private ensureLogDirectory(): void {
    try {
      fs.mkdirSync(path.dirname(this.logPath), { recursive: true });
    } catch {
      // Directory might already exist
    }
  }
  
  private formatMessage(level: LogLevel, message: string, data?: any): string {
    const timestamp = new Date().toISOString();
    const dataStr = data ? ` | ${JSON.stringify(data)}` : '';
    return `${timestamp} ${level.padEnd(5)} | [${this.component}] ${message}${dataStr}`;
  }
  
  private writeLog(level: LogLevel, message: string, data?: any): void {
    try {
      const formattedMessage = this.formatMessage(level, message, data);
      fs.appendFileSync(this.logPath, formattedMessage + '\n', 'utf8');
    } catch {
      // Silent failure if we can't write logs
    }
  }
  
  /**
   * Log human-readable startup preamble without timestamps
   */
  startupPreamble(lines: string[]): void {
    try {
      const preamble = lines.map(line => `[${this.component}] ${line}`).join('\n');
      fs.appendFileSync(this.logPath, preamble + '\n', 'utf8');
    } catch {
      // Silent failure
    }
  }
  
  /**
   * Log machine-readable initialization complete
   */
  initComplete(config: Record<string, any>): void {
    this.isInitComplete = true;
    this.info('Initialization complete', config);
  }
  
  debug(message: string, data?: any): void {
    if (process.env.THREEPIO_DEBUG === '1') {
      this.writeLog('DEBUG', message, data);
    }
  }
  
  info(message: string, data?: any): void {
    this.writeLog('INFO', message, data);
  }
  
  warn(message: string, data?: any): void {
    this.writeLog('WARN', message, data);
  }
  
  error(message: string, error?: Error | any, data?: any): void {
    const errorData = {
      ...data,
      ...(error && {
        error: error.message || String(error),
        stack: error.stack
      })
    };
    this.writeLog('ERROR', message, errorData);
  }
  
  /**
   * Log lifecycle events with consistent narrative structure
   */
  lifecycle(event: string, details?: any): void {
    this.info(`Lifecycle: ${event}`, details);
  }
  
  /**
   * Log test execution flow
   */
  testFlow(action: string, testFile?: string, details?: any): void {
    const message = testFile 
      ? `Test flow: ${action} for ${testFile}`
      : `Test flow: ${action}`;
    this.info(message, details);
  }
  
  /**
   * Log IPC events
   */
  ipc(direction: 'send' | 'receive', eventType: string, details?: any): void {
    this.debug(`IPC ${direction}: ${eventType}`, details);
  }
  
  /**
   * Log command execution
   */
  command(cmd: string, args?: string[]): void {
    this.info(`Executing command: ${cmd}`, { args });
  }
  
  /**
   * Log decision points
   */
  decision(description: string, choice: string, reason?: string): void {
    this.info(`Decision: ${description}`, { choice, reason });
  }
}