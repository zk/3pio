import { describe, it, expect } from 'vitest';

describe('Package B String Operations', () => {
  it('should concatenate strings', () => {
    expect('hello' + ' ' + 'world').toBe('hello world');
    expect('foo' + 'bar').toBe('foobar');
  });

  it('should convert to uppercase', () => {
    expect('hello'.toUpperCase()).toBe('HELLO');
    expect('world'.toUpperCase()).toBe('WORLD');
  });

  it('should check string length', () => {
    expect('test'.length).toBe(4);
    expect('monorepo'.length).toBe(8);
  });
});