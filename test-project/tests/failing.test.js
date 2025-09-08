import { describe, it, expect } from 'vitest';

describe('Failing test suite', () => {
  it('should pass this test', () => {
    expect(true).toBe(true);
  });

  it('should fail this test intentionally', () => {
    console.error('This test is designed to fail!');
    expect(1 + 1).toBe(3);
  });

  it.skip('should skip this test', () => {
    expect(false).toBe(true);
  });
});