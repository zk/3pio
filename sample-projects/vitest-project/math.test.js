import { describe, it, expect } from 'vitest';

describe('Math operations', () => {
  it('should add numbers correctly', () => {
    console.log('Testing addition...');
    expect(1 + 1).toBe(2);
    expect(10 + 5).toBe(15);
    console.log('Addition tests passed!');
  });

  it('should multiply numbers correctly', () => {
    console.log('Testing multiplication...');
    expect(2 * 3).toBe(6);
    expect(5 * 5).toBe(25);
    console.log('Multiplication tests passed!');
  });

  it('should handle division', () => {
    console.log('Testing division...');
    expect(10 / 2).toBe(5);
    expect(20 / 4).toBe(5);
    console.log('Division tests passed!');
  });
});