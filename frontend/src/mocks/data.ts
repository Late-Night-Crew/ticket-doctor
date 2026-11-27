/**
 * Deterministic fake data for the mock API: leagues, teams, profiles and
 * fixtures for the coming week. Numbers are invented, not real stats.
 */
import type { Fixture, League, Team } from '../api/types'
import { RULES } from '../config/rules'
import type { TeamProfile } from './scorer'

export const LEAGUES: League[] = [
  { id: 'epl', name: 'Premier League', country: 'England' },
  { id: 'laliga', name: 'La Liga', country: 'Spain' },
  { id: 'bundesliga', name: 'Bundesliga', country: 'Germany' },
]

/** Team names; a trailing * marks a newly promoted side with too few matches. */
const TEAM_NAMES: Record<string, string[]> = {
  epl: ['Arsenal', 'Aston Villa', 'Brentford', 'Brighton', 'Chelsea', 'Everton', 'Fulham', 'Liverpool', 'Man City', 'Newcastle', 'Tottenham', 'Sunderland*'],
  laliga: ['Athletic Club', 'Atlético Madrid', 'Barcelona', 'Betis', 'Celta Vigo', 'Girona', 'Real Madrid', 'Real Sociedad', 'Sevilla', 'Valencia', 'Villarreal', 'Levante*'],
  bundesliga: ['Bayer Leverkusen', 'Bayern Munich', 'Dortmund', 'Eintracht Frankfurt', 'Freiburg', 'Hoffenheim', 'Mainz', 'RB Leipzig', 'Stuttgart', 'Werder Bremen', 'Wolfsburg', 'Hamburg*'],
}

/* ---------- Seeded random ---------- */

function hash(s: string): number {
  let h = 2166136261
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i)
    h = Math.imul(h, 16777619)
  }
  return h >>> 0
}

export function seeded(seed: number): () => number {
  let a = seed >>> 0
  return () => {
    a = (a + 0x6d2b79f5) >>> 0
    let t = a
    t = Math.imul(t ^ (t >>> 15), t | 1)
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61)
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }
}

const slug = (s: string) =>
  s
    .normalize('NFD')
    .replace(/[̀-ͯ]/g, '')
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')

/* ---------- Teams and profiles ---------- */

interface MockTeam extends Team {
  league_id: string
  strength: number
  profiles: Record<'home' | 'away', TeamProfile>
}

const clamp = (x: number, lo = 0, hi = 1) => Math.min(hi, Math.max(lo, x))

function makeProfile(name: string, venue: 'home' | 'away', strength: number, sample: number): TeamProfile {
  const rnd = seeded(hash(`${name}:${venue}`))
  const n = sample
  const c = (rate: number) => Math.round(clamp(rate) * n) / n
  const homeEdge = venue === 'home' ? 0.08 : -0.05

  const win = c(0.12 + 0.6 * strength + homeEdge + (rnd() - 0.5) * 0.1)
  const draw = c(Math.min(1 - win, 0.18 + rnd() * 0.15))
  const loss = Math.max(0, Math.round((1 - win - draw) * n) / n)
  const attack = clamp(strength + (rnd() - 0.5) * 0.3)
  const defence = clamp(strength + (rnd() - 0.5) * 0.3)
  const over15 = c(0.6 + rnd() * 0.3)
  const over25 = c(Math.min(over15, 0.35 + rnd() * 0.3))
  const under35 = c(Math.max(1 - over25, 0.6 + rnd() * 0.3))

  return {
    team: name,
    venue,
    sample_size: n,
    win_rate: win,
    draw_rate: draw,
    loss_rate: loss,
    scored_rate: c(0.5 + 0.45 * attack + homeEdge),
    clean_sheet_rate: c(0.1 + 0.4 * defence + homeEdge / 2),
    btts_rate: c(0.35 + rnd() * 0.35),
    over15_rate: over15,
    over25_rate: over25,
    under35_rate: under35,
  }
}

function formString(name: string): string {
  const rnd = seeded(hash(`${name}:form`))
  return Array.from({ length: 5 }, () => {
    const r = rnd()
    return r < 0.45 ? 'W' : r < 0.7 ? 'D' : 'L'
  }).join('-')
}

export const TEAMS: MockTeam[] = Object.entries(TEAM_NAMES).flatMap(([leagueId, names]) =>
  names.map((raw, i) => {
    const promoted = raw.endsWith('*')
    const name = raw.replace('*', '')
    const strength = clamp(1 - i / names.length + (seeded(hash(name))() - 0.5) * 0.3)
    const sample = promoted ? 4 : RULES.formWindow
    return {
      id: slug(name),
      name,
      league_id: leagueId,
      form: formString(name),
      strength,
      profiles: {
        home: makeProfile(name, 'home', strength, sample),
        away: makeProfile(name, 'away', strength, sample),
      },
    }
  }),
)

const TEAM_BY_ID = new Map(TEAMS.map((t) => [t.id, t]))

export function profilesFor(fixture: Fixture): { home: TeamProfile; away: TeamProfile } {
  const home = TEAM_BY_ID.get(fixture.home.id)
  const away = TEAM_BY_ID.get(fixture.away.id)
  if (!home || !away) throw new Error(`No profiles for fixture ${fixture.id}`)
  return { home: home.profiles.home, away: away.profiles.away }
}

/* ---------- Fixtures ---------- */

/** "Today" and "this weekend" are Central Africa Time days, as on the server (SRS addendum A5). */
const CAT_ZONE = 'Africa/Blantyre'

/** The CAT calendar date of an instant, as YYYY-MM-DD. */
export function catDate(d: Date): string {
  return d.toLocaleDateString('en-CA', { timeZone: CAT_ZONE })
}

function addDays(ymd: string, n: number): string {
  const d = new Date(`${ymd}T00:00:00Z`)
  d.setUTCDate(d.getUTCDate() + n)
  return d.toISOString().slice(0, 10)
}

/** Today and this weekend's Saturday and Sunday, as CAT dates (see fixturesIn). */
function weekendDays(now: Date): { today: string; saturday: string; sunday: string } {
  const today = catDate(now)
  const weekday = new Date(`${today}T00:00:00Z`).getUTCDay() // 0 Sunday … 6 Saturday
  if (weekday === 0) return { today, saturday: today, sunday: today } // the rest of the weekend is today
  const saturday = addDays(today, 6 - weekday)
  return { today, saturday, sunday: addDays(saturday, 1) }
}

const KICKOFFS = [[19, 30], [20, 0], [14, 0], [16, 30], [17, 30], [21, 0]]

function buildFixtures(now = new Date()): Fixture[] {
  const { today, saturday, sunday } = weekendDays(now)
  const days = [today, saturday, sunday]
  const out: Fixture[] = []
  for (const league of LEAGUES) {
    const teams = TEAMS.filter((t) => t.league_id === league.id)
    const rnd = seeded(hash(`${league.id}:${today}`))
    const order = [...teams].sort(() => rnd() - 0.5)
    // Two matches today, the rest across the weekend.
    for (let i = 0; i + 1 < order.length; i += 2) {
      const matchNo = i / 2
      const day = matchNo < 2 ? days[0] : days[matchNo % 2 === 0 ? 1 : 2]
      const [hh, mm] = KICKOFFS[matchNo % KICKOFFS.length]
      // Kick-off times are CAT wall-clock times (UTC+2, no daylight saving).
      const kickoff = new Date(`${day}T${String(hh).padStart(2, '0')}:${String(mm).padStart(2, '0')}:00+02:00`)
      const home = order[i]
      const away = order[i + 1]
      out.push({
        id: `${league.id}-${home.id}-${away.id}`,
        league_id: league.id,
        kickoff: kickoff.toISOString(),
        home: { id: home.id, name: home.name, form: home.form },
        away: { id: away.id, name: away.name, form: away.form },
      })
    }
  }
  return out.sort((a, b) => a.kickoff.localeCompare(b.kickoff))
}

export const FIXTURES = buildFixtures()
const FIXTURE_BY_ID = new Map(FIXTURES.map((f) => [f.id, f]))

export function fixtureById(id: string): Fixture | undefined {
  return FIXTURE_BY_ID.get(id)
}

/**
 * Fixtures in a window, by CAT date: today; this weekend's Saturday and Sunday
 * (on a weekend day, the rest of that weekend); or everything. Unlike the
 * server, the mock doesn't hide games that have already kicked off today.
 */
export function fixturesIn(when: 'today' | 'weekend' | 'upcoming', leagueId?: string, now = new Date()): Fixture[] {
  const { today, saturday, sunday } = weekendDays(now)
  return FIXTURES.filter((f) => {
    if (leagueId && f.league_id !== leagueId) return false
    if (when === 'upcoming') return true
    const day = catDate(new Date(f.kickoff))
    if (when === 'today') return day === today
    return day >= saturday && day <= sunday
  })
}
