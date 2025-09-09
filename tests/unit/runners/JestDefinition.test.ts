import { describe, it, expect } from 'vitest';
import { JestDefinition } from '../../../src/runners/jest/JestDefinition';

describe('JestDefinition', () => {
  describe('buildMainCommand', () => {
    const jestDef = new JestDefinition();
    const adapterPath = '/path/to/adapter.js';

    describe('npm run test commands', () => {
      it('should place --reporters AFTER test file paths', () => {
        const args = ['npm', 'run', 'test', './test/system/mcp-tools'];
        const result = jestDef.buildMainCommand(args, adapterPath);
        
        expect(result).toEqual([
          'npm', 'run', 'test',
          '--',
          './test/system/mcp-tools',  // test files FIRST
          '--reporters', adapterPath   // reporters LAST
        ]);
      });

      it('should handle multiple test file arguments', () => {
        const args = ['npm', 'run', 'test', 'file1.test.js', 'file2.test.js'];
        const result = jestDef.buildMainCommand(args, adapterPath);
        
        expect(result).toEqual([
          'npm', 'run', 'test',
          '--',
          'file1.test.js', 'file2.test.js',  // test files FIRST
          '--reporters', adapterPath          // reporters LAST
        ]);
      });

      it('should handle npm test (without run)', () => {
        const args = ['npm', 'test', './test/file.test.js'];
        const result = jestDef.buildMainCommand(args, adapterPath);
        
        expect(result).toEqual([
          'npm', 'test',
          '--',
          './test/file.test.js',      // test files FIRST
          '--reporters', adapterPath   // reporters LAST
        ]);
      });

      it('should handle when -- separator already exists', () => {
        const args = ['npm', 'run', 'test', '--', './test/file.test.js', '--coverage'];
        const result = jestDef.buildMainCommand(args, adapterPath);
        
        expect(result).toEqual([
          'npm', 'run', 'test', '--', './test/file.test.js', '--coverage',
          '--reporters', adapterPath   // reporters at the END
        ]);
      });

      it('should handle npm run test without file arguments', () => {
        const args = ['npm', 'run', 'test'];
        const result = jestDef.buildMainCommand(args, adapterPath);
        
        expect(result).toEqual([
          'npm', 'run', 'test',
          '--',
          '--reporters', adapterPath
        ]);
      });

      it('should handle custom test scripts', () => {
        const args = ['npm', 'run', 'test:unit', './src/utils'];
        const result = jestDef.buildMainCommand(args, adapterPath);
        
        expect(result).toEqual([
          'npm', 'run', 'test:unit',
          '--',
          './src/utils',              // test path FIRST
          '--reporters', adapterPath   // reporters LAST
        ]);
      });

      it('should prevent "Could not resolve a module for a custom reporter" error', () => {
        // This is the exact scenario from the error report
        const args = ['npm', 'run', 'test', './test/system/mcp-tools'];
        const result = jestDef.buildMainCommand(args, adapterPath);
        
        // The test file path MUST come before --reporters
        // Otherwise Jest will try to load './test/system/mcp-tools' as a reporter module
        const reportersIndex = result.indexOf('--reporters');
        const testPathIndex = result.indexOf('./test/system/mcp-tools');
        
        expect(testPathIndex).toBeGreaterThan(-1);
        expect(reportersIndex).toBeGreaterThan(-1);
        expect(testPathIndex).toBeLessThan(reportersIndex);
        
        // Also verify the exact structure
        expect(result).toEqual([
          'npm', 'run', 'test',
          '--',
          './test/system/mcp-tools',  // MUST be before --reporters
          '--reporters', adapterPath
        ]);
      });
    });

    describe('direct jest commands', () => {
      it('should append --reporters for direct jest command', () => {
        const args = ['jest', 'file.test.js'];
        const result = jestDef.buildMainCommand(args, adapterPath);
        
        expect(result).toEqual([
          'jest', 'file.test.js',
          '--reporters', adapterPath
        ]);
      });

      it('should append --reporters for npx jest command', () => {
        const args = ['npx', 'jest', '--coverage', 'file.test.js'];
        const result = jestDef.buildMainCommand(args, adapterPath);
        
        expect(result).toEqual([
          'npx', 'jest', '--coverage', 'file.test.js',
          '--reporters', adapterPath
        ]);
      });
    });

    describe('existing --reporters flag', () => {
      it('should only add adapter path when --reporters already exists', () => {
        const args = ['npm', 'run', 'test', '--', '--reporters', 'default'];
        const result = jestDef.buildMainCommand(args, adapterPath);
        
        expect(result).toEqual([
          'npm', 'run', 'test', '--', '--reporters', 'default',
          adapterPath
        ]);
      });
    });

    describe('yarn and pnpm support', () => {
      it('should handle yarn test commands', () => {
        const args = ['yarn', 'test', './test/file.test.js'];
        const result = jestDef.buildMainCommand(args, adapterPath);
        
        expect(result).toEqual([
          'yarn', 'test',
          '--',
          './test/file.test.js',
          '--reporters', adapterPath
        ]);
      });

      it('should handle pnpm test commands', () => {
        const args = ['pnpm', 'test', './test/file.test.js'];
        const result = jestDef.buildMainCommand(args, adapterPath);
        
        expect(result).toEqual([
          'pnpm', 'test',
          '--',
          './test/file.test.js',
          '--reporters', adapterPath
        ]);
      });
    });
  });

  describe('matches', () => {
    const jestDef = new JestDefinition();

    it('should match direct jest command', () => {
      expect(jestDef.matches(['jest'])).toBe(true);
    });

    it('should match npx jest command', () => {
      expect(jestDef.matches(['npx', 'jest'])).toBe(true);
    });

    it('should match yarn jest command', () => {
      expect(jestDef.matches(['yarn', 'jest'])).toBe(true);
    });

    it('should match npm test when package.json has jest', () => {
      const packageJson = JSON.stringify({
        scripts: { test: 'jest' }
      });
      expect(jestDef.matches(['npm', 'test'], packageJson)).toBe(true);
    });

    it('should match npm run test when package.json has jest', () => {
      const packageJson = JSON.stringify({
        scripts: { test: 'jest --coverage' }
      });
      expect(jestDef.matches(['npm', 'run', 'test'], packageJson)).toBe(true);
    });

    it('should not match vitest commands', () => {
      expect(jestDef.matches(['vitest'])).toBe(false);
      expect(jestDef.matches(['npx', 'vitest'])).toBe(false);
    });

    it('should handle malformed package.json gracefully', () => {
      expect(jestDef.matches(['npm', 'test'], 'invalid json')).toBe(false);
    });
  });
});