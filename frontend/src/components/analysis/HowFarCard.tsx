import { useState } from 'react'
import type { AnalysedLeg } from '../../api/types'
import { COPY } from '../../config/copy'
import { market } from '../../config/markets'
import { kickoffLabel, oneIn, pct } from '../../lib/format'
import { chanceToReachLast, expectedGamesSurvived, likeliestEnd, riskiestIsEarly, survival } from '../../lib/survival'

const legPick = (l: AnalysedLeg) => market(l.market).plain(l.fixture.home.name, l.fixture.away.name)

/**
 * "How far will it get?" for cash-out chasers: the chance the ticket is still
 * alive after each game, in kick-off order. It never shows money: cash-out
 * offers are the bookmaker's, and we don't suggest whether to take one (RG-6).
 */
export function HowFarCard({ legs }: { legs: AnalysedLeg[] }) {
  const c = COPY.howFar
  const steps = survival(legs.map((l) => ({ ...l, kickoff: l.fixture.kickoff })))
  const n = steps.length
  const [left, setLeft] = useState(() => Math.max(1, Math.min(4, Math.floor(n / 2))))
  if (n < 3) return null

  const remaining = Math.min(left, n - 1)
  const reach = chanceToReachLast(steps, remaining)
  const end = likeliestEnd(steps)
  const riskiest = steps.reduce((w, s) => (s.leg.estimate < w.leg.estimate ? s : w))

  return (
    <section className="td-surface p-3 mt-4" aria-labelledby="howfar-title">
      <h2 id="howfar-title" className="h5 mb-1">
        {c.title}
      </h2>
      <p className="small text-body-secondary">{c.lead}</p>

      <ol className="list-unstyled td-howfar mb-3">
        {steps.map((s) => (
          <li key={s.leg.id} className={s.position > n - remaining ? 'td-howfar__row td-howfar__row--late' : 'td-howfar__row'}>
            <span className="td-howfar__pos td-num" aria-hidden="true">
              {s.position}
            </span>
            <div className="td-howfar__body">
              <div className="small text-body-secondary">
                {s.leg.fixture.home.name} v {s.leg.fixture.away.name} · {kickoffLabel(s.leg.fixture.kickoff)}
                {s.sameTimeAsPrevious && ` · ${c.sameTime}`}
              </div>
              <div className="fw-semibold">
                {legPick(s.leg)} <span className="text-body-secondary fw-normal td-num">({pct(s.leg.estimate)})</span>
                {end?.leg.id === s.leg.id && <span className="badge text-bg-light border ms-2">{c.likeliestEnd}</span>}
              </div>
              <div className="td-howfar__bar" aria-hidden="true">
                <span style={{ width: `${s.alive * 100}%` }} />
              </div>
            </div>
            <div className="td-howfar__alive text-end">
              <span className="fw-bold td-num">{pct(s.alive)}</span>
              <span className="d-block small text-body-secondary">{c.stillAlive}</span>
            </div>
          </li>
        ))}
      </ol>

      <div className="td-howfar__cashout p-3 mb-3">
        <label htmlFor="howfar-left" className="form-label small fw-semibold mb-1">
          {c.cashOutLabel}
        </label>
        <select
          id="howfar-left"
          className="form-select form-select-sm w-auto mb-2"
          value={remaining}
          onChange={(e) => setLeft(Number(e.target.value))}
        >
          {Array.from({ length: n - 1 }, (_, i) => i + 1).map((k) => (
            <option key={k} value={k}>
              {c.gamesLeft(k)}
            </option>
          ))}
        </select>
        <p className="mb-0" role="status">
          {c.reach(n - remaining, remaining)} <strong className="td-num">{pct(reach)}</strong>
          <span className="text-body-secondary"> (about 1 in {oneIn(reach)})</span>
        </p>
      </div>

      <p className="small mb-2">{c.average(expectedGamesSurvived(steps).toFixed(1), n)}</p>
      {riskiestIsEarly(steps) && <p className="small mb-2">{c.riskyEarly(legPick(riskiest.leg))}</p>}
      <p className="small text-body-secondary mb-0">{c.cashOutNote}</p>
    </section>
  )
}
