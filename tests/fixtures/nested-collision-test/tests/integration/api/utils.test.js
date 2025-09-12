describe('Utils (tests/integration/api)', () => {
  test('API endpoint test', () => {
    const endpoint = '/api/v1/users';
    expect(endpoint).toContain('/api');
  });

  test('Response validation', () => {
    const response = { status: 200, data: [] };
    expect(response.status).toBe(200);
  });
});