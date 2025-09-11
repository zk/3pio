// Simple math operations tests for ES module project

describe('Math operations', () => {
  test('should add numbers correctly', () => {
    expect(2 + 2).toBe(4);
  });

  test('should multiply numbers correctly', () => {
    expect(3 * 4).toBe(12);
  });

  test('should handle division', () => {
    expect(10 / 2).toBe(5);
  });
});