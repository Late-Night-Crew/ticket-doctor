/**
 * Mock admin API (B6), for development without the Go API and for tests.
 * Sign in as "admin" with the password "mock password 123". The real API keeps
 * the session in an HttpOnly cookie; here it's a flag in memory.
 */
import { delay, http, HttpResponse } from 'msw'
import type { Admin, AdminAccount, AdminMatch, Calibration, Dashboard, Threshold } from '../api/admin'
import { RULES } from '../config/rules'

const API = '*/api/admin'
export const MOCK_ADMIN = { username: 'admin', password: 'mock password 123' }
/** With setupOpen(true), /admin/setup takes this code (and no accounts exist). */
export const MOCK_SETUP_CODE = 'mock setup code 123'


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
let open = false
let password = MOCK_ADMIN.password
const seedAccounts = (): AdminAccount[] => [
  { id: '1', username: MOCK_ADMIN.username, role: 'admin', last_login_at: null, disabled: false, created_at: hoursAgo(72) },
  { id: '2', username: 'grace', role: 'admin', last_login_at: hoursAgo(30), disabled: false, created_at: hoursAgo(48) },
]
let accounts = seedAccounts()

/** Back to signed out with the original games and accounts (tests call this between cases). */
export function resetAdminMock() {
  signedIn = false
  matches = seedMatches()
  open = false
  password = MOCK_ADMIN.password
  accounts = seedAccounts()
}

/** Pretend no account exists yet, so first-admin setup is open. */
export function setupOpen(v: boolean) {
  open = v
  if (v) accounts = []
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
  { key: 'prior_games', label: 'League prior', description: "Each team's last-games record is pulled towards its league's average, as if it had this many extra average games, so ten games of form don't overshoot (backtest, 2026-11-28). 0 turns it off.", value: RULES.priorGames, default: 10, unit: 'count' },
  { key: 'max_estimate', label: 'Estimate cap', description: 'No estimate goes above this, so nothing looks certain (addendum A7).', value: RULES.maxEstimate, default: 0.95, unit: 'percent' },
]

const awaiting = () => matches.filter((m) => m.status !== 'finished')

export const adminHandlers = [
  http.post(`${API}/login`, async ({ request }) => {
    await delay(150)
    const body = (await request.json()) as { username?: string; password?: string }
    if (body.username?.trim().toLowerCase() !== MOCK_ADMIN.username || body.password !== password) {
      return problem(401, 'Wrong username or password.')
    }
    signedIn = true
    return HttpResponse.json({ admin: ME, expires_at: new Date(Date.now() + 12 * 3600_000).toISOString() })
  }),

  http.post(`${API}/logout`, () => {
    signedIn = false
    return new HttpResponse(null, { status: 204 })
  }),

  http.get(`${API}/setup`, () => HttpResponse.json({ open })),

  http.post(`${API}/setup`, async ({ request }) => {
    await delay(150)
    const body = (await request.json()) as { code: string; username: string; password: string }
    if (!open) return problem(403, 'Setup is closed: an admin account already exists. Sign in instead.')
    if (body.code !== MOCK_SETUP_CODE) return problem(401, "That setup code isn't right.")
    open = false
    signedIn = true
    accounts = [{ ...ME, username: body.username, disabled: false, created_at: new Date().toISOString() }]
    return HttpResponse.json({ admin: { ...ME, username: body.username }, expires_at: new Date(Date.now() + 12 * 3600_000).toISOString() })
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
        { id: '11', started_at: hoursAgo(23), finished_at: hoursAgo(22.98), status: 'succeeded', error: null, summary: { leagues: [{ league: 'laliga', inserted: 0, updated: 0, corrected: ['Betis v Vallecano 0-1 → 1-1 (was football-data.org)'], fixtures: { inserted: 0, updated: 1, results: 2, unmatched: ['Real Oviedo'] } }] } },
      ],
      score_checks: [
        { id: '14', started_at: hoursAgo(0.25), finished_at: hoursAgo(0.24), status: 'succeeded', error: null, summary: { leagues: [{ league: 'epl', results: 3 }, { league: 'bundesliga', results: 1 }] } },
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

  http.get(`${API}/calibration`, ({ request }) => {
    const source = new URL(request.url).searchParams.get('source') ?? 'backtest'
    return HttpResponse.json(source === 'tickets' ? EMPTY_CALIBRATION : BACKTEST)
  }),

  http.get(`${API}/users`, () => HttpResponse.json(accounts)),

  http.post(`${API}/users`, async ({ request }) => {
    const body = (await request.json()) as { username: string; password: string }
    if (accounts.some((a) => a.username === body.username)) return problem(422, `There is already an account called "${body.username}".`)
    const a: AdminAccount = { id: String(accounts.length + 10), username: body.username, role: 'admin', last_login_at: null, disabled: false, created_at: new Date().toISOString() }
    accounts = [...accounts, a].sort((x, y) => x.username.localeCompare(y.username))
    return HttpResponse.json(a, { status: 201 })
  }),

  http.put(`${API}/users/:username/disabled`, async ({ params, request }) => {
    const { disabled } = (await request.json()) as { disabled: boolean }
    if (params.username === MOCK_ADMIN.username) return problem(422, "You can't disable your own account.")
    accounts = accounts.map((a) => (a.username === params.username ? { ...a, disabled } : a))
    return new HttpResponse(null, { status: 204 })
  }),

  http.put(`${API}/me/password`, async ({ request }) => {
    const body = (await request.json()) as { current_password: string; new_password: string }
    if (body.current_password !== password) return problem(422, "Your current password isn't right.")
    password = body.new_password
    return HttpResponse.json({ admin: ME, expires_at: new Date(Date.now() + 12 * 3600_000).toISOString() })
  }),
]

/** The real backtest from 2026-11-28 (1,009 games), trimmed to a few markets. */
const BACKTEST: Calibration = {
  source: 'backtest',
  n: 15135,
  matches: 1009,
  skipped_low_confidence: 386,
  brier: 0.2153,
  bins: [
    [0.07, 0.194, 382], [0.16, 0.218, 1029], [0.239, 0.277, 1292], [0.334, 0.361, 2023], [0.429, 0.441, 1792],
    [0.527, 0.516, 2304], [0.63, 0.615, 2069], [0.728, 0.705, 2006], [0.825, 0.794, 1517], [0.92, 0.818, 721],
  ].map(([predicted, actual, n], i) => ({ from: i / 10, to: (i + 1) / 10, n, predicted, actual })),
  bands: [
    { band: 'low', n: 5246, predicted: 0.769, actual: 0.731 },
    { band: 'medium', n: 4194, predicted: 0.533, actual: 0.53 },
    { band: 'high', n: 5695, predicted: 0.276, actual: 0.314 },
  ],
  markets: [
    { market: 'HOME_WIN', n: 1009, predicted: 0.454, actual: 0.445, brier: 0.2316, base_brier: 0.247, skill: 0.062 },
    { market: 'BTTS_YES', n: 1009, predicted: 0.573, actual: 0.576, brier: 0.2563, base_brier: 0.2443, skill: -0.049 },
    { market: 'OVER_2_5', n: 1009, predicted: 0.557, actual: 0.561, brier: 0.2525, base_brier: 0.2463, skill: -0.025 },
  ],
}

const EMPTY_CALIBRATION: Calibration = {
  source: 'tickets', n: 0, matches: 0, skipped_low_confidence: 0, brier: 0,
  bins: BACKTEST.bins.map((b) => ({ ...b, n: 0, predicted: 0, actual: 0 })), bands: [], markets: [],
}
