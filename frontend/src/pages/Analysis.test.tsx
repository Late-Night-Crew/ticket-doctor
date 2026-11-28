import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import type { Fixture, MarketKey } from '../api/types'
import { MARKETS, market } from '../config/markets'
import { FIXTURES, profilesFor } from '../mocks/data'
import { estimateMarket, suggestSwaps } from '../mocks/scorer'
import { TICKET_STORAGE_KEY, type TicketState } from '../state/ticketReducer'
import { renderPage } from '../test/render'
import { server } from '../test/server'
import Analysis from './Analysis'

beforeAll(() => server.listen({ onUnhandledFrame: 'error' }))
afterEach(() => {
  server.resetHandlers()
  sessionStorage.clear()
})
afterAll(() => server.close())

/** Find a leg in the mock data that is risky and has swap suggestions. */
function riskyLeg(): { fixture: Fixture; market: MarketKey; swapTo: MarketKey } {
  for (const fixture of FIXTURES) {
    const { home, away } = profilesFor(fixture)
    for (const m of MARKETS) {
      const e = estimateMarket(m.key, home, away)
      const swaps = suggestSwaps(e, home, away)
      if (e.risk_band === 'high' && swaps.length) return { fixture, market: m.key, swapTo: swaps[0].market }
    }
  }
  throw new Error('mock data has no risky leg with swaps')
}

/** A Low-risk leg on a different match. */
function safeLeg(exclude: string): { fixture: Fixture; market: MarketKey } {
  for (const fixture of FIXTURES.filter((f) => f.id !== exclude)) {
    const { home, away } = profilesFor(fixture)
    for (const m of MARKETS) {
      if (estimateMarket(m.key, home, away).risk_band === 'low') return { fixture, market: m.key }
    }
  }
  throw new Error('mock data has no low-risk leg')
}

function seed(legs: { fixture: Fixture; market: MarketKey; odds?: number | null }[]) {
  const state: TicketState = {
    ticketId: 'test-ticket',
    origin: 'manual',
    legs: legs.map((l) => ({ fixture: l.fixture, market: l.market, odds: l.odds ?? null })),
    swaps: [],
    baseline: null,
  }
  sessionStorage.setItem(TICKET_STORAGE_KEY, JSON.stringify(state))
}

const plain = (f: Fixture, m: MarketKey) => market(m).plain(f.home.name, f.away.name)
const renderAnalysis = () => renderPage(<Analysis />, { path: '/ticket/analysis' })

describe('Analysis screen (FR-4 to FR-6)', () => {
  it('shows an empty state with no legs', () => {
    renderAnalysis()
    expect(screen.getByRole('heading', { name: 'Nothing to analyse yet' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Go to the ticket builder' })).toHaveAttribute('href', '/ticket')
  })

  it('shows each leg with risk, reason, sample size, the disclaimer and combined chance', async () => {
    const risky = riskyLeg()
    const safe = safeLeg(risky.fixture.id)
    seed([risky, safe])
    renderAnalysis()

    expect(await screen.findByRole('heading', { name: plain(risky.fixture, risky.market) })).toBeInTheDocument()
    expect(screen.getByRole('note')).toHaveTextContent(/not guarantees/) // RG-1
    expect(screen.getAllByText(/High risk/).length).toBeGreaterThan(0)
    expect(screen.getAllByText(/last 10 matches/).length).toBeGreaterThan(0) // NFR-5
    expect(screen.getByText(/about 1 in \d+/)).toBeInTheDocument() // RG-2
    expect(screen.getByText('Weakest leg', { selector: '.td-eyebrow' })).toBeInTheDocument() // FR-6.3
  })

  it('accepts a swap in one tap, then shows before and after, and can undo (FR-5.3, FR-6.2)', async () => {
    const risky = riskyLeg()
    seed([{ ...risky, odds: 4.5 }])
    const user = userEvent.setup()
    renderAnalysis()

    const swapTo = plain(risky.fixture, risky.swapTo)
    await user.click(await screen.findByRole('button', { name: `Swap to ${swapTo}` }))

    expect(await screen.findByRole('heading', { name: swapTo })).toBeInTheDocument()
    await waitFor(() => expect(screen.getByText(/^was /)).toBeInTheDocument())
    const notice = screen.getByText(/Swapped from/).closest('.td-swapped') as HTMLElement
    expect(notice).toHaveTextContent(plain(risky.fixture, risky.market))
    expect(notice).toHaveTextContent(/Odds were cleared/)

    const saved = JSON.parse(sessionStorage.getItem(TICKET_STORAGE_KEY)!) as TicketState
    expect(saved.legs[0]).toMatchObject({ market: risky.swapTo, odds: null })

    await user.click(within(notice).getByRole('button', { name: 'Undo' }))
    expect(await screen.findByRole('heading', { name: plain(risky.fixture, risky.market) })).toBeInTheDocument()
    expect(screen.queryByText(/^was /)).not.toBeInTheDocument()
  })

  it('shows combined odds and bookmaker comparison when every leg has odds (FR-6.4)', async () => {
    const risky = riskyLeg()
    const safe = safeLeg(risky.fixture.id)
    seed([
      { ...risky, odds: 4 },
      { ...safe, odds: 1.5 },
    ])
    renderAnalysis()
    expect(await screen.findByText('6.00')).toBeInTheDocument()
    expect(screen.getByText(/price implies 16\.7%/)).toBeInTheDocument()
  })

  it('removes a leg', async () => {
    const risky = riskyLeg()
    const safe = safeLeg(risky.fixture.id)
    seed([risky, safe])
    const user = userEvent.setup()
    renderAnalysis()
    await user.click(await screen.findByRole('button', { name: `Remove leg: ${plain(risky.fixture, risky.market)}` }))
    await waitFor(() =>
      expect(screen.queryByRole('heading', { name: plain(risky.fixture, risky.market) })).not.toBeInTheDocument(),
    )
    expect(screen.getByRole('heading', { name: plain(safe.fixture, safe.market) })).toBeInTheDocument()
  })

  it('explains when limited data blocks swaps (FR-5.4)', async () => {
    const thin = FIXTURES.find((f) => /Sunderland|Levante|Hamburg/.test(f.home.name + f.away.name))!
    const { home, away } = profilesFor(thin)
    const risky = MARKETS.map((m) => estimateMarket(m.key, home, away)).find((e) => e.risk_band !== 'low')!
    seed([{ fixture: thin, market: risky.market }])
    renderAnalysis()
    expect(await screen.findByText(/too few recent matches/)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /^Swap to/ })).not.toBeInTheDocument()
  })

  it('shows the API error message', async () => {
    server.use(
      http.post('*/api/tickets/analyse', () =>
        HttpResponse.json({ type: 'about:blank', title: 'Unprocessable Entity', status: 422, detail: 'Only one leg per match.' }, { status: 422 }),
      ),
    )
    seed([riskyLeg()])
    renderAnalysis()
    expect(await screen.findByText('Only one leg per match.')).toBeInTheDocument()
  })
})

describe('How far will it get? (cash-out view)', () => {
  it('orders games by kick-off, shows the chance still alive, and answers for a cash-out point', async () => {
    const legs = [...FIXTURES].slice(0, 3).reverse() // deliberately not in kick-off order
    seed(legs.map((fixture) => ({ fixture, market: 'OVER_1_5' as MarketKey })))
    const user = userEvent.setup()
    renderAnalysis()

    const card = await screen.findByRole('region', { name: 'How far will it get?' })
    const rows = within(card).getAllByRole('listitem')
    expect(rows).toHaveLength(3)
    // Kick-off order, earliest first.
    const kickoffs = [...legs].sort((a, b) => Date.parse(a.kickoff) - Date.parse(b.kickoff))
    rows.forEach((row, i) => expect(row).toHaveTextContent(kickoffs[i].home.name))
    // "Still alive" only ever goes down.
    const alive = rows.map((r) => Number(r.querySelector('.td-howfar__alive .fw-bold')!.textContent!.replace('%', '')))
    expect(alive).toEqual([...alive].sort((a, b) => b - a))

    await user.selectOptions(within(card).getByLabelText('Cash out with'), '1 game left')
    const answer = within(card).getByRole('status')
    expect(answer).toHaveTextContent('Chance the first 2 games all win, so you reach the last 1 still alive:')
    expect(answer).toHaveTextContent(`${alive[1]}%`)
    // Never money, never advice on whether to cash out.
    expect(card).toHaveTextContent('we do not suggest whether to cash out')
  })

  it('is not shown for one or two legs', async () => {
    seed(FIXTURES.slice(0, 2).map((fixture) => ({ fixture, market: 'OVER_1_5' as MarketKey })))
    renderAnalysis()
    await screen.findAllByRole('article')
    expect(screen.queryByRole('region', { name: 'How far will it get?' })).not.toBeInTheDocument()
  })
})
