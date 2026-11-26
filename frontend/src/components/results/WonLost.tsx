import type { HitCount } from '../../api/types'
import { COPY } from '../../config/copy'
import { pct } from '../../lib/format'
import { hitRate, isSmallSample, settled } from '../../lib/results'

/** Won and lost bar; both segments get equal visual weight (RG-3). */
export function WonLostBar({ count }: { count: HitCount }) {
  const n = settled(count)
  if (n === 0) return <div className="td-wl" aria-hidden="true" />
  return (
    <div className="td-wl" aria-hidden="true">
      <span className="td-wl__won" style={{ flexGrow: count.won }} />
      <span className="td-wl__lost" style={{ flexGrow: count.lost }} />
    </div>
  )
}

/** "12 won · 6 lost", with won and lost in the same size and weight. */
export function WonLostText({ count }: { count: HitCount }) {
  const c = COPY.results
  return (
    <span className="td-wl-text td-num">
      <span className="td-wl-text__won">
        {count.won} {c.won}
      </span>
      <span className="td-wl-text__lost">
        {count.lost} {c.lost}
      </span>
    </span>
  )
}

/** Compact cell for the results table. */
export function HitCell({ count }: { count: HitCount }) {
  const c = COPY.results
  const rate = hitRate(count)
  if (rate === null) return <span className="small text-body-secondary">{c.noneSettled}</span>
  return (
    <div className="td-hitcell">
      <div className="d-flex align-items-baseline gap-2 flex-wrap">
        <span className="fw-bold td-num">{pct(rate)}</span>
        {isSmallSample(count) && <span className="td-small-sample">{c.smallSample}</span>}
      </div>
      <WonLostText count={count} />
      <WonLostBar count={count} />
    </div>
  )
}
