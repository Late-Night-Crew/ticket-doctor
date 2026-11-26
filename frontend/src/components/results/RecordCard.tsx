import type { HitCount } from '../../api/types'
import { COPY } from '../../config/copy'
import { pct } from '../../lib/format'
import { hitRate, isSmallSample } from '../../lib/results'
import { WonLostBar, WonLostText } from './WonLost'

export function RecordCard({ title, help, count }: { title: string; help: string; count: HitCount }) {
  const c = COPY.results
  const rate = hitRate(count)
  return (
    <section className="td-surface p-3 h-100 td-record" aria-label={title}>
      <h2 className="h6 mb-0">{title}</h2>
      <p className="small text-body-secondary mb-2">{help}</p>
      {rate === null ? (
        <p className="text-body-secondary mb-0">{c.noneSettled}</p>
      ) : (
        <>
          <div className="d-flex align-items-baseline gap-2 flex-wrap mb-1">
            <span className="td-record__rate td-num">{pct(rate)}</span>
            <span className="text-body-secondary small">{c.hitRate}</span>
            {isSmallSample(count) && <span className="td-small-sample">{c.smallSample}</span>}
          </div>
          <div className="td-record__counts mb-2">
            <WonLostText count={count} />
          </div>
          <WonLostBar count={count} />
        </>
      )}
    </section>
  )
}
