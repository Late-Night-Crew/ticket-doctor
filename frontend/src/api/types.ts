/**
 * API contract between the React app and the Go API.
 * Keep in sync with the Go structs; field names are snake_case on the wire.
 */

export type MarketKey =
  | 'HOME_WIN'
  | 'AWAY_WIN'
  | 'DRAW'
  | 'DC_1X'
  | 'DC_X2'
  | 'BTTS_YES'
  | 'BTTS_NO'
  | 'OVER_1_5'
  | 'UNDER_1_5'
  | 'OVER_2_5'
  | 'UNDER_2_5'
  | 'OVER_3_5'
  | 'UNDER_3_5'
  | 'HOME_TO_SCORE'
  | 'AWAY_TO_SCORE'

export type RiskBand = 'low' | 'medium' | 'high'
export type TicketOrigin = 'manual' | 'wizard'
export type Venue = 'home' | 'away'

export interface League {
  id: string
  name: string
  country: string
}

export interface Team {
  id: string
  name: string
  /** Last five results, most recent last, e.g. "W-D-L-W-W" (FR-2.4). */
  form?: string
}

export interface Fixture {
  id: string
  league_id: string
  /** ISO 8601 kickoff time. */
  kickoff: string
  home: Team
  away: Team
}

export type FixtureWindow = 'today' | 'weekend' | 'upcoming'

/* ---------- Ticket analysis (FR-4 to FR-6) ---------- */

export interface TicketLegInput {
  match_id: string
  market: MarketKey
  /** Decimal odds entered by the user (FR-3.3). */
  odds?: number | null
}

export interface AnalyseRequest {
  ticket_id: string
  origin: TicketOrigin
  legs: TicketLegInput[]
}

export interface Suggestion {
  id: string
  market: MarketKey
  estimate: number
  risk_band: RiskBand
  reason: string
}

export interface AnalysedLeg {
  id: string
  match_id: string
  fixture: Fixture
  market: MarketKey
  odds: number | null
  /** Model estimate, 0–1. */
  estimate: number
  risk_band: RiskBand
  reason: string
  /** Smallest sample behind the estimate, and the form window used (NFR-5). */
  sample_size: number
  window_n: number
  low_confidence: boolean
  /** 1 ÷ odds, when odds were entered (FR-4.4). */
  implied_prob: number | null
  is_value: boolean | null
  suggestions: Suggestion[]
}

export interface CombinedProbability {
  probability: number
  /** Rounded "about 1 in N" (RG-2). */
  one_in: number
  /** Product of decimal odds when every leg has odds (FR-6.4). */
  odds: number | null
}

export interface AnalyseResponse {
  ticket_id: string
  legs: AnalysedLeg[]
  combined: CombinedProbability
  weakest_leg_id: string | null
}

export interface AcceptSuggestionResponse {
  ok: true
}

/* ---------- Ticket wizard (FR-8) ---------- */

export type WizardPath = 'beginner' | 'guided'
export type WizardRisk = 'safe' | 'balanced' | 'bold'
export type WizardFocus = 'wins' | 'goals' | 'mix'

export interface WizardRequest {
  path: WizardPath
  risk: WizardRisk
  focus: WizardFocus
  /** Exact number of games wanted: 2–10, or 2–4 on the beginner path. */
  legs: number
  /** "soonest" (beginner path): today if its games can fill the ticket, otherwise this weekend. */
  when: Exclude<FixtureWindow, 'upcoming'> | 'soonest'
  /** Changes on Regenerate (FR-8.9). */
  seed: number
}

export interface WizardLeg {
  fixture: Fixture
  market: MarketKey
  estimate: number
  risk_band: RiskBand
  reason: string
  /** Smallest sample behind the estimate, and the form window used (NFR-5). */
  sample_size: number
  window_n: number
  /** 1 ÷ estimate, the app's own estimate of fair odds. */
  fair_odds: number
}

export interface WizardResponse {
  legs: WizardLeg[]
  /** The window the ticket was built from (differs from the request only for "soonest"). */
  when: Exclude<FixtureWindow, 'upcoming'>
  /** Set when fewer legs than asked qualified (rule 4.6.6). */
  shortfall_reason: string | null
}

/* ---------- Results log (FR-7) ---------- */

export interface HitCount {
  won: number
  lost: number
}

export interface ResultsRow {
  league_id: string
  market: MarketKey
  /** Users' own picks in this market (a swap they accepted is not counted here). */
  original: HitCount
  /** Our suggestions in this market, accepted or not. */
  suggested: HitCount
  /** Users' picks in this market that we suggested a swap for: the like-for-like group (SRS addendum A6). */
  original_with_suggestion: HitCount
}

export interface ResultsResponse {
  /** When results were last imported; null before the first import. */
  updated_at: string | null
  rows: ResultsRow[]
}

/* ---------- Saved tickets (V2, FR-9) ---------- */

export type PickOutcome = 'won' | 'lost' | 'pending'

export interface SavedPick {
  market: MarketKey
  odds?: number | null
  estimate?: number
  risk_band?: RiskBand
  outcome: PickOutcome
}

export interface SavedSuggestion {
  market: MarketKey
  estimate: number
  risk_band: RiskBand
  accepted: boolean
  outcome: PickOutcome
}

export interface SavedLeg {
  fixture: Fixture
  match_status: 'scheduled' | 'postponed' | 'finished'
  score: { home: number; away: number } | null
  /** The user's own pick. */
  original: SavedPick
  /** The pick on the ticket in the end (differs when a swap was accepted). */
  final: SavedPick
  /** Every suggestion shown for this leg, accepted or not (FR-9.3). */
  suggestions: SavedSuggestion[]
}

/** For each suggestion shown, how it did against the user's own pick (FR-9.6). */
export interface PairCounts {
  suggestion_won_pick_lost: number
  pick_won_suggestion_lost: number
  both_won: number
  both_lost: number
}

export interface SavedSummary {
  legs: number
  settled: number
  original: HitCount
  final: HitCount
  pairs: PairCounts
  original_ticket: PickOutcome
  final_ticket: PickOutcome
}

export interface SavedTicket {
  /** The private link id: /t/<id>. */
  id: string
  origin: TicketOrigin
  created_at: string
  updated_at: string
  status: 'pending' | 'settled'
  settled_at: string | null
  /** When it will be deleted (90 days after settling); null until settled. */
  expires_at: string | null
  summary: SavedSummary
  /** Only on a single ticket, not in lists. */
  legs?: SavedLeg[]
}
