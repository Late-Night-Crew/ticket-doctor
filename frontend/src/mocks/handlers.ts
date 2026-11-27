import { delay, http, HttpResponse } from 'msw'
import type {
  AnalyseRequest,
  AnalyseResponse,
  AnalysedLeg,
  ResultsResponse,
  WizardRequest,
  WizardResponse,
} from '../api/types'
import { RULES } from '../config/rules'
import { adminHandlers } from './admin'
import { fixtureById, fixturesIn, LEAGUES, profilesFor, seeded } from './data'
import {
  buildWizardTicket,
  combinedProbability,
  estimateMarket,
  impliedProbability,
  isValue,
  suggestSwaps,
  wizardCandidates,
} from './scorer'

const API = '*/api'

/** Errors in the same RFC 7807 shape as the Go API (pkg/problemdetail). */
function problem(status: number, detail: string) {
  return HttpResponse.json(
    { type: 'about:blank', title: status === 404 ? 'Not Found' : 'Unprocessable Entity', status, detail },
    { status, headers: { 'Content-Type': 'application/problem+json' } },
  )
}
const LATENCY = 250

let legSeq = 0

export const handlers = [
  ...adminHandlers,
  http.get(`${API}/leagues`, async () => {
    await delay(LATENCY)
    return HttpResponse.json(LEAGUES)
  }),

  http.get(`${API}/leagues/:id/fixtures`, async ({ params, request }) => {
    await delay(LATENCY)
    const when = (new URL(request.url).searchParams.get('when') ?? 'upcoming') as 'today' | 'weekend' | 'upcoming'
    return HttpResponse.json(fixturesIn(when, String(params.id)))
  }),

  http.post(`${API}/tickets/analyse`, async ({ request }) => {
    await delay(LATENCY * 2)
    const body = (await request.json()) as AnalyseRequest

    if (!body.legs?.length || body.legs.length > RULES.maxLegs) {
      return problem(422, `A ticket holds 1–${RULES.maxLegs} legs.`)
    }
    const seen = new Set<string>()
    for (const l of body.legs) {
      if (seen.has(l.match_id)) {
        return problem(422, 'Only one leg per match.')
      }
      seen.add(l.match_id)
    }

    const legs: AnalysedLeg[] = []
    for (const input of body.legs) {
      const fixture = fixtureById(input.match_id)
      if (!fixture) return problem(404, `Unknown match ${input.match_id}`)
      const { home, away } = profilesFor(fixture)
      const e = estimateMarket(input.market, home, away)
      const id = `leg-${++legSeq}`
      const odds = input.odds ?? null
      legs.push({
        id,
        match_id: fixture.id,
        fixture,
        market: input.market,
        odds,
        estimate: e.estimate,
        risk_band: e.risk_band,
        reason: e.reason,
        sample_size: e.sample_size,
        window_n: RULES.formWindow,
        low_confidence: e.low_confidence,
        implied_prob: odds ? impliedProbability(odds) : null,
        is_value: odds ? isValue(e.estimate, odds) : null,
        suggestions: suggestSwaps(e, home, away).map((s, i) => ({
          id: `${id}-s${i + 1}`,
          market: s.market,
          estimate: s.estimate,
          risk_band: s.risk_band,
          reason: s.reason,
        })),
      })
    }

    const p = combinedProbability(legs.map((l) => l.estimate))
    const allOdds = legs.every((l) => l.odds)
    const weakest = legs.reduce((w, l) => (l.estimate < w.estimate ? l : w), legs[0])
    const res: AnalyseResponse = {
      ticket_id: body.ticket_id,
      legs,
      combined: {
        probability: p,
        one_in: Math.max(1, Math.round(1 / p)),
        odds: allOdds ? legs.reduce((o, l) => o * (l.odds as number), 1) : null,
      },
      weakest_leg_id: weakest.id,
    }
    return HttpResponse.json(res)
  }),

  http.post(`${API}/suggestions/:id/accept`, async () => {
    await delay(LATENCY / 2)
    return HttpResponse.json({ ok: true })
  }),

  http.post(`${API}/wizard`, async ({ request }) => {
    await delay(LATENCY * 2)
    const body = (await request.json()) as WizardRequest
    const max = body.path === 'beginner' ? RULES.wizard.beginnerMaxLegs : RULES.wizard.maxLegs
    const target = Math.min(max, Math.max(RULES.wizard.minLegs, Math.round(Number(body.legs) || 0)))

    let matches = fixturesIn(body.when).map((f) => {
      const { home, away } = profilesFor(f)
      return wizardCandidates(f.id, home, away, body.focus)
    })

    if (body.seed > 0) {
      // Regenerate: drop a random third of matches so a different ticket appears.
      const rnd = seeded(body.seed)
      const kept = matches.filter(() => rnd() > 0.33)
      if (kept.length >= target) matches = kept
    }

    const picked = buildWizardTicket(matches, body.risk, target)
    const day = body.when === 'today' ? 'today' : 'this weekend'
    const res: WizardResponse = {
      legs: picked.map((c) => ({
        fixture: fixtureById(c.matchId)!,
        market: c.market,
        estimate: c.estimate,
        risk_band: c.risk_band,
        reason: c.reason,
        sample_size: c.sample_size,
        window_n: RULES.formWindow,
        fair_odds: 1 / c.estimate,
      })),
      shortfall_reason:
        picked.length < target
          ? `Only ${picked.length} of the ${target} games you asked for met the ${body.risk} rules for ${day}, so the ticket is shorter. We never loosen the rules to fill a ticket.`
          : null,
    }
    return HttpResponse.json(res)
  }),

  http.get(`${API}/results`, async ({ request }) => {
    await delay(LATENCY)
    const q = new URL(request.url).searchParams
    const rows = RESULTS.rows.filter(
      (r) => (!q.get('league') || r.league_id === q.get('league')) && (!q.get('market') || r.market === q.get('market')),
    )
    return HttpResponse.json({ ...RESULTS, rows } satisfies ResultsResponse)
  }),
]

export const RESULTS: ResultsResponse = {
  updated_at: new Date().toISOString(),
  rows: [
    { league_id: 'epl', market: 'HOME_WIN', original: { won: 41, lost: 37 }, suggested: { won: 12, lost: 6 }, original_with_suggestion: { won: 13, lost: 18 } },
    { league_id: 'epl', market: 'AWAY_WIN', original: { won: 18, lost: 33 }, suggested: { won: 21, lost: 9 }, original_with_suggestion: { won: 6, lost: 16 } },
    { league_id: 'epl', market: 'OVER_2_5', original: { won: 29, lost: 24 }, suggested: { won: 15, lost: 7 }, original_with_suggestion: { won: 9, lost: 12 } },
    { league_id: 'epl', market: 'BTTS_YES', original: { won: 22, lost: 19 }, suggested: { won: 9, lost: 6 }, original_with_suggestion: { won: 7, lost: 9 } },
    { league_id: 'laliga', market: 'HOME_WIN', original: { won: 35, lost: 28 }, suggested: { won: 10, lost: 8 }, original_with_suggestion: { won: 11, lost: 14 } },
    { league_id: 'laliga', market: 'UNDER_3_5', original: { won: 20, lost: 6 }, suggested: { won: 26, lost: 7 }, original_with_suggestion: { won: 6, lost: 3 } },
    { league_id: 'laliga', market: 'DRAW', original: { won: 6, lost: 19 }, suggested: { won: 0, lost: 0 }, original_with_suggestion: { won: 2, lost: 9 } },
    { league_id: 'bundesliga', market: 'OVER_2_5', original: { won: 31, lost: 18 }, suggested: { won: 8, lost: 5 }, original_with_suggestion: { won: 10, lost: 9 } },
    { league_id: 'bundesliga', market: 'AWAY_WIN', original: { won: 15, lost: 22 }, suggested: { won: 14, lost: 9 }, original_with_suggestion: { won: 5, lost: 11 } },
    { league_id: 'bundesliga', market: 'HOME_TO_SCORE', original: { won: 27, lost: 5 }, suggested: { won: 19, lost: 4 }, original_with_suggestion: { won: 9, lost: 2 } },
  ],
}
