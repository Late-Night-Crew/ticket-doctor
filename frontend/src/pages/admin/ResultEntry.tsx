import { useState, type FormEvent } from 'react'
import { useAdminMatches, useEnterResult, type AdminMatch, type MatchView } from '../../api/admin'
import { useLeagues } from '../../api/hooks'
import { PageSpinner } from '../../components/AppLayout'
import { PageHeader } from '../../components/PageHeader'
import { kickoffLabel } from '../../lib/format'

const MAX_GOALS = 20

/** Manual result entry (FR-1.4): games that kicked off with no result, or recent results to correct. */
export default function ResultEntry() {
  const [view, setView] = useState<MatchView>('awaiting')
  const [league, setLeague] = useState('')
  const leagues = useLeagues()
  const matches = useAdminMatches(view, league)
  const leagueName = (id: string) => leagues.data?.find((l) => l.id === id)?.name ?? id

  return (
    <>
      <PageHeader eyebrow="Admin" title="Enter results">
        For games the daily import hasn't got a result for yet. If the import later has a different score for the same
        game, its score replaces yours. Team stats pick results up at the next import.
      </PageHeader>

      <div className="d-flex flex-wrap gap-2 align-items-center mb-3">
        <div className="btn-group" role="group" aria-label="Which games">
          {(
            [
              ['awaiting', 'Awaiting a result'],
              ['recent', 'Recent results'],
            ] as const
          ).map(([v, label]) => (
            <button
              key={v}
              type="button"
              className={`btn btn-sm ${view === v ? 'btn-primary' : 'btn-outline-primary'}`}
              aria-pressed={view === v}
              onClick={() => setView(v)}
            >
              {label}
            </button>
          ))}
        </div>
        <label className="visually-hidden" htmlFor="admin-league">
          League
        </label>
        <select
          id="admin-league"
          className="form-select form-select-sm w-auto"
          value={league}
          onChange={(e) => setLeague(e.target.value)}
        >
          <option value="">All leagues</option>
          {leagues.data?.map((l) => (
            <option key={l.id} value={l.id}>
              {l.name}
            </option>
          ))}
        </select>
      </div>

      {matches.isPending ? (
        <PageSpinner />
      ) : matches.isError ? (
        <div className="alert alert-danger">Couldn't load the games: {matches.error.message}</div>
      ) : matches.data.length === 0 ? (
        <p className="td-surface p-3 text-body-secondary">
          {view === 'awaiting'
            ? 'Nothing waiting: every game that has kicked off has a result.'
            : 'No results in the last two weeks.'}
        </p>
      ) : (
        <ul className="list-unstyled d-grid gap-2">
          {matches.data.map((m) => (
            // Keyed on the score too, so the inputs reset when it changes.
            <ResultRow key={`${m.id}-${m.home_goals}-${m.away_goals}`} match={m} league={leagueName(m.league_id)} />
          ))}
        </ul>
      )}
    </>
  )
}

function ResultRow({ match: m, league }: { match: AdminMatch; league: string }) {
  const [home, setHome] = useState(m.home_goals?.toString() ?? '')
  const [away, setAway] = useState(m.away_goals?.toString() ?? '')
  const save = useEnterResult()
  const finished = m.status === 'finished'

  const valid = (v: string) => /^\d{1,2}$/.test(v) && Number(v) <= MAX_GOALS
  const changed = home !== (m.home_goals?.toString() ?? '') || away !== (m.away_goals?.toString() ?? '')
  const canSave = valid(home) && valid(away) && changed && !save.isPending

  const submit = (e: FormEvent) => {
    e.preventDefault()
    if (canSave) save.mutate({ id: m.id, home: Number(home), away: Number(away) })
  }

  const label = `${m.home} v ${m.away}`
  return (
    <li className="td-surface p-3">
      <form onSubmit={submit} className="td-result-row" aria-label={label}>
        <div className="td-result-row__match">
          <div className="small text-body-secondary">
            {league} · {kickoffLabel(m.kickoff)}
            {m.status === 'postponed' && <span className="badge text-bg-warning ms-2">postponed</span>}
            {finished && <span className="badge text-bg-light border ms-2">{m.source === 'manual' ? 'entered by hand' : 'from import'}</span>}
          </div>
          <div className="fw-semibold">{label}</div>
        </div>
        <div className="td-result-row__score">
          <ScoreInput id={`h-${m.id}`} label={`${m.home} goals`} value={home} onChange={setHome} invalid={home !== '' && !valid(home)} />
          <span aria-hidden="true">–</span>
          <ScoreInput id={`a-${m.id}`} label={`${m.away} goals`} value={away} onChange={setAway} invalid={away !== '' && !valid(away)} />
          <button type="submit" className="btn btn-primary btn-sm" disabled={!canSave}>
            {save.isPending ? 'Saving…' : finished ? 'Correct' : 'Save'}
          </button>
        </div>
      </form>
      {save.isError && (
        <div className="text-danger small mt-2" role="alert">
          {save.error.message}
        </div>
      )}
    </li>
  )
}

function ScoreInput(props: { id: string; label: string; value: string; onChange: (v: string) => void; invalid: boolean }) {
  return (
    <>
      <label htmlFor={props.id} className="visually-hidden">
        {props.label}
      </label>
      <input
        id={props.id}
        className={`form-control form-control-sm td-num td-score-input${props.invalid ? ' is-invalid' : ''}`}
        inputMode="numeric"
        pattern="[0-9]*"
        maxLength={2}
        value={props.value}
        onChange={(e) => props.onChange(e.target.value.trim())}
      />
    </>
  )
}
