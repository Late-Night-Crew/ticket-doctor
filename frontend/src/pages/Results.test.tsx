import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { RESULTS } from '../mocks/handlers'
import { pct } from '../lib/format'
import { hitRate, totals } from '../lib/results'
import { renderPage } from '../test/render'
import { server } from '../test/server'
import Results from './Results'

beforeAll(() => server.listen({ onUnhandledFrame: 'error' }))
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

const renderResults = (url = '/results') => renderPage(<Results />, { path: '/results', url })

describe('Results page (FR-7.3, RG-3)', () => {
  it('compares original picks with suggested swaps, showing losses as clearly as wins', async () => {
    renderResults()
    const sum = totals(RESULTS.rows)

    const original = await screen.findByRole('region', { name: 'Original picks' })
    expect(within(original).getByText(pct(hitRate(sum.original)!))).toBeInTheDocument()
    expect(within(original).getByText(`${sum.original.won} won`)).toBeInTheDocument()
    expect(within(original).getByText(`${sum.original.lost} lost`)).toBeInTheDocument()

    const suggested = screen.getByRole('region', { name: 'Suggested swaps' })
    expect(within(suggested).getByText(`${sum.suggested.lost} lost`)).toBeInTheDocument()

    // Won and lost use the same element type and size: neither is hidden or shrunk.
    const won = within(original).getByText(`${sum.original.won} won`)
    const lost = within(original).getByText(`${sum.original.lost} lost`)
    expect(won.tagName).toBe(lost.tagName)
    expect(won.parentElement).toBe(lost.parentElement)
  })

  it('says the two groups are not a like-for-like comparison', async () => {
    renderResults()
    expect(await screen.findByText(/not a like-for-like comparison/)).toBeInTheDocument()
  })

  it('lists a row per market and league, flagging small samples and unsettled cells', async () => {
    renderResults()
    const table = await screen.findByRole('table', { name: 'Results by market and league' })
    expect(within(table).getAllByRole('row')).toHaveLength(RESULTS.rows.length + 1)
    expect(within(table).getAllByText('small sample').length).toBeGreaterThan(0)
    // La Liga draws: no suggested swaps settled
    expect(within(table).getByText('None settled yet')).toBeInTheDocument()
  })

  it('filters by league and market, keeping filters in the URL', async () => {
    const user = userEvent.setup()
    renderResults()
    await user.click(await screen.findByRole('button', { name: 'Bundesliga' }))
    await waitFor(() => {
      const rows = within(screen.getByRole('table')).getAllByRole('row').slice(1)
      expect(rows).toHaveLength(RESULTS.rows.filter((r) => r.league_id === 'bundesliga').length)
    })

    await user.selectOptions(screen.getByLabelText('Market'), 'OVER_2_5')
    await waitFor(() => expect(within(screen.getByRole('table')).getAllByRole('row')).toHaveLength(2))
  })

  it('reads filters from the URL', async () => {
    renderResults('/results?league=laliga&market=DRAW')
    expect(await screen.findByRole('button', { name: 'La Liga', pressed: true })).toBeInTheDocument()
    expect(screen.getByLabelText('Market')).toHaveValue('DRAW')
    await waitFor(() => expect(within(screen.getByRole('table')).getAllByRole('row')).toHaveLength(2))
  })

  it('shows an empty state', async () => {
    server.use(http.get('*/api/results', () => HttpResponse.json({ updated_at: new Date().toISOString(), rows: [] })))
    renderResults()
    expect(await screen.findByText('No settled results for these filters yet.')).toBeInTheDocument()
  })

  it('shows an error with retry', async () => {
    server.use(http.get('*/api/results', () => HttpResponse.json({ error: 'down' }, { status: 500 })))
    renderResults()
    expect(await screen.findByText('Could not load the track record.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Try again' })).toBeInTheDocument()
  })
})
