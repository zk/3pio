import { TestRunnerDefinition } from '../base/TestRunnerDefinition';
import * as path from 'path';

export class PyTestDefinition implements TestRunnerDefinition {
  name = 'pytest';
  
  matches(args: string[], packageJsonContent?: string): boolean {
    const command = args[0];
    
    // Direct pytest detection
    if (command === 'pytest') return true;
    
    // Python module execution: python -m pytest
    if (command === 'python' || command === 'python3') {
      if (args[1] === '-m' && args[2] === 'pytest') {
        return true;
      }
    }
    
    // py.test (alternative pytest command)
    if (command === 'py.test') return true;
    
    // npm script detection for projects that use npm to run Python tests
    if ((command === 'npm' && (args[1] === 'test' || args[1] === 'run')) && packageJsonContent) {
      try {
        const packageJson = JSON.parse(packageJsonContent);
        const scriptName = args[1] === 'test' ? 'test' : args[2];
        const testScript = packageJson.scripts?.[scriptName];
        if (testScript?.includes('pytest') || testScript?.includes('py.test')) {
          return true;
        }
      } catch {
        return false;
      }
    }
    
    return false;
  }
  
  async getTestFiles(args: string[]): Promise<string[]> {
    // Extract test files from command arguments (like Vitest approach)
    // pytest follows these patterns: test_*.py, *_test.py, or files explicitly passed
    const pythonTestPatterns = [
      /test_.*\.py$/,
      /.*_test\.py$/,
      /tests?\/.*\.py$/  // Files in test/tests directories
    ];
    
    const providedFiles = args.filter(arg => {
      // Skip flags and options
      if (arg.startsWith('-')) return false;
      
      // Check if it's a Python file
      if (arg.endsWith('.py')) {
        return true;
      }
      
      // Check if it matches test patterns
      return pythonTestPatterns.some(pattern => pattern.test(arg));
    });
    
    return providedFiles;
  }
  
  buildMainCommand(args: string[], adapterPath: string): string[] {
    // For pytest, we use the -p flag with the module name (without .py)
    // The CLI will add the dist directory to PYTHONPATH so it can be imported
    const adapterModuleName = path.basename(adapterPath, '.py');
    
    // Handle different command patterns
    const command = args[0];
    
    if (command === 'npm' && args[1] === 'run') {
      // npm run test -- -p module_name [other args]
      const separatorIndex = args.indexOf('--');
      
      if (separatorIndex !== -1) {
        // Insert plugin flag after separator
        return [
          ...args.slice(0, separatorIndex + 1),
          '-p', adapterModuleName,
          ...args.slice(separatorIndex + 1)
        ];
      } else {
        // Add separator and plugin flag
        return [...args, '--', '-p', adapterModuleName];
      }
    } else if (command === 'python' || command === 'python3') {
      // python -m pytest -p module_name [other args]
      // Insert after 'pytest' but before other arguments
      const pytestIndex = args.indexOf('pytest');
      if (pytestIndex !== -1) {
        return [
          ...args.slice(0, pytestIndex + 1),
          '-p', adapterModuleName,
          ...args.slice(pytestIndex + 1)
        ];
      }
      // Fallback: append at end
      return [...args, '-p', adapterModuleName];
    } else {
      // Direct pytest command: pytest -p module_name [other args]
      // Insert plugin flag early in the command
      return [
        args[0],  // pytest
        '-p', adapterModuleName,
        ...args.slice(1)
      ];
    }
  }
  
  getAdapterFileName(): string {
    // Return the Python adapter filename
    return 'pytest_adapter.py';
  }
  
  interpretExitCode(exitCode: number): 'success' | 'test-failure' | 'system-error' {
    // pytest exit codes:
    // 0: All tests passed
    // 1: Tests were collected and run but some failed
    // 2: Test execution was interrupted
    // 3: Internal error occurred
    // 4: pytest command line usage error
    // 5: No tests were collected
    
    switch (exitCode) {
      case 0: 
        return 'success';
      case 1: 
        return 'test-failure';
      case 5:
        // No tests collected is considered a system error
        return 'system-error';
      default: 
        return 'system-error';
    }
  }
}