import {
  bandFor,
  combinedProbability,
  contradicts,
  estimateMarket,
  isValue,
  buildWizardTicket,
  suggestSwaps,
  type TeamProfile,
  type MatchCandidates,
} from './scorer'

/** Profiles matching the worked example in SRS 4.4. */
const teamA: TeamProfile = {
  team: 'Team A',
  venue: 'home',
  sample_size: 10,
  win_rate: 0.4,
  draw_rate: 0.3,
  loss_rate: 0.3,
  scored_rate: 0.6,
  clean_sheet_rate: 0.3,
  btts_rate: 0.5,
  over15_rate: 0.6,
  over25_rate: 0.3,
  under35_rate: 0.8,
}

const teamB: TeamProfile = {
  team: 'Team B',
  venue: 'away',
  sample_size: 10,
  win_rate: 0.2,
  draw_rate: 0.3,
  loss_rate: 0.5,
  scored_rate: 0.8,
  clean_sheet_rate: 0.2,
  btts_rate: 0.5,
  over15_rate: 0.6,
  over25_rate: 0.3,
  under35_rate: 0.8,
}

describe('SRS 4.4 worked example', () => {
  const pick = estimateMarket('AWAY_WIN', teamA, teamB)

  it('rates Team B away win at 25%, High risk', () => {
    expect(pick.estimate).toBeCloseTo(0.25)
    expect(pick.risk_band).toBe('high')
    expect(pick.reason).toContain('Team B won 2 of last 10 away games')
    expect(pick.reason).toContain('Team A lost 3 of last 10 at home')
  })

  it('estimates Team B to score at 75% and Under 3.5 at 80%', () => {
    expect(estimateMarket('AWAY_TO_SCORE', teamA, teamB).estimate).toBeCloseTo(0.75)
    expect(estimateMarket('UNDER_3_5', teamA, teamB).estimate).toBeCloseTo(0.8)
  })

  it('suggests Under 3.5 (80%) then Team B to score (75%)', () => {
    const swaps = suggestSwaps(pick, teamA, teamB)
    expect(swaps.map((s) => s.market)).toEqual(['UNDER_3_5', 'AWAY_TO_SCORE'])
  })
})

describe('risk bands (4.2)', () => {
  it('uses 65% and 45% thresholds', () => {
    expect(bandFor(0.65)).toBe('low')
    expect(bandFor(0.6499)).toBe('medium')
    expect(bandFor(0.45)).toBe('medium')
    expect(bandFor(0.4499)).toBe('high')
  })

  it('raises the band one step and says "limited data" on small samples', () => {
    const thin = { ...teamB, sample_size: 4 }
    const e = estimateMarket('UNDER_3_5', teamA, thin)
    expect(e.estimate).toBeCloseTo(0.8)
    expect(e.risk_band).toBe('medium')
    expect(e.low_confidence).toBe(true)
    expect(e.reason).toContain('limited data')
  })
})

describe('swap rules (4.3)', () => {
  it('skips Low-risk legs', () => {
    const low = estimateMarket('UNDER_3_5', teamA, teamB)
    expect(suggestSwaps(low, teamA, teamB)).toEqual([])
  })

  it('never suggests from a low-confidence profile', () => {
    const thin = { ...teamB, sample_size: 3 }
    const pick = estimateMarket('AWAY_WIN', teamA, thin)
    expect(suggestSwaps(pick, teamA, thin)).toEqual([])
  })

  it('never suggests a market that contradicts the pick', () => {
    expect(contradicts('AWAY_WIN', 'DC_1X')).toBe(true)
    expect(contradicts('OVER_2_5', 'UNDER_2_5')).toBe(true)
    expect(contradicts('OVER_2_5', 'UNDER_3_5')).toBe(false)
    expect(contradicts('BTTS_YES', 'UNDER_1_5')).toBe(true)
    expect(contradicts('AWAY_WIN', 'UNDER_3_5')).toBe(false)
  })
})

describe('combined probability (4.5)', () => {
  it('multiplies leg estimates', () => {
    expect(combinedProbability([0.7, 0.7, 0.7, 0.7, 0.7])).toBeCloseTo(0.168, 3)
  })

  it('marks value when the estimate beats implied by 5 points', () => {
    expect(isValue(0.55, 2.0)).toBe(true) // implied 50%
    expect(isValue(0.54, 2.0)).toBe(false)
  })
})

describe('wizard selection (4.6, amended)', () => {
  /** One match offering one pick in each band: low 80%, medium 55%, high 40%, and a 30% pick below the floor. */
  const match = (id: string, bump = 0): MatchCandidates => ({
    matchId: id,
    candidates: [
      { market: 'UNDER_3_5', estimate: 0.8 + bump, risk_band: 'low', reason: '', sample_size: 10, low_confidence: false },
      { market: 'BTTS_YES', estimate: 0.55 + bump, risk_band: 'medium', reason: '', sample_size: 10, low_confidence: false },
      { market: 'AWAY_WIN', estimate: 0.4 + bump, risk_band: 'high', reason: '', sample_size: 10, low_confidence: false },
      { market: 'DRAW', estimate: 0.3, risk_band: 'high', reason: '', sample_size: 10, low_confidence: false },
    ],
  })
  const many = (n: number) => Array.from({ length: n }, (_, i) => match(`m${i}`, i * 0.001))
  const bands = (legs: { risk_band: string }[]) => legs.map((l) => l.risk_band).sort()
  const product = (legs: { estimate: number }[]) => legs.reduce((p, l) => p * l.estimate, 1)

  it('Safe uses only Low legs', () => {
    expect(bands(buildWizardTicket(many(6), 'safe', 5))).toEqual(['low', 'low', 'low', 'low', 'low'])
  })

  it('Balanced uses half Medium, rest Low', () => {
    expect(bands(buildWizardTicket(many(6), 'balanced', 4))).toEqual(['low', 'low', 'medium', 'medium'])
    expect(bands(buildWizardTicket(many(6), 'balanced', 3))).toEqual(['low', 'low', 'medium'])
  })

  it('Bold really picks longer odds: half High, rest Medium, never below 35%', () => {
    const legs = buildWizardTicket(many(6), 'bold', 4)
    expect(bands(legs)).toEqual(['high', 'high', 'medium', 'medium'])
    expect(Math.min(...legs.map((l) => l.estimate))).toBeGreaterThanOrEqual(0.35)
  })

  it('one leg per match', () => {
    const legs = buildWizardTicket(many(10), 'bold', 10)
    expect(new Set(legs.map((l) => l.matchId)).size).toBe(legs.length)
  })

  it('2 Bold games carry longer odds than 5 Safe games, with a lower chance', () => {
    const bold2 = product(buildWizardTicket(many(6), 'bold', 2))
    const safe5 = product(buildWizardTicket(many(6), 'safe', 5))
    expect(1 / bold2).toBeGreaterThan(1 / safe5) // bigger potential return
    expect(bold2).toBeLessThan(safe5) // less likely to win
  })

  it('falls back to safer bands, never riskier', () => {
    const onlyLow: MatchCandidates[] = many(4).map((m) => ({ ...m, candidates: m.candidates.filter((c) => c.risk_band === 'low') }))
    expect(bands(buildWizardTicket(onlyLow, 'bold', 4))).toEqual(['low', 'low', 'low', 'low'])
    const onlyHigh: MatchCandidates[] = many(4).map((m) => ({ ...m, candidates: m.candidates.filter((c) => c.risk_band === 'high') }))
    expect(buildWizardTicket(onlyHigh, 'safe', 3)).toEqual([])
    expect(buildWizardTicket(onlyHigh, 'balanced', 3)).toEqual([])
  })

  it('returns a shorter ticket rather than loosening the rules', () => {
    expect(buildWizardTicket(many(2), 'safe', 5)).toHaveLength(2)
  })
})
