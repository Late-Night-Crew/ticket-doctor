import { parseOdds } from './odds'

describe('parseOdds', () => {
  it('treats empty as no odds', () => {
    expect(parseOdds('  ')).toEqual({ ok: true, value: null })
  })

  it('accepts dot or comma decimals', () => {
    expect(parseOdds('1.85')).toEqual({ ok: true, value: 1.85 })
    expect(parseOdds('2,5')).toEqual({ ok: true, value: 2.5 })
    expect(parseOdds('3')).toEqual({ ok: true, value: 3 })
  })

  it('rejects fractional, American and out-of-range odds', () => {
    expect(parseOdds('5/2').ok).toBe(false)
    expect(parseOdds('+150').ok).toBe(false)
    expect(parseOdds('1').ok).toBe(false)
    expect(parseOdds('1001').ok).toBe(false)
  })
})
