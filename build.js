const esbuild = require('esbuild');
const path = require('path');
const fs = require('fs');

// Check for watch mode
const isWatch = process.argv.includes('--watch');

// Common build options
const commonOptions = {
  bundle: true,
  platform: 'node',
  target: 'node18',
  sourcemap: true,
  external: [
    'jest',
    'vitest',
    '@jest/reporters',
    'chokidar',
    'commander',
    'lodash.debounce',
    'zx'
  ],
  logLevel: 'info',
};

// Options for adapters that need to be self-contained
const adapterOptions = {
  bundle: true,
  platform: 'node',
  target: 'node18',
  sourcemap: true,
  external: [
    'jest',
    'vitest',
    '@jest/reporters',
    'fsevents'  // Native module, can't be bundled
  ],
  logLevel: 'info',
};

// Build configurations
const builds = [
  {
    entryPoints: ['src/cli.ts'],
    outfile: 'dist/cli.js',
    format: 'cjs',
    options: commonOptions
  },
  {
    entryPoints: ['src/adapters/jest.ts'],
    outfile: 'dist/jest.js',
    format: 'cjs',
    options: adapterOptions
  },
  {
    entryPoints: ['src/adapters/vitest.ts'],
    outfile: 'dist/vitest.js',
    format: 'esm',
    options: adapterOptions
  },
  {
    entryPoints: ['src/index.ts'],
    outfile: 'dist/index.js',
    format: 'cjs',
    options: commonOptions
  }
];

// Create index.ts if it doesn't exist
if (!fs.existsSync('src/index.ts')) {
  fs.writeFileSync('src/index.ts', `
export { IPCManager } from './ipc';
export { ReportManager } from './ReportManager';
export * from './types/events';
`);
}

async function build() {
  // Clean dist directory
  if (fs.existsSync('dist')) {
    fs.rmSync('dist', { recursive: true });
  }
  fs.mkdirSync('dist');

  // Build all targets
  for (const config of builds) {
    const baseOptions = config.options || commonOptions;
    const options = { 
      ...baseOptions, 
      entryPoints: config.entryPoints,
      outfile: config.outfile,
      format: config.format
    };

    if (isWatch) {
      const ctx = await esbuild.context(options);
      await ctx.watch();
      console.log(`Watching ${config.entryPoints[0]}...`);
    } else {
      await esbuild.build(options);
      console.log(`Built ${config.outfile}`);
    }
  }

  // Add shebang and make CLI executable
  if (!isWatch) {
    const cliPath = 'dist/cli.js';
    const cliContent = fs.readFileSync(cliPath, 'utf8');
    // Only add shebang if not already present
    if (!cliContent.startsWith('#!/usr/bin/env node')) {
      fs.writeFileSync(cliPath, '#!/usr/bin/env node\n' + cliContent);
    }
    fs.chmodSync(cliPath, '755');
  }

  // Copy Python adapter to dist
  const pythonAdapterSrc = 'src/adapters/pytest/pytest_adapter.py';
  const pythonAdapterDest = 'dist/pytest_adapter.py';
  if (fs.existsSync(pythonAdapterSrc)) {
    fs.copyFileSync(pythonAdapterSrc, pythonAdapterDest);
    fs.chmodSync(pythonAdapterDest, '755');
    console.log('Copied Python pytest adapter');
  }

  // Create package.json exports map for adapters
  const packageJsonPath = path.join(__dirname, 'package.json');
  const packageJson = JSON.parse(fs.readFileSync(packageJsonPath, 'utf8'));
  
  packageJson.exports = {
    '.': './dist/index.js',
    './jest': './dist/jest.js',
    './vitest': './dist/vitest.js'
  };
  
  fs.writeFileSync(packageJsonPath, JSON.stringify(packageJson, null, 2));
  
  console.log(isWatch ? 'Watching for changes...' : 'Build complete!');
}

build().catch(error => {
  console.error('Build failed:', error);
  process.exit(1);
});