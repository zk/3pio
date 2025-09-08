import { $ } from 'zx';
import { TestRunnerDefinition } from '../base/TestRunnerDefinition';

export class JestDefinition implements TestRunnerDefinition {
  name = 'jest';
  
  matches(args: string[], packageJsonContent?: string): boolean {
    const command = args[0];
    
    // Direct detection from command
    if (command === 'jest') return true;
    if ((command === 'npx' || command === 'yarn' || command === 'pnpm') && args[1] === 'jest') {
      return true;
    }
    
    // npm script detection
    if ((command === 'npm' && (args[1] === 'test' || args[1] === 'run')) && packageJsonContent) {
      try {
        const packageJson = JSON.parse(packageJsonContent);
        
        // Check test script
        const scriptName = args[1] === 'test' ? 'test' : args[2];
        const testScript = packageJson.scripts?.[scriptName];
        if (testScript?.includes('jest')) return true;
        
        // Check dependencies
        const deps = { ...packageJson.dependencies, ...packageJson.devDependencies };
        return !!deps.jest;
      } catch {
        return false;
      }
    }
    
    return false;
  }
  
  async getTestFiles(args: string[]): Promise<string[]> {
    // Check if specific test files provided
    const testFileExtensions = ['.test.js', '.test.ts', '.test.mjs', '.test.jsx', '.test.tsx', '.spec.js', '.spec.ts', '.spec.mjs'];
    const providedFiles = args.filter(arg => 
      !arg.startsWith('-') && testFileExtensions.some(ext => arg.includes(ext))
    );
    
    if (providedFiles.length > 0) {
      return providedFiles;
    }
    
    try {
      // Jest dry run with --listTests
      const dryRunCommand = args.join(' ') + ' --listTests';
      const result = await $`sh -c ${dryRunCommand}`;
      
      // Jest outputs JSON array
      try {
        return JSON.parse(result.stdout);
      } catch {
        // Fallback to line parsing
        return result.stdout
          .split('\n')
          .filter(line => line.trim() && (line.endsWith('.test.js') || line.endsWith('.test.ts')));
      }
    } catch {
      return [];
    }
  }
  
  buildMainCommand(args: string[], adapterPath: string): string[] {
    const hasReporters = args.some(arg => arg.includes('--reporters'));
    if (hasReporters) {
      return [...args, adapterPath];
    } else {
      return [...args, '--reporters', 'default', '--reporters', adapterPath];
    }
  }
  
  getAdapterFileName(): string {
    return 'jest.js';
  }
  
  interpretExitCode(exitCode: number): 'success' | 'test-failure' | 'system-error' {
    switch (exitCode) {
      case 0: return 'success';
      case 1: return 'test-failure';
      default: return 'system-error';
    }
  }
}