import { Footer } from './Footer'
import { render, screen, within } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { TicketProvider } from '../state/TicketContext'
import { AppLayout } from './AppLayout'
import userEvent from '@testing-library/user-event'
import { Disclaimer } from './Disclaimer'
import { GlossaryTerm } from './GlossaryTerm'
import { ProbabilityMeter } from './ProbabilityMeter'
import { RiskBadge } from './RiskBadge'

describe('RiskBadge', () => {
  it('names the band in text, not only colour', () => {
    render(<RiskBadge band="high" limitedData />)
    expect(screen.getByText(/High risk/)).toBeInTheDocument()
    expect(screen.getByText(/limited data/)).toBeInTheDocument()
  })
})

describe('ProbabilityMeter', () => {
  it('shows a percentage and "about 1 in N" (RG-2)', () => {
    render(<ProbabilityMeter probability={0.7 ** 5} />)
    expect(screen.getByText('16.8%')).toBeInTheDocument()
    expect(screen.getByText('about 1 in 6')).toBeInTheDocument()
  })
})

describe('Disclaimer', () => {
  it('says estimates are not guarantees and stakes can be lost (RG-1)', () => {
    render(<Disclaimer />)
    expect(screen.getByRole('note')).toHaveTextContent(/not guarantees/)
    expect(screen.getByRole('note')).toHaveTextContent(/risk of losing/)
  })
})

describe('GlossaryTerm', () => {
  it('opens a short explanation on tap (FR-8.10)', async () => {
    render(<GlossaryTerm id="btts">BTTS</GlossaryTerm>)
    await userEvent.click(screen.getByRole('button', { name: 'BTTS' }))
    expect(await screen.findByText(/each team scores at least one goal/)).toBeInTheDocument()
  })
})

describe('Footer (RG-4)', () => {
  it('links international help and the Malawi regulator, opening in a new tab', () => {
    render(<Footer />)
    for (const name of ['BeGambleAware', 'Gamblers Anonymous', 'GamCare', 'MAGLA']) {
      expect(screen.getByRole('link', { name })).toHaveAttribute('rel', 'noopener noreferrer')
    }
    expect(screen.getByRole('link', { name: 'MAGLA' })).toHaveAttribute('href', 'https://magla.org.mw')
    expect(screen.getByText(/In Malawi, the gambling regulator/)).toBeInTheDocument()
  })
})

describe('Bottom navigation (phones)', () => {
  it('shows short labels but announces the full ones, each short label inside its full name', () => {
    render(
      <MemoryRouter>
        <TicketProvider>
          <AppLayout />
        </TicketProvider>
      </MemoryRouter>,
    )
    const nav = screen.getAllByRole('navigation', { name: 'Main' }).find((n) => n.classList.contains('td-bottomnav'))!
    const links = within(nav).getAllByRole('link')
    expect(links.map((l) => l.textContent)).toEqual(['Home', 'Check', 'Build', 'Record', 'Tickets'])
    expect(links.map((l) => l.getAttribute('aria-label'))).toEqual(['Home', 'Check a ticket', 'Build for me', 'Track record', 'My tickets'])
    for (const l of links) expect(l.getAttribute('aria-label')!.toLowerCase()).toContain(l.textContent!.toLowerCase())
  })
})
