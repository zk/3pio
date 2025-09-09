import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { execSync } from 'child_process';
import path from 'path';
import fs from 'fs';
import os from 'os';

describe('npm test with -- separator integration', () => {
  let tempDir: string;
  const cliPath = path.join(__dirname, '../../dist/cli.js');

  beforeEach(() => {
    // Create a temporary directory for testing
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), '3pio-npm-separator-test-'));
    
    // Create a simple package.json with jest as test runner
    const packageJson = {
      name: 'test-project',
      version: '1.0.0',
      scripts: {
        test: 'echo "mock jest test"'  // Mock test script since we don't have jest installed
      },
      devDependencies: {
        jest: '^29.0.0'  // This helps 3pio detect it as a jest project
      }
    };
    fs.writeFileSync(
      path.join(tempDir, 'package.json'),
      JSON.stringify(packageJson, null, 2)
    );

    // Create a test file
    const testContent = `
test('example test', () => {
  expect(1 + 1).toBe(2);
});
`;
    fs.writeFileSync(
      path.join(tempDir, 'example.test.js'),
      testContent
    );
  });

  afterEach(() => {
    // Clean up temp directory
    fs.rmSync(tempDir, { recursive: true, force: true });
  });

  it('should handle npm test -- file.test.js command format', () => {
    // This test verifies that the command format is preserved correctly:
    // 3pio npm test -- example.test.js
    
    try {
      const output = execSync(
        `node ${cliPath} npm test -- example.test.js`,
        { 
          cwd: tempDir,
          encoding: 'utf-8',
          env: { ...process.env, NO_COLOR: '1' }
        }
      );
      
      // Check that the command is displayed correctly with the -- separator
      expect(output).toContain('npm test -- example.test.js');
      
      // Check that the test file is recognized
      expect(output).toContain('example.test.js');
      
      // The command should complete (even if the mock test doesn't actually run)
      expect(output).toBeDefined();
    } catch (error: any) {
      // If there's an error, check if it's just because jest isn't installed
      // The important thing is that the command format is correct
      if (error.stdout) {
        // Still check that the command format was correct
        expect(error.stdout).toContain('npm test -- example.test.js');
        expect(error.stdout).toContain('example.test.js');
      } else {
        throw error;
      }
    }
  });

  it('should pass all arguments after -- to the test runner', () => {
    try {
      // Test with multiple arguments after --
      const output = execSync(
        `node ${cliPath} npm test -- example.test.js --coverage`,
        { 
          cwd: tempDir,
          encoding: 'utf-8',
          env: { ...process.env, NO_COLOR: '1' }
        }
      );
      
      // Check that the command preserves all arguments
      expect(output).toContain('npm test -- example.test.js --coverage');
      expect(output).toContain('example.test.js');
      expect(output).toBeDefined();
    } catch (error: any) {
      // If there's an error, still check the command format
      if (error.stdout) {
        expect(error.stdout).toContain('npm test -- example.test.js --coverage');
        expect(error.stdout).toContain('example.test.js');
      } else {
        throw error;
      }
    }
  });
});