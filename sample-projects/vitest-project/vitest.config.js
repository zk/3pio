import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    include: ['**/*.test.js', '**/*.spec.js'],
    exclude: ['node_modules/**', 'dist/**']
  }
});