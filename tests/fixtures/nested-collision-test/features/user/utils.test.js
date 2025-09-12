describe('Utils (features/user)', () => {
  test('user validation', () => {
    const user = { name: 'John', age: 30 };
    expect(user.name).toBe('John');
  });

  test('user permissions', () => {
    const permissions = ['read', 'write'];
    expect(permissions).toContain('read');
  });
});