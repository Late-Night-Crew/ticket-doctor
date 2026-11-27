import { hitRate, isSmallSample, totals } from './results'

describe('results helpers', () => {
  it('computes hit rate over settled legs only', () => {
    expect(hitRate({ won: 3, lost: 1 })).toBe(0.75)
    expect(hitRate({ won: 0, lost: 0 })).toBeNull()
  })

  it('flags small samples but not empty ones', () => {
    expect(isSmallSample({ won: 5, lost: 4 })).toBe(true)
    expect(isSmallSample({ won: 15, lost: 5 })).toBe(false)
    expect(isSmallSample({ won: 0, lost: 0 })).toBe(false)
  })

  it('totals rows', () => {
    const t = totals([
      { league_id: 'epl', market: 'HOME_WIN', original: { won: 1, lost: 2 }, suggested: { won: 3, lost: 0 }, original_with_suggestion: { won: 1, lost: 1 } },
      { league_id: 'epl', market: 'DRAW', original: { won: 4, lost: 5 }, suggested: { won: 0, lost: 1 }, original_with_suggestion: { won: 0, lost: 3 } },
    ])
    expect(t).toEqual({
      original: { won: 5, lost: 7 },
      suggested: { won: 3, lost: 1 },
      original_with_suggestion: { won: 1, lost: 4 },
    })
  })
})
