import { defineConfig } from 'vitest/config';
import path from 'path';

export default defineConfig({
  test: {
    globals: true,
    environment: 'node',
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      exclude: [
        'node_modules/**',
        'dist/**',
        'tests/**',
        'test-project/**',
        '*.config.ts',
        'build.js'
      ]
    },
    testTimeout: 10000,
    hookTimeout: 10000,
    include: [
      'tests/**/*.test.ts',
      'tests/**/*.test.js',
      'tests/**/*.spec.ts',
      'tests/**/*.spec.js'
    ],
    exclude: [
      'node_modules/**',
      'dist/**',
      'test-project/**',
      'tests/system/console-output/jest-project/**'
    ]
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src')
    }
  }
});