/**
 * Mock admin API (B6), for development without the Go API and for tests.
 * Sign in as "admin" with the password "mock password 123". The real API keeps
 * the session in an HttpOnly cookie; here it's a flag in memory.
 */
import { delay, http, HttpResponse } from 'msw'
import type { Admin, AdminMatch, Dashboard, Threshold } from '../api/admin'
import { RULES } from '../config/rules'

const API = '*/api/admin'
export const MOCK_ADMIN = { username: 'admin', password: 'mock password 123' }

const problem = (status: number, detail: string) =>
  HttpResponse.json({ type: 'about:blank', title: 'Error', status, detail }, { status, headers: { 'Content-Type': 'application/problem+json' } })

const hoursAgo = (h: number) => new Date(Date.now() - h * 3600_000).toISOString()

function seedMatches(): AdminMatch[] {
  return [
    { id: '9001', league_id: 'epl', kickoff: hoursAgo(3), status: 'scheduled', home: 'Arsenal', away: 'Chelsea', home_goals: null, away_goals: null, source: 'football-data.org' },
    { id: '9002', league_id: 'laliga', kickoff: hoursAgo(26), status: 'postponed', home: 'Sevilla', away: 'Valencia', home_goals: null, away_goals: null, source: 'football-data.org' },
    { id: '9003', league_id: 'bundesliga', kickoff: hoursAgo(50), status: 'finished', home: 'Mainz', away: 'Freiburg', home_goals: 1, away_goals: 1, source: 'football-data.co.uk' },
  ]
}

let signedIn = false
let matches = seedMatches()

/** Back to signed out with the original games (tests call this between cases). */
export function resetAdminMock() {
  signedIn = false
  matches = seedMatches()
}

const ME: Admin = { id: '1', username: MOCK_ADMIN.username, role: 'admin', last_login_at: null }

const THRESHOLDS: Threshold[] = [
  { key: 'form_window', label: 'Form window', description: 'How many recent home or away games a team profile uses (FR-2.1).', value: RULES.formWindow, default: 10, unit: 'count' },
  { key: 'low_confidence_below', label: 'Low confidence below', description: 'Profiles built from fewer games are marked low confidence (FR-2.3).', value: RULES.lowConfidenceBelow, default: 5, unit: 'count' },
  { key: 'low_from', label: 'Low risk from', description: 'Estimates at or above this are Low risk (4.2).', value: RULES.band.lowFrom, default: 0.65, unit: 'percent' },
  { key: 'medium_from', label: 'Medium risk from', description: 'Estimates at or above this, but below Low, are Medium risk; anything lower is High (4.2).', value: RULES.band.mediumFrom, default: 0.45, unit: 'percent' },
  { key: 'swap_min_gain', label: 'Swap minimum gain', description: 'A swap is only suggested if its estimate is at least this much higher (4.3).', value: RULES.swap.minGain, default: 0.1, unit: 'percent' },
  { key: 'max_suggestions', label: 'Suggestions per leg', description: 'The most swaps offered for one leg (4.3).', value: RULES.swap.maxSuggestions, default: 2, unit: 'count' },
  { key: 'value_margin', label: 'Value margin', description: "A pick is marked value when its estimate beats the odds' implied chance by this much (4.5).", value: RULES.valueMargin, default: 0.05, unit: 'percent' },
  { key: 'bold_floor', label: 'Bold floor', description: 'The lowest estimate the ticket wizard will ever use (4.6).', value: RULES.wizard.boldFloor, default: 0.35, unit: 'percent' },
  { key: 'max_estimate', label: 'Estimate cap', description: 'No estimate goes above this, so nothing looks certain (addendum A7).', value: RULES.maxEstimate, default: 0.95, unit: 'percent' },
]

const awaiting = () => matches.filter((m) => m.status !== 'finished')

export const adminHandlers = [
  http.post(`${API}/login`, async ({ request }) => {
    await delay(150)
    const body = (await request.json()) as { username?: string; password?: string }
    if (body.username?.trim().toLowerCase() !== MOCK_ADMIN.username || body.password !== MOCK_ADMIN.password) {
      return problem(401, 'Wrong username or password.')
    }
    signedIn = true
    return HttpResponse.json({ admin: ME, expires_at: new Date(Date.now() + 12 * 3600_000).toISOString() })
  }),

  http.post(`${API}/logout`, () => {
    signedIn = false
    return new HttpResponse(null, { status: 204 })
  }),

  // Everything below needs a session.
  http.all(`${API}/*`, () => (signedIn ? undefined : problem(401, 'Please sign in.'))),

  http.get(`${API}/me`, () => HttpResponse.json(ME)),

  http.get(`${API}/dashboard`, async () => {
    await delay(150)
    const d: Dashboard = {
      matches: { upcoming: 73, awaiting_result: awaiting().length, finished: 1395 + matches.filter((m) => m.status === 'finished').length },
      tickets: { total: 42, last_7_days: 17, legs: 268 },
      suggestions: { shown: 121, accepted: 38 },
      imports: [
        { id: '12', started_at: hoursAgo(5), finished_at: hoursAgo(4.98), status: 'succeeded', error: null, summary: { leagues: [{ league: 'epl', inserted: 3, updated: 1, fixtures: { inserted: 2, updated: 4 } }] } },
        { id: '11', started_at: hoursAgo(23), finished_at: hoursAgo(22.98), status: 'succeeded', error: null, summary: { leagues: [{ league: 'laliga', inserted: 0, updated: 0, fixtures: { inserted: 0, updated: 1, unmatched: ['Real Oviedo'] } }] } },
      ],
    }
    return HttpResponse.json(d)
  }),

  http.get(`${API}/matches`, async ({ request }) => {
    await delay(150)
    const q = new URL(request.url).searchParams
    const view = q.get('view') ?? 'awaiting'
    const league = q.get('league')
    const list = (view === 'recent' ? matches.filter((m) => m.status === 'finished') : awaiting()).filter(
      (m) => !league || m.league_id === league,
    )
    return HttpResponse.json(list)
  }),

  http.put(`${API}/matches/:id/result`, async ({ params, request }) => {
    await delay(150)
    const m = matches.find((x) => x.id === params.id)
    if (!m) return problem(404, `No match with id "${params.id}".`)
    const { home_goals, away_goals } = (await request.json()) as { home_goals: number; away_goals: number }
    if (![home_goals, away_goals].every((g) => Number.isInteger(g) && g >= 0 && g <= 20)) {
      return problem(422, 'Goals must be between 0 and 20.')
    }
    Object.assign(m, { status: 'finished', home_goals, away_goals, source: 'manual' })
    return new HttpResponse(null, { status: 204 })
  }),

  http.get(`${API}/thresholds`, () => HttpResponse.json(THRESHOLDS)),
]
