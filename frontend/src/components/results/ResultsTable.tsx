import type { League, ResultsRow } from '../../api/types'
import { COPY } from '../../config/copy'
import { market } from '../../config/markets'
import { GlossaryTerm } from '../GlossaryTerm'
import { HitCell } from './WonLost'

/** A table on wide screens; each row becomes a card on phones (CSS only, one DOM). */
export function ResultsTable({ rows, leagues }: { rows: ResultsRow[]; leagues: League[] }) {
  const c = COPY.results
  const leagueName = (id: string) => leagues.find((l) => l.id === id)?.name ?? id
  return (
    <table className="td-results-table">
      <caption className="visually-hidden">{c.tableCaption}</caption>
      <thead>
        <tr>
          <th scope="col">{c.marketLabel}</th>
          <th scope="col">{c.league}</th>
          <th scope="col">{c.original}</th>
          <th scope="col">{c.suggested}</th>
        </tr>
      </thead>
      <tbody>
        {rows.map((r) => {
          const def = market(r.market)
          return (
            <tr key={`${r.league_id}-${r.market}`}>
              <th scope="row" data-label={c.marketLabel}>
                <GlossaryTerm id={def.glossary}>{def.short}</GlossaryTerm>
              </th>
              <td data-label={c.league}>{leagueName(r.league_id)}</td>
              <td data-label={c.original}>
                <HitCell count={r.original} />
              </td>
              <td data-label={c.suggested}>
                <HitCell count={r.suggested} />
              </td>
            </tr>
          )
        })}
      </tbody>
    </table>
  )
}
