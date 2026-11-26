import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render } from '@testing-library/react'
import type { ReactElement } from 'react'
import { MemoryRouter, Route, Routes, useLocation } from 'react-router'
import { TicketProvider } from '../state/TicketContext'

function LocationProbe() {
  const loc = useLocation()
  return <div data-testid="location">{loc.pathname}</div>
}

/** Render a page with the app's providers at a given URL. */
export function renderPage(ui: ReactElement, opts: { path?: string; url?: string } = {}) {
  const path = opts.path ?? '/'
  const url = opts.url ?? path
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[url]}>
        <TicketProvider>
          <Routes>
            <Route path={path} element={ui} />
            <Route path="*" element={null} />
          </Routes>
          <LocationProbe />
        </TicketProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}
