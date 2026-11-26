import { COPY } from '../../config/copy'
import { oneIn, pct } from '../../lib/format'
import { GlossaryTerm } from '../GlossaryTerm'

/** Beginner explainer shown with a generated ticket (FR-8.7). */
export function HowItWorksCard({ legs, combined }: { legs: number; combined: number }) {
  const h = COPY.wizard.how
  return (
    <section className="td-how" aria-labelledby="how-title">
      <h2 id="how-title" className="h6 mb-2">
        {h.title}
      </h2>
      <ul className="mb-0 small">
        <li>{h.allLegs(legs)}</li>
        <li>{h.combined(pct(combined, 1), oneIn(combined))}</li>
        <li>
          {h.fair} <GlossaryTerm id="fair-odds">More about fair odds</GlossaryTerm>
        </li>
      </ul>
    </section>
  )
}
