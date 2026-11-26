import type { ReactNode } from 'react'
import type { Fixture, MarketKey, RiskBand } from '../api/types'
import { COPY } from '../config/copy'
import { market } from '../config/markets'
import { decimalOdds, kickoffLabel, pct } from '../lib/format'
import { GlossaryTerm } from './GlossaryTerm'
import { RiskBadge } from './RiskBadge'

export interface LegCardProps {
  id?: string
  fixture: Fixture
  marketKey: MarketKey
  estimate: number
  band: RiskBand
  reason: string
  sampleSize: number
  windowN: number
  lowConfidence?: boolean
  odds?: number | null
  impliedProb?: number | null
  isValue?: boolean | null
  weakest?: boolean
  /** Extra content under a perforated divider, e.g. swap suggestions. */
  footer?: ReactNode
  actions?: ReactNode
}

export function LegCard(p: LegCardProps) {
  const def = market(p.marketKey)
  const { home, away } = p.fixture
  return (
    <article id={p.id} className={`td-leg td-leg--${p.band} ${p.weakest ? 'td-leg--weakest' : ''}`} tabIndex={-1}>
      <div className="td-leg__body">
        <div className="d-flex justify-content-between align-items-start gap-2">
          <div className="min-w-0">
            <div className="td-leg__match">
              {home.name} v {away.name} · {kickoffLabel(p.fixture.kickoff)}
            </div>
            <h3 className="td-leg__pick">{def.plain(home.name, away.name)}</h3>
            <div className="d-flex flex-wrap align-items-center gap-2">
              <RiskBadge band={p.band} limitedData={p.lowConfidence} />
              <span className="small text-body-secondary">
                <GlossaryTerm id={def.glossary}>{def.short}</GlossaryTerm>
              </span>
              {p.weakest && <span className="small fw-bold text-danger">Weakest leg</span>}
            </div>
          </div>
          <div className="text-end flex-none">
            <div className="td-leg__estimate td-num">{pct(p.estimate)}</div>
            <div className="small text-body-secondary">{COPY.probability.estimate.toLowerCase()}</div>
          </div>
        </div>

        <p className="td-leg__reason">{p.reason}</p>

        <div className="d-flex flex-wrap align-items-center column-gap-3 row-gap-1 td-leg__meta">
          <span>{COPY.probability.basedOn(p.sampleSize, p.windowN)}</span>
          {p.odds != null && p.impliedProb != null && (
            <span className="td-num">
              Odds {decimalOdds(p.odds)} → {COPY.probability.implied.toLowerCase()} {pct(p.impliedProb)}
            </span>
          )}
          {p.isValue && <span className="td-value-tag">{COPY.probability.value}</span>}
        </div>
        {p.actions && <div className="mt-2">{p.actions}</div>}
      </div>
      {p.footer && (
        <>
          <hr className="td-leg__perf" />
          <div className="td-leg__foot">{p.footer}</div>
        </>
      )}
    </article>
  )
}
