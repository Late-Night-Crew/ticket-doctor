import type { RiskBand } from '../api/types'
import { RULES } from '../config/rules'

/** Display-only band for a probability; the API decides leg bands. */
export function bandFor(p: number): RiskBand {
  if (p >= RULES.band.lowFrom) return 'low'
  if (p >= RULES.band.mediumFrom) return 'medium'
  return 'high'
}
