import { describe, it, expect, beforeAll } from 'vitest';
import { execSync } from 'child_process';
import * as path from 'path';
import * as fs from 'fs';

describe('Console Output System Test', () => {
  const testProjectDir = path.join(__dirname, 'jest-project');
  const cliPath = path.resolve(__dirname, '../../../dist/cli.js');
  
  beforeAll(() => {
    // Ensure the CLI is built
    if (!fs.existsSync(cliPath)) {
      execSync('npm run build', { cwd: path.resolve(__dirname, '../../..') });
    }
    
    // Copy sample jest project files to test directory
    const sourceDir = path.resolve(__dirname, '../../../sample-projects/jest-project');
    
    // Copy package.json
    fs.copyFileSync(
      path.join(sourceDir, 'package.json'),
      path.join(testProjectDir, 'package.json')
    );
    
    // Copy test files
    fs.copyFileSync(
      path.join(sourceDir, 'math.test.js'),
      path.join(testProjectDir, 'math.test.js')
    );
    
    fs.copyFileSync(
      path.join(sourceDir, 'string.test.js'),
      path.join(testProjectDir, 'string.test.js')
    );
    
    // Install dependencies
    execSync('npm install', { cwd: testProjectDir, stdio: 'ignore' });
  });
  
  it('should produce expected console output format', () => {
    // Run the CLI in the test project directory (expect it to fail since tests fail)
    let output: string;
    try {
      output = execSync(`node ${cliPath} npm test`, {
        cwd: testProjectDir,
        encoding: 'utf-8',
        env: { ...process.env, NO_COLOR: '1' }
      });
    } catch (error: any) {
      // The command will fail because tests fail, but we still want the output
      output = error.stdout;
    }
    
    // Replace dynamic timestamp with placeholder for comparison
    const normalizedOutput = output.replace(/\d{4}-\d{2}-\d{2}T\d{6,9}Z/g, 'TIMESTAMP');
    
    const expectedOutput = `Greetings! I will now execute the test command:
\`npm test\`

Full report: .3pio/runs/TIMESTAMP/test-run.md

The following 2 test files will be run:
- /Users/zk/code/3pio/tests/system/console-output/jest-project/string.test.js
- /Users/zk/code/3pio/tests/system/console-output/jest-project/math.test.js

Beginning test execution now...

RUNNING  ./string.test.js
FAIL     ./string.test.js
  String operations
    ✕ should fail this test (3 ms)
  String operations
    ✕ should skip this test (0 ms)
  See .3pio/runs/TIMESTAMP/logs/math.test.js.log
    
RUNNING  ./math.test.js
PASS     ./math.test.js

Test Suites: 1 failed,  1 passed, 2 total
Tests: 1 failed,  1 skipped,  5 passed, 7 total
Snapshots:   0 total
Time:        0.098s`;
    
    expect(normalizedOutput).toBe(expectedOutput);
  });
});