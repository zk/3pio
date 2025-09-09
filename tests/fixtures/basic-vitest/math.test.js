import { describe, it, expect } from 'vitest';

describe('Math operations', () => {
  it('should add numbers correctly', () => {
    console.log('Testing addition...');
    expect(2 + 2).toBe(4);
    console.log('Addition tests passed!');
  });
  
  it('should multiply numbers correctly', () => {
    console.log('Testing multiplication...');
    expect(3 * 4).toBe(12);
    console.log('Multiplication tests passed!');
  });
  
  it('should handle division', () => {
    console.log('Testing division...');
    expect(10 / 2).toBe(5);
    console.log('Division tests passed!');
  });
});