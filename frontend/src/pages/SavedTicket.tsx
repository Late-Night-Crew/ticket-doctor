import { useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router'
import { ApiError } from '../api/client'
import { awaitingResult, useDeleteSaved, useSavedTicket } from '../api/hooks'
import type { PairCounts, SavedLeg, SavedSummary, SavedTicket as Saved } from '../api/types'
import { PageSpinner } from '../components/AppLayout'
import { PageHeader } from '../components/PageHeader'
import { CopyLink } from '../components/saved/CopyLink'
import { OutcomeBadge } from '../components/saved/Outcome'
import { COPY } from '../config/copy'
import { market } from '../config/markets'
import { kickoffLabel, pct } from '../lib/format'
import { forgetSaved, ownerOf } from '../lib/savedStore'
import { byKickoff, expectedGamesSurvived, gamesSurvived, survival } from '../lib/survival'

const clock = (iso: string) => new Date(iso).toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' })
const day = (iso: string) => new Date(iso).toLocaleDateString(undefined, { day: 'numeric', month: 'short', year: 'numeric' })

/** A saved ticket, leg by leg: your pick, our suggestions, your final pick, and how each did (FR-9.5 to 9.7). */
export default function SavedTicket() {
  const c = COPY.saved
  const { id } = useParams()
  const ticket = useSavedTicket(id)

  if (ticket.isPending) return <PageSpinner />
  if (ticket.isError) {
    const missing = ticket.error instanceof ApiError && ticket.error.status === 404
    return (
      <>
        <PageHeader eyebrow={c.eyebrow} title={missing ? 'Ticket not found' : 'Something went wrong'} />
        <div className={`alert ${missing ? 'alert-info' : 'alert-danger'}`} role="alert">
          {missing ? c.notFound : ticket.error.message}
        </div>
        <Link to="/tickets" className="btn btn-outline-primary">
          {COPY.nav.saved}
        </Link>
      </>
    )
  }

  const t = ticket.data
  return (
    <>
      <PageHeader eyebrow={c.eyebrow} title={c.title}>
        <span className="d-inline-flex flex-wrap gap-2 align-items-center">
          <span className={`badge ${t.status === 'settled' ? 'text-bg-primary' : 'text-bg-secondary'}`}>
            {t.status === 'settled' ? c.settled : c.pending}
          </span>
          <span>{c.savedOn(day(t.updated_at))}</span>
        </span>
      </PageHeader>

      <div className="row g-4">
        <div className="col-12 col-lg-5 order-lg-2">
          <div className="td-sticky d-grid gap-3">
            <TicketOutcomes summary={t.summary} legs={t.legs ?? []} checkedAt={t.results_checked_at} />
            <Pairs pairs={t.summary.pairs} anySuggestions={(t.legs ?? []).some((l) => l.suggestions.length > 0)} />
            <Manage ticket={t} />
          </div>
        </div>
        <div className="col-12 col-lg-7 order-lg-1">
          <h2 className="visually-hidden">Games</h2>
          <div className="d-grid gap-3">
            {(t.legs ?? []).map((l) => (
              <LegReview key={l.fixture.id} leg={l} />
            ))}
          </div>
        </div>
      </div>
    </>
  )
}

/** The estimate behind the pick that ended up on the ticket (the accepted swap's, if any). */
function finalEstimate(l: SavedLeg): number | undefined {
  if (l.final.market === l.original.market) return l.original.estimate
  return l.suggestions.find((s) => s.market === l.final.market)?.estimate
}

/** How far we expected the ticket to get, in kick-off order, against how far it got. */
function HowFarResult({ legs }: { legs: SavedLeg[] }) {
  const c = COPY.howFar
  const rows = legs.flatMap((l, i) => {
    const estimate = finalEstimate(l)
    return estimate === undefined ? [] : [{ id: String(i), kickoff: l.fixture.kickoff, estimate, leg: l }]
  })
  if (legs.length < 3 || rows.length !== legs.length) return null
  const steps = survival(rows)
  const actual = gamesSurvived(byKickoff(rows).map((r) => r.leg.final.outcome))
  const allWon = !actual.ended && actual.survived === legs.length
  return (
    <p className="small mt-2 mb-0">
      {c.expected(expectedGamesSurvived(steps).toFixed(1), legs.length)}{' '}
      {allWon ? c.allWon : actual.ended ? c.gotThrough(actual.survived) : c.soFar(actual.survived)}
    </p>
  )
}

function TicketOutcomes({ summary: s, legs, checkedAt }: { summary: SavedSummary; legs: SavedLeg[]; checkedAt: string | null }) {
  const c = COPY.saved
  return (
    <section className="td-surface p-3" aria-label="Ticket result">
      <p className="small text-body-secondary mb-2">{c.legsSettled(s.settled, s.legs)}</p>
      <dl className="td-saved-outcomes mb-0">
        <dt>{c.ticketAsPicked}</dt>
        <dd>
          <OutcomeBadge outcome={s.original_ticket} label={c.ticketOutcome[s.original_ticket]} />
        </dd>
        <dt>{c.ticketAsPlaced}</dt>
        <dd>
          <OutcomeBadge outcome={s.final_ticket} label={c.ticketOutcome[s.final_ticket]} />
        </dd>
      </dl>
      <HowFarResult legs={legs} />
      {s.settled < s.legs && (
        <p className="small text-body-secondary mt-2 mb-0">
          {c.pendingNote}
          {checkedAt && <> {c.checkedAt(clock(checkedAt))}</>}
        </p>
      )}
    </section>
  )
}

function Pairs({ pairs, anySuggestions }: { pairs: PairCounts; anySuggestions: boolean }) {
  const c = COPY.saved
  const keys = Object.keys(c.pairs) as (keyof PairCounts)[]
  return (
    <section className="td-surface p-3" aria-labelledby="pairs-title">
      <h2 id="pairs-title" className="h6 mb-1">
        {c.pairsTitle}
      </h2>
      {anySuggestions ? (
        <>
          <p className="small text-body-secondary">{c.pairsHelp}</p>
          <ul className="list-unstyled mb-0 td-pairs">
            {keys.map((k) => (
              <li key={k}>
                <span>{c.pairs[k]}</span>
                <span className="td-num fw-bold">{pairs[k]}</span>
              </li>
            ))}
          </ul>
        </>
      ) : (
        <p className="small text-body-secondary mb-0">{c.noPairs}</p>
      )}
    </section>
  )
}

function Manage({ ticket: t }: { ticket: Saved }) {
  const c = COPY.saved
  const owner = ownerOf(t.id)
  const del = useDeleteSaved()
  const navigate = useNavigate()
  const [error, setError] = useState(false)
  const remove = () => {
    if (!owner || !window.confirm(c.deleteConfirm)) return
    del.mutate(
      { id: t.id, ticketId: owner.ticketId },
      {
        onSuccess: () => {
          forgetSaved(t.id)
          navigate('/tickets', { replace: true })
        },
        onError: () => setError(true),
      },
    )
  }
  return (
    <section className="td-surface p-3" aria-labelledby="manage-title">
      <h2 id="manage-title" className="h6 mb-2">
        {c.copyLinkTitle}
      </h2>
      <CopyLink id={t.id} />
      <p className="small text-body-secondary mt-2 mb-2">{t.expires_at ? c.expires(day(t.expires_at)) : c.keepNote}</p>
      {owner ? (
        <button type="button" className="btn btn-sm btn-outline-danger" onClick={remove} disabled={del.isPending}>
          {c.delete}
        </button>
      ) : (
        <p className="small text-body-secondary mb-0">{c.notOwner}</p>
      )}
      {error && (
        <p className="small text-danger mt-2 mb-0" role="alert">
          {c.deleteError}
        </p>
      )}
    </section>
  )
}

function LegReview({ leg }: { leg: SavedLeg }) {
  const c = COPY.saved
  const { home, away } = leg.fixture
  const pick = (m: SavedLeg['original']['market']) => market(m).plain(home.name, away.name)
  const swapped = leg.final.market !== leg.original.market
  const title = `${home.name} v ${away.name}`

  return (
    <article className="td-surface p-3 td-review" aria-label={title}>
      <header className="d-flex flex-wrap align-items-baseline gap-2 mb-2">
        <h3 className="h6 mb-0">{title}</h3>
        <span className="small text-body-secondary">
          {leg.score ? (
            <span className="td-num fw-semibold text-body">
              {leg.score.home}–{leg.score.away}
            </span>
          ) : leg.match_status === 'postponed' ? (
            c.postponed
          ) : awaitingResult(leg) ? (
            c.awaitingResult
          ) : (
            kickoffLabel(leg.fixture.kickoff)
          )}
        </span>
      </header>

      <div className="td-review__grid">
        <div>
          <div className="td-eyebrow mb-1">{c.yourPick}</div>
          <div className="fw-semibold">{pick(leg.original.market)}</div>
          {leg.original.estimate !== undefined && (
            <div className="small text-body-secondary td-num">Our estimate {pct(leg.original.estimate)}</div>
          )}
          <OutcomeBadge outcome={leg.original.outcome} />
        </div>

        <div>
          <div className="td-eyebrow mb-1">{c.ourSuggestions}</div>
          {leg.suggestions.length === 0 ? (
            <div className="small text-body-secondary">{c.noSuggestions}</div>
          ) : (
            <ul className="list-unstyled mb-0 d-grid gap-2">
              {leg.suggestions.map((s) => (
                <li key={s.market}>
                  <div className="fw-semibold">
                    {pick(s.market)}
                    {s.accepted && <span className="badge text-bg-light border ms-1">{c.accepted}</span>}
                  </div>
                  <div className="small text-body-secondary td-num">Our estimate {pct(s.estimate)}</div>
                  <OutcomeBadge outcome={s.outcome} />
                </li>
              ))}
            </ul>
          )}
        </div>

        <div>
          <div className="td-eyebrow mb-1">{c.finalPick}</div>
          <div className="fw-semibold">{pick(leg.final.market)}</div>
          {swapped && <div className="small text-body-secondary">Swapped from your pick</div>}
          <OutcomeBadge outcome={leg.final.outcome} />
        </div>
      </div>
    </article>
  )
}
