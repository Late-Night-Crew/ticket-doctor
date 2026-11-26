import type { Fixture, Suggestion } from '../../api/types'
import { COPY } from '../../config/copy'
import { market } from '../../config/markets'
import { pct } from '../../lib/format'
import { RiskBadge } from '../RiskBadge'

interface Props {
  fixture: Fixture
  suggestion: Suggestion
  disabled?: boolean
  onAccept: (s: Suggestion) => void
}

/** One suggested swap with its reason and a one-tap accept (FR-5.2, FR-5.3). */
export function SwapOption({ fixture, suggestion, disabled, onAccept }: Props) {
  const pick = market(suggestion.market).plain(fixture.home.name, fixture.away.name)
  return (
    <div className="td-swap">
      <div className="min-w-0">
        <div className="d-flex flex-wrap align-items-center gap-2">
          <span className="fw-semibold">{pick}</span>
          <span className="td-num fw-bold">{pct(suggestion.estimate)}</span>
          <RiskBadge band={suggestion.risk_band} />
        </div>
        <div className="small text-body-secondary mt-1">{suggestion.reason}</div>
      </div>
      <button
        type="button"
        className="btn btn-sm btn-outline-primary flex-none"
        disabled={disabled}
        onClick={() => onAccept(suggestion)}
        aria-label={COPY.analysis.swapTo(pick)}
      >
        {COPY.analysis.swap}
      </button>
    </div>
  )
}
