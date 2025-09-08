#!/usr/bin/env node

import { Command } from 'commander';
import { $ } from 'zx';
import { promises as fs } from 'fs';
import path from 'path';
import { IPCManager } from './ipc';
import { ReportManager } from './ReportManager';
import { IPCEvent } from './types/events';
import { TestRunnerManager, TestRunnerName } from './TestRunnerManager';
import { TestRunnerDefinition } from './runners/base/TestRunnerDefinition';
import { Logger } from './utils/logger';

// Disable zx verbosity
$.verbose = false;

// Removed old TestRunner interface - now using TestRunnerDefinition

class CLIOrchestrator {
  private runId: string;
  private ipcPath: string;
  private ipcManager: IPCManager | null = null;
  private reportManager: ReportManager | null = null;
  private logger: Logger;

  constructor() {
    this.runId = new Date().toISOString().replace(/[:.]/g, '').replace('T', 'T');
    this.ipcPath = '';
    this.logger = Logger.create('cli-orchestrator');
  }

  async run(commandArgs: string[]): Promise<void> {
    try {
      const testCommand = commandArgs.join(' ');
      
      // Log startup preamble
      this.logger.startupPreamble([
        '=========================================',
        'Starting 3pio test runner adapter v1.0.0',
        'Configuration:',
        `  - Run ID: ${this.runId}`,
        `  - Working directory: ${process.cwd()}`,
        `  - Command: ${testCommand}`,
        `  - Debug mode: ${process.env.THREEPIO_DEBUG === '1' ? 'enabled' : 'disabled'}`,
        '========================================='
      ]);
      
      console.log(`Greetings! I will now execute the test command:`);
      console.log(`\`${testCommand}\``);
      console.log();

      this.logger.lifecycle('Starting test execution', { command: testCommand });
      
      // Detect test runner
      this.logger.info('Detecting test runner from command');
      const runnerName = await this.detectTestRunner(commandArgs);
      
      if (!runnerName) {
        this.logger.error('Test runner detection failed', null, { 
          command: commandArgs,
          supportedRunners: TestRunnerManager.getAvailableRunners() 
        });
        console.error('Oh dear! I cannot determine which test runner you are using.');
        console.error(`3pio currently supports: ${TestRunnerManager.getAvailableRunners().join(', ')}`);
        process.exit(1);
      }
      
      this.logger.decision('Test runner detected', runnerName, `Based on command: ${commandArgs[0]}`);
      
      const runner = TestRunnerManager.getDefinition(runnerName);
      this.logger.info(`Using ${runnerName} test runner adapter`);

      // Perform dry run to get test files
      this.logger.lifecycle('Performing dry run to discover test files');
      const testFiles = await runner.getTestFiles(commandArgs);
      this.logger.info(`Discovered ${testFiles.length} test files`, { files: testFiles });
      
      if (testFiles.length === 0) {
        this.logger.warn('No test files discovered during dry run', { command: commandArgs });
        this.logger.decision('Proceeding without file list', 'continue', 'Test runner will handle file discovery');
        console.log('No test files found to run. Proceeding without file list...');
        // Continue anyway - the test runner will handle finding files
      }

      // Initialize IPC and Report
      this.logger.lifecycle('Initializing IPC and report systems');
      await this.initialize(testCommand, testFiles, runnerName);

      // Print preamble
      this.printPreamble(testFiles);

      // Start IPC listening
      this.logger.lifecycle('Starting IPC event monitoring');
      this.startIPCListening();

      // Execute main command with adapter injected
      this.logger.lifecycle('Executing test runner with adapter', { runner: runnerName });
      const exitCode = await this.executeMainCommand(runnerName, commandArgs);
      this.logger.info(`Test execution completed with exit code ${exitCode}`);

      // Finalize
      this.logger.lifecycle('Finalizing reports and cleanup');
      await this.finalize(exitCode);
      
      this.logger.lifecycle('Test run complete', { exitCode, runId: this.runId });
      process.exit(exitCode);
    } catch (error) {
      this.logger.error('Catastrophic error in test execution', error as Error);
      console.error('Oh no! A catastrophic error occurred:', error);
      process.exit(1);
    }
  }

  private async detectTestRunner(args: string[]): Promise<TestRunnerName | null> {
    let packageJsonContent: string | undefined;
    
    // Only read package.json if we might need it (npm commands)
    if (args[0] === 'npm' && (args[1] === 'test' || args[1] === 'run')) {
      this.logger.debug('Reading package.json for npm command detection');
      try {
        packageJsonContent = await fs.readFile(path.join(process.cwd(), 'package.json'), 'utf8');
        this.logger.debug('Successfully read package.json');
      } catch (error) {
        this.logger.debug('package.json not found or not readable', { error: (error as Error).message });
        // package.json not found - continue without it
      }
    }
    
    const runner = await TestRunnerManager.detect(args, packageJsonContent);
    this.logger.debug('Test runner detection result', { runner, args });
    return runner;
  }

  // Removed performDryRun - now handled by TestRunnerDefinition.getTestFiles()

  private async initialize(testCommand: string, testFiles: string[], runnerName: TestRunnerName): Promise<void> {
    // Create IPC directory and file
    this.logger.info('Creating IPC directory and communication channel');
    const ipcDir = await IPCManager.ensureIPCDirectory();
    this.ipcPath = path.join(ipcDir, `${this.runId}.jsonl`);
    this.logger.info(`IPC path configured: ${this.ipcPath}`);
    
    // Set environment variable for adapters
    process.env.THREEPIO_IPC_PATH = this.ipcPath;
    this.logger.debug('THREEPIO_IPC_PATH environment variable set');
    
    // Get the output parser for this test runner
    const parser = TestRunnerManager.getParser(runnerName);
    
    // Initialize managers
    this.logger.info('Initializing IPC and Report managers');
    this.ipcManager = new IPCManager(this.ipcPath);
    this.reportManager = new ReportManager(this.runId, testCommand, parser);
    
    // Initialize report with test files
    this.logger.info('Initializing report structure with test files');
    await this.reportManager.initialize(testFiles);
    
    this.logger.initComplete({
      runId: this.runId,
      testRunner: runnerName,
      testFiles: testFiles.length,
      ipcPath: this.ipcPath
    });
  }

  private printPreamble(testFiles: string[]): void {
    const reportPath = this.reportManager!.getReportPath();
    this.logger.info('Report will be generated at', { path: reportPath });
    console.log(`Full report: ${reportPath}`);
    console.log();

    if (testFiles.length <= 10) {
      // Short list - show all files
      console.log(`The following ${testFiles.length} test file${testFiles.length === 1 ? '' : 's'} will be run:`);
      testFiles.forEach(file => {
        console.log(`- ${file}`);
      });
    } else if (testFiles.length <= 25) {
      // Medium list - show first 10 and count
      console.log(`The following ${testFiles.length} test files will be run:`);
      testFiles.slice(0, 10).forEach(file => {
        console.log(`- ${file}`);
      });
      console.log(`- ...and ${testFiles.length - 10} more.`);
    } else {
      // Long list - show breakdown by directory
      console.log(`Running ${testFiles.length} total test files.`);
      console.log();
      console.log('Breakdown by directory:');
      
      const dirCounts = new Map<string, string[]>();
      testFiles.forEach(file => {
        const dir = path.dirname(file);
        if (!dirCounts.has(dir)) {
          dirCounts.set(dir, []);
        }
        dirCounts.get(dir)!.push(path.basename(file));
      });

      // Show first few directories
      const dirs = Array.from(dirCounts.entries()).slice(0, 4);
      dirs.forEach(([dir, files]) => {
        console.log(`- ${dir}/ (${files.length} files)`);
        if (files.length <= 3) {
          files.forEach(file => console.log(`  - ${file}`));
        } else {
          files.slice(0, 3).forEach(file => console.log(`  - ${file}`));
          console.log(`  - ...and ${files.length - 3} more.`);
        }
      });
    }

    console.log();
    console.log('Beginning test execution now...');
    console.log();
    this.logger.lifecycle('Test execution starting');
  }

  private startIPCListening(): void {
    if (!this.ipcManager || !this.reportManager) {
      this.logger.error('Cannot start IPC listening: managers not initialized');
      return;
    }

    this.logger.info('Starting IPC event listener on', { path: this.ipcPath });
    
    // Process events sequentially to avoid concurrent file writes
    const eventQueue: IPCEvent[] = [];
    let processing = false;

    const processQueue = async () => {
      if (processing || eventQueue.length === 0) return;
      processing = true;

      while (eventQueue.length > 0) {
        const event = eventQueue.shift()!;
        this.logger.ipc('receive', event.eventType, { filePath: event.payload?.filePath });
        try {
          await this.reportManager!.handleEvent(event);
        } catch (error) {
          this.logger.error('Error handling IPC event', error as Error, { eventType: event.eventType });
          console.error('Error handling event:', error);
        }
      }

      processing = false;
    };

    this.ipcManager.watchEvents((event: IPCEvent) => {
      this.logger.debug('IPC event received', { type: event.eventType, queueLength: eventQueue.length });
      eventQueue.push(event);
      processQueue();
    });
  }

  private async executeMainCommand(
    runnerName: TestRunnerName,
    args: string[]
  ): Promise<number> {
    const runner = TestRunnerManager.getDefinition(runnerName);
    
    // Use absolute path to the adapter
    const adapterPath = path.join(__dirname, runner.getAdapterFileName());
    this.logger.debug('Adapter path resolved', { adapter: adapterPath });
    
    // Build command using runner definition
    const commandArgs = runner.buildMainCommand(args, adapterPath);
    this.logger.command(commandArgs[0], commandArgs.slice(1));

    try {
      // Set THREEPIO_IPC_PATH for the child process
      const env = { ...process.env, THREEPIO_IPC_PATH: this.ipcPath };
      this.logger.debug('Environment prepared for child process', { THREEPIO_IPC_PATH: this.ipcPath });
      
      // Execute the command with zx using proper array syntax
      // This avoids shell interpretation issues
      this.logger.lifecycle('Spawning test runner process');
      const proc = $({ env })`${commandArgs}`;
      
      // Pipe output to console so user can see test progress
      proc.stdout.pipe(process.stdout);
      proc.stderr.pipe(process.stderr);
      
      // Wait for process to complete
      const result = await proc;
      
      this.logger.info('Test runner process completed successfully');
      // Return the actual exit code (0 on success)
      return 0;
    } catch (error: any) {
      // zx throws on non-zero exit codes
      const exitCode = error.exitCode || 1;
      this.logger.warn(`Test runner process exited with code ${exitCode}`);
      return exitCode;
    }
  }

  private async finalize(exitCode: number): Promise<void> {
    // Give adapters a grace period to write final events
    // This prevents race conditions where the test runner exits before
    // the adapter has finished writing all events
    // Jest in particular needs more time for all reporter callbacks
    this.logger.info('Waiting for adapter to flush final events (1s grace period)');
    await new Promise(resolve => setTimeout(resolve, 1000));

    if (this.reportManager) {
      this.logger.info('Finalizing report generation');
      await this.reportManager.finalize(exitCode);
      
      // Validate that we received test results
      const summary = this.reportManager.getSummary();
      this.logger.info('Test run summary', summary);
      
      if (summary.totalFiles === 0 && exitCode === 0) {
        this.logger.warn('No test results captured despite successful exit', { exitCode });
        console.error('\nWarning: No test results were captured. This may indicate:');
        console.error('- The test runner adapter failed to inject properly');
        console.error('- The test runner exited before results could be written');
        console.error('- An incompatibility with your test runner version');
        console.error('\nCheck the IPC file for debugging:', this.ipcPath);
      }
    }

    if (this.ipcManager) {
      this.logger.info('Cleaning up IPC resources');
      await this.ipcManager.cleanup();
    }
    
    this.logger.lifecycle('Shutdown complete');
  }
}

// Main entry point
const program = new Command();

program
  .name('3pio')
  .description('AI-first test runner adapter')
  .version('0.1.0');

program
  .command('run <command...>')
  .description('Run tests with 3pio adapter')
  .action(async (commandArgs: string[]) => {
    const orchestrator = new CLIOrchestrator();
    await orchestrator.run(commandArgs);
  });

// Parse arguments
program.parse(process.argv);

// Show help if no command provided
if (!process.argv.slice(2).length) {
  program.outputHelp();
}