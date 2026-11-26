// @vitest-environment node
import { setupServer } from 'msw/node'
import type { AnalyseResponse, Fixture, League, WizardResponse } from '../api/types'
import { handlers } from './handlers'

const server = setupServer(...handlers)
beforeAll(() => server.listen({ onUnhandledFrame: 'error' }))
afterAll(() => server.close())

const BASE = 'http://localhost/api'
const get = async <T,>(p: string) => (await fetch(`${BASE}${p}`)).json() as Promise<T>
const post = (p: string, body: unknown) =>
  fetch(`${BASE}${p}`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) })

describe('mock API contract', () => {
  it('lists the three MVP leagues with fixtures', async () => {
    const leagues = await get<League[]>('/leagues')
    expect(leagues.map((l) => l.id)).toEqual(['epl', 'laliga', 'bundesliga'])
    const fixtures = await get<Fixture[]>('/leagues/epl/fixtures?when=upcoming')
    expect(fixtures.length).toBeGreaterThan(0)
  })

  it('analyses a ticket and returns combined probability and weakest leg', async () => {
    const fixtures = await get<Fixture[]>('/leagues/epl/fixtures?when=upcoming')
    const legs = fixtures.slice(0, 3).map((f) => ({ match_id: f.id, market: 'AWAY_WIN', odds: 3.2 }))
    const res = await post('/tickets/analyse', { ticket_id: 't1', origin: 'manual', legs })
    expect(res.ok).toBe(true)
    const body = (await res.json()) as AnalyseResponse
    expect(body.legs).toHaveLength(3)
    const product = body.legs.reduce((p, l) => p * l.estimate, 1)
    expect(body.combined.probability).toBeCloseTo(product)
    expect(body.combined.odds).toBeCloseTo(3.2 ** 3)
    const weakest = body.legs.find((l) => l.id === body.weakest_leg_id)!
    expect(weakest.estimate).toBe(Math.min(...body.legs.map((l) => l.estimate)))
    for (const l of body.legs) {
      expect(l.suggestions.length).toBeLessThanOrEqual(2)
      if (l.risk_band === 'low') expect(l.suggestions).toHaveLength(0)
    }
  })

  it('rejects two legs on the same match (FR-3.4)', async () => {
    const [f] = await get<Fixture[]>('/leagues/epl/fixtures?when=upcoming')
    const res = await post('/tickets/analyse', {
      ticket_id: 't2',
      origin: 'manual',
      legs: [
        { match_id: f.id, market: 'HOME_WIN' },
        { match_id: f.id, market: 'BTTS_YES' },
      ],
    })
    expect(res.status).toBe(422)
  })

  it('builds a wizard ticket with one leg per match and fair odds', async () => {
    const res = await post('/wizard', { path: 'guided', risk: 'balanced', focus: 'mix', legs: 5, when: 'weekend', seed: 0 })
    const body = (await res.json()) as WizardResponse
    const ids = body.legs.map((l) => l.fixture.id)
    expect(new Set(ids).size).toBe(ids.length)
    for (const l of body.legs) {
      expect(l.risk_band).not.toBe('high')
      expect(l.fair_odds).toBeCloseTo(1 / l.estimate)
    }
    expect(body.legs.length).toBeLessThanOrEqual(5)
    if (body.legs.length < 5) expect(body.shortfall_reason).toBeTruthy()
  })

  it('caps beginner tickets at 4 games', async () => {
    const res = await post('/wizard', { path: 'beginner', risk: 'safe', focus: 'mix', legs: 9, when: 'weekend', seed: 0 })
    const body = (await res.json()) as WizardResponse
    expect(body.legs.length).toBeLessThanOrEqual(4)
  })

  it('Bold picks longer odds than Safe for the same games', async () => {
    const ticket = async (risk: string) =>
      ((await (await post('/wizard', { path: 'guided', risk, focus: 'mix', legs: 2, when: 'weekend', seed: 0 })).json()) as WizardResponse).legs
    const safe = await ticket('safe')
    const bold = await ticket('bold')
    const odds = (legs: WizardResponse['legs']) => legs.reduce((o, l) => o * l.fair_odds, 1)
    expect(bold.some((l) => l.risk_band === 'high')).toBe(true)
    expect(odds(bold)).toBeGreaterThan(odds(safe))
  })
})
