import type { Fixture } from '../api/types'
import { emptyTicket, ticketReducer, type DraftLeg } from './ticketReducer'

const fixture = (id: string): Fixture => ({
  id,
  league_id: 'epl',
  kickoff: '2026-11-28T15:00:00Z',
  home: { id: `${id}-h`, name: 'Home' },
  away: { id: `${id}-a`, name: 'Away' },
})

const leg = (id: string): DraftLeg => ({ fixture: fixture(id), market: 'HOME_WIN', odds: null })

describe('ticketReducer', () => {
  it('keeps one leg per match (FR-3.4)', () => {
    let s = ticketReducer(emptyTicket(), { type: 'add', leg: leg('m1') })
    s = ticketReducer(s, { type: 'add', leg: { ...leg('m1'), market: 'BTTS_YES' } })
    expect(s.legs).toHaveLength(1)
    expect(s.legs[0].market).toBe('BTTS_YES')
  })

  it('caps the ticket at 15 legs (FR-3.4)', () => {
    let s = emptyTicket()
    for (let i = 0; i < 20; i++) s = ticketReducer(s, { type: 'add', leg: leg(`m${i}`) })
    expect(s.legs).toHaveLength(15)
  })

  it('clears odds when the market changes', () => {
    let s = ticketReducer(emptyTicket(), { type: 'add', leg: { ...leg('m1'), odds: 2.1 } })
    s = ticketReducer(s, { type: 'setMarket', matchId: 'm1', market: 'UNDER_3_5' })
    expect(s.legs[0].odds).toBeNull()
  })

  it('gets a new ticket id on clear', () => {
    const s = emptyTicket()
    expect(ticketReducer(s, { type: 'clear' }).ticketId).not.toBe(s.ticketId)
  })
})

describe('swaps (FR-5.3, FR-6.2)', () => {
  const start = () => ticketReducer(emptyTicket(), { type: 'add', leg: { ...leg('m1'), market: 'AWAY_WIN', odds: 3.6 } })
  const accept = (s: ReturnType<typeof start>, market: 'UNDER_3_5' | 'AWAY_TO_SCORE' = 'UNDER_3_5', baseline = 0.25) =>
    ticketReducer(s, { type: 'acceptSwap', matchId: 'm1', suggestionId: 'x', market, baseline })

  it('swaps the market, clears odds and records the baseline', () => {
    const s = accept(start())
    expect(s.legs[0]).toMatchObject({ market: 'UNDER_3_5', odds: null })
    expect(s.baseline).toBe(0.25)
    expect(s.swaps).toEqual([{ matchId: 'm1', suggestionId: 'x', from: 'AWAY_WIN', fromOdds: 3.6, to: 'UNDER_3_5' }])
  })

  it('keeps the first baseline across later swaps', () => {
    const s = accept(accept(start()), 'AWAY_TO_SCORE', 0.8)
    expect(s.baseline).toBe(0.25)
    expect(s.swaps[0]).toMatchObject({ from: 'AWAY_WIN', fromOdds: 3.6, to: 'AWAY_TO_SCORE' })
  })

  it('undo restores the original pick and odds, and drops the baseline when no swaps remain', () => {
    const s = ticketReducer(accept(start()), { type: 'undoSwap', matchId: 'm1' })
    expect(s.legs[0]).toMatchObject({ market: 'AWAY_WIN', odds: 3.6 })
    expect(s.swaps).toEqual([])
    expect(s.baseline).toBeNull()
  })

  it('a manual edit resets swaps and baseline', () => {
    const s = ticketReducer(accept(start()), { type: 'add', leg: leg('m2') })
    expect(s.swaps).toEqual([])
    expect(s.baseline).toBeNull()
  })

  it('setting odds keeps swaps', () => {
    const s = ticketReducer(accept(start()), { type: 'setOdds', matchId: 'm1', odds: 1.4 })
    expect(s.swaps).toHaveLength(1)
  })
})
