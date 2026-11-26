import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { FIXTURES } from '../mocks/data'
import { server } from '../test/server'
import { renderPage } from '../test/render'
import { TICKET_STORAGE_KEY, type TicketState } from '../state/ticketReducer'
import Builder from './Builder'

beforeAll(() => server.listen({ onUnhandledFrame: 'error' }))
afterEach(() => {
  server.resetHandlers()
  sessionStorage.clear()
})
afterAll(() => server.close())

const epl = FIXTURES.filter((f) => f.league_id === 'epl')
const first = epl[0]
/** jsdom trims hidden separators ("BrightonvFulham"); Chrome keeps the spaces. */
const matchRow = (home: string, away: string) => new RegExp(`${home}\\s*v\\s*${away}`)
const fixtureButton = (name: RegExp | string) => screen.findByRole('button', { name })
const slip = () => screen.getByRole('complementary', { name: 'Your ticket' })

function renderBuilder() {
  renderPage(<Builder />, { path: '/ticket' })
}

describe('Ticket builder (FR-3)', () => {
  it('lists fixtures for the first league', async () => {
    renderBuilder()
    expect(await screen.findByRole('button', { name: 'Premier League', pressed: true })).toBeInTheDocument()
    for (const f of epl) {
      expect(await fixtureButton(matchRow(f.home.name, f.away.name))).toBeInTheDocument()
    }
  })

  it('adds a leg with a market and odds', async () => {
    const user = userEvent.setup()
    renderBuilder()
    await user.click(await fixtureButton(matchRow(first.home.name, first.away.name)))

    const sheet = await screen.findByRole('dialog')
    const add = within(sheet).getByRole('button', { name: 'Choose a pick first' })
    expect(add).toBeDisabled()

    await user.click(within(sheet).getByRole('button', { name: /3 or more goals in the match/ }))
    await user.type(within(sheet).getByLabelText(/Bookmaker odds/), '1,9')
    await user.click(within(sheet).getByRole('button', { name: 'Add to ticket' }))

    await waitFor(() => expect(within(slip()).getByText('1 of 15 legs')).toBeInTheDocument())
    expect(within(slip()).getByRole('button', { name: /Change pick: 3 or more goals/ })).toBeInTheDocument()
    expect(within(slip()).getByDisplayValue('1.90')).toBeInTheDocument()
    expect(await fixtureButton(/on ticket:\s*Over 2\.5/)).toBeInTheDocument()

    const saved = JSON.parse(sessionStorage.getItem(TICKET_STORAGE_KEY)!) as TicketState
    expect(saved.legs).toMatchObject([{ market: 'OVER_2_5', odds: 1.9 }])
  })

  it('keeps one leg per match: re-picking updates the leg (FR-3.4)', async () => {
    const user = userEvent.setup()
    renderBuilder()
    const row = matchRow(first.home.name, first.away.name)

    await user.click(await fixtureButton(row))
    await user.click(within(await screen.findByRole('dialog')).getByRole('button', { name: /Both teams to score/ }))
    await user.click(screen.getByRole('button', { name: 'Add to ticket' }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())

    await user.click(await fixtureButton(row))
    const sheet = await screen.findByRole('dialog')
    expect(within(sheet).getByRole('button', { name: /Both teams to score/, pressed: true })).toBeInTheDocument()
    await user.click(within(sheet).getByRole('button', { name: /to draw/ }))
    await user.click(within(sheet).getByRole('button', { name: 'Update leg' }))

    await waitFor(() => expect(within(slip()).getByText('1 of 15 legs')).toBeInTheDocument())
    expect(await fixtureButton(/on ticket:\s*Draw/)).toBeInTheDocument()
  })

  it('blocks invalid odds', async () => {
    const user = userEvent.setup()
    renderBuilder()
    await user.click(await fixtureButton(matchRow(first.home.name, first.away.name)))
    const sheet = await screen.findByRole('dialog')
    await user.click(within(sheet).getByRole('button', { name: /to draw/ }))
    await user.type(within(sheet).getByLabelText(/Bookmaker odds/), '5/2')
    expect(within(sheet).getByRole('button', { name: 'Add to ticket' })).toBeDisabled()
    await user.tab()
    expect(within(sheet).getByText('Use decimal odds, e.g. 1.85')).toBeInTheDocument()
  })

  it('removes a leg from the slip', async () => {
    seedTicket(1)
    const user = userEvent.setup()
    renderBuilder()
    await user.click(within(slip()).getByRole('button', { name: /^Remove / }))
    expect(within(slip()).getByText('0 of 15 legs')).toBeInTheDocument()
  })

  it('stops adding new matches at 15 legs (FR-3.4)', async () => {
    seedTicket(15)
    renderBuilder()
    expect(await screen.findByText(/maximum 15 legs/)).toBeInTheDocument()
    const row = await fixtureButton(matchRow(first.home.name, first.away.name))
    expect(row).toBeDisabled()
  })

  it('goes to the analysis screen', async () => {
    seedTicket(2)
    const user = userEvent.setup()
    renderBuilder()
    await user.click(within(slip()).getByRole('button', { name: 'Analyse ticket' }))
    expect(screen.getByTestId('location')).toHaveTextContent('/ticket/analysis')
  })
})

/** Put n legs on the session ticket, using fixtures that are not in the EPL list. */
function seedTicket(n: number) {
  const legs = Array.from({ length: n }, (_, i) => ({
    fixture: { ...first, id: `seed-${i}`, home: { id: `h${i}`, name: `Home ${i}` }, away: { id: `a${i}`, name: `Away ${i}` } },
    market: 'HOME_WIN',
    odds: null,
  }))
  const state = { ticketId: 'seed', origin: 'manual', legs, swaps: [], baseline: null } as TicketState
  sessionStorage.setItem(TICKET_STORAGE_KEY, JSON.stringify(state))
}
