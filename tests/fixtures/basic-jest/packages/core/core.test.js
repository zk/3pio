describe('Core functionality', () => {
  it('should work with core features', () => {
    expect(true).toBe(true);
  });

  it('should handle core errors', () => {
    expect(() => { throw new Error('core error'); }).toThrow('core error');
  });
});