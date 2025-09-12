describe('Utils (tests/unit/helpers)', () => {
  test('helper function test', () => {
    const formatDate = (date) => date.toISOString();
    const now = new Date('2024-01-01');
    expect(formatDate(now)).toContain('2024');
  });

  test('utility validation', () => {
    const isValid = (val) => val !== null && val !== undefined;
    expect(isValid('test')).toBe(true);
  });
});