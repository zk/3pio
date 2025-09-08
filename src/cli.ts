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

// Disable zx verbosity
$.verbose = false;

// Removed old TestRunner interface - now using TestRunnerDefinition

class CLIOrchestrator {
  private runId: string;
  private ipcPath: string;
  private ipcManager: IPCManager | null = null;
  private reportManager: ReportManager | null = null;

  constructor() {
    this.runId = new Date().toISOString().replace(/[:.]/g, '').replace('T', 'T');
    this.ipcPath = '';
  }

  async run(commandArgs: string[]): Promise<void> {
    try {
      const testCommand = commandArgs.join(' ');
      console.log(`Greetings! I will now execute the test command:`);
      console.log(`\`${testCommand}\``);
      console.log();

      // Detect test runner
      const runnerName = await this.detectTestRunner(commandArgs);
      
      if (!runnerName) {
        console.error('Oh dear! I cannot determine which test runner you are using.');
        console.error(`3pio currently supports: ${TestRunnerManager.getAvailableRunners().join(', ')}`);
        process.exit(1);
      }
      
      const runner = TestRunnerManager.getDefinition(runnerName);

      // Perform dry run to get test files
      const testFiles = await runner.getTestFiles(commandArgs);
      
      if (testFiles.length === 0) {
        console.log('No test files found to run. Proceeding without file list...');
        // Continue anyway - the test runner will handle finding files
      }

      // Initialize IPC and Report
      await this.initialize(testCommand, testFiles, runnerName);

      // Print preamble
      this.printPreamble(testFiles);

      // Start IPC listening
      this.startIPCListening();

      // Execute main command with adapter injected
      const exitCode = await this.executeMainCommand(runnerName, commandArgs);

      // Finalize
      await this.finalize(exitCode);
      
      process.exit(exitCode);
    } catch (error) {
      console.error('Oh no! A catastrophic error occurred:', error);
      process.exit(1);
    }
  }

  private async detectTestRunner(args: string[]): Promise<TestRunnerName | null> {
    let packageJsonContent: string | undefined;
    
    // Only read package.json if we might need it (npm commands)
    if (args[0] === 'npm' && (args[1] === 'test' || args[1] === 'run')) {
      try {
        packageJsonContent = await fs.readFile(path.join(process.cwd(), 'package.json'), 'utf8');
      } catch {
        // package.json not found - continue without it
      }
    }
    
    return TestRunnerManager.detect(args, packageJsonContent);
  }

  // Removed performDryRun - now handled by TestRunnerDefinition.getTestFiles()

  private async initialize(testCommand: string, testFiles: string[], runnerName: TestRunnerName): Promise<void> {
    // Create IPC directory and file
    const ipcDir = await IPCManager.ensureIPCDirectory();
    this.ipcPath = path.join(ipcDir, `${this.runId}.jsonl`);
    
    // Set environment variable for adapters
    process.env.THREEPIO_IPC_PATH = this.ipcPath;
    
    // Get the output parser for this test runner
    const parser = TestRunnerManager.getParser(runnerName);
    
    // Initialize managers
    this.ipcManager = new IPCManager(this.ipcPath);
    this.reportManager = new ReportManager(this.runId, testCommand, parser);
    
    // Initialize report with test files
    await this.reportManager.initialize(testFiles);
  }

  private printPreamble(testFiles: string[]): void {
    const reportPath = this.reportManager!.getReportPath();
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
  }

  private startIPCListening(): void {
    if (!this.ipcManager || !this.reportManager) return;

    // Process events sequentially to avoid concurrent file writes
    const eventQueue: IPCEvent[] = [];
    let processing = false;

    const processQueue = async () => {
      if (processing || eventQueue.length === 0) return;
      processing = true;

      while (eventQueue.length > 0) {
        const event = eventQueue.shift()!;
        try {
          await this.reportManager!.handleEvent(event);
        } catch (error) {
          console.error('Error handling event:', error);
        }
      }

      processing = false;
    };

    this.ipcManager.watchEvents((event: IPCEvent) => {
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
    
    // Build command using runner definition
    const modifiedCommand = runner.buildMainCommand(args, adapterPath);

    try {
      // Set THREEPIO_IPC_PATH for the child process
      const env = { ...process.env, THREEPIO_IPC_PATH: this.ipcPath };
      
      // Execute the command and pipe output to console
      const proc = $({ env })`sh -c ${modifiedCommand}`;
      proc.stdout.pipe(process.stdout);
      proc.stderr.pipe(process.stderr);
      
      const result = await proc;
      return 0;
    } catch (error: any) {
      // zx throws on non-zero exit codes
      return error.exitCode || 1;
    }
  }

  private async finalize(exitCode: number): Promise<void> {
    // Give adapters a small grace period to write final events
    // This prevents race conditions where the test runner exits before
    // the adapter has finished writing all events
    await new Promise(resolve => setTimeout(resolve, 500));

    if (this.reportManager) {
      await this.reportManager.finalize(exitCode);
      
      // Validate that we received test results
      const summary = this.reportManager.getSummary();
      if (summary.totalFiles === 0 && exitCode === 0) {
        console.error('\nWarning: No test results were captured. This may indicate:');
        console.error('- The test runner adapter failed to inject properly');
        console.error('- The test runner exited before results could be written');
        console.error('- An incompatibility with your test runner version');
        console.error('\nCheck the IPC file for debugging:', this.ipcPath);
      }
    }

    if (this.ipcManager) {
      await this.ipcManager.cleanup();
    }
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