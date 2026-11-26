import type { RiskBand } from '../api/types'
import { COPY } from '../config/copy'
import { oneIn, pct } from '../lib/format'
import { bandFor } from '../lib/bands'

interface Props {
  probability: number
  label?: string
  /** Show the earlier value, e.g. before swaps (FR-6.2). */
  before?: number | null
}

/** A probability shown as a percentage and in plain words (RG-2). */
export function ProbabilityMeter({ probability, label = COPY.probability.combinedLabel, before = null }: Props) {
  const band: RiskBand = bandFor(probability)
  const width = Math.max(2, Math.round(probability * 100))
  return (
    <div className="td-prob" aria-label={`${label}: ${pct(probability, 1)}, ${COPY.probability.oneIn(oneIn(probability))}`}>
      <div className="td-eyebrow mb-1">{label}</div>
      <div className="d-flex align-items-baseline flex-wrap gap-2 mb-2">
        <span className="td-prob__value td-num">{pct(probability, 1)}</span>
        <span className="td-prob__words">{COPY.probability.oneIn(oneIn(probability))}</span>
        {before !== null && (
          <span className="small text-body-secondary td-num">
            was {pct(before, 1)} ({COPY.probability.oneIn(oneIn(before))})
          </span>
        )}
      </div>
      <div className="td-prob__bar" role="presentation">
        <div className={`td-prob__fill td-prob__fill--${band}`} style={{ width: `${width}%` }} />
      </div>
    </div>
  )
}
