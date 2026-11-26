/**
 * Mock implementation of SRS section 4 (risk rules and swap logic).
 *
 * Only the MSW mock API uses this. The Go API is the source of truth; this file
 * exists so the UI can be built end to end first, and its tests double as the
 * expected values for the Go unit tests (NFR-7).
 */
import type { MarketKey, RiskBand } from '../api/types'
import { MARKETS, market } from '../config/markets'
import { RULES } from '../config/rules'

/** One team's record over its last N home or away matches (FR-2.2). */
export interface TeamProfile {
  team: string
  venue: 'home' | 'away'
  sample_size: number
  win_rate: number
  draw_rate: number
  loss_rate: number
  scored_rate: number
  clean_sheet_rate: number
  btts_rate: number
  over15_rate: number
  over25_rate: number
  under35_rate: number
}

export interface MarketEstimate {
  market: MarketKey
  estimate: number
  risk_band: RiskBand
  reason: string
  sample_size: number
  low_confidence: boolean
}

const EPS = 1e-9
const avg = (a: number, b: number) => (a + b) / 2
const count = (rate: number, n: number) => Math.round(rate * n)
const pctStr = (p: number) => `${Math.round(p * 100)}%`

/** Base band from thresholds in 4.2. */
export function bandFor(estimate: number): RiskBand {
  if (estimate + EPS >= RULES.band.lowFrom) return 'low'
  if (estimate + EPS >= RULES.band.mediumFrom) return 'medium'
  return 'high'
}

function raiseBand(band: RiskBand): RiskBand {
  return band === 'low' ? 'medium' : 'high'
}

/**
 * Estimate one market for a fixture (4.1).
 *
 * Markets the SRS does not define directly use complements (proposed SRS
 * addition): BTTS no = 1 − BTTS yes, Under 1.5 = 1 − Over 1.5,
 * Under 2.5 = 1 − Over 2.5, Over 3.5 = 1 − Under 3.5.
 */
export function estimateMarket(key: MarketKey, home: TeamProfile, away: TeamProfile): MarketEstimate {
  const h = home
  const a = away
  const hn = h.sample_size
  const an = a.sample_size
  let estimate: number
  let reason: string

  const homeWin = avg(h.win_rate, a.loss_rate)
  const awayWin = avg(a.win_rate, h.loss_rate)
  const btts = avg(h.btts_rate, a.btts_rate)
  const over15 = avg(h.over15_rate, a.over15_rate)
  const over25 = avg(h.over25_rate, a.over25_rate)
  const under35 = avg(h.under35_rate, a.under35_rate)

  switch (key) {
    case 'HOME_WIN':
      estimate = homeWin
      reason = `${h.team} won ${count(h.win_rate, hn)} of last ${hn} home games; ${a.team} lost ${count(a.loss_rate, an)} of last ${an} away.`
      break
    case 'AWAY_WIN':
      estimate = awayWin
      reason = `${a.team} won ${count(a.win_rate, an)} of last ${an} away games; ${h.team} lost ${count(h.loss_rate, hn)} of last ${hn} at home.`
      break
    case 'DRAW':
      estimate = avg(h.draw_rate, a.draw_rate)
      reason = `${h.team} drew ${count(h.draw_rate, hn)} of last ${hn} at home; ${a.team} drew ${count(a.draw_rate, an)} of last ${an} away.`
      break
    case 'DC_1X':
      estimate = 1 - awayWin
      reason = `${a.team} won only ${count(a.win_rate, an)} of last ${an} away; ${h.team} lost ${count(h.loss_rate, hn)} of last ${hn} at home.`
      break
    case 'DC_X2':
      estimate = 1 - homeWin
      reason = `${h.team} won ${count(h.win_rate, hn)} of last ${hn} at home; ${a.team} lost ${count(a.loss_rate, an)} of last ${an} away.`
      break
    case 'HOME_TO_SCORE':
      estimate = avg(h.scored_rate, 1 - a.clean_sheet_rate)
      reason = `${h.team} scored in ${count(h.scored_rate, hn)} of last ${hn} home games; ${a.team} kept ${count(a.clean_sheet_rate, an)} clean sheets in ${an} away.`
      break
    case 'AWAY_TO_SCORE':
      estimate = avg(a.scored_rate, 1 - h.clean_sheet_rate)
      reason = `${a.team} scored in ${count(a.scored_rate, an)} of last ${an} away games; ${h.team} kept ${count(h.clean_sheet_rate, hn)} clean sheets in ${hn} at home.`
      break
    case 'BTTS_YES':
      estimate = btts
      reason = `Both teams scored in ${count(h.btts_rate, hn)} of ${h.team}'s last ${hn} home games and ${count(a.btts_rate, an)} of ${a.team}'s last ${an} away.`
      break
    case 'BTTS_NO':
      estimate = 1 - btts
      reason = `At least one side failed to score in ${hn - count(h.btts_rate, hn)} of ${h.team}'s last ${hn} home games and ${an - count(a.btts_rate, an)} of ${a.team}'s last ${an} away.`
      break
    case 'OVER_1_5':
      estimate = over15
      reason = goalsReason('2 or more goals', h.over15_rate, a.over15_rate, h, a)
      break
    case 'UNDER_1_5':
      estimate = 1 - over15
      reason = goalsReason('1 goal or none', 1 - h.over15_rate, 1 - a.over15_rate, h, a)
      break
    case 'OVER_2_5':
      estimate = over25
      reason = goalsReason('3 or more goals', h.over25_rate, a.over25_rate, h, a)
      break
    case 'UNDER_2_5':
      estimate = 1 - over25
      reason = goalsReason('2 goals or fewer', 1 - h.over25_rate, 1 - a.over25_rate, h, a)
      break
    case 'UNDER_3_5':
      estimate = under35
      reason = goalsReason('3 goals or fewer', h.under35_rate, a.under35_rate, h, a)
      break
    case 'OVER_3_5':
      estimate = 1 - under35
      reason = goalsReason('4 or more goals', 1 - h.under35_rate, 1 - a.under35_rate, h, a)
      break
  }

  const sample = Math.min(hn, an)
  const lowConfidence = sample < RULES.lowConfidenceBelow
  let band = bandFor(estimate)
  if (lowConfidence) {
    band = raiseBand(band)
    reason = `${reason} (limited data)`
  }

  return { market: key, estimate, risk_band: band, reason, sample_size: sample, low_confidence: lowConfidence }
}

function goalsReason(what: string, hRate: number, aRate: number, h: TeamProfile, a: TeamProfile): string {
  return `${what} in ${count(hRate, h.sample_size)} of ${h.team}'s last ${h.sample_size} home games and ${count(aRate, a.sample_size)} of ${a.team}'s last ${a.sample_size} away.`
}

/* ---------- Compatibility ---------- */

type GoalLine = { side: 'over' | 'under'; line: number }

function goalLine(key: MarketKey): GoalLine | null {
  const m = /^(OVER|UNDER)_(\d)_5$/.exec(key)
  return m ? { side: m[1] === 'OVER' ? 'over' : 'under', line: Number(m[2]) + 0.5 } : null
}

const RESULT_CLASH: Partial<Record<MarketKey, MarketKey[]>> = {
  HOME_WIN: ['AWAY_WIN', 'DRAW', 'DC_X2'],
  AWAY_WIN: ['HOME_WIN', 'DRAW', 'DC_1X'],
  DRAW: ['HOME_WIN', 'AWAY_WIN'],
  DC_1X: ['AWAY_WIN'],
  DC_X2: ['HOME_WIN'],
  BTTS_YES: ['BTTS_NO', 'UNDER_1_5'],
  BTTS_NO: ['BTTS_YES'],
}

/**
 * True when two markets can never both win. Swaps exclude these so a swap
 * never bets against the user's own pick (proposed SRS addition; implied by
 * the worked example in 4.4, which does not suggest 1X for an away-win pick).
 */
export function contradicts(a: MarketKey, b: MarketKey): boolean {
  if (RESULT_CLASH[a]?.includes(b) || RESULT_CLASH[b]?.includes(a)) return true
  const ga = goalLine(a)
  const gb = goalLine(b)
  if (ga && gb && ga.side !== gb.side) {
    const over = ga.side === 'over' ? ga : gb
    const under = ga.side === 'under' ? ga : gb
    // over x needs ≥ x+0.5 goals; under y needs ≤ y−0.5 goals
    return over.line >= under.line
  }
  return false
}

/* ---------- Swaps (4.3) ---------- */

export function suggestSwaps(pick: MarketEstimate, home: TeamProfile, away: TeamProfile): MarketEstimate[] {
  if (pick.risk_band === 'low') return []
  return MARKETS.map((m) => m.key)
    .filter((k) => k !== pick.market && !contradicts(k, pick.market))
    .map((k) => estimateMarket(k, home, away))
    .filter((c) => c.estimate + EPS >= pick.estimate + RULES.swap.minGain)
    .filter((c) => !c.low_confidence)
    .sort((x, y) => y.estimate - x.estimate)
    .slice(0, RULES.swap.maxSuggestions)
}

/* ---------- Combined probability (4.5) ---------- */

export function combinedProbability(estimates: number[]): number {
  return estimates.reduce((p, e) => p * e, 1)
}

export function impliedProbability(odds: number): number {
  return 1 / odds
}

export function isValue(estimate: number, odds: number): boolean {
  return estimate + EPS >= impliedProbability(odds) + RULES.valueMargin
}

/* ---------- Wizard selection (4.6, amended) ---------- */

/** Every usable market for one match: filtered by focus, low-confidence dropped (4.6.1–4.6.2). */
export interface MatchCandidates {
  matchId: string
  candidates: MarketEstimate[]
}

export interface WizardCandidate extends MarketEstimate {
  matchId: string
}

export function wizardCandidates(
  matchId: string,
  home: TeamProfile,
  away: TeamProfile,
  focus: 'wins' | 'goals' | 'mix',
): MatchCandidates {
  const keys = MARKETS.filter((m) => focus === 'mix' || m.group === focus).map((m) => m.key)
  return {
    matchId,
    candidates: keys.map((k) => estimateMarket(k, home, away)).filter((c) => !c.low_confidence),
  }
}

interface Slot {
  band: RiskBand
  count: number
  /** Safer bands to use when this band runs short. Never riskier. */
  fallback: RiskBand[]
}

/**
 * Risk level decides which band each leg is drawn from, so Bold really means
 * longer odds rather than "the same picks, with riskier ones allowed".
 *   Safe:     all Low.
 *   Balanced: half Medium (rounded down), rest Low.
 *   Bold:     half High (rounded down, never below the 35% floor), rest Medium.
 * This keeps the SRS limits: Balanced ≤ half Medium; Bold ≥ half Low/Medium.
 */
function slotsFor(risk: 'safe' | 'balanced' | 'bold', n: number): Slot[] {
  const half = Math.floor(n / 2)
  switch (risk) {
    case 'safe':
      return [{ band: 'low', count: n, fallback: [] }]
    case 'balanced':
      return [
        { band: 'medium', count: half, fallback: ['low'] },
        { band: 'low', count: n - half, fallback: [] },
      ]
    case 'bold':
      return [
        { band: 'high', count: half, fallback: ['medium', 'low'] },
        { band: 'medium', count: n - half, fallback: ['low'] },
      ]
  }
}

function bestInBand(m: MatchCandidates, band: RiskBand): WizardCandidate | null {
  const best = m.candidates
    .filter((c) => c.risk_band === band && c.estimate + EPS >= RULES.wizard.boldFloor)
    .sort((x, y) => y.estimate - x.estimate)[0]
  return best ? { ...best, matchId: m.matchId } : null
}

/**
 * Build a wizard ticket of n legs, one per match (4.6.3–4.6.6). Within a band,
 * the best-supported pick wins. Returns fewer legs when too few qualify; it
 * never loosens the rules.
 */
export function buildWizardTicket(
  matches: MatchCandidates[],
  risk: 'safe' | 'balanced' | 'bold',
  n: number,
): WizardCandidate[] {
  const used = new Set<string>()
  const out: WizardCandidate[] = []

  const take = (band: RiskBand, count: number): number => {
    const pool = matches
      .filter((m) => !used.has(m.matchId))
      .map((m) => bestInBand(m, band))
      .filter((c): c is WizardCandidate => c !== null)
      .sort((x, y) => y.estimate - x.estimate)
    const picked = pool.slice(0, count)
    for (const c of picked) {
      used.add(c.matchId)
      out.push(c)
    }
    return count - picked.length
  }

  const slots = slotsFor(risk, n)
  // First every slot from its own band, then any gaps from safer bands.
  const missing = slots.map((s) => take(s.band, s.count))
  slots.forEach((s, i) => {
    for (const band of s.fallback) {
      if (missing[i] <= 0) break
      missing[i] = take(band, missing[i])
    }
  })

  return out.sort((x, y) => y.estimate - x.estimate)
}

export function plainMarket(key: MarketKey, home: string, away: string): string {
  return market(key).plain(home, away)
}

export { pctStr }
