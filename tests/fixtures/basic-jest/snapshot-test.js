// Test to understand how Jest reports snapshot failures
test('snapshot test', () => {
  const obj = {
    value: 'actual value',
    nested: { deep: true }
  };
  expect(obj).toMatchInlineSnapshot(`
    Object {
      "value": "expected value",
      "nested": Object {
        "deep": false,
      },
    }
  `);
});