import { describe, test, expect } from 'vitest';

describe('duplicate test names', () => {
  // First test with this name
  test('duplicate test name', () => {
    expect(1).toBe(1);
  });

  // Second test with same name - should get [index] suffix
  test('duplicate test name', () => {
    expect(2).toBe(2);
  });

  // Third test with same name - should get different [index] suffix
  test('duplicate test name', () => {
    expect(3).toBe(3);
  });

  // Different test name - should not be affected
  test('unique test name', () => {
    expect(true).toBe(true);
  });

  // Another duplicate pair
  test('another duplicate', () => {
    expect('a').toBe('a');
  });

  test('another duplicate', () => {
    expect('b').toBe('b');
  });
});

// Test duplicates across different describe blocks
describe('first suite', () => {
  test('cross-suite duplicate', () => {
    expect(1).toBe(1);
  });
});

describe('second suite', () => {
  test('cross-suite duplicate', () => {
    expect(2).toBe(2);
  });
});