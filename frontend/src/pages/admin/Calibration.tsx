import { useState } from 'react'
import { useCalibration, type Calibration as Cal, type CalibrationQuery } from '../../api/admin'
import { useLeagues } from '../../api/hooks'
import type { MarketKey } from '../../api/types'
import { ReliabilityChart } from '../../components/admin/ReliabilityChart'
import { PageSpinner } from '../../components/AppLayout'
import { PageHeader } from '../../components/PageHeader'
import { MARKETS, market } from '../../config/markets'
import { pct } from '../../lib/format'

/** "+2.4 points, too cautious" style reading of a gap between said and happened. */
function gap(predicted: number, actual: number): string {
  const d = (actual - predicted) * 100
  if (Math.abs(d) < 1) return 'about right'
  return `${d > 0 ? '+' : '−'}${Math.abs(d).toFixed(1)} pts, ${d > 0 ? 'too cautious' : 'too confident'}`
}

const BAND_LABEL = { low: 'Low risk', medium: 'Medium risk', high: 'High risk' } as const

/** Are the estimates right? Backtest over past seasons, or the estimates people were shown. */
export default function Calibration() {
  const [q, setQ] = useState<CalibrationQuery>({ source: 'backtest', league: '', market: '', kind: 'picks', low: false })
  const set = (patch: Partial<CalibrationQuery>) => setQ((old) => ({ ...old, ...patch }))
  const leagues = useLeagues()
  const cal = useCalibration(q)

  return (
    <>
      <PageHeader eyebrow="Admin" title="Calibration">
        When we say 70%, does it happen about 70% of the time? <strong>Past seasons</strong> replays every finished
        game, rating each only from the games before it, with the rules in use now. <strong>Real tickets</strong> uses
        the estimates people were actually shown, once their games finish.
      </PageHeader>

      {/* One filter row above everything it scopes */}
      <div className="d-flex flex-wrap gap-2 align-items-center mb-3" role="group" aria-label="Filters">
        <div className="btn-group" role="group" aria-label="Source">
          {(
            [
              ['backtest', 'Past seasons'],
              ['tickets', 'Real tickets'],
            ] as const
          ).map(([v, label]) => (
            <button
              key={v}
              type="button"
              className={`btn btn-sm ${q.source === v ? 'btn-primary' : 'btn-outline-primary'}`}
              aria-pressed={q.source === v}
              onClick={() => set({ source: v })}
            >
              {label}
            </button>
          ))}
        </div>
        {q.source === 'tickets' && (
          <select
            className="form-select form-select-sm w-auto"
            aria-label="Whose picks"
            value={q.kind}
            onChange={(e) => set({ kind: e.target.value as CalibrationQuery['kind'] })}
          >
            <option value="picks">Users' own picks</option>
            <option value="suggestions">Our suggestions</option>
          </select>
        )}
        <select
          className="form-select form-select-sm w-auto"
          aria-label="League"
          value={q.league}
          onChange={(e) => set({ league: e.target.value })}
        >
          <option value="">All leagues</option>
          {leagues.data?.map((l) => (
            <option key={l.id} value={l.id}>
              {l.name}
            </option>
          ))}
        </select>
        <select
          className="form-select form-select-sm w-auto"
          aria-label="Market"
          value={q.market}
          onChange={(e) => set({ market: e.target.value })}
        >
          <option value="">All markets</option>
          {MARKETS.map((m) => (
            <option key={m.key} value={m.key}>
              {m.short}
            </option>
          ))}
        </select>
        {q.source === 'backtest' && (
          <div className="form-check mb-0">
            <input
              id="cal-low"
              type="checkbox"
              className="form-check-input"
              checked={q.low}
              onChange={(e) => set({ low: e.target.checked })}
            />
            <label htmlFor="cal-low" className="form-check-label small">
              Include games with thin team data
            </label>
          </div>
        )}
      </div>

      {cal.isPending ? (
        <PageSpinner />
      ) : cal.isError ? (
        <div className="alert alert-danger">Couldn't load calibration: {cal.error.message}</div>
      ) : cal.data.n === 0 ? (
        <p className="td-surface p-3 text-body-secondary">
          {q.source === 'tickets'
            ? 'No analysed picks have finished games yet. This fills in as results arrive.'
            : 'No finished games to rate for this filter.'}
        </p>
      ) : (
        <div className={cal.isPlaceholderData ? 'td-updating' : undefined}>
          <Report data={cal.data} />
        </div>
      )}
    </>
  )
}

function Report({ data }: { data: Cal }) {
  const bins = data.bins.filter((b) => b.n > 0)
  return (
    <>
      <p className="mb-3">
        <strong>{data.n.toLocaleString()}</strong> estimates
        {data.source === 'backtest' && (
          <>
            {' '}
            from <strong>{data.matches.toLocaleString()}</strong> games
            {data.skipped_low_confidence > 0 && ` (${data.skipped_low_confidence} left out for thin team data)`}
          </>
        )}
        .
      </p>

      <div className="row g-4 mb-4">
        <div className="col-lg-6">
          <section className="td-surface p-3 h-100" aria-labelledby="rel-title">
            <h2 id="rel-title" className="h6 mb-1">
              What we said vs what happened
            </h2>
            <p className="small text-body-secondary">On the thin diagonal is spot on; below it we were too confident, above it too cautious.</p>
            <ReliabilityChart bins={data.bins} label="Reliability chart: what we said against what happened" />
          </section>
        </div>
        <div className="col-lg-6">
          <section className="td-surface p-0 h-100 overflow-hidden" aria-labelledby="bins-title">
            <h2 id="bins-title" className="h6 px-3 pt-3 mb-2">
              By estimate
            </h2>
            <div className="table-responsive">
              <table className="table table-sm align-middle mb-0 td-num text-nowrap">
                <thead>
                  <tr>
                    <th scope="col">We said</th>
                    <th scope="col" className="text-end">
                      Picks
                    </th>
                    <th scope="col" className="text-end">
                      Average
                    </th>
                    <th scope="col" className="text-end">
                      Happened
                    </th>
                    <th scope="col">Gap</th>
                  </tr>
                </thead>
                <tbody>
                  {bins.map((b) => (
                    <tr key={b.from}>
                      <td>
                        {pct(b.from)}–{pct(b.to)}
                      </td>
                      <td className="text-end">{b.n.toLocaleString()}</td>
                      <td className="text-end">{pct(b.predicted, 1)}</td>
                      <td className="text-end fw-semibold">{pct(b.actual, 1)}</td>
                      <td className="small text-body-secondary">{gap(b.predicted, b.actual)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>
        </div>
      </div>

      <section className="td-surface p-0 overflow-hidden mb-4" aria-labelledby="bands-title">
        <h2 id="bands-title" className="h6 px-3 pt-3 mb-2">
          By risk band
        </h2>
        <div className="table-responsive">
          <table className="table table-sm align-middle mb-0 td-num text-nowrap">
            <thead>
              <tr>
                <th scope="col">Band</th>
                <th scope="col" className="text-end">
                  Picks
                </th>
                <th scope="col" className="text-end">
                  We said
                </th>
                <th scope="col" className="text-end">
                  Happened
                </th>
                <th scope="col">Gap</th>
              </tr>
            </thead>
            <tbody>
              {data.bands.map((b) => (
                <tr key={b.band}>
                  <td>{BAND_LABEL[b.band]}</td>
                  <td className="text-end">{b.n.toLocaleString()}</td>
                  <td className="text-end">{pct(b.predicted, 1)}</td>
                  <td className="text-end fw-semibold">{pct(b.actual, 1)}</td>
                  <td className="small text-body-secondary">{gap(b.predicted, b.actual)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      <section className="td-surface p-0 overflow-hidden" aria-labelledby="markets-title">
        <div className="px-3 pt-3">
          <h2 id="markets-title" className="h6 mb-1">
            By market: do the team stats help?
          </h2>
          <p className="small text-body-secondary mb-2">
            <strong>Skill</strong> compares our estimates with simply guessing each market's average rate every time.
            Above 0 means the team stats add something; 0 or below means guessing the average would have done as
            well. Lower Brier is better.
          </p>
        </div>
        <div className="table-responsive">
          <table className="table table-sm align-middle mb-0 td-num text-nowrap">
            <thead>
              <tr>
                <th scope="col">Market</th>
                <th scope="col" className="text-end">
                  Picks
                </th>
                <th scope="col" className="text-end">
                  We said
                </th>
                <th scope="col" className="text-end">
                  Happened
                </th>
                <th scope="col" className="text-end">
                  Brier
                </th>
                <th scope="col" className="text-end">
                  Guess-the-average
                </th>
                <th scope="col" className="text-end">
                  Skill
                </th>
              </tr>
            </thead>
            <tbody>
              {data.markets.map((m) => (
                <tr key={m.market}>
                  <td>{market(m.market as MarketKey).short}</td>
                  <td className="text-end">{m.n.toLocaleString()}</td>
                  <td className="text-end">{pct(m.predicted, 1)}</td>
                  <td className="text-end">{pct(m.actual, 1)}</td>
                  <td className="text-end">{m.brier.toFixed(3)}</td>
                  <td className="text-end">{m.base_brier.toFixed(3)}</td>
                  <td className="text-end fw-semibold">
                    {m.skill > 0 ? '+' : m.skill < 0 ? '−' : ''}
                    {Math.abs(m.skill).toFixed(3)}
                    <span className="visually-hidden">{m.skill > 0 ? ' (better than guessing)' : ' (no better than guessing)'}</span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
    </>
  )
}
