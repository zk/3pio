import { describe, it, expect } from 'vitest';

describe('String operations (v2)', () => {
  it('concatenates strings', () => {
    expect('hello' + ' ' + 'world').toBe('hello world');
  });

  it('should fail intentionally', () => {
    expect('foo').toBe('bar');
  });

  it.skip('skips this test', () => {
    expect(true).toBe(false);
  });
});
