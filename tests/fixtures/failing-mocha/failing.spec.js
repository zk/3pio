const assert = require('assert');

describe('Failing suite', () => {
  it('fails intentionally', () => {
    assert.strictEqual(1 + 1, 3);
  });
});

