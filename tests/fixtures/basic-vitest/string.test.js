import { describe, it, expect } from 'vitest';

describe('String operations', () => {
  it('should concatenate strings', () => {
    console.log('Testing string concatenation...');
    expect('hello' + ' ' + 'world').toBe('hello world');
    console.log('String concatenation passed!');
  });
  
  it('should fail this test', () => {
    console.log('This test is expected to fail');
    console.error('Error: Intentional failure for testing');
    expect('foo').toBe('bar'); // This will fail
  });
  
  it.skip('should skip this test', () => {
    console.log('This should not run');
    expect(true).toBe(true);
  });
  
  it('should convert to uppercase', () => {
    console.log('Testing uppercase conversion...');
    expect('hello'.toUpperCase()).toBe('HELLO');
    console.log('Uppercase test passed!');
  });
});