import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    // Look for tests in all packages
    include: ['packages/*/**.test.js'],
  },
});