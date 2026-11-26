import { oneIn, pct } from './format'

describe('format', () => {
  it('formats percentages', () => {
    expect(pct(0.256)).toBe('26%')
    expect(pct(0.256, 1)).toBe('25.6%')
  })

  it('gives "about 1 in N" for the SRS five-leg example', () => {
    // Five legs at 70% each ≈ 16.8% → about 1 in 6 (section 4.5)
    expect(oneIn(0.7 ** 5)).toBe(6)
  })
})
