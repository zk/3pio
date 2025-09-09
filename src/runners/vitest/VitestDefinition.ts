import { TestRunnerDefinition } from '../base/TestRunnerDefinition';

export class VitestDefinition implements TestRunnerDefinition {
  name = 'vitest';
  
  matches(args: string[], packageJsonContent?: string): boolean {
    const command = args[0];
    
    // Direct detection
    if (command === 'vitest') return true;
    if ((command === 'npx' || command === 'yarn' || command === 'pnpm') && args[1] === 'vitest') {
      return true;
    }
    
    // npm script detection
    if ((command === 'npm' && (args[1] === 'test' || args[1] === 'run')) && packageJsonContent) {
      try {
        const packageJson = JSON.parse(packageJsonContent);
        
        const scriptName = args[1] === 'test' ? 'test' : args[2];
        const testScript = packageJson.scripts?.[scriptName];
        if (testScript?.includes('vitest')) return true;
        
        const deps = { ...packageJson.dependencies, ...packageJson.devDependencies };
        return !!deps.vitest;
      } catch {
        return false;
      }
    }
    
    return false;
  }
  
  async getTestFiles(args: string[]): Promise<string[]> {
    // Vitest list doesn't work reliably, so extract from args or return empty
    const testFileExtensions = ['.test.js', '.test.ts', '.test.mjs', '.test.jsx', '.test.tsx', '.spec.js', '.spec.ts', '.spec.mjs'];
    const providedFiles = args.filter(arg =>
      !arg.startsWith('-') && testFileExtensions.some(ext => arg.includes(ext))
    );
    
    return providedFiles;
  }
  
  buildMainCommand(args: string[], adapterPath: string): string[] {
    const hasReporter = args.some(arg => arg.includes('--reporter'));
    
    // Handle npm run commands - need to use -- separator
    if (args[0] === 'npm' && args[1] === 'run') {
      // Check if -- separator already exists
      const separatorIndex = args.indexOf('--');
      if (separatorIndex === -1) {
        // No separator, add it before reporter args
        if (hasReporter) {
          return [...args, '--', '--reporter', adapterPath];
        } else {
          return [...args, '--', '--reporter', 'default', '--reporter', adapterPath];
        }
      } else {
        // Separator exists, add reporter args after it
        if (hasReporter) {
          return [...args, '--reporter', adapterPath];
        } else {
          return [...args, '--reporter', 'default', '--reporter', adapterPath];
        }
      }
    }
    
    // Direct vitest or npx vitest commands
    if (hasReporter) {
      return [...args, '--reporter', adapterPath];
    } else {
      return [...args, '--reporter', 'default', '--reporter', adapterPath];
    }
  }
  
  getAdapterFileName(): string {
    return 'vitest.js';
  }
  
  interpretExitCode(exitCode: number): 'success' | 'test-failure' | 'system-error' {
    switch (exitCode) {
      case 0: return 'success';
      case 1: return 'test-failure';
      default: return 'system-error';
    }
  }
}