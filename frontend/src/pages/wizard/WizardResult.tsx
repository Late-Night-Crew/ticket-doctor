import { useNavigate } from 'react-router'
import { useWizardTicket } from '../../api/hooks'
import type { WizardRequest } from '../../api/types'
import { PageSpinner } from '../../components/AppLayout'
import { Disclaimer } from '../../components/Disclaimer'
import { LegCard } from '../../components/LegCard'
import { ProbabilityMeter } from '../../components/ProbabilityMeter'
import { HowItWorksCard } from '../../components/wizard/HowItWorksCard'
import { COPY } from '../../config/copy'
import { decimalOdds } from '../../lib/format'
import { useTicket } from '../../state/useTicket'

interface Props {
  request: WizardRequest
  onRegenerate: () => void
  onChangeAnswers: () => void
}

/** The generated ticket (FR-8.5–8.9). */
export function WizardResult({ request, onRegenerate, onChangeAnswers }: Props) {
  const r = COPY.wizard.result
  const w = COPY.wizard
  const navigate = useNavigate()
  const { ticket, dispatch } = useTicket()
  const result = useWizardTicket(request)

  const answers = [
    (request.path === 'beginner' ? w.risk.beginnerOptions : w.risk.options)[request.risk].label,
    w.focus.options[request.focus].label,
    w.games.label(request.legs),
    // The beginner path asks for the soonest games; the answer says which window it used.
    (() => {
      const when = result.data?.when ?? (request.when === 'soonest' ? null : request.when)
      return when ? w.when.options[when].label : w.when.soonestLabel
    })(),
  ]

  function openInAnalyser() {
    if (!result.data) return
    if (ticket.legs.length > 0 && !window.confirm(r.replaceConfirm(ticket.legs.length))) return
    dispatch({
      type: 'load',
      origin: 'wizard',
      legs: result.data.legs.map((l) => ({ fixture: l.fixture, market: l.market, odds: null })),
    })
    navigate('/ticket/analysis')
  }

  const legs = result.data?.legs ?? []
  const combined = legs.reduce((p, l) => p * l.estimate, 1)

  return (
    <section aria-labelledby="result-title">
      <h1 id="result-title" className="h3 mb-2">
        {r.title}
      </h1>
      <ul className="td-answers list-unstyled mb-3" aria-label="Your answers">
        {answers.map((a) => (
          <li key={a}>{a}</li>
        ))}
      </ul>

      <Disclaimer className="mb-3" />

      {result.isPending ? (
        <div role="status">
          <PageSpinner />
          <p className="text-center small text-body-secondary">{r.loading}</p>
        </div>
      ) : result.isError ? (
        <div className="alert alert-danger" role="alert">
          {r.error}{' '}
          <button type="button" className="btn btn-sm btn-link" onClick={() => result.refetch()}>
            {COPY.builder.retry}
          </button>
        </div>
      ) : (
        <>
          {result.data.shortfall_reason && legs.length > 0 && (
            <div className="alert alert-warning small" role="status">
              {result.data.shortfall_reason}
            </div>
          )}

          {legs.length === 0 ? (
            <div className="td-surface p-3 mb-3">
              <p className="fw-semibold mb-1">{r.none}</p>
              <p className="small text-body-secondary mb-0">{result.data.shortfall_reason ?? r.noneHint}</p>
            </div>
          ) : (
            <>
              <section className="td-surface p-3 mb-3" aria-labelledby="reward-title">
                <h2 id="reward-title" className="visually-hidden">
                  {r.rewardTitle}
                </h2>
                <div className="td-reward">
                  <ProbabilityMeter probability={combined} />
                  <div>
                    <div className="td-eyebrow mb-1">{r.ticketOdds}</div>
                    <div className="td-reward__odds td-num mb-2">{decimalOdds(1 / combined)}</div>
                    <p className="small text-body-secondary mb-0">{r.ticketOddsNote(decimalOdds(1 / combined))}</p>
                  </div>
                </div>
              </section>

              {request.path === 'beginner' && <HowItWorksCard legs={legs.length} combined={combined} />}

              <div className="d-grid gap-3 my-3">
                {legs.map((l) => (
                  <LegCard
                    key={l.fixture.id}
                    fixture={l.fixture}
                    marketKey={l.market}
                    estimate={l.estimate}
                    band={l.risk_band}
                    reason={l.reason}
                    sampleSize={l.sample_size}
                    windowN={l.window_n}
                    footer={
                      <span>
                        <span className="fw-semibold td-num">{r.fairOdds(decimalOdds(l.fair_odds))}</span>{' '}
                        <span className="text-body-secondary small">{r.fairOddsNote}</span>
                      </span>
                    }
                  />
                ))}
              </div>
            </>
          )}

          <div className="d-grid d-sm-flex gap-2">
            {legs.length > 0 && (
              <button type="button" className="btn btn-primary" onClick={openInAnalyser}>
                {r.open}
              </button>
            )}
            <button type="button" className="btn btn-outline-primary" onClick={onRegenerate} disabled={result.isFetching}>
              {r.regenerate}
            </button>
            <button type="button" className="btn btn-link text-body-secondary" onClick={onChangeAnswers}>
              {r.change}
            </button>
          </div>
        </>
      )}
    </section>
  )
}
