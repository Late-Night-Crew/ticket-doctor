import type { Fixture } from '../api/types'
import { Disclaimer } from '../components/Disclaimer'
import { FormString } from '../components/FormString'
import { GlossaryTerm } from '../components/GlossaryTerm'
import { IconSwap } from '../components/Icons'
import { LegCard } from '../components/LegCard'
import { PageHeader } from '../components/PageHeader'
import { ProbabilityMeter } from '../components/ProbabilityMeter'
import { RiskBadge } from '../components/RiskBadge'

const fixture: Fixture = {
  id: 'demo',
  league_id: 'epl',
  kickoff: new Date(Date.now() + 2 * 86400000).toISOString(),
  home: { id: 'a', name: 'Team A', form: 'W-D-L-W-W' },
  away: { id: 'b', name: 'Team B', form: 'L-L-D-W-L' },
}

/** Dev-only page showing F1 components with the SRS 4.4 worked example. */
export default function Styleguide() {
  return (
    <>
      <PageHeader eyebrow="Dev only" title="Component styleguide">
        F1 building blocks, using the worked example from SRS 4.4.
      </PageHeader>

      <section className="mb-4">
        <h2 className="h6 td-eyebrow">Risk badges</h2>
        <div className="d-flex flex-wrap gap-2">
          <RiskBadge band="low" />
          <RiskBadge band="medium" />
          <RiskBadge band="high" />
          <RiskBadge band="medium" limitedData />
        </div>
      </section>

      <section className="mb-4">
        <h2 className="h6 td-eyebrow">Form</h2>
        <FormString form="W-D-L-W-W" />
      </section>

      <section className="mb-4 d-grid gap-3">
        <h2 className="h6 td-eyebrow mb-0">Leg cards</h2>
        <LegCard
          fixture={fixture}
          marketKey="AWAY_WIN"
          estimate={0.25}
          band="high"
          reason="Team B won 2 of last 10 away games; Team A lost 3 of last 10 at home."
          sampleSize={10}
          windowN={10}
          odds={3.6}
          impliedProb={1 / 3.6}
          isValue={false}
          weakest
          footer={
            <div className="d-grid gap-2">
              <div className="fw-semibold d-flex align-items-center gap-1">
                <IconSwap width={16} height={16} /> Better-supported picks in this match
              </div>
              {[
                { label: 'Under 3.5 goals', est: '80%', why: '3 goals or fewer in 8 of 10 for both sides.' },
                { label: 'Team B to score', est: '75%', why: 'Team B scored in 8 of last 10 away; Team A kept 3 clean sheets in 10 at home.' },
              ].map((s) => (
                <div key={s.label} className="d-flex justify-content-between align-items-center gap-2">
                  <div>
                    <div className="fw-semibold">
                      {s.label} <span className="td-num">({s.est})</span>
                    </div>
                    <div className="small text-body-secondary">{s.why}</div>
                  </div>
                  <button type="button" className="btn btn-sm btn-outline-primary flex-none">
                    Swap
                  </button>
                </div>
              ))}
            </div>
          }
        />
        <LegCard
          fixture={fixture}
          marketKey="UNDER_3_5"
          estimate={0.8}
          band="low"
          reason="3 goals or fewer in 8 of Team A's last 10 home games and 8 of Team B's last 10 away."
          sampleSize={10}
          windowN={10}
          odds={1.4}
          impliedProb={1 / 1.4}
          isValue
        />
      </section>

      <section className="mb-4 td-surface p-3">
        <ProbabilityMeter probability={0.7 ** 5} before={0.25 * 0.7 ** 4} />
      </section>

      <section className="mb-4">
        <p>
          Tap a term: <GlossaryTerm id="btts">BTTS</GlossaryTerm>, <GlossaryTerm id="double-chance">double chance</GlossaryTerm>,{' '}
          <GlossaryTerm id="fair-odds">fair odds</GlossaryTerm>.
        </p>
        <Disclaimer />
      </section>

      <section className="mb-4 d-flex flex-wrap gap-2">
        <button className="btn btn-primary">Primary</button>
        <button className="btn btn-outline-primary">Outline</button>
        <button className="btn btn-light">Light</button>
        <button className="btn btn-link">Link</button>
      </section>
    </>
  )
}
