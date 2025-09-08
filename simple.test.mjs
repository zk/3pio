import { describe, it, expect } from 'vitest';

describe('Simple test', () => {
  it('should pass', () => {
    console.log('Test is running!');
    expect(1 + 1).toBe(2);
  });

  it('should fail', () => {
    console.error('This test will fail');
    expect(1 + 1).toBe(3);
  });
});