import type { Fixture } from '../../api/types'
import { COPY } from '../../config/copy'
import { market } from '../../config/markets'
import { dayLabel, timeLabel } from '../../lib/format'
import type { DraftLeg } from '../../state/ticketReducer'
import { FormString } from '../FormString'
import { IconCheck } from '../Icons'

interface Props {
  fixtures: Fixture[]
  legs: DraftLeg[]
  isFull: boolean
  onSelect: (f: Fixture) => void
}

export function FixtureList({ fixtures, legs, isFull, onSelect }: Props) {
  const legByMatch = new Map(legs.map((l) => [l.fixture.id, l]))
  const groups = new Map<string, Fixture[]>()
  for (const f of fixtures) {
    const key = dayLabel(f.kickoff)
    groups.set(key, [...(groups.get(key) ?? []), f])
  }

  return (
    <div className="d-grid gap-3">
      {[...groups].map(([day, list]) => (
        <section key={day} aria-label={day}>
          <h2 className="td-eyebrow mb-2">{day}</h2>
          <ul className="td-fixtures list-unstyled mb-0">
            {list.map((f) => {
              const leg = legByMatch.get(f.id)
              const disabled = isFull && !leg
              return (
                <li key={f.id}>
                  <button
                    type="button"
                    className={`td-fixture ${leg ? 'is-on-ticket' : ''}`}
                    onClick={() => onSelect(f)}
                    disabled={disabled}
                  >
                    <span className="td-fixture__time td-num">{timeLabel(f.kickoff)}</span>
                    <span className="visually-hidden">, </span>
                    <span className="td-fixture__teams">
                      <span className="td-fixture__team">
                        <span className="text-truncate">{f.home.name}</span>
                        <span aria-hidden="true">
                          <FormString form={f.home.form} />
                        </span>
                      </span>
                      <span className="visually-hidden"> v </span>
                      <span className="td-fixture__team">
                        <span className="text-truncate">{f.away.name}</span>
                        <span aria-hidden="true">
                          <FormString form={f.away.form} />
                        </span>
                      </span>
                    </span>
                    <span className="visually-hidden">, </span>
                    <span className="td-fixture__action">
                      {leg ? (
                        <span className="td-chip td-chip--on">
                          <IconCheck width={14} height={14} />
                          <span className="visually-hidden">on ticket: </span>
                          {market(leg.market).short}
                        </span>
                      ) : (
                        <span className="td-chip">+ {COPY.builder.add}</span>
                      )}
                    </span>
                  </button>
                </li>
              )
            })}
          </ul>
        </section>
      ))}
    </div>
  )
}
