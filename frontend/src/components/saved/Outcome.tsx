import type { PickOutcome } from '../../api/types'
import { COPY } from '../../config/copy'
import { IconCheck, IconDot } from '../Icons'

/** Won, lost or not played yet, in the same size and weight either way (RG-3). */
export function OutcomeBadge({ outcome, label }: { outcome: PickOutcome; label?: string }) {
  return (
    <span className={`td-outcome td-outcome--${outcome}`}>
      {outcome === 'won' ? <IconCheck width={14} height={14} /> : outcome === 'lost' ? <span aria-hidden="true">✕</span> : <IconDot width={10} height={10} />}
      {label ?? COPY.saved.outcome[outcome]}
    </span>
  )
}
