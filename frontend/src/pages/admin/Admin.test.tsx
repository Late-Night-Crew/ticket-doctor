import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, useLocation } from 'react-router'
import App from '../../App'
import { MOCK_ADMIN, resetAdminMock } from '../../mocks/admin'
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
})
