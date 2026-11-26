import { useId } from 'react'
import { useSearchParams } from 'react-router'
import { useLeagues, useResults } from '../api/hooks'
import type { MarketKey } from '../api/types'
import { PageSpinner } from '../components/AppLayout'
import { LeaguePicker } from '../components/builder/LeaguePicker'
import { PageHeader } from '../components/PageHeader'
import { RecordCard } from '../components/results/RecordCard'
import { ResultsTable } from '../components/results/ResultsTable'
import { COPY } from '../config/copy'
import { MARKETS } from '../config/markets'
import { RULES } from '../config/rules'
import { totals } from '../lib/results'

const ALL = 'all'

/** Public track record: original picks vs suggested swaps (FR-7.3, RG-3). */
export default function Results() {
  const c = COPY.results
  const marketId = useId()
  const [params, setParams] = useSearchParams()
  const league = params.get('league') ?? undefined
  const marketParam = params.get('market') ?? undefined
  const marketFilter = MARKETS.some((m) => m.key === marketParam) ? (marketParam as MarketKey) : undefined

  const leagues = useLeagues()
  const results = useResults({ league, market: marketFilter })

  function setFilter(key: 'league' | 'market', value: string) {
    const next = new URLSearchParams(params)
    if (value === ALL || value === '') next.delete(key)
    else next.set(key, value)
    setParams(next, { replace: true })
  }

  const rows = results.data?.rows ?? []
  const sum = totals(rows)

  return (
    <>
      <PageHeader eyebrow={c.eyebrow} title={c.title}>
        {c.lead}
      </PageHeader>

      <div className="d-grid gap-2 mb-4">
        <LeaguePicker
          leagues={[{ id: ALL, name: c.allLeagues, country: '' }, ...(leagues.data ?? [])]}
          value={league ?? ALL}
          onChange={(id) => setFilter('league', id)}
        />
        <div className="td-market-filter">
          <label htmlFor={marketId} className="form-label small fw-semibold mb-1">
            {c.marketLabel}
          </label>
          <select
            id={marketId}
            className="form-select"
            value={marketFilter ?? ALL}
            onChange={(e) => setFilter('market', e.target.value)}
          >
            <option value={ALL}>{c.allMarkets}</option>
            {MARKETS.map((m) => (
              <option key={m.key} value={m.key}>
                {m.short}
              </option>
            ))}
          </select>
        </div>
      </div>

      {results.isPending ? (
        <PageSpinner />
      ) : results.isError ? (
        <div className="alert alert-danger d-flex justify-content-between align-items-center gap-2" role="alert">
          <span className="small">{c.error}</span>
          <button type="button" className="btn btn-sm btn-outline-danger flex-none" onClick={() => results.refetch()}>
            {COPY.builder.retry}
          </button>
        </div>
      ) : rows.length === 0 ? (
        <p className="td-surface p-3 text-body-secondary">{c.empty}</p>
      ) : (
        <>
          <div className="row g-3 mb-4">
            <div className="col-12 col-md-6">
              <RecordCard title={c.original} help={c.originalHelp} count={sum.original} />
            </div>
            <div className="col-12 col-md-6">
              <RecordCard title={c.suggested} help={c.suggestedHelp} count={sum.suggested} />
            </div>
          </div>

          <ResultsTable rows={rows} leagues={leagues.data ?? []} />
        </>
      )}

      {results.data && (
        <ul className="small text-body-secondary mt-3 ps-3">
          <li>{c.groupsNote}</li>
          <li>{c.pendingNote}</li>
          <li>{c.smallSampleNote(RULES.results.smallSample)}</li>
          <li>{c.pastNote}</li>
          <li>
            {c.updated(
              new Date(results.data.updated_at).toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' }),
            )}
          </li>
        </ul>
      )}
    </>
  )
}
