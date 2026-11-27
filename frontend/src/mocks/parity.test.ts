// @vitest-environment node
/**
 * Cross-language parity: the Go rules engine (backend/engine) must give exactly
 * the same answers as this mock scorer. Both sides test against one golden file,
 * backend/engine/testdata/parity.json, so a rule changed in only one language
 * fails CI.
 *
 * Regenerate after an intended rule change (then update the Go engine to match):
 *   UPDATE_PARITY=1 npx vitest run src/mocks/parity.test.ts
 */
import { existsSync, readFileSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'
import { MARKETS } from '../config/markets'
import { LEAGUES, TEAMS } from './data'
import { buildWizardTicket, estimateMarket, suggestSwaps, wizardCandidates, type TeamProfile } from './scorer'

const GOLDEN = join(process.cwd(), '..', 'backend', 'engine', 'testdata', 'parity.json')

interface Pair {
  match_id: string
  home: TeamProfile
  away: TeamProfile
}

/** Neighbouring teams in each league, plus the last pair reversed (a promoted side at home). */
function pairs(): Pair[] {
  const out: Pair[] = []
  for (const league of LEAGUES) {
    const teams = TEAMS.filter((t) => t.league_id === league.id)
    for (let i = 0; i + 1 < teams.length; i++) {
      out.push({ match_id: `${teams[i].id}-v-${teams[i + 1].id}`, home: teams[i].profiles.home, away: teams[i + 1].profiles.away })
    }
    const [a, b] = teams.slice(-2)
    out.push({ match_id: `${b.id}-v-${a.id}`, home: b.profiles.home, away: a.profiles.away })
  }
  return out
}

function build() {
  const ps = pairs()
  const matches = ps.map((p) => {
    const estimates = MARKETS.map((m) => estimateMarket(m.key, p.home, p.away))
    return {
      match_id: p.match_id,
      home: p.home,
      away: p.away,
      estimates,
      swaps: Object.fromEntries(estimates.map((e) => [e.market, suggestSwaps(e, p.home, p.away).map((s) => s.market)])),
    }
  })

  const wizard = []
  for (const focus of ['wins', 'goals', 'mix'] as const) {
    const candidates = ps.map((p) => wizardCandidates(p.match_id, p.home, p.away, focus))
    for (const risk of ['safe', 'balanced', 'bold'] as const) {
      for (const legs of [2, 3, 5, 10]) {
        wizard.push({
          focus,
          risk,
          legs,
          ticket: buildWizardTicket(candidates, risk, legs).map((l) => ({ match_id: l.matchId, market: l.market })),
        })
      }
    }
  }
  return { generated_by: 'frontend/src/mocks/parity.test.ts', matches, wizard }
}

describe('engine parity (TypeScript mock vs Go engine)', () => {
  it('matches the golden file shared with backend/engine', () => {
    const actual = build()
    if (process.env.UPDATE_PARITY === '1') {
      writeFileSync(GOLDEN, JSON.stringify(actual, null, 1) + '\n')
      return
    }
    expect(existsSync(GOLDEN), `${GOLDEN} missing: run with UPDATE_PARITY=1`).toBe(true)
    expect(actual).toEqual(JSON.parse(readFileSync(GOLDEN, 'utf8')))
  })
})
