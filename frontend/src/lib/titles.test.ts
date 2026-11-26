import { titleFor } from './titles'

describe('titleFor', () => {
  it('names each page', () => {
    expect(titleFor('/')).toMatch(/^Ticket Doctor: /)
    expect(titleFor('/ticket')).toBe('Check a ticket · Ticket Doctor')
    expect(titleFor('/ticket/analysis')).toBe('Ticket analysis · Ticket Doctor')
    expect(titleFor('/wizard/beginner')).toBe('Build a ticket for me · Ticket Doctor')
    expect(titleFor('/results')).toBe('Track record · Ticket Doctor')
    expect(titleFor('/nope')).toBe('Page not found · Ticket Doctor')
  })
})
