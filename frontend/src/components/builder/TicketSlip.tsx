import { COPY } from '../../config/copy'
import { market } from '../../config/markets'
import { RULES } from '../../config/rules'
import type { DraftLeg, TicketAction } from '../../state/ticketReducer'
import { OddsInput } from './OddsInput'

interface Props {
  legs: DraftLeg[]
  dispatch: React.Dispatch<TicketAction>
  onEdit: (leg: DraftLeg) => void
  onAnalyse: () => void
}

/** The legs on the ticket, with odds editing and the Analyse action. */
export function TicketSlip({ legs, dispatch, onEdit, onAnalyse }: Props) {
  const s = COPY.builder.slip

  function clear() {
    if (window.confirm(s.confirmClear)) dispatch({ type: 'clear' })
  }

  return (
    <div className="td-slip">
      <div className="d-flex justify-content-between align-items-baseline mb-2">
        <h2 className="h6 mb-0">{s.title}</h2>
        <span className="small text-body-secondary td-num">{s.count(legs.length, RULES.maxLegs)}</span>
      </div>

      {legs.length === 0 ? (
        <p className="small text-body-secondary mb-0">{s.empty}</p>
      ) : (
        <ol className="td-slip__legs list-unstyled mb-3">
          {legs.map((l) => {
            const pick = market(l.market).plain(l.fixture.home.name, l.fixture.away.name)
            return (
              <li key={l.fixture.id} className="td-slip__leg">
                <div className="min-w-0 flex-grow-1">
                  <div className="small text-body-secondary text-truncate">
                    {l.fixture.home.name} v {l.fixture.away.name}
                  </div>
                  <button type="button" className="td-slip__pick" onClick={() => onEdit(l)} aria-label={s.editLeg(pick)}>
                    {pick}
                  </button>
                </div>
                <div className="td-slip__odds">
                  <OddsInput
                    value={l.odds}
                    onCommit={(v) => dispatch({ type: 'setOdds', matchId: l.fixture.id, odds: v })}
                    label={`${COPY.builder.sheet.oddsLabel}: ${pick}`}
                    hideLabel
                    size="sm"
                    placeholder={s.oddsPlaceholder}
                  />
                </div>
                <button
                  type="button"
                  className="btn-close flex-none"
                  aria-label={s.removeLeg(pick)}
                  onClick={() => dispatch({ type: 'remove', matchId: l.fixture.id })}
                />
              </li>
            )
          })}
        </ol>
      )}

      <div className="d-grid gap-2">
        <button type="button" className="btn btn-primary" disabled={legs.length === 0} onClick={onAnalyse}>
          {s.analyse}
        </button>
        {legs.length > 0 && (
          <button type="button" className="btn btn-link btn-sm text-body-secondary" onClick={clear}>
            {s.clear}
          </button>
        )}
      </div>
    </div>
  )
}
