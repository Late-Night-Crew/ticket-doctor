import { useMemo } from 'react'
import { Link, useNavigate } from 'react-router'
import { ApiError } from '../api/client'
import { useAcceptSuggestion, useAnalysis } from '../api/hooks'
import type { AnalyseRequest, AnalysedLeg, Suggestion } from '../api/types'
import { HowFarCard } from '../components/analysis/HowFarCard'
import { SwapOption } from '../components/analysis/SwapOption'
import { TicketSummary } from '../components/analysis/TicketSummary'
import { PageSpinner } from '../components/AppLayout'
import { SaveTicketCard } from '../components/saved/SaveTicketCard'
import { Disclaimer } from '../components/Disclaimer'
import { IconSwap } from '../components/Icons'
import { LegCard } from '../components/LegCard'
import { PageHeader } from '../components/PageHeader'
import { COPY } from '../config/copy'
import { market } from '../config/markets'
import type { AcceptedSwap } from '../state/ticketReducer'
import { useTicket } from '../state/useTicket'

const legDomId = (matchId: string) => `leg-${matchId}`

export default function Analysis() {
  const c = COPY.analysis
  const navigate = useNavigate()
  const { ticket, dispatch } = useTicket()
  const acceptLog = useAcceptSuggestion()

  const request = useMemo<AnalyseRequest | null>(
    () =>
      ticket.legs.length
        ? {
            ticket_id: ticket.ticketId,
            origin: ticket.origin,
            legs: ticket.legs.map((l) => ({ match_id: l.fixture.id, market: l.market, odds: l.odds })),
          }
        : null,
    [ticket.ticketId, ticket.origin, ticket.legs],
  )
  const analysis = useAnalysis(request)
  const updating = analysis.isPlaceholderData || analysis.isFetching

  if (!request) {
    return (
      <>
        <PageHeader eyebrow={c.eyebrow} title={c.emptyTitle}>
          {c.emptyBody}
        </PageHeader>
        <Link to="/ticket" className="btn btn-primary">
          {c.toBuilder}
        </Link>
      </>
    )
  }

  function accept(leg: AnalysedLeg, s: Suggestion) {
    if (!analysis.data) return
    // Logging the acceptance must never block the user (FR-7.1).
    acceptLog.mutate(s.id, { onError: (e) => console.warn('Could not log accepted swap', e) })
    dispatch({
      type: 'acceptSwap',
      matchId: leg.match_id,
      suggestionId: s.id,
      market: s.market,
      baseline: analysis.data.combined.probability,
    })
  }

  function jumpTo(matchId: string) {
    const el = document.getElementById(legDomId(matchId))
    el?.scrollIntoView({ behavior: 'smooth', block: 'center' })
    el?.focus({ preventScroll: true })
  }

  function startOver() {
    if (window.confirm(c.confirmNew)) {
      dispatch({ type: 'clear' })
      navigate('/ticket')
    }
  }

  return (
    <>
      <PageHeader eyebrow={c.eyebrow} title={c.title} />
      <Disclaimer className="mb-4" />

      {analysis.isError && !analysis.data ? (
        <div className="alert alert-danger" role="alert">
          <h2 className="h6">{c.errorTitle}</h2>
          <p className="small mb-2">
            {analysis.error instanceof ApiError ? analysis.error.message : COPY.builder.loadError}
          </p>
          <div className="d-flex gap-2">
            <button type="button" className="btn btn-sm btn-outline-danger" onClick={() => analysis.refetch()}>
              {COPY.builder.retry}
            </button>
            <Link to="/ticket" className="btn btn-sm btn-link">
              {c.editTicket}
            </Link>
          </div>
        </div>
      ) : !analysis.data ? (
        <PageSpinner />
      ) : (
        <div className="row g-4" aria-busy={updating}>
          <div className="col-12 col-lg-5 order-lg-2">
            <div className="td-sticky">
              <TicketSummary
                data={analysis.data}
                baseline={ticket.swaps.length ? ticket.baseline : null}
                onJumpToLeg={jumpTo}
              />
              {updating && (
                <p className="small text-body-secondary mt-2 mb-0" role="status">
                  {c.updating}
                </p>
              )}
              <SaveTicketCard ticketId={ticket.ticketId} updating={updating} changed={JSON.stringify(request)} />
            </div>
          </div>

          <div className={`col-12 col-lg-7 order-lg-1 ${updating ? 'td-updating' : ''}`}>
            <h2 className="visually-hidden">Legs</h2>
            <div className="d-grid gap-3">
              {analysis.data.legs.map((leg) => (
                <AnalysedLegCard
                  key={leg.id}
                  leg={leg}
                  weakest={analysis.data.legs.length > 1 && leg.id === analysis.data.weakest_leg_id}
                  swap={ticket.swaps.find((s) => s.matchId === leg.match_id && s.to === leg.market)}
                  disabled={updating}
                  onAccept={(s) => accept(leg, s)}
                  onUndo={() => dispatch({ type: 'undoSwap', matchId: leg.match_id })}
                  onRemove={() => dispatch({ type: 'remove', matchId: leg.match_id })}
                />
              ))}
            </div>

            <HowFarCard legs={analysis.data.legs} />

            <div className="d-flex flex-wrap gap-2 mt-4">
              <Link to="/ticket" className="btn btn-outline-primary">
                {c.editTicket}
              </Link>
              <button type="button" className="btn btn-link text-body-secondary" onClick={startOver}>
                {c.newTicket}
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  )
}

interface LegProps {
  leg: AnalysedLeg
  weakest: boolean
  swap: AcceptedSwap | undefined
  disabled: boolean
  onAccept: (s: Suggestion) => void
  onUndo: () => void
  onRemove: () => void
}

function AnalysedLegCard({ leg, weakest, swap, disabled, onAccept, onUndo, onRemove }: LegProps) {
  const c = COPY.analysis
  const { home, away } = leg.fixture
  const pick = market(leg.market).plain(home.name, away.name)
  const risky = leg.risk_band !== 'low'

  let footer = null
  if (swap || risky) {
    footer = (
      <div className="d-grid gap-2">
        {swap && (
          <div className="td-swapped" role="status">
            <IconSwap width={16} height={16} className="flex-none" />
            <span className="flex-grow-1">
              {c.swappedFrom(market(swap.from).plain(home.name, away.name))}
              {swap.fromOdds != null && <span className="d-block small text-body-secondary">{c.swappedOdds}</span>}
            </span>
            <button type="button" className="btn btn-sm btn-link flex-none py-0" onClick={onUndo} disabled={disabled}>
              {c.undo}
            </button>
          </div>
        )}
        {risky &&
          (leg.suggestions.length > 0 ? (
            <>
              <div className="fw-semibold d-flex align-items-center gap-1">
                <IconSwap width={16} height={16} /> {c.swapsTitle}
              </div>
              {leg.suggestions.map((s) => (
                <SwapOption key={s.id} fixture={leg.fixture} suggestion={s} disabled={disabled} onAccept={onAccept} />
              ))}
            </>
          ) : (
            <p className="small text-body-secondary mb-0">{leg.low_confidence ? c.noSwapsLimited : c.noSwaps}</p>
          ))}
      </div>
    )
  }

  return (
    <LegCard
      id={legDomId(leg.match_id)}
      fixture={leg.fixture}
      marketKey={leg.market}
      estimate={leg.estimate}
      band={leg.risk_band}
      reason={leg.reason}
      sampleSize={leg.sample_size}
      windowN={leg.window_n}
      lowConfidence={leg.low_confidence}
      odds={leg.odds}
      impliedProb={leg.implied_prob}
      isValue={leg.is_value}
      weakest={weakest}
      footer={footer}
      actions={
        <button
          type="button"
          className="btn btn-sm btn-link text-body-secondary px-0"
          onClick={onRemove}
          disabled={disabled}
          aria-label={c.removeLeg(pick)}
        >
          {c.remove}
        </button>
      }
    />
  )
}
