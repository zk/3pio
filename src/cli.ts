#!/usr/bin/env node

import { Command } from 'commander';
import { $ } from 'zx';
import { promises as fs } from 'fs';
import path from 'path';
import { IPCManager } from './ipc';
import { ReportManager } from './ReportManager';
import { IPCEvent } from './types/events';

// Disable zx verbosity
$.verbose = false;

interface TestRunner {
  name: 'jest' | 'vitest' | 'unknown';
  command: string;
}

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
      const runner = await this.detectTestRunner(commandArgs);
      
      if (runner.name === 'unknown') {
        console.error('Oh dear! I cannot determine which test runner you are using.');
        console.error('3pio currently supports Jest and Vitest.');
        process.exit(1);
      }

      // Perform dry run to get test files
      const testFiles = await this.performDryRun(runner, commandArgs);
      
      if (testFiles.length === 0) {
        console.log('No test files found to run. Proceeding without file list...');
        // Continue anyway - the test runner will handle finding files
      }

      // Initialize IPC and Report
      await this.initialize(testCommand, testFiles);

      // Print preamble
      this.printPreamble(testFiles);

      // Start IPC listening
      this.startIPCListening();

      // Execute main command with adapter injected
      const exitCode = await this.executeMainCommand(runner, commandArgs);

      // Finalize
      await this.finalize(exitCode);
      
      process.exit(exitCode);
    } catch (error) {
      console.error('Oh no! A catastrophic error occurred:', error);
      process.exit(1);
    }
  }

  private async detectTestRunner(args: string[]): Promise<TestRunner> {
    const command = args[0];
    
    // Direct runner commands
    if (command === 'jest') {
      return { name: 'jest', command: 'jest' };
    }
    if (command === 'vitest') {
      return { name: 'vitest', command: 'vitest' };
    }
    
    // npx/yarn/pnpm commands
    if ((command === 'npx' || command === 'yarn' || command === 'pnpm') && args[1]) {
      if (args[1] === 'jest') {
        return { name: 'jest', command: args.slice(0, 2).join(' ') };
      }
      if (args[1] === 'vitest') {
        return { name: 'vitest', command: args.slice(0, 2).join(' ') };
      }
    }

    // NPM scripts
    if (command === 'npm' && (args[1] === 'test' || args[1] === 'run')) {
      try {
        const packageJson = JSON.parse(
          await fs.readFile(path.join(process.cwd(), 'package.json'), 'utf8')
        );

        // Check scripts
        const scriptName = args[1] === 'test' ? 'test' : args[2];
        const script = packageJson.scripts?.[scriptName];
        
        if (script) {
          if (script.includes('jest')) {
            return { name: 'jest', command: script };
          }
          if (script.includes('vitest')) {
            return { name: 'vitest', command: script };
          }
        }

        // Check dependencies
        const deps = {
          ...packageJson.dependencies,
          ...packageJson.devDependencies
        };

        if (deps.jest) return { name: 'jest', command: script || 'jest' };
        if (deps.vitest) return { name: 'vitest', command: script || 'vitest' };
      } catch (error) {
        // Package.json not found or invalid
      }
    }

    return { name: 'unknown', command: args[0] };
  }

  private async performDryRun(
    runner: TestRunner,
    args: string[]
  ): Promise<string[]> {
    // Check if specific test files are provided
    const testFileExtensions = ['.test.js', '.test.ts', '.test.mjs', '.test.jsx', '.test.tsx', '.spec.js', '.spec.ts', '.spec.mjs'];
    const providedFiles = args.filter(arg => 
      !arg.startsWith('-') && testFileExtensions.some(ext => arg.includes(ext))
    );
    
    if (providedFiles.length > 0) {
      // User provided specific test files, use those
      return providedFiles;
    }

    try {
      let dryRunCommand: string;
      
      if (runner.name === 'jest') {
        // For Jest, use --listTests
        dryRunCommand = args.join(' ') + ' --listTests';
      } else if (runner.name === 'vitest') {
        // For Vitest, we can't easily list files without running them
        // So we'll just return empty and let the adapter handle it
        console.log('Note: Vitest dry run not available. Proceeding...');
        return [];
      } else {
        return [];
      }

      // Execute dry run
      const result = await $`sh -c ${dryRunCommand}`;
      
      // Parse output
      if (runner.name === 'jest') {
        // Jest outputs JSON array
        try {
          return JSON.parse(result.stdout);
        } catch {
          // Fallback to line parsing
          return result.stdout
            .split('\n')
            .filter(line => line.trim() && line.endsWith('.test.js') || line.endsWith('.test.ts'));
        }
      } else {
        return [];
      }
    } catch (error) {
      console.error('Warning: Dry run failed. Proceeding optimistically...');
      return [];
    }
  }

  private async initialize(testCommand: string, testFiles: string[]): Promise<void> {
    // Create IPC directory and file
    const ipcDir = await IPCManager.ensureIPCDirectory();
    this.ipcPath = path.join(ipcDir, `${this.runId}.jsonl`);
    
    // Set environment variable for adapters
    process.env.THREEPIO_IPC_PATH = this.ipcPath;
    
    // Initialize managers
    this.ipcManager = new IPCManager(this.ipcPath);
    this.reportManager = new ReportManager(this.runId, testCommand);
    
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

    this.ipcManager.watchEvents((event: IPCEvent) => {
      this.reportManager!.handleEvent(event).catch(console.error);
    });
  }

  private async executeMainCommand(
    runner: TestRunner,
    args: string[]
  ): Promise<number> {
    let modifiedCommand: string;
    // Use absolute path to the adapter
    const adapterPath = runner.name === 'jest' 
      ? path.join(__dirname, 'jest.js')
      : path.join(__dirname, 'vitest.js');

    if (runner.name === 'jest') {
      // Inject Jest reporter
      const hasReporters = args.some(arg => arg.includes('--reporters'));
      if (hasReporters) {
        // Append to existing reporters
        modifiedCommand = args.join(' ') + ` ${adapterPath}`;
      } else {
        // Add default and 3pio reporters
        modifiedCommand = args.join(' ') + ` --reporters default --reporters ${adapterPath}`;
      }
    } else if (runner.name === 'vitest') {
      // Inject Vitest reporter
      const hasReporter = args.some(arg => arg.includes('--reporter'));
      if (hasReporter) {
        // Append to existing reporters
        modifiedCommand = args.join(' ') + ` --reporter ${adapterPath}`;
      } else {
        // Add default and 3pio reporters
        modifiedCommand = args.join(' ') + ` --reporter default --reporter ${adapterPath}`;
      }
    } else {
      modifiedCommand = args.join(' ');
    }

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
    if (this.reportManager) {
      await this.reportManager.finalize(exitCode);
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