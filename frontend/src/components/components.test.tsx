import { render, screen } from '@testing-library/react'
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
