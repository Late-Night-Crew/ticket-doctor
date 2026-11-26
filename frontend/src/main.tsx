import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router'
import App from './App'
import { TicketProvider } from './state/TicketContext'
import './styles/main.scss'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: 1, refetchOnWindowFocus: false },
  },
})

async function enableMocks() {
  if (import.meta.env.VITE_USE_MOCKS !== 'true') return
  const { worker } = await import('./mocks/browser')
  await worker.start({ onUnhandledFrame: 'bypass', quiet: true })
}

enableMocks()
  .catch((err) => console.warn('Mock API failed to start; continuing without it.', err))
  .then(() => {
  createRoot(document.getElementById('root')!).render(
    <StrictMode>
      <QueryClientProvider client={queryClient}>
        <BrowserRouter>
          <TicketProvider>
            <App />
          </TicketProvider>
        </BrowserRouter>
      </QueryClientProvider>
    </StrictMode>,
  )
})
