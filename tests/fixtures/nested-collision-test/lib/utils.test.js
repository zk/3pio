describe('Utils (lib)', () => {
  test('string concatenation', () => {
    expect('hello' + ' ' + 'world').toBe('hello world');
  });

  test('array operations', () => {
    expect([1, 2, 3].length).toBe(3);
  });
});