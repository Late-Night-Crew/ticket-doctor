export const MIN_ODDS = 1.01
export const MAX_ODDS = 1000

export type OddsParse = { ok: true; value: number | null } | { ok: false; error: string }

/**
 * Parse user-entered decimal odds (FR-3.3). Empty means "no odds".
 * Accepts a comma as the decimal separator.
 */
export function parseOdds(raw: string): OddsParse {
  const s = raw.trim().replace(',', '.')
  if (s === '') return { ok: true, value: null }
  if (!/^\d+(\.\d+)?$/.test(s)) return { ok: false, error: 'Use decimal odds, e.g. 1.85' }
  const v = Number(s)
  if (v < MIN_ODDS) return { ok: false, error: `Decimal odds start at ${MIN_ODDS.toFixed(2)}` }
  if (v > MAX_ODDS) return { ok: false, error: `Odds above ${MAX_ODDS} are not supported` }
  return { ok: true, value: Math.round(v * 100) / 100 }
}

export function oddsToInput(v: number | null): string {
  return v == null ? '' : v.toFixed(2)
}
