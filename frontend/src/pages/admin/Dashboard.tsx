import { Link } from 'react-router'
import { useDashboard, type ImportRun } from '../../api/admin'
import { PageSpinner } from '../../components/AppLayout'
import { PageHeader } from '../../components/PageHeader'
import { pct } from '../../lib/format'

const dateTime = (iso: string) =>
  new Date(iso).toLocaleString(undefined, { weekday: 'short', day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' })

export default function Dashboard() {
  const { data, isPending, isError, error } = useDashboard()
  if (isPending) return <PageSpinner />
  if (isError) return <div className="alert alert-danger">Couldn't load the dashboard: {error.message}</div>

  const { matches, tickets, suggestions, imports, score_checks } = data
  const lastOk = imports.find((r) => r.status === 'succeeded')
  const lastRun = imports[0]

  return (
    <>
      <PageHeader eyebrow="Admin" title="Dashboard" />

      {lastRun?.status === 'failed' && (
        <div className="alert alert-danger" role="alert">
          The last import failed{lastRun.error ? `: ${lastRun.error}` : '.'} Check the Import workflow on GitHub.
        </div>
      )}
      {matches.awaiting_result > 0 && (
        <div className="alert alert-warning d-flex flex-wrap align-items-center gap-2" role="status">
          <span>
            <strong className="td-num">{matches.awaiting_result}</strong>{' '}
            {matches.awaiting_result === 1 ? 'game has' : 'games have'} kicked off with no result yet.
          </span>
          <Link to="/admin/results" className="alert-link ms-auto">
            Enter results
          </Link>
        </div>
      )}

      <div className="td-admin-stats mb-4">
        <Stat label="Upcoming games" value={matches.upcoming} />
        <Stat label="Awaiting a result" value={matches.awaiting_result} />
        <Stat label="Finished games" value={matches.finished} />
        <Stat label="Tickets analysed" value={tickets.total} note={`${tickets.last_7_days} in the last 7 days`} />
        <Stat label="Picks logged" value={tickets.legs} />
        <Stat
          label="Swaps accepted"
          value={suggestions.accepted}
          note={suggestions.shown ? `${pct(suggestions.accepted / suggestions.shown)} of ${suggestions.shown} offered` : 'none offered yet'}
        />
      </div>

      <ScoreChecks runs={score_checks} />

      <section className="td-surface p-3">
        <div className="d-flex flex-wrap align-items-baseline gap-2 mb-2">
          <h2 className="h6 mb-0">Imports</h2>
          <span className="small text-body-secondary">
            {lastOk ? `Last success ${dateTime(lastOk.finished_at ?? lastOk.started_at)}` : 'No successful import yet'}
            {' · '}runs at 06:15 and 00:15 Malawi time
          </span>
        </div>
        {imports.length === 0 ? (
          <p className="text-body-secondary mb-0">No imports have run yet.</p>
        ) : (
          <div className="table-responsive">
            <table className="table table-sm align-middle mb-0">
              <thead>
                <tr>
                  <th scope="col">Started</th>
                  <th scope="col">Status</th>
                  <th scope="col">What changed</th>
                </tr>
              </thead>
              <tbody>
                {imports.map((r) => (
                  <ImportRow key={r.id} run={r} />
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </>
  )
}

function Stat({ label, value, note }: { label: string; value: number; note?: string }) {
  return (
    <div className="td-surface p-3">
      <div className="small text-body-secondary">{label}</div>
      <div className="td-admin-stat td-num">{value.toLocaleString()}</div>
      {note && <div className="small text-body-secondary">{note}</div>}
    </div>
  )
}

const STATUS_CLASS = { succeeded: 'text-bg-success', failed: 'text-bg-danger', running: 'text-bg-secondary' } as const

function ImportRow({ run }: { run: ImportRun }) {
  const leagues = run.summary.leagues ?? []
  const results = leagues.reduce((n, l) => n + (l.inserted ?? 0) + (l.updated ?? 0), 0)
  const quick = leagues.reduce((n, l) => n + (l.fixtures?.results ?? 0), 0)
  const corrected = leagues.flatMap((l) => (l.corrected ?? []).map((c) => `${l.league}: ${c}`))
  const fixtures = leagues.reduce((n, l) => n + (l.fixtures ? l.fixtures.inserted + l.fixtures.updated : 0), 0)
  const problems = leagues.flatMap((l) => [
    ...(l.error ? [`${l.league}: ${l.error}`] : []),
    ...(l.fixtures?.error ? [`${l.league} fixtures: ${l.fixtures.error}`] : []),
    ...(l.fixtures?.unmatched?.length ? [`${l.league}: unmatched ${l.fixtures.unmatched.join(', ')}`] : []),
  ])
  return (
    <tr>
      <td className="text-nowrap">{dateTime(run.started_at)}</td>
      <td>
        <span className={`badge ${STATUS_CLASS[run.status]}`}>{run.status}</span>
      </td>
      <td className="small">
        {run.error ?? `${results} results, ${fixtures} fixtures new or changed`}
        {quick > 0 && <div className="text-body-secondary">{quick} early scores from football-data.org</div>}
        {corrected.map((c) => (
          <div key={c} className="text-warning-emphasis">
            Score corrected by the results file: {c}
          </div>
        ))}
        {problems.map((p) => (
          <div key={p} className="text-danger">
            {p}
          </div>
        ))}
      </td>
    </tr>
  )
}

/**
 * Quick score checks (A10): every 15 minutes on match days, full-time scores
 * from football-data.org so saved tickets settle soon after the final whistle.
 */
function ScoreChecks({ runs }: { runs: ImportRun[] }) {
  const last = runs[0]
  const failed = runs.filter((r) => r.status === 'failed')
  return (
    <section className="td-surface p-3 mb-4" aria-labelledby="scores-title">
      <div className="d-flex flex-wrap align-items-baseline gap-2">
        <h2 id="scores-title" className="h6 mb-0">
          Score checks
        </h2>
        <span className="small text-body-secondary">
          every 15 minutes on match days: weekends 13:00–01:45, weekday evenings 19:00–01:45 Malawi time
        </span>
      </div>
      <p className="small mb-0 mt-2">
        {last ? (
          <>
            Last check {dateTime(last.finished_at ?? last.started_at)}:{' '}
            {last.status === 'failed'
              ? 'failed'
              : `${(last.summary.leagues ?? []).reduce((n, l) => n + (l.results ?? 0), 0)} new full-time scores`}
            .
          </>
        ) : (
          'No checks yet. A check only records itself when a game is awaiting its result.'
        )}
      </p>
      {failed.slice(0, 1).map((r) => (
        <p key={r.id} className="small text-danger mb-0 mt-1">
          {dateTime(r.started_at)}: {r.error ?? 'failed'}
          {(r.summary.leagues ?? [])
            .filter((l) => l.error)
            .map((l) => (
              <span key={l.league} className="d-block">
                {l.league}: {l.error}
              </span>
            ))}
        </p>
      ))}
    </section>
  )
}
