import type { RiskBand } from '../api/types'
import { COPY } from '../config/copy'
import { IconAlert, IconCheck, IconDot } from './Icons'

const ICON = { low: IconCheck, medium: IconDot, high: IconAlert }

export function RiskBadge({ band, limitedData = false }: { band: RiskBand; limitedData?: boolean }) {
  const Icon = ICON[band]
  return (
    <span className={`td-risk td-risk--${band}`}>
      <Icon />
      {COPY.risk[band]}
      {limitedData && <span className="fw-normal">· {COPY.risk.limitedData}</span>}
    </span>
  )
}
