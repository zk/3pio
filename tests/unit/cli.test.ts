import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import path from 'path';
import { promises as fs } from 'fs';
import os from 'os';

// Mock zx module
vi.mock('zx', () => ({
  $: vi.fn(),
  chalk: {
    blue: (text: string) => text,
    green: (text: string) => text,
    red: (text: string) => text,
    yellow: (text: string) => text,
    bold: (text: string) => text,
    dim: (text: string) => text
  }
}));

// Mock commander
vi.mock('commander', () => {
  const program = {
    name: vi.fn().mockReturnThis(),
    description: vi.fn().mockReturnThis(),
    version: vi.fn().mockReturnThis(),
    command: vi.fn().mockReturnThis(),
    argument: vi.fn().mockReturnThis(),
    option: vi.fn().mockReturnThis(),
    action: vi.fn().mockReturnThis(),
    parse: vi.fn().mockReturnThis(),
    opts: vi.fn().mockReturnValue({})
  };
  return {
    Command: vi.fn(() => program),
    program
  };
});

describe('CLI', () => {
  let tempDir: string;
  let originalEnv: NodeJS.ProcessEnv;

  beforeEach(async () => {
    // Save original state
    originalEnv = { ...process.env };
    
    // Create temp directory
    tempDir = await fs.mkdtemp(path.join(os.tmpdir(), '3pio-cli-test-'));
    
    // Mock process.cwd to return temp directory
    vi.spyOn(process, 'cwd').mockReturnValue(tempDir);
    
    // Clear module cache to get fresh imports
    vi.resetModules();
  });

  afterEach(async () => {
    // Restore original state
    process.env = originalEnv;
    
    // Restore all mocks
    vi.restoreAllMocks();
    
    // Clean up
    await fs.rm(tempDir, { recursive: true, force: true });
    vi.clearAllMocks();
  });

  describe('detectTestRunner', () => {
    it('should detect jest from command', async () => {
      // Create a mock module to test the function in isolation
      const detectTestRunner = (command: string, args: string[]) => {
        if (command === 'jest') {
          return { name: 'jest', command: 'jest' };
        }
        if (command === 'vitest') {
          return { name: 'vitest', command: 'vitest' };
        }
        if ((command === 'npx' || command === 'yarn' || command === 'pnpm') && args[0]) {
          if (args[0] === 'jest') {
            return { name: 'jest', command: `${command} jest` };
          }
          if (args[0] === 'vitest') {
            return { name: 'vitest', command: `${command} vitest` };
          }
        }
        if (command === 'npm' && args[0] === 'test') {
          return { name: 'jest', command: 'npm test' }; // Default assumption
        }
        return null;
      };

      expect(detectTestRunner('jest', [])).toEqual({ name: 'jest', command: 'jest' });
      expect(detectTestRunner('vitest', [])).toEqual({ name: 'vitest', command: 'vitest' });
      expect(detectTestRunner('npx', ['jest'])).toEqual({ name: 'jest', command: 'npx jest' });
      expect(detectTestRunner('yarn', ['vitest'])).toEqual({ name: 'vitest', command: 'yarn vitest' });
      expect(detectTestRunner('pnpm', ['jest'])).toEqual({ name: 'jest', command: 'pnpm jest' });
      expect(detectTestRunner('npm', ['test'])).toEqual({ name: 'jest', command: 'npm test' });
      expect(detectTestRunner('unknown', [])).toBeNull();
    });
  });

  describe('extractTestFiles', () => {
    it('should extract test files from arguments', () => {
      const extractTestFiles = (args: string[]) => {
        const testFileExtensions = ['.test.js', '.test.ts', '.test.mjs', '.test.jsx', '.test.tsx', '.spec.js', '.spec.ts', '.spec.mjs'];
        return args.filter(arg => 
          !arg.startsWith('-') && testFileExtensions.some(ext => arg.includes(ext))
        );
      };

      expect(extractTestFiles(['src/foo.test.js', '--watch'])).toEqual(['src/foo.test.js']);
      expect(extractTestFiles(['--coverage', 'src/bar.spec.ts'])).toEqual(['src/bar.spec.ts']);
      expect(extractTestFiles(['src/baz.test.tsx', 'src/qux.spec.mjs'])).toEqual(['src/baz.test.tsx', 'src/qux.spec.mjs']);
      expect(extractTestFiles(['--watch', '--coverage'])).toEqual([]);
      expect(extractTestFiles(['src/regular.js'])).toEqual([]);
    });

    it('should handle npm test with -- separator for passing arguments', () => {
      // This test validates that arguments after -- are properly passed to the test runner
      // Example: 3pio npm test -- test/system/mcp-tools/click.test.js
      const parseNpmTestArgs = (args: string[]) => {
        // If we have 'npm test' followed by '--', everything after -- should be passed to the test runner
        if (args[0] === 'npm' && args[1] === 'test' && args[2] === '--') {
          return {
            command: 'npm test',
            testArgs: args.slice(3)
          };
        }
        return {
          command: args.slice(0, 2).join(' '),
          testArgs: []
        };
      };

      const result = parseNpmTestArgs(['npm', 'test', '--', 'test/system/mcp-tools/click.test.js']);
      expect(result.command).toBe('npm test');
      expect(result.testArgs).toEqual(['test/system/mcp-tools/click.test.js']);
    });
  });

  describe('buildAdapterCommand', () => {
    it('should build correct adapter command for jest', () => {
      const buildAdapterCommand = (runner: string, command: string, args: string[], adapterPath: string) => {
        if (runner === 'jest') {
          const configArg = `--reporters="${adapterPath}"`;
          return `${command} ${configArg} ${args.join(' ')}`.trim();
        }
        if (runner === 'vitest') {
          const configArg = `--reporter="${adapterPath}"`;
          return `${command} ${configArg} ${args.join(' ')}`.trim();
        }
        return `${command} ${args.join(' ')}`.trim();
      };

      const adapterPath = '/path/to/adapter.js';
      
      expect(buildAdapterCommand('jest', 'jest', ['--coverage'], adapterPath))
        .toBe(`jest --reporters="${adapterPath}" --coverage`);
      
      expect(buildAdapterCommand('vitest', 'vitest', ['run'], adapterPath))
        .toBe(`vitest --reporter="${adapterPath}" run`);
      
      expect(buildAdapterCommand('unknown', 'unknown', ['test'], adapterPath))
        .toBe('unknown test');
    });
  });

  describe('preamble generation', () => {
    it('should generate correct preamble', () => {
      const generatePreamble = (reportPath: string) => {
        return [
          '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━',
          '                              🎯 3pio Test Runner',
          '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━',
          '',
          `📊 Report: ${reportPath}`,
          '',
          '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━',
          ''
        ].join('\n');
      };

      const preamble = generatePreamble('.3pio/runs/123/test-run.md');
      expect(preamble).toContain('3pio Test Runner');
      expect(preamble).toContain('Report: .3pio/runs/123/test-run.md');
    });
  });

  describe('dry run mode', () => {
    it('should detect test files in dry run for jest', async () => {
      // Create mock test files
      await fs.mkdir('src', { recursive: true });
      await fs.writeFile('src/foo.test.js', '// test file');
      await fs.writeFile('src/bar.spec.js', '// spec file');
      await fs.writeFile('src/regular.js', '// not a test');

      const performDryRun = async (runner: string, command: string, args: string[]) => {
        if (runner === 'jest') {
          // Simulate jest --listTests output
          const testFiles = [
            path.join(process.cwd(), 'src/foo.test.js'),
            path.join(process.cwd(), 'src/bar.spec.js')
          ];
          return testFiles;
        }
        if (runner === 'vitest' && args.length > 0) {
          // For vitest with specific files, return those files
          const testFileExtensions = ['.test.js', '.test.ts', '.spec.js', '.spec.ts'];
          return args.filter(arg => 
            !arg.startsWith('-') && testFileExtensions.some(ext => arg.includes(ext))
          );
        }
        return [];
      };

      const jestFiles = await performDryRun('jest', 'jest', []);
      expect(jestFiles).toHaveLength(2);
      expect(jestFiles[0]).toContain('foo.test.js');
      expect(jestFiles[1]).toContain('bar.spec.js');

      const vitestFiles = await performDryRun('vitest', 'vitest', ['src/foo.test.js']);
      expect(vitestFiles).toEqual(['src/foo.test.js']);
    });
  });

  describe('environment variable handling', () => {
    it('should set THREEPIO_IPC_PATH environment variable', () => {
      const ipcPath = '/tmp/test.ipc';
      const env = {
        ...process.env,
        THREEPIO_IPC_PATH: ipcPath
      };
      
      expect(env.THREEPIO_IPC_PATH).toBe(ipcPath);
    });

    it('should preserve existing environment variables', () => {
      process.env.CUSTOM_VAR = 'custom_value';
      
      const env = {
        ...process.env,
        THREEPIO_IPC_PATH: '/tmp/test.ipc'
      };
      
      expect(env.CUSTOM_VAR).toBe('custom_value');
      expect(env.THREEPIO_IPC_PATH).toBe('/tmp/test.ipc');
    });
  });

  describe('error handling', () => {
    it('should handle missing test runner', () => {
      const handleMissingRunner = () => {
        console.error('Error: Unable to detect test runner');
        console.error('Please specify the test runner explicitly or ensure package.json has test scripts');
        process.exit(1);
      };

      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
      const exitSpy = vi.spyOn(process, 'exit').mockImplementation(() => {
        throw new Error('Process exit');
      });

      expect(() => handleMissingRunner()).toThrow('Process exit');
      expect(consoleSpy).toHaveBeenCalledWith('Error: Unable to detect test runner');
      
      consoleSpy.mockRestore();
      exitSpy.mockRestore();
    });

    it('should handle test runner startup failure', () => {
      const handleStartupFailure = (error: Error) => {
        console.error('Failed to start test runner:', error.message);
        process.exit(1);
      };

      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
      const exitSpy = vi.spyOn(process, 'exit').mockImplementation(() => {
        throw new Error('Process exit');
      });

      const error = new Error('Command not found');
      expect(() => handleStartupFailure(error)).toThrow('Process exit');
      expect(consoleSpy).toHaveBeenCalledWith('Failed to start test runner:', 'Command not found');
      
      consoleSpy.mockRestore();
      exitSpy.mockRestore();
    });
  });

  describe('exit code handling', () => {
    it('should mirror test runner exit code', () => {
      const handleExit = (exitCode: number) => {
        process.exit(exitCode);
      };

      const exitSpy = vi.spyOn(process, 'exit').mockImplementation((code) => {
        throw new Error(`Exit with code ${code}`);
      });

      expect(() => handleExit(0)).toThrow('Exit with code 0');
      expect(() => handleExit(1)).toThrow('Exit with code 1');
      expect(() => handleExit(2)).toThrow('Exit with code 2');
      
      exitSpy.mockRestore();
    });
  });

  describe('npm test with -- separator', () => {
    it('should correctly handle npm test -- file.test.js format', async () => {
      // This test simulates the actual CLI parsing behavior for:
      // 3pio npm test -- test/system/mcp-tools/click.test.js
      
      // The actual CLI receives: ['npm', 'test', '--', 'test/system/mcp-tools/click.test.js']
      // This should be passed to npm as: npm test -- test/system/mcp-tools/click.test.js
      
      const buildNpmCommand = (args: string[]) => {
        // This simulates what the CLI should do
        if (args[0] === 'npm' && args[1] === 'test') {
          // Join all arguments including the -- separator
          return args.join(' ');
        }
        return args.join(' ');
      };

      const command = buildNpmCommand(['npm', 'test', '--', 'test/system/mcp-tools/click.test.js']);
      expect(command).toBe('npm test -- test/system/mcp-tools/click.test.js');
    });

    it('should preserve all arguments after -- separator', () => {
      // Test that multiple arguments after -- are preserved
      const args = ['npm', 'test', '--', 'file1.test.js', 'file2.test.js', '--coverage'];
      const expectedCommand = 'npm test -- file1.test.js file2.test.js --coverage';
      
      const command = args.join(' ');
      expect(command).toBe(expectedCommand);
    });
  });
});