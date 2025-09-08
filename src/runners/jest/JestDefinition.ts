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
      // For npm scripts, we'll use a direct jest command to avoid running tests twice
      const isNpmScript = args[0] === 'npm' || args[0] === 'yarn' || args[0] === 'pnpm';
      let dryRunCommand: string;
      
      if (isNpmScript) {
        // Instead of running npm test which might execute tests,
        // directly call jest with --listTests
        // We need to find jest in node_modules
        dryRunCommand = 'npx jest --listTests 2>/dev/null';
      } else {
        // For direct jest/npx jest commands
        dryRunCommand = args.join(' ') + ' --listTests';
      }
      
      const result = await $`sh -c ${dryRunCommand}`;
      
      // Jest outputs test file paths, one per line
      return result.stdout
        .split('\n')
        .filter(line => line.trim() && (line.endsWith('.test.js') || line.endsWith('.test.ts') || 
                                        line.endsWith('.spec.js') || line.endsWith('.spec.ts')));
    } catch {
      return [];
    }
  }
  
  buildMainCommand(args: string[], adapterPath: string): string[] {
    const hasReporters = args.some(arg => arg.includes('--reporters'));
    
    // For npm/yarn scripts, we need to use -- to pass arguments to the underlying command
    const isNpmScript = args[0] === 'npm' || args[0] === 'yarn' || args[0] === 'pnpm';
    
    if (hasReporters) {
      // User has already specified reporters, just add ours
      return [...args, adapterPath];
    } else if (isNpmScript) {
      // For npm scripts, we need to use -- to pass arguments to jest
      // Use ONLY our adapter to prevent duplicate output
      // e.g., npm test -> npm test -- --reporters /path/to/adapter
      const scriptIndex = args.findIndex(arg => arg === 'test' || arg.startsWith('test:'));
      if (scriptIndex !== -1) {
        return [
          ...args.slice(0, scriptIndex + 1),
          '--',  // This is crucial for npm to pass arguments to the underlying command
          '--reporters', adapterPath,
          ...args.slice(scriptIndex + 1)
        ];
      }
      // Fallback for other npm scripts
      return [...args, '--', '--reporters', adapterPath];
    } else {
      // For direct jest/npx jest commands
      return [...args, '--reporters', adapterPath];
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