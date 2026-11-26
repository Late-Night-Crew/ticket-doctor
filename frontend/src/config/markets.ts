import type { MarketKey } from '../api/types'

export type MarketGroup = 'wins' | 'goals'

export interface MarketDef {
  key: MarketKey
  /** Short label for chips and lists. */
  short: string
  group: MarketGroup
  /** Plain-words description for a given fixture (FR-8.6). */
  plain: (home: string, away: string) => string
  /** Glossary entry id (FR-8.10). */
  glossary: string
}

export const MARKETS: MarketDef[] = [
  { key: 'HOME_WIN', short: 'Home win', group: 'wins', glossary: 'win', plain: (h) => `${h} to win` },
  { key: 'DRAW', short: 'Draw', group: 'wins', glossary: 'draw', plain: (h, a) => `${h} and ${a} to draw` },
  { key: 'AWAY_WIN', short: 'Away win', group: 'wins', glossary: 'win', plain: (_h, a) => `${a} to win` },
  { key: 'DC_1X', short: 'Double chance 1X', group: 'wins', glossary: 'double-chance', plain: (h) => `${h} to win or draw` },
  { key: 'DC_X2', short: 'Double chance X2', group: 'wins', glossary: 'double-chance', plain: (_h, a) => `${a} to win or draw` },
  { key: 'BTTS_YES', short: 'BTTS yes', group: 'goals', glossary: 'btts', plain: () => 'Both teams to score' },
  { key: 'BTTS_NO', short: 'BTTS no', group: 'goals', glossary: 'btts', plain: () => 'At least one team fails to score' },
  { key: 'OVER_1_5', short: 'Over 1.5', group: 'goals', glossary: 'over-under', plain: () => '2 or more goals in the match' },
  { key: 'UNDER_1_5', short: 'Under 1.5', group: 'goals', glossary: 'over-under', plain: () => '1 goal or none in the match' },
  { key: 'OVER_2_5', short: 'Over 2.5', group: 'goals', glossary: 'over-under', plain: () => '3 or more goals in the match' },
  { key: 'UNDER_2_5', short: 'Under 2.5', group: 'goals', glossary: 'over-under', plain: () => '2 goals or fewer in the match' },
  { key: 'OVER_3_5', short: 'Over 3.5', group: 'goals', glossary: 'over-under', plain: () => '4 or more goals in the match' },
  { key: 'UNDER_3_5', short: 'Under 3.5', group: 'goals', glossary: 'over-under', plain: () => '3 goals or fewer in the match' },
  { key: 'HOME_TO_SCORE', short: 'Home to score', group: 'goals', glossary: 'team-to-score', plain: (h) => `${h} to score at least one goal` },
  { key: 'AWAY_TO_SCORE', short: 'Away to score', group: 'goals', glossary: 'team-to-score', plain: (_h, a) => `${a} to score at least one goal` },
]

const BY_KEY = new Map(MARKETS.map((m) => [m.key, m]))

export function market(key: MarketKey): MarketDef {
  const def = BY_KEY.get(key)
  if (!def) throw new Error(`Unknown market ${key}`)
  return def
}
