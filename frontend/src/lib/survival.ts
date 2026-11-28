/**
 * "How far will it get?": a ticket's chance of still being alive after each
 * game, in kick-off order. Every leg must win, so the chance of reaching the
 * last k games is the product of the estimates of all games before them. Legs
 * are treated as independent, as in the combined chance (FR-6).
 */
export interface SurvivalLeg {
  id: string
  kickoff: string
  estimate: number
}

export interface SurvivalStep<T extends SurvivalLeg = SurvivalLeg> {
  leg: T
  /** 1-based position in kick-off order. */
  position: number
  /** Chance this leg is reached with every earlier leg won. */
  reached: number
  /** Chance the ticket is still alive after this leg. */
  alive: number
  /** Chance this is the leg that ends the ticket. */
  endsHere: number
  /** Kicks off at the same time as the previous leg (order between them is arbitrary). */
  sameTimeAsPrevious: boolean
}

/** Legs in kick-off order; ties keep the ticket's order. */
export function byKickoff<T extends SurvivalLeg>(legs: T[]): T[] {
  return legs
    .map((leg, i) => ({ leg, i }))
    .sort((a, b) => Date.parse(a.leg.kickoff) - Date.parse(b.leg.kickoff) || a.i - b.i)
    .map((x) => x.leg)
}

export function survival<T extends SurvivalLeg>(legs: T[]): SurvivalStep<T>[] {
  let alive = 1
  return byKickoff(legs).map((leg, i, ordered) => {
    const reached = alive
    alive *= leg.estimate
    return {
      leg,
      position: i + 1,
      reached,
      alive,
      endsHere: reached * (1 - leg.estimate),
      sameTimeAsPrevious: i > 0 && ordered[i - 1].kickoff === leg.kickoff,
    }
  })
}

/** Chance of reaching the last `remaining` games with every earlier leg won. */
export function chanceToReachLast(steps: SurvivalStep[], remaining: number): number {
  const before = steps.length - remaining
  return before <= 0 ? 1 : steps[before - 1].alive
}

/** Expected number of games won before the first loss (or all of them). */
export function expectedGamesSurvived(steps: SurvivalStep[]): number {
  return steps.reduce((sum, s) => sum + s.alive, 0)
}

/** The leg most likely to end the ticket. */
export function likeliestEnd<T extends SurvivalLeg>(steps: SurvivalStep<T>[]): SurvivalStep<T> | undefined {
  return steps.reduce<SurvivalStep<T> | undefined>((best, s) => (!best || s.endsHere > best.endsHere ? s : best), undefined)
}

/**
 * For a cash-out, a risky leg hurts least when it kicks off last: every game
 * before it then has a chance to count. True when the riskiest leg is in the
 * first half of the ticket and a safer leg kicks off after it.
 */
export function riskiestIsEarly(steps: SurvivalStep[]): boolean {
  if (steps.length < 3) return false
  const riskiest = steps.reduce((w, s) => (s.leg.estimate < w.leg.estimate ? s : w))
  return riskiest.position <= Math.floor(steps.length / 2) && steps.some((s) => s.position > riskiest.position && s.leg.estimate > riskiest.leg.estimate)
}

/** How many games the ticket actually got through, in kick-off order (saved tickets). */
export function gamesSurvived(outcomes: ('won' | 'lost' | 'pending')[]): { survived: number; ended: boolean } {
  let survived = 0
  for (const o of outcomes) {
    if (o === 'lost') return { survived, ended: true }
    if (o === 'pending') return { survived, ended: false }
    survived++
  }
  return { survived, ended: false }
}
