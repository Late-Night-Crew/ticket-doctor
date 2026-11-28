import { byKickoff, chanceToReachLast, expectedGamesSurvived, gamesSurvived, likeliestEnd, riskiestIsEarly, survival } from './survival'

const leg = (id: string, hour: number, estimate: number) => ({ id, kickoff: `2026-11-28T${String(hour).padStart(2, '0')}:00:00Z`, estimate })

describe('survival ("How far will it get?")', () => {
  // Out of order on purpose: kick-off order is c (12:00), a (15:00), b (17:30 → 17:00).
  const legs = [leg('a', 15, 0.8), leg('b', 17, 0.5), leg('c', 12, 0.9)]
  const steps = survival(legs)

  it('orders by kick-off and multiplies as it goes', () => {
    expect(steps.map((s) => s.leg.id)).toEqual(['c', 'a', 'b'])
    expect(steps.map((s) => s.alive)).toEqual([0.9, 0.9 * 0.8, 0.9 * 0.8 * 0.5])
    expect(steps.map((s) => s.reached)).toEqual([1, 0.9, 0.9 * 0.8])
  })

  it('chance of reaching the last k games', () => {
    expect(chanceToReachLast(steps, 1)).toBeCloseTo(0.72) // c and a won
    expect(chanceToReachLast(steps, 2)).toBeCloseTo(0.9)
    expect(chanceToReachLast(steps, 3)).toBe(1)
  })

  it('where it most likely ends, and how far it gets on average', () => {
    // Ends at c: 10%, at a: 18%, at b: 36%.
    expect(steps.map((s) => s.endsHere)).toEqual([expect.closeTo(0.1), expect.closeTo(0.18), expect.closeTo(0.36)])
    expect(likeliestEnd(steps)?.leg.id).toBe('b')
    expect(expectedGamesSurvived(steps)).toBeCloseTo(0.9 + 0.72 + 0.36)
    // The probabilities of ending somewhere plus surviving everything add to 1.
    expect(steps.reduce((s, x) => s + x.endsHere, 0) + steps[2].alive).toBeCloseTo(1)
  })

  it('flags a riskiest leg that kicks off early', () => {
    expect(riskiestIsEarly(steps)).toBe(false) // the 50% leg is last: best place for it
    expect(riskiestIsEarly(survival([leg('x', 12, 0.4), leg('y', 15, 0.8), leg('z', 18, 0.9)]))).toBe(true)
  })

  it('keeps ticket order for games at the same time, and marks them', () => {
    const tied = survival([leg('p', 15, 0.7), leg('q', 15, 0.6)])
    expect(byKickoff(tied.map((s) => s.leg)).map((l) => l.id)).toEqual(['p', 'q'])
    expect(tied.map((s) => s.sameTimeAsPrevious)).toEqual([false, true])
  })

  it('counts how far a played ticket actually got', () => {
    expect(gamesSurvived(['won', 'won', 'lost', 'won'])).toEqual({ survived: 2, ended: true })
    expect(gamesSurvived(['won', 'pending'])).toEqual({ survived: 1, ended: false })
  })
})
