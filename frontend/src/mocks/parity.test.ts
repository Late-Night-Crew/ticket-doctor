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

/**
 * A home side with a perfect record against an away side that lost every game:
 * several raw estimates reach 100%, so the 95% cap (RULES.maxEstimate) is checked.
 */
const PERFECT: Pair = {
  match_id: 'perfect-v-hopeless',
  home: {
    team: 'Perfect', venue: 'home', sample_size: 10, win_rate: 1, draw_rate: 0, loss_rate: 0, scored_rate: 1,
    clean_sheet_rate: 0.6, btts_rate: 0.4, over15_rate: 1, over25_rate: 0.8, under35_rate: 0.5,
  },
  away: {
    team: 'Hopeless', venue: 'away', sample_size: 10, win_rate: 0, draw_rate: 0, loss_rate: 1, scored_rate: 0.4,
    clean_sheet_rate: 0, btts_rate: 0.4, over15_rate: 1, over25_rate: 0.7, under35_rate: 0.5,
  },
}

/** Neighbouring teams in each league, plus the last pair reversed (a promoted side at home), plus PERFECT. */
function pairs(): Pair[] {
  const out: Pair[] = [PERFECT]
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
  it('caps estimates at 95%, keeping the real record in the reason', () => {
    const win = estimateMarket('HOME_WIN', PERFECT.home, PERFECT.away)
    expect(win.estimate).toBe(0.95)
    expect(win.reason).toContain('won 10 of last 10 home games')
    for (const m of MARKETS) expect(estimateMarket(m.key, PERFECT.home, PERFECT.away).estimate).toBeLessThanOrEqual(0.95)
  })

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
