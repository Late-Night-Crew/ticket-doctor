/**
 * Rule thresholds from SRS section 4 (NFR-6).
 * The Go API owns the real copy; the frontend uses these only for the mock
 * API and for wording. Keep the two in step.
 */
export const RULES = {
  formWindow: 10,
  lowConfidenceBelow: 5,
  band: {
    lowFrom: 0.65,
    mediumFrom: 0.45,
  },
  swap: {
    minGain: 0.1,
    maxSuggestions: 2,
  },
  valueMargin: 0.05,
  maxLegs: 15,
  wizard: {
    boldFloor: 0.35,
    minLegs: 2,
    maxLegs: 10,
    /** Beginners choose 2–4 games, keeping first tickets short. */
    beginnerMaxLegs: 4,
  },
  results: {
    /** Below this many settled legs, a hit rate is flagged as a small sample. */
    smallSample: 20,
  },
} as const
