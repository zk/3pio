describe('Utility functions', () => {
  it('should format strings', () => {
    expect('test'.toUpperCase()).toBe('TEST');
  });

  it('should parse numbers', () => {
    expect(parseInt('123')).toBe(123);
  });
});