describe('Jest Test', () => {
  test('should add numbers', () => {
    expect(1 + 2).toBe(3);
  });

  test('should handle strings', () => {
    expect('hello' + ' world').toBe('hello world');
  });
});