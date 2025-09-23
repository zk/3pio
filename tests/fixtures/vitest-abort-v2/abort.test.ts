import { describe, it, beforeAll, expect } from 'vitest'

describe('abort suite', () => {
  beforeAll(() => {
    throw new Error('boom')
  })

  it('should not run 1', () => {
    expect(true).toBe(false)
  })

  it('should not run 2', () => {
    expect(true).toBe(false)
  })
})

