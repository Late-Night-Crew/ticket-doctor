import type { Fixture, MarketKey, TicketOrigin } from '../api/types'
import { RULES } from '../config/rules'

/** A leg as the user built it, before analysis. */
export interface DraftLeg {
  fixture: Fixture
  market: MarketKey
  odds: number | null
}

/** A swap the user accepted on the analysis screen (FR-5.3). */
export interface AcceptedSwap {
  matchId: string
  suggestionId: string
  from: MarketKey
  fromOdds: number | null
  to: MarketKey
}

export interface TicketState {
  /** Random per-session id; no personal data (NFR-8). */
  ticketId: string
  origin: TicketOrigin
  legs: DraftLeg[]
  swaps: AcceptedSwap[]
  /** Combined probability before the first accepted swap (FR-6.2). */
  baseline: number | null
}

export type TicketAction =
  | { type: 'add'; leg: DraftLeg }
  | { type: 'remove'; matchId: string }
  | { type: 'setMarket'; matchId: string; market: MarketKey }
  | { type: 'setOdds'; matchId: string; odds: number | null }
  | { type: 'load'; origin: TicketOrigin; legs: DraftLeg[] }
  | { type: 'acceptSwap'; matchId: string; suggestionId: string; market: MarketKey; baseline: number }
  | { type: 'undoSwap'; matchId: string }
  | { type: 'clear' }

export const TICKET_STORAGE_KEY = 'td.ticket'

function newId(): string {
  return typeof crypto !== 'undefined' && 'randomUUID' in crypto
    ? crypto.randomUUID()
    : `t-${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`
}

export function emptyTicket(): TicketState {
  return { ticketId: newId(), origin: 'manual', legs: [], swaps: [], baseline: null }
}

/** Manual edits change the ticket itself, so "before swaps" no longer applies. */
const resetSwaps = { swaps: [], baseline: null } satisfies Pick<TicketState, 'swaps' | 'baseline'>

export function ticketReducer(state: TicketState, action: TicketAction): TicketState {
  switch (action.type) {
    case 'add': {
      // One leg per match (FR-3.4): adding a match that is already on the ticket replaces its market.
      const exists = state.legs.some((l) => l.fixture.id === action.leg.fixture.id)
      if (exists) {
        return {
          ...state,
          ...resetSwaps,
          legs: state.legs.map((l) => (l.fixture.id === action.leg.fixture.id ? action.leg : l)),
        }
      }
      if (state.legs.length >= RULES.maxLegs) return state
      return { ...state, ...resetSwaps, legs: [...state.legs, action.leg] }
    }
    case 'remove':
      return { ...state, ...resetSwaps, legs: state.legs.filter((l) => l.fixture.id !== action.matchId) }
    case 'setMarket':
      return {
        ...state,
        ...resetSwaps,
        legs: state.legs.map((l) => (l.fixture.id === action.matchId ? { ...l, market: action.market, odds: null } : l)),
      }
    case 'setOdds':
      return {
        ...state,
        legs: state.legs.map((l) => (l.fixture.id === action.matchId ? { ...l, odds: action.odds } : l)),
      }
    case 'acceptSwap': {
      const leg = state.legs.find((l) => l.fixture.id === action.matchId)
      if (!leg) return state
      // Swapping the same leg twice keeps the original "from" so Undo goes back to the user's pick.
      const prior = state.swaps.find((s) => s.matchId === action.matchId)
      const swap: AcceptedSwap = {
        matchId: action.matchId,
        suggestionId: action.suggestionId,
        from: prior?.from ?? leg.market,
        fromOdds: prior ? prior.fromOdds : leg.odds,
        to: action.market,
      }
      return {
        ...state,
        baseline: state.baseline ?? action.baseline,
        swaps: [...state.swaps.filter((s) => s.matchId !== action.matchId), swap],
        // Odds were for the old market, so they no longer apply.
        legs: state.legs.map((l) => (l.fixture.id === action.matchId ? { ...l, market: action.market, odds: null } : l)),
      }
    }
    case 'undoSwap': {
      const swap = state.swaps.find((s) => s.matchId === action.matchId)
      if (!swap) return state
      const swaps = state.swaps.filter((s) => s.matchId !== action.matchId)
      return {
        ...state,
        swaps,
        baseline: swaps.length ? state.baseline : null,
        legs: state.legs.map((l) => (l.fixture.id === action.matchId ? { ...l, market: swap.from, odds: swap.fromOdds } : l)),
      }
    }
    case 'load':
      return { ...emptyTicket(), origin: action.origin, legs: action.legs.slice(0, RULES.maxLegs) }
    case 'clear':
      return emptyTicket()
  }
}

export function loadInitial(): TicketState {
  try {
    const raw = sessionStorage.getItem(TICKET_STORAGE_KEY)
    if (raw) {
      const parsed = JSON.parse(raw) as Partial<TicketState>
      if (parsed?.ticketId && Array.isArray(parsed.legs)) {
        return {
          ticketId: parsed.ticketId,
          origin: parsed.origin ?? 'manual',
          legs: parsed.legs,
          swaps: Array.isArray(parsed.swaps) ? parsed.swaps : [],
          baseline: typeof parsed.baseline === 'number' ? parsed.baseline : null,
        }
      }
    }
  } catch {
    /* storage unavailable or corrupt: start fresh */
  }
  return emptyTicket()
}
