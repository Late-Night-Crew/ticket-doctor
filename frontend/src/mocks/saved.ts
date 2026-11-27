/**
 * Mock saved tickets (V2), for development and tests. The mock analyser
 * records each ticket's last analysis here, and saving snapshots it. Games in
 * the mock are all in the future, so a pre-made settled ticket (DEMO_SAVED_ID)
 * shows the review screen with results.
 */
import { http, HttpResponse } from 'msw'
import type { AnalyseResponse, SavedLeg, SavedSummary, SavedTicket } from '../api/types'
import { FIXTURES } from './data'

const API = '*/api/saved'
export const DEMO_SAVED_ID = 'demoSettledTicket00001'
export const DEMO_TICKET_ID = 'demo-ticket-0001'

const analyses = new Map<string, AnalyseResponse>()
export const recordAnalysis = (res: AnalyseResponse) => analyses.set(res.ticket_id, res)

const problem = (status: number, detail: string) =>
  HttpResponse.json({ type: 'about:blank', title: 'Error', status, detail }, { status, headers: { 'Content-Type': 'application/problem+json' } })

function summarise(legs: SavedLeg[]): SavedSummary {
  const s: SavedSummary = {
    legs: legs.length,
    settled: 0,
    original: { won: 0, lost: 0 },
    final: { won: 0, lost: 0 },
    pairs: { suggestion_won_pick_lost: 0, pick_won_suggestion_lost: 0, both_won: 0, both_lost: 0 },
    original_ticket: 'pending',
    final_ticket: 'pending',
  }
  for (const l of legs) {
    if (l.original.outcome === 'pending') continue
    s.settled++
    s.original[l.original.outcome]++
    if (l.final.outcome !== 'pending') s.final[l.final.outcome]++
    for (const sg of l.suggestions) {
      const pickWon = l.original.outcome === 'won'
      if (sg.outcome === 'won') s.pairs[pickWon ? 'both_won' : 'suggestion_won_pick_lost']++
      else s.pairs[pickWon ? 'pick_won_suggestion_lost' : 'both_lost']++
    }
  }
  const ticket = (h: { won: number; lost: number }): SavedSummary['final_ticket'] =>
    h.lost > 0 ? 'lost' : h.won === legs.length ? 'won' : 'pending'
  s.original_ticket = ticket(s.original)
  s.final_ticket = ticket(s.final)
  return s
}

function demo(): SavedTicket {
  const [a, b] = FIXTURES
  const legs: SavedLeg[] = [
    {
      fixture: a,
      match_status: 'finished',
      score: { home: 0, away: 2 },
      original: { market: 'HOME_WIN', estimate: 0.38, risk_band: 'high', outcome: 'lost' },
      final: { market: 'DC_X2', outcome: 'won' },
      suggestions: [
        { market: 'DC_X2', estimate: 0.62, risk_band: 'medium', accepted: true, outcome: 'won' },
        { market: 'BTTS_NO', estimate: 0.55, risk_band: 'medium', accepted: false, outcome: 'won' },
      ],
    },
    {
      fixture: b,
      match_status: 'finished',
      score: { home: 2, away: 1 },
      original: { market: 'OVER_1_5', estimate: 0.81, risk_band: 'low', outcome: 'won' },
      final: { market: 'OVER_1_5', outcome: 'won' },
      suggestions: [],
    },
  ]
  const settledAt = new Date(Date.now() - 86400_000).toISOString()
  return {
    id: DEMO_SAVED_ID,
    origin: 'manual',
    created_at: new Date(Date.now() - 3 * 86400_000).toISOString(),
    updated_at: new Date(Date.now() - 3 * 86400_000).toISOString(),
    status: 'settled',
    settled_at: settledAt,
    expires_at: new Date(Date.now() + 89 * 86400_000).toISOString(),
    summary: summarise(legs),
    legs,
  }
}

const saved = new Map<string, { ticketId: string; ticket: SavedTicket }>()
export function resetSavedMock() {
  analyses.clear()
  saved.clear()
  saved.set(DEMO_SAVED_ID, { ticketId: DEMO_TICKET_ID, ticket: demo() })
}
resetSavedMock()

const newId = () => {
  const b = new Uint8Array(16)
  crypto.getRandomValues(b)
  return btoa(String.fromCharCode(...b)).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')
}

export const savedHandlers = [
  http.post(API, async ({ request }) => {
    const { ticket_id } = (await request.json()) as { ticket_id: string }
    const a = analyses.get(ticket_id)
    if (!a) return problem(404, 'Analyse the ticket before saving it.')
    const existing = [...saved.entries()].find(([, v]) => v.ticketId === ticket_id)
    const id = existing?.[0] ?? newId()
    const legs: SavedLeg[] = a.legs.map((l) => ({
      fixture: l.fixture,
      match_status: 'scheduled',
      score: null,
      original: { market: l.market, odds: l.odds, estimate: l.estimate, risk_band: l.risk_band, outcome: 'pending' },
      final: { market: l.market, outcome: 'pending' },
      suggestions: l.suggestions.map((s) => ({ market: s.market, estimate: s.estimate, risk_band: s.risk_band, accepted: false, outcome: 'pending' })),
    }))
    const now = new Date().toISOString()
    const ticket: SavedTicket = {
      id,
      origin: 'manual',
      created_at: existing?.[1].ticket.created_at ?? now,
      updated_at: now,
      status: 'pending',
      settled_at: null,
      expires_at: null,
      summary: summarise(legs),
      legs,
    }
    saved.set(id, { ticketId: ticket_id, ticket })
    return HttpResponse.json(ticket)
  }),

  http.get(API, ({ request }) => {
    const ids = (new URL(request.url).searchParams.get('ids') ?? '').split(',').filter(Boolean)
    return HttpResponse.json(ids.flatMap((id) => (saved.has(id) ? [{ ...saved.get(id)!.ticket, legs: undefined }] : [])))
  }),

  http.get(`${API}/:id`, ({ params }) => {
    const t = saved.get(String(params.id))
    return t
      ? HttpResponse.json(t.ticket)
      : problem(404, 'No saved ticket with that link. Saved tickets are kept for 90 days after their last game.')
  }),

  http.delete(`${API}/:id`, async ({ params, request }) => {
    const { ticket_id } = (await request.json()) as { ticket_id: string }
    const t = saved.get(String(params.id))
    if (!t || t.ticketId !== ticket_id) return problem(404, 'No saved ticket of yours with that link.')
    saved.delete(String(params.id))
    return new HttpResponse(null, { status: 204 })
  }),
]
