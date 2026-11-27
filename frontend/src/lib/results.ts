import type { HitCount, ResultsRow } from '../api/types'
import { RULES } from '../config/rules'

export function settled(c: HitCount): number {
  return c.won + c.lost
}

/** Share of settled legs that won, or null when nothing has settled. */
export function hitRate(c: HitCount): number | null {
  const n = settled(c)
  return n === 0 ? null : c.won / n
}

export function isSmallSample(c: HitCount): boolean {
  const n = settled(c)
  return n > 0 && n < RULES.results.smallSample
}

type CountKey = 'original' | 'suggested' | 'original_with_suggestion'

export function totals(rows: ResultsRow[]): Record<CountKey, HitCount> {
  const sum = (key: CountKey) =>
    rows.reduce((acc, r) => ({ won: acc.won + r[key].won, lost: acc.lost + r[key].lost }), { won: 0, lost: 0 })
  return { original: sum('original'), suggested: sum('suggested'), original_with_suggestion: sum('original_with_suggestion') }
}
