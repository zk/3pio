describe('Index tests', () => {
  test('root level test', () => {
    expect(true).toBe(true);
  });

  test('main application test', () => {
    const app = 'running';
    expect(app).toBe('running');
  });
});