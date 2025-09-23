import { describe, it, expect, expectTypeOf } from 'vitest'
import { add } from '../src/add'

describe('add', () => {
  it('sums numbers', () => {
    expect(add(2, 3)).toBe(5)
  })

  it('has correct types', () => {
    expectTypeOf(add).parameter(0).toEqualTypeOf<number>()
    expectTypeOf(add).parameter(1).toEqualTypeOf<number>()
    expectTypeOf(add(1, 2)).toEqualTypeOf<number>()
  })
})

