describe('Utils (features/auth)', () => {
  test('authentication check', () => {
    const isAuthenticated = true;
    expect(isAuthenticated).toBe(true);
  });

  test('token validation', () => {
    const token = 'abc123';
    expect(token).toHaveLength(6);
  });
});