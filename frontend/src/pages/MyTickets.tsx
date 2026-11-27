import { useEffect, useMemo } from 'react'
import { Link } from 'react-router'
import { useSavedList } from '../api/hooks'
import type { HitCount, PairCounts, SavedTicket } from '../api/types'
import { PageSpinner } from '../components/AppLayout'
import { PageHeader } from '../components/PageHeader'
import { RecordCard } from '../components/results/RecordCard'
import { OutcomeBadge } from '../components/saved/Outcome'
import { COPY } from '../config/copy'
import { keepOnly, loadSaved } from '../lib/savedStore'

const day = (iso: string) => new Date(iso).toLocaleDateString(undefined, { weekday: 'short', day: 'numeric', month: 'short' })

/** The tickets saved on this device (FR-9.8) and a personal record across them (FR-9.9). */
export default function MyTickets() {
  const c = COPY.saved
  const ids = useMemo(() => loadSaved().map((e) => e.id), [])
  const list = useSavedList(ids)

  // Forget tickets the server no longer has (deleted, or past 90 days).
  useEffect(() => {
    if (list.data) keepOnly(new Set(list.data.map((t) => t.id)))
  }, [list.data])

  return (
    <>
      <PageHeader eyebrow={COPY.saved.eyebrow} title={c.listTitle}>
        {c.listLead}
      </PageHeader>

      {ids.length === 0 ? (
        <div className="td-surface p-3">
          <p className="mb-3">{c.listEmpty}</p>
          <Link to="/ticket" className="btn btn-primary">
            {c.listCheck}
          </Link>
        </div>
      ) : list.isPending ? (
        <PageSpinner />
      ) : list.isError ? (
        <div className="alert alert-danger" role="alert">
          {list.error.message}
        </div>
      ) : list.data.length === 0 ? (
        <div className="td-surface p-3">
          <p className="mb-0">{c.listEmpty}</p>
        </div>
      ) : (
        <>
          <PersonalRecord tickets={list.data} />
          <ul className="list-unstyled d-grid gap-2 mt-4">
            {list.data.map((t) => (
              <li key={t.id}>
                <TicketRow ticket={t} />
              </li>
            ))}
          </ul>
        </>
      )}
      <p className="small text-body-secondary mt-4">{c.deviceNote}</p>
    </>
  )
}

function TicketRow({ ticket: t }: { ticket: SavedTicket }) {
  const c = COPY.saved
  const s = t.summary
  return (
    <Link to={`/t/${t.id}`} className="td-surface p-3 d-flex flex-wrap align-items-center gap-2 text-decoration-none td-saved-row">
      <div className="flex-grow-1">
        <div className="fw-semibold text-body">
          {c.savedOn(day(t.updated_at))} · {c.legsCount(s.legs)}
        </div>
        <div className="small text-body-secondary">{c.legsSettled(s.settled, s.legs)}</div>
      </div>
      <span className={`badge ${t.status === 'settled' ? 'text-bg-primary' : 'text-bg-secondary'}`}>
        {t.status === 'settled' ? c.settled : c.pending}
      </span>
      <OutcomeBadge outcome={s.final_ticket} label={c.ticketOutcome[s.final_ticket]} />
    </Link>
  )
}

/** Sums every played game across the saved tickets (FR-9.9). */
function PersonalRecord({ tickets }: { tickets: SavedTicket[] }) {
  const c = COPY.saved
  const add = (a: HitCount, b: HitCount): HitCount => ({ won: a.won + b.won, lost: a.lost + b.lost })
  const own = tickets.reduce((h, t) => add(h, t.summary.original), { won: 0, lost: 0 })
  const pairs = tickets.reduce(
    (p, t) => {
      const q = t.summary.pairs
      return {
        suggestion_won_pick_lost: p.suggestion_won_pick_lost + q.suggestion_won_pick_lost,
        pick_won_suggestion_lost: p.pick_won_suggestion_lost + q.pick_won_suggestion_lost,
        both_won: p.both_won + q.both_won,
        both_lost: p.both_lost + q.both_lost,
      }
    },
    { suggestion_won_pick_lost: 0, pick_won_suggestion_lost: 0, both_won: 0, both_lost: 0 } as PairCounts,
  )
  // Suggestions won when they won outright or alongside the pick.
  const ours: HitCount = {
    won: pairs.suggestion_won_pick_lost + pairs.both_won,
    lost: pairs.pick_won_suggestion_lost + pairs.both_lost,
  }

  return (
    <section aria-labelledby="record-title">
      <h2 id="record-title" className="h5 mb-1">
        {c.recordTitle}
      </h2>
      <p className="small text-body-secondary">{c.recordHelp}</p>
      <div className="row g-3">
        <div className="col-sm-6">
          <RecordCard title={c.yourPicks} help="Every game you picked yourself." count={own} />
        </div>
        <div className="col-sm-6">
          <RecordCard title={c.ourPicks} help="Every swap we suggested on your tickets, taken or not." count={ours} />
        </div>
      </div>
    </section>
  )
}
