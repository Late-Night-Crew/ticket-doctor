import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, useLocation } from 'react-router'
import App from '../App'
import { FIXTURES } from '../mocks/data'
import { awaitingResult } from '../api/hooks'
import type { SavedLeg } from '../api/types'
import { DEMO_LIVE_ID, DEMO_SAVED_ID, DEMO_TICKET_ID, resetSavedMock } from '../mocks/saved'
import { loadSaved, rememberSaved } from '../lib/savedStore'
import { TicketProvider } from '../state/TicketContext'
import { TICKET_STORAGE_KEY, type TicketState } from '../state/ticketReducer'
import { server } from '../test/server'

beforeAll(() => server.listen({ onUnhandledFrame: 'error' }))
afterEach(() => {
  server.resetHandlers()
  resetSavedMock()
  sessionStorage.clear()
  localStorage.clear()
})
afterAll(() => server.close())

function Where() {
  return <div data-testid="location">{useLocation().pathname}</div>
}

function renderAt(url: string) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[url]}>
        <TicketProvider>
          <App />
          <Where />
        </TicketProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('Saved tickets (V2, FR-9)', () => {
  it('saves an analysed ticket without an account and opens it from its link', async () => {
    const ticket: TicketState = {
      ticketId: 'b7a4c1d2-1111-4222-8333-944455556666',
      origin: 'manual',
      legs: [{ fixture: FIXTURES[0], market: 'OVER_1_5', odds: null }],
      swaps: [],
      baseline: null,
    }
    sessionStorage.setItem(TICKET_STORAGE_KEY, JSON.stringify(ticket))
    const user = userEvent.setup()
    renderAt('/ticket/analysis')

    const save = await screen.findByRole('button', { name: 'Save ticket' })
    await waitFor(() => expect(save).toBeEnabled())
    await user.click(save)

    const link = (await screen.findByRole('textbox', { name: 'Private link' })) as HTMLInputElement
    expect(link.value).toMatch(/\/t\/[A-Za-z0-9_-]{22}$/)
    // Kept on this device, with the ticket's own id for deleting later.
    expect(loadSaved()).toEqual([expect.objectContaining({ ticketId: ticket.ticketId, id: link.value.split('/t/')[1] })])
    // Nothing changed since saving, so no "Save changes" button.
    expect(screen.queryByRole('button', { name: 'Save changes' })).not.toBeInTheDocument()

    await user.click(screen.getByRole('link', { name: 'Open saved ticket' }))
    expect(await screen.findByRole('heading', { name: 'How your ticket did' })).toBeInTheDocument()
    expect(screen.getAllByText('Waiting for results').length).toBeGreaterThan(0)
    const leg = screen.getByRole('article', { name: `${FIXTURES[0].home.name} v ${FIXTURES[0].away.name}` })
    expect(within(leg).getAllByText('Not played yet').length).toBeGreaterThanOrEqual(2)
    expect(screen.getByRole('button', { name: 'Delete saved ticket' })).toBeInTheDocument()
  })

  it('compares picks with suggestions side by side once played, misses included', async () => {
    renderAt(`/t/${DEMO_SAVED_ID}`)
    const [a] = FIXTURES
    const leg = await screen.findByRole('article', { name: `${a.home.name} v ${a.away.name}` })
    expect(within(leg).getByText('0–2')).toBeInTheDocument()
    // The user's own pick lost; the swap they took and the one they didn't both won.
    expect(within(leg).getByText(`${a.home.name} to win`).parentElement).toHaveTextContent('Lost')
    expect(within(leg).getByText('you took this')).toBeInTheDocument()
    expect(within(leg).getAllByText('Won')).toHaveLength(3)

    const result = screen.getByRole('region', { name: 'Ticket result' })
    expect(within(result).getByText('Your ticket as you first picked it').nextElementSibling).toHaveTextContent('Lost')
    expect(within(result).getByText('Your ticket with the swaps you chose').nextElementSibling).toHaveTextContent('Won')

    const pairs = screen.getByRole('region', { name: 'Your picks next to our suggestions' })
    expect(within(pairs).getByText('Our suggestion won, your pick lost').nextElementSibling).toHaveTextContent('2')
    expect(within(pairs).getByText('Both lost').nextElementSibling).toHaveTextContent('0')

    // Someone with only the link can view but not delete.
    expect(screen.queryByRole('button', { name: 'Delete saved ticket' })).not.toBeInTheDocument()
    expect(screen.getByText('Only the device that saved this ticket can delete it.')).toBeInTheDocument()
    expect(screen.getByText(/Kept until/)).toBeInTheDocument()
  })

  it('says when a game has kicked off and when results were last checked (A10)', async () => {
    renderAt(`/t/${DEMO_LIVE_ID}`)
    const [a, b, c] = FIXTURES
    const played = await screen.findByRole('article', { name: `${a.home.name} v ${a.away.name}` })
    expect(within(played).getByText('2–0')).toBeInTheDocument()
    const live = screen.getByRole('article', { name: `${b.home.name} v ${b.away.name}` })
    expect(within(live).getByText('Kicked off, result soon')).toBeInTheDocument()
    const toCome = screen.getByRole('article', { name: `${c.home.name} v ${c.away.name}` })
    expect(within(toCome).queryByText('Kicked off, result soon')).not.toBeInTheDocument()

    const result = screen.getByRole('region', { name: 'Ticket result' })
    expect(result).toHaveTextContent('1 of 3 games played')
    expect(result).toHaveTextContent(/15 to 30 minutes after full time.*Results last checked at \d/)
  })

  it('only looks again for games that have kicked off and are not postponed', () => {
    const now = Date.parse('2026-11-28T17:00:00Z')
    const leg = (kickoff: string, over: Partial<SavedLeg> = {}) =>
      ({ fixture: { kickoff }, match_status: 'scheduled', score: null, ...over }) as SavedLeg
    expect(awaitingResult(leg('2026-11-28T15:00:00Z'), now)).toBe(true)
    expect(awaitingResult(leg('2026-11-28T19:00:00Z'), now)).toBe(false)
    expect(awaitingResult(leg('2026-11-28T15:00:00Z', { match_status: 'postponed' }), now)).toBe(false)
    expect(awaitingResult(leg('2026-11-28T15:00:00Z', { match_status: 'finished', score: { home: 1, away: 0 } }), now)).toBe(false)
  })

  it('lists this device’s tickets with a personal record, forgets ones that are gone, and deletes', async () => {
    rememberSaved({ id: 'goneGoneGoneGoneGone00', ticketId: 'old-ticket-0001', savedAt: '2026-08-01T10:00:00Z' })
    rememberSaved({ id: DEMO_SAVED_ID, ticketId: DEMO_TICKET_ID, savedAt: new Date().toISOString() })
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    const user = userEvent.setup()
    renderAt('/tickets')

    expect(await screen.findByRole('heading', { name: 'Your record across saved tickets' })).toBeInTheDocument()
    const own = screen.getByRole('region', { name: 'Your own picks' })
    expect(own).toHaveTextContent('1 won')
    expect(own).toHaveTextContent('1 lost')
    expect(screen.getByRole('region', { name: 'Our suggestions' })).toHaveTextContent('2 won')
    await waitFor(() => expect(loadSaved().map((e) => e.id)).toEqual([DEMO_SAVED_ID]))

    await user.click(screen.getByRole('link', { name: /2 games/ }))
    await user.click(await screen.findByRole('button', { name: 'Delete saved ticket' }))
    await waitFor(() => expect(screen.getByTestId('location')).toHaveTextContent('/tickets'))
    expect(loadSaved()).toEqual([])
    expect(await screen.findByText(/No saved tickets on this device yet/)).toBeInTheDocument()
  })
})
