import { useState } from 'react'
import Offcanvas from 'react-bootstrap/Offcanvas'
import type { Fixture, MarketKey } from '../../api/types'
import { COPY } from '../../config/copy'
import { MARKETS, type MarketGroup } from '../../config/markets'
import { kickoffLabel } from '../../lib/format'
import type { DraftLeg } from '../../state/ticketReducer'
import { FormString } from '../FormString'
import { GlossaryTerm } from '../GlossaryTerm'
import { OddsInput } from './OddsInput'

interface Props {
  fixture: Fixture | null
  existing: DraftLeg | undefined
  onSave: (leg: DraftLeg) => void
  onRemove: (matchId: string) => void
  onClose: () => void
}

const GROUPS: { group: MarketGroup; title: string; glossary: [string, string][] }[] = [
  { group: 'wins', title: COPY.builder.sheet.result, glossary: [['win', 'Win'], ['draw', 'Draw'], ['double-chance', 'Double chance']] },
  { group: 'goals', title: COPY.builder.sheet.goals, glossary: [['btts', 'BTTS'], ['over-under', 'Over/under'], ['team-to-score', 'Team to score']] },
]

/** Bottom sheet for choosing a market and optional odds for one match (FR-3.1–3.3). */
export function MarketSheet({ fixture, existing, onSave, onRemove, onClose }: Props) {
  return (
    <Offcanvas show={!!fixture} onHide={onClose} placement="bottom" className="td-sheet" aria-labelledby="market-sheet-title">
      {fixture && (
        // Keyed so the form resets for each match.
        <SheetBody key={fixture.id} fixture={fixture} existing={existing} onSave={onSave} onRemove={onRemove} onClose={onClose} />
      )}
    </Offcanvas>
  )
}

function SheetBody({ fixture, existing, onSave, onRemove, onClose }: Props & { fixture: Fixture }) {
  const [picked, setPicked] = useState<MarketKey | null>(existing?.market ?? null)
  const [odds, setOdds] = useState<number | null>(existing?.odds ?? null)
  const [oddsValid, setOddsValid] = useState(true)
  const { home, away } = fixture
  const s = COPY.builder.sheet

  function save() {
    if (!picked || !oddsValid) return
    onSave({ fixture, market: picked, odds })
    onClose()
  }

  // Odds belong to a market: switching market clears odds carried over from the saved leg.
  function pick(key: MarketKey) {
    if (key !== picked && existing && odds === existing.odds) setOdds(null)
    setPicked(key)
  }

  return (
    <>
      <Offcanvas.Header closeButton>
        <div>
          <Offcanvas.Title as="h2" id="market-sheet-title" className="h5 mb-0">
            {home.name} v {away.name}
          </Offcanvas.Title>
          <div className="small text-body-secondary">{kickoffLabel(fixture.kickoff)}</div>
          {(home.form || away.form) && (
            <dl className="td-sheet__form small mb-0 mt-1">
              <div>
                <dt>{home.name}</dt>
                <dd>
                  <FormString form={home.form} />
                </dd>
              </div>
              <div>
                <dt>{away.name}</dt>
                <dd>
                  <FormString form={away.form} />
                </dd>
              </div>
            </dl>
          )}
        </div>
      </Offcanvas.Header>
      <Offcanvas.Body className="pt-0">
        {GROUPS.map(({ group, title, glossary }) => (
          <fieldset key={group} className="mb-3">
            <legend className="td-eyebrow mb-2 d-flex flex-wrap gap-2 align-items-baseline">
              <span>{title}</span>
              <span className="text-transform-none fw-normal small">
                {glossary.map(([id, label], i) => (
                  <span key={id}>
                    <GlossaryTerm id={id}>{label}</GlossaryTerm>
                    {i < glossary.length - 1 && ', '}
                  </span>
                ))}
              </span>
            </legend>
            <div className="td-market-grid">
              {MARKETS.filter((m) => m.group === group).map((m) => (
                <button
                  key={m.key}
                  type="button"
                  className="td-market"
                  aria-pressed={picked === m.key}
                  onClick={() => pick(m.key)}
                >
                  <span className="td-market__plain">{m.plain(home.name, away.name)}</span>
                  <span className="visually-hidden">, </span>
                  <span className="td-market__short">{m.short}</span>
                </button>
              ))}
            </div>
          </fieldset>
        ))}

        <div className="mb-3">
          <OddsInput
            key={picked ?? 'none'}
            value={odds}
            onCommit={setOdds}
            onValidChange={(ok, v) => {
              setOddsValid(ok)
              if (ok) setOdds(v)
            }}
            label={s.oddsLabel}
            help={s.oddsHelp}
          />
        </div>
      </Offcanvas.Body>
      <div className="td-sheet__footer">
        {existing && (
          <button
            type="button"
            className="btn btn-link text-danger px-0 me-auto"
            onClick={() => {
              onRemove(fixture.id)
              onClose()
            }}
          >
            {s.remove}
          </button>
        )}
        <button type="button" className="btn btn-primary flex-grow-1 flex-sm-grow-0" disabled={!picked || !oddsValid} onClick={save}>
          {!picked ? s.pickFirst : existing ? s.update : s.add}
        </button>
      </div>
    </>
  )
}
