import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, useLocation } from 'react-router'
import App from '../../App'
import { MOCK_ADMIN, MOCK_SETUP_CODE, resetAdminMock, setupOpen } from '../../mocks/admin'
import { TicketProvider } from '../../state/TicketContext'
import { server } from '../../test/server'

beforeAll(() => server.listen({ onUnhandledFrame: 'error' }))
afterEach(() => {
  server.resetHandlers()
  resetAdminMock()
})
afterAll(() => server.close())

function Where() {
  const loc = useLocation()
  return <div data-testid="location">{loc.pathname + loc.search}</div>
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

async function signIn(user: ReturnType<typeof userEvent.setup>, password = MOCK_ADMIN.password) {
  await user.type(await screen.findByLabelText('Username'), MOCK_ADMIN.username)
  await user.type(screen.getByLabelText('Password'), password)
  await user.click(screen.getByRole('button', { name: 'Sign in' }))
}

describe('Admin area (B6)', () => {
  it('sends a signed-out visitor to sign in, then back to the page they wanted', async () => {
    const user = userEvent.setup()
    renderAt('/admin/thresholds')
    await waitFor(() => expect(screen.getByTestId('location')).toHaveTextContent('/admin/login?next=%2Fadmin%2Fthresholds'))

    await signIn(user, 'wrong password!!')
    expect(await screen.findByRole('alert')).toHaveTextContent('Wrong username or password.')
    expect(screen.getByLabelText('Password')).toHaveValue('') // cleared after a failure

    await user.type(screen.getByLabelText('Password'), MOCK_ADMIN.password)
    await user.click(screen.getByRole('button', { name: 'Sign in' }))
    expect(await screen.findByRole('heading', { name: 'Rule thresholds' })).toBeInTheDocument()
    expect(screen.getByTestId('location')).toHaveTextContent(/^\/admin\/thresholds$/)
    // Shown as percentages, with the SRS default beside the value in use.
    const cap = screen.getByText('Estimate cap').closest('tr')!
    expect(within(cap).getAllByText('95%')).toHaveLength(2)
  })

  it('never follows a sign-in redirect off the admin area', async () => {
    const user = userEvent.setup()
    renderAt('/admin/login?next=https://evil.example')
    await signIn(user)
    await waitFor(() => expect(screen.getByTestId('location')).toHaveTextContent(/^\/admin$/))
  })

  it('shows the dashboard with games awaiting a result and the import history', async () => {
    const user = userEvent.setup()
    renderAt('/admin')
    await signIn(user)
    expect(await screen.findByRole('heading', { name: 'Dashboard' })).toBeInTheDocument()
    expect(screen.getByRole('status')).toHaveTextContent('2 games have kicked off with no result yet.')
    expect(screen.getByText('laliga: unmatched Real Oviedo')).toBeInTheDocument()
    expect(screen.getByRole('navigation', { name: 'Admin' })).toBeInTheDocument()
  })

  it('enters a result, which moves the game to recent results', async () => {
    const user = userEvent.setup()
    renderAt('/admin/results')
    await signIn(user)

    const row = await screen.findByRole('form', { name: 'Arsenal v Chelsea' })
    const save = within(row).getByRole('button', { name: 'Save' })
    expect(save).toBeDisabled()
    await user.type(within(row).getByLabelText('Arsenal goals'), '2')
    await user.type(within(row).getByLabelText('Chelsea goals'), '99') // too many: maxLength 2, over 20
    expect(save).toBeDisabled()
    await user.clear(within(row).getByLabelText('Chelsea goals'))
    await user.type(within(row).getByLabelText('Chelsea goals'), '1')
    await user.click(save)

    await waitFor(() => expect(screen.queryByRole('form', { name: 'Arsenal v Chelsea' })).not.toBeInTheDocument())
    await user.click(screen.getByRole('button', { name: 'Recent results' }))
    const done = await screen.findByRole('form', { name: 'Arsenal v Chelsea' })
    expect(within(done).getByLabelText('Arsenal goals')).toHaveValue('2')
    expect(within(done).getByText('entered by hand')).toBeInTheDocument()
    expect(within(done).getByRole('button', { name: 'Correct' })).toBeDisabled() // nothing changed yet
  })

  it('signs out', async () => {
    const user = userEvent.setup()
    renderAt('/admin')
    await signIn(user)
    await user.click(await screen.findByRole('button', { name: 'Sign out' }))
    await waitFor(() => expect(screen.getByTestId('location')).toHaveTextContent(/^\/admin\/login$/))
  })

  it('sets up the first admin with the setup code, then signs them in', async () => {
    setupOpen(true)
    const user = userEvent.setup()
    renderAt('/admin/login')
    await user.click(await screen.findByRole('link', { name: 'Set up the first admin' }))

    await user.type(await screen.findByLabelText('Setup code'), 'wrong code')
    await user.type(screen.getByLabelText('Username'), 'Joseph')
    await user.type(screen.getByLabelText('Password'), 'short')
    expect(screen.getByText('Use at least 12 characters.')).toBeInTheDocument()
    const submit = screen.getByRole('button', { name: 'Create admin and sign in' })
    expect(submit).toBeDisabled()

    await user.type(screen.getByLabelText('Password'), ' but now long enough')
    await user.type(screen.getByLabelText('Same password again'), 'short but now long enough')
    await user.click(submit)
    expect(await screen.findByRole('alert')).toHaveTextContent("That setup code isn't right.")

    await user.clear(screen.getByLabelText('Setup code'))
    await user.type(screen.getByLabelText('Setup code'), MOCK_SETUP_CODE)
    await user.click(submit)
    expect(await screen.findByRole('heading', { name: 'Dashboard' })).toBeInTheDocument()
    expect(screen.getByText('joseph')).toBeInTheDocument() // usernames are lowercased
  })

  it('says setup is closed once an admin exists', async () => {
    renderAt('/admin/setup')
    expect(await screen.findByText(/Setup is closed/)).toBeInTheDocument()
    expect(screen.queryByLabelText('Setup code')).not.toBeInTheDocument()
  })

  it('adds an admin, disables another, and changes your own password', async () => {
    const user = userEvent.setup()
    renderAt('/admin/admins')
    await signIn(user)

    const list = await screen.findByRole('region', { name: 'Accounts' })
    expect(await within(list).findByText('grace', { selector: '.fw-semibold' })).toBeInTheDocument()
    // You can't disable yourself, so there's no button on your own row.
    expect(within(list).queryByRole('button', { name: /Disable admin/ })).not.toBeInTheDocument()
    await user.click(within(list).getByRole('button', { name: 'Disable grace' }))
    expect(await within(list).findByText('disabled')).toBeInTheDocument()
    expect(within(list).getByRole('button', { name: 'Enable grace' })).toBeInTheDocument()

    const add = screen.getByRole('region', { name: 'Add an admin' })
    await user.type(within(add).getByLabelText('Username'), 'tamanda')
    await user.type(within(add).getByLabelText('Password'), 'tamanda first password')
    await user.type(within(add).getByLabelText('Same password again'), 'tamanda first password')
    await user.click(within(add).getByRole('button', { name: 'Add admin' }))
    expect(await within(add).findByRole('status')).toHaveTextContent('Added tamanda.')
    expect(await within(list).findByText('tamanda', { selector: '.fw-semibold' })).toBeInTheDocument()

    const change = screen.getByRole('region', { name: 'Change your password' })
    await user.type(within(change).getByLabelText('Current password'), 'not my password')
    await user.type(within(change).getByLabelText('New password'), 'a brand new password')
    await user.type(within(change).getByLabelText('Same password again'), 'a brand new password')
    await user.click(within(change).getByRole('button', { name: 'Change password' }))
    expect(await within(change).findByRole('alert')).toHaveTextContent("Your current password isn't right.")
    await user.clear(within(change).getByLabelText('Current password'))
    await user.type(within(change).getByLabelText('Current password'), MOCK_ADMIN.password)
    await user.click(within(change).getByRole('button', { name: 'Change password' }))
    expect(await within(change).findByRole('status')).toHaveTextContent('Password changed.')
  })

  it('shows calibration: what we said vs what happened, by band and market', async () => {
    const user = userEvent.setup()
    renderAt('/admin/calibration')
    await signIn(user)
    expect(await screen.findByRole('img', { name: /Reliability chart/ })).toBeInTheDocument()

    // The table holds every value the chart shows (tooltips never gate).
    const bins = screen.getByRole('region', { name: 'By estimate' })
    const top = within(bins).getByText('90%–100%').closest('tr')!
    expect(top).toHaveTextContent('81.8%')
    expect(top).toHaveTextContent('10.2 pts, too confident')

    const markets = screen.getByRole('region', { name: 'By market: do the team stats help?' })
    expect(within(markets).getByText('BTTS yes').closest('tr')).toHaveTextContent('no better than guessing')

    // Real tickets: empty until games finish.
    await user.click(screen.getByRole('button', { name: 'Real tickets' }))
    expect(await screen.findByText(/No analysed picks have finished games yet/)).toBeInTheDocument()
  })
})
