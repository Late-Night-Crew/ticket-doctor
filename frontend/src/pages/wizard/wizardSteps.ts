import type { WizardFocus, WizardPath, WizardRisk } from '../../api/types'
import { RULES } from '../../config/rules'

export type When = 'today' | 'weekend' | 'soonest'
export type StepId = 'stake' | 'risk' | 'bold' | 'focus' | 'games' | 'when' | 'result'

export interface Answers {
  stakeOk: boolean
  risk: WizardRisk | null
  boldOk: boolean
  focus: WizardFocus | null
  /** Exact number of games, 2–10 (2–4 for beginners). */
  legs: number | null
  when: When | null
}

/**
 * Beginner path: stake warning (RG-6), risk per game (default Safe; Bold needs
 * an extra confirmation, FR-8.4), focus, number of games (2–4).
 * Guided path: risk per game, focus, number of games (2–10), when (FR-8.2).
 */
export function stepsFor(path: WizardPath, risk: WizardRisk | null): StepId[] {
  if (path === 'beginner') return ['stake', 'risk', ...(risk === 'bold' ? (['bold'] as const) : []), 'focus', 'games', 'result']
  return ['risk', 'focus', 'games', 'when', 'result']
}

/**
 * Beginners are not asked when: they get today's games if there are enough,
 * otherwise this weekend's (open question 9, decided 2026-11-27).
 */
export const BEGINNER_WHEN: When = 'soonest'

export const initial = (path: WizardPath): Answers =>
  path === 'beginner'
    ? { stakeOk: false, risk: 'safe', boldOk: false, focus: null, legs: null, when: BEGINNER_WHEN }
    : { stakeOk: true, risk: null, boldOk: true, focus: null, legs: null, when: null }

export function answered(step: StepId, a: Answers): boolean {
  switch (step) {
    case 'stake':
      return a.stakeOk
    case 'risk':
      return a.risk !== null
    case 'bold':
      return a.boldOk
    case 'focus':
      return a.focus !== null
    case 'games':
      return a.legs !== null
    case 'when':
      return a.when !== null
    case 'result':
      return true
  }
}

/** Range of games a path may choose. */
export function legRange(path: WizardPath): { min: number; max: number } {
  return {
    min: RULES.wizard.minLegs,
    max: path === 'beginner' ? RULES.wizard.beginnerMaxLegs : RULES.wizard.maxLegs,
  }
}
