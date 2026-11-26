import type { AnalyseResponse, RiskBand } from '../../api/types'
import { COPY } from '../../config/copy'
import { market } from '../../config/markets'
import { decimalOdds, pct } from '../../lib/format'
import { ProbabilityMeter } from '../ProbabilityMeter'

interface Props {
  data: AnalyseResponse
  /** Combined probability before accepted swaps, if any (FR-6.2). */
  baseline: number | null
  onJumpToLeg: (matchId: string) => void
}

const BANDS: RiskBand[] = ['high', 'medium', 'low']

/** Combined probability, risk mix, weakest leg and combined odds (FR-6). */
export function TicketSummary({ data, baseline, onJumpToLeg }: Props) {
  const c = COPY.analysis
  const { combined, legs } = data
  const weakest = legs.find((l) => l.id === data.weakest_leg_id)
  const counts = BANDS.map((b) => [b, legs.filter((l) => l.risk_band === b).length] as const)

  return (
    <section className="td-surface p-3 td-summary" aria-labelledby="summary-title">
      <h2 id="summary-title" className="visually-hidden">
        {c.summaryTitle}
      </h2>

      <ProbabilityMeter probability={combined.probability} before={baseline} />
      <p className="small text-body-secondary mt-2 mb-3">
        {c.everyLeg} {c.legs(legs.length)}.
      </p>

      <ul className="td-summary__bands list-unstyled" aria-label="Legs by risk">
        {counts.map(([band, n]) => (
          <li key={band} className={`td-summary__band td-summary__band--${band}`}>
            <span className="td-num fw-bold">{n}</span> {c.bandCount[band]}
          </li>
        ))}
      </ul>

      {weakest && legs.length > 1 && (
        <div className="td-summary__row">
          <div className="min-w-0">
            <div className="td-eyebrow">{c.weakest}</div>
            <div className="fw-semibold">
              {market(weakest.market).plain(weakest.fixture.home.name, weakest.fixture.away.name)}{' '}
              <span className="td-num">({pct(weakest.estimate)})</span>
            </div>
          </div>
          <button type="button" className="btn btn-sm btn-link flex-none" onClick={() => onJumpToLeg(weakest.match_id)}>
            {c.jumpTo}
          </button>
        </div>
      )}

      <div className="td-summary__row d-block">
        {combined.odds != null ? (
          <>
            <div className="td-eyebrow">{c.combinedOdds}</div>
            <div className="fw-bold td-num fs-5">{decimalOdds(combined.odds)}</div>
            <p className="small text-body-secondary mb-0">
              {c.bookmakerChance(pct(1 / combined.odds, 1))} {c.fairOdds(decimalOdds(1 / combined.probability))}
            </p>
          </>
        ) : (
          <p className="small text-body-secondary mb-0">{c.addOddsHint}</p>
        )}
      </div>
    </section>
  )
}
