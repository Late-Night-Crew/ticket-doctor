import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import type { WizardRequest } from '../api/types'
import { FIXTURES } from '../mocks/data'
import { TICKET_STORAGE_KEY, type TicketState } from '../state/ticketReducer'
import { renderPage } from '../test/render'
import { server } from '../test/server'
import Wizard from './Wizard'

beforeAll(() => server.listen({ onUnhandledFrame: 'error' }))
afterEach(() => {
  server.resetHandlers()
  sessionStorage.clear()
  vi.restoreAllMocks()
})
afterAll(() => server.close())

const renderWizard = (url = '/wizard') => renderPage(<Wizard />, { path: '/wizard/*', url })
const choice = (name: RegExp) => screen.getByRole('button', { name })
const games = (n: number) => screen.getByRole('button', { name: `${n} games` })

/** Record what the wizard sends to the API. */
function captureRequests(): WizardRequest[] {
  const seen: WizardRequest[] = []
  server.events.on('request:start', async ({ request }) => {
    if (request.url.endsWith('/api/wizard')) seen.push((await request.clone().json()) as WizardRequest)
  })
  return seen
}

describe('Ticket wizard (FR-8)', () => {
  afterEach(() => server.events.removeAllListeners())

  it('offers the beginner and guided paths (FR-8.1)', () => {
    renderWizard()
    expect(screen.getByRole('link', { name: /I'm new to betting/ })).toHaveAttribute('href', '/wizard/beginner')
    expect(screen.getByRole('link', { name: /I know the basics/ })).toHaveAttribute('href', '/wizard/guided')
  })

  it('beginner path: stake warning, Safe by default, two questions, explainer, then the analyser', async () => {
    const requests = captureRequests()
    const user = userEvent.setup()
    renderWizard('/wizard/beginner')

    // RG-6: the stake warning comes before anything else
    expect(screen.getByRole('heading', { name: 'Before we start' })).toBeInTheDocument()
    expect(screen.getByText(/lose all the money you staked/)).toBeInTheDocument()
    expect(screen.getByText(/never tell you how much to stake/)).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'I understand' }))

    // FR-8.4: Safe is preselected
    expect(choice(/Careful \(Safe\)/)).toHaveAttribute('aria-pressed', 'true')
    await user.click(screen.getByRole('button', { name: 'Continue' }))

    await user.click(choice(/^A mix/))

    // Beginners choose 2–4 games
    expect(screen.getByRole('heading', { name: 'How many games on your ticket?' })).toBeInTheDocument()
    expect(screen.getAllByRole('button', { name: /^\d+ games$/ }).map((b) => b.textContent)).toEqual(['2', '3', '4'])
    await user.click(games(3))

    expect(await screen.findByRole('heading', { name: 'Your suggested ticket' })).toBeInTheDocument()
    expect(screen.getByRole('note')).toHaveTextContent(/not guarantees/)
    await screen.findAllByText(/Fair odds \(our estimate\)/)
    expect(screen.getByRole('heading', { name: 'How this ticket works' })).toBeInTheDocument() // FR-8.7
    expect(screen.getAllByText(/Based on each team's last 10 matches/).length).toBeGreaterThan(0) // NFR-5

    // Q9: beginners aren't asked when; the ticket says which window it used.
    expect(requests.at(-1)).toMatchObject({ path: 'beginner', risk: 'safe', focus: 'mix', legs: 3, when: 'soonest' })
    const answers = within(screen.getByRole('list', { name: 'Your answers' }))
    expect(answers.queryByText('Today or this weekend')).not.toBeInTheDocument()
    expect(answers.getByText(/^(Today|This weekend)$/)).toBeInTheDocument()
    expect(screen.getByText('Ticket fair odds')).toBeInTheDocument()
    const legs = screen.getAllByRole('article')
    expect(legs.length).toBeGreaterThan(0)
    expect(legs.length).toBeLessThanOrEqual(3)

    // FR-8.8: open in the analyser
    await user.click(screen.getByRole('button', { name: 'Open in analyser' }))
    expect(screen.getByTestId('location')).toHaveTextContent('/ticket/analysis')
    const saved = JSON.parse(sessionStorage.getItem(TICKET_STORAGE_KEY)!) as TicketState
    expect(saved.origin).toBe('wizard')
    expect(saved.legs).toHaveLength(legs.length)
  })

  it('beginner Bold needs an extra confirmation screen (FR-8.4)', async () => {
    const user = userEvent.setup()
    renderWizard('/wizard/beginner')
    await user.click(screen.getByRole('button', { name: 'I understand' }))

    await user.click(choice(/Risky \(Bold\)/))
    expect(screen.getByRole('heading', { name: 'About Bold' })).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Switch to Safe' }))
    expect(screen.getByRole('group', { name: 'What kind of picks?' })).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /Back/ }))
    expect(choice(/Careful \(Safe\)/)).toHaveAttribute('aria-pressed', 'true')

    await user.click(choice(/Risky \(Bold\)/))
    await user.click(screen.getByRole('button', { name: 'Keep Bold' }))
    expect(screen.getByRole('group', { name: 'What kind of picks?' })).toBeInTheDocument()
  })

  it('guided path asks four questions, and Regenerate asks for a different ticket (FR-8.2, FR-8.9)', async () => {
    const requests = captureRequests()
    const user = userEvent.setup()
    renderWizard('/wizard/guided')

    expect(screen.getByText('Step 1 of 4')).toBeInTheDocument()
    await user.click(choice(/^Balanced/))
    expect(screen.getByText('Step 2 of 4')).toBeInTheDocument()
    await user.click(choice(/^Goals/))
    expect(screen.getAllByRole('button', { name: /^\d+ games$/ })).toHaveLength(9) // 2–10
    await user.click(games(5))
    await user.click(choice(/^This weekend/))

    expect(await screen.findByRole('heading', { name: 'Your suggested ticket' })).toBeInTheDocument()
    await waitFor(() => expect(requests).toHaveLength(1))
    expect(requests[0]).toEqual({ path: 'guided', risk: 'balanced', focus: 'goals', legs: 5, when: 'weekend', seed: 0 })
    expect(screen.queryByRole('heading', { name: 'How this ticket works' })).not.toBeInTheDocument()

    await user.click(await screen.findByRole('button', { name: 'Show a different ticket' }))
    await waitFor(() => expect(requests).toHaveLength(2))
    expect(requests[1].seed).toBe(1)
  })

  it('shows the reward next to the chance: fair odds are 1 ÷ combined chance', async () => {
    const user = userEvent.setup()
    renderWizard('/wizard/guided')
    await user.click(choice(/^Bold/))
    await user.click(choice(/^A mix/))
    await user.click(games(2))
    await user.click(choice(/^This weekend/))
    await screen.findAllByRole('article')
    const pctText = screen.getByText(/^\d+\.\d%$/).textContent!
    const odds = Number(screen.getByText('Ticket fair odds').nextElementSibling!.textContent)
    expect(odds).toBeCloseTo(100 / parseFloat(pctText), 0)
  })

  it('explains an empty ticket instead of loosening the rules (4.6.6)', async () => {
    server.use(
      http.post('*/api/wizard', () =>
        HttpResponse.json({ legs: [], shortfall_reason: 'Only 0 matches met the safe rules for today.' }),
      ),
    )
    const user = userEvent.setup()
    renderWizard('/wizard/guided')
    await user.click(choice(/^Safe/))
    await user.click(choice(/^Goals/))
    await user.click(games(2))
    await user.click(choice(/^Today/))

    expect(await screen.findByText('No matches met your rules.')).toBeInTheDocument()
    expect(screen.getByText(/Only 0 matches met the safe rules/)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Open in analyser' })).not.toBeInTheDocument()
  })

  it('asks before replacing an existing ticket', async () => {
    const existing: TicketState = {
      ticketId: 'old',
      origin: 'manual',
      legs: [{ fixture: FIXTURES[0], market: 'HOME_WIN', odds: null }],
      swaps: [],
      baseline: null,
    }
    sessionStorage.setItem(TICKET_STORAGE_KEY, JSON.stringify(existing))
    const confirm = vi.spyOn(window, 'confirm').mockReturnValue(false)
    const user = userEvent.setup()
    renderWizard('/wizard/guided')
    await user.click(choice(/^Balanced/))
    await user.click(choice(/^A mix/))
    await user.click(games(3))
    await user.click(choice(/^This weekend/))

    await user.click(await screen.findByRole('button', { name: 'Open in analyser' }))
    expect(confirm).toHaveBeenCalledWith(expect.stringContaining('1 legs'))
    expect(screen.getByTestId('location')).toHaveTextContent('/wizard/guided')
    const saved = JSON.parse(sessionStorage.getItem(TICKET_STORAGE_KEY)!) as TicketState
    expect(saved.ticketId).toBe('old')
  })
})

describe('result chips', () => {
  it('shows the answers', async () => {
    const user = userEvent.setup()
    renderWizard('/wizard/guided')
    await user.click(choice(/^Bold/))
    await user.click(choice(/^Match results/))
    await user.click(games(7))
    await user.click(choice(/^Today/))
    const chips = await screen.findByRole('list', { name: 'Your answers' })
    expect(within(chips).getAllByRole('listitem').map((li) => li.textContent)).toEqual([
      'Bold',
      'Match results',
      '7 games',
      'Today',
    ])
  })
})
