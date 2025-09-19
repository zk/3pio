const assert = require('assert');

describe('String utilities', () => {
  it('uppercases', () => {
    const s = 'hello';
    assert.strictEqual(s.toUpperCase(), 'HELLO');
  });
});

