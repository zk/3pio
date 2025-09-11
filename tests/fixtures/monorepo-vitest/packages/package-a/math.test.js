import { describe, it, expect } from 'vitest';

describe('Package A Math Operations', () => {
  it('should add numbers correctly', () => {
    expect(1 + 1).toBe(2);
    expect(10 + 20).toBe(30);
  });

  it('should subtract numbers correctly', () => {
    expect(5 - 3).toBe(2);
    expect(100 - 50).toBe(50);
  });

  it('should multiply numbers correctly', () => {
    expect(3 * 4).toBe(12);
    expect(7 * 8).toBe(56);
  });
});