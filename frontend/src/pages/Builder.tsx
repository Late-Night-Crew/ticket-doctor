import { useMemo, useState } from 'react'
import Offcanvas from 'react-bootstrap/Offcanvas'
import { useNavigate, useSearchParams } from 'react-router'
import { useFixtures, useLeagues } from '../api/hooks'
import type { Fixture, FixtureWindow } from '../api/types'
import { PageSpinner } from '../components/AppLayout'
import { FixtureList } from '../components/builder/FixtureList'
import { LeaguePicker } from '../components/builder/LeaguePicker'
import { MarketSheet } from '../components/builder/MarketSheet'
import { TicketSlip } from '../components/builder/TicketSlip'
import { WindowToggle } from '../components/builder/WindowToggle'
import { PageHeader } from '../components/PageHeader'
import { COPY } from '../config/copy'
import { RULES } from '../config/rules'
import type { DraftLeg } from '../state/ticketReducer'
import { useTicket } from '../state/useTicket'

const WINDOWS: FixtureWindow[] = ['today', 'weekend', 'upcoming']

export default function Builder() {
  const c = COPY.builder
  const navigate = useNavigate()
  const { ticket, dispatch, isFull } = useTicket()
  const [params, setParams] = useSearchParams()
  const [editing, setEditing] = useState<Fixture | null>(null)
  const [slipOpen, setSlipOpen] = useState(false)

  const leagues = useLeagues()
  const leagueId = params.get('league') ?? leagues.data?.[0]?.id ?? null
  const whenParam = params.get('when') as FixtureWindow | null
  const when: FixtureWindow = whenParam && WINDOWS.includes(whenParam) ? whenParam : 'upcoming'
  const fixtures = useFixtures(leagueId, when)

  const existing = useMemo(
    () => (editing ? ticket.legs.find((l) => l.fixture.id === editing.id) : undefined),
    [editing, ticket.legs],
  )

  function setParam(key: string, value: string) {
    const next = new URLSearchParams(params)
    next.set(key, value)
    setParams(next, { replace: true })
  }

  function edit(leg: DraftLeg) {
    setSlipOpen(false)
    setEditing(leg.fixture)
  }

  function analyse() {
    setSlipOpen(false)
    navigate('/ticket/analysis')
  }

  const slip = <TicketSlip legs={ticket.legs} dispatch={dispatch} onEdit={edit} onAnalyse={analyse} />

  return (
    <div className="td-builder">
      <PageHeader eyebrow={c.eyebrow} title={c.title}>
        {c.intro}
      </PageHeader>

      <div className="row g-4">
        <div className="col-12 col-lg-7">
          {leagues.isPending ? (
            <PageSpinner />
          ) : leagues.isError ? (
            <LoadError onRetry={() => leagues.refetch()} />
          ) : (
            <>
              <div className="d-grid gap-2 mb-3">
                <LeaguePicker leagues={leagues.data} value={leagueId} onChange={(id) => setParam('league', id)} />
                <WindowToggle value={when} onChange={(w) => setParam('when', w)} />
              </div>

              {isFull && (
                <div className="alert alert-warning small" role="status">
                  {c.full(RULES.maxLegs)}
                </div>
              )}

              {fixtures.isPending ? (
                <PageSpinner />
              ) : fixtures.isError ? (
                <LoadError onRetry={() => fixtures.refetch()} />
              ) : fixtures.data.length === 0 ? (
                <p className="td-surface p-3 text-body-secondary mb-0">{c.noFixtures}</p>
              ) : (
                <FixtureList fixtures={fixtures.data} legs={ticket.legs} isFull={isFull} onSelect={setEditing} />
              )}
            </>
          )}
        </div>

        {/* Desktop: slip beside the fixtures */}
        <aside className="col-lg-5 d-none d-lg-block" aria-label={c.slip.title}>
          <div className="td-surface p-3 td-sticky">{slip}</div>
        </aside>
      </div>

      {/* Mobile: summary bar above the bottom nav, slip in a sheet */}
      <div className="td-slipbar d-lg-none">
        <button type="button" className="btn btn-light flex-grow-1 text-start" onClick={() => setSlipOpen(true)}>
          <span className="fw-bold td-num">{c.slip.count(ticket.legs.length, RULES.maxLegs)}</span>
          <span className="d-block small text-body-secondary">{c.slip.view}</span>
        </button>
        <button type="button" className="btn btn-primary" disabled={ticket.legs.length === 0} onClick={analyse}>
          {c.slip.analyse}
        </button>
      </div>

      <Offcanvas
        show={slipOpen}
        onHide={() => setSlipOpen(false)}
        placement="bottom"
        className="td-sheet d-lg-none"
        aria-labelledby="slip-sheet-title"
      >
        <Offcanvas.Header closeButton>
          <Offcanvas.Title as="h2" className="visually-hidden" id="slip-sheet-title">
            {c.slip.title}
          </Offcanvas.Title>
        </Offcanvas.Header>
        <Offcanvas.Body className="pt-0">{slip}</Offcanvas.Body>
      </Offcanvas>

      <MarketSheet
        fixture={editing}
        existing={existing}
        onSave={(leg) => dispatch({ type: 'add', leg })}
        onRemove={(matchId) => dispatch({ type: 'remove', matchId })}
        onClose={() => setEditing(null)}
      />
    </div>
  )
}

function LoadError({ onRetry }: { onRetry: () => void }) {
  return (
    <div className="alert alert-danger d-flex justify-content-between align-items-center gap-2" role="alert">
      <span className="small">{COPY.builder.loadError}</span>
      <button type="button" className="btn btn-sm btn-outline-danger flex-none" onClick={onRetry}>
        {COPY.builder.retry}
      </button>
    </div>
  )
}
