import { useEffect, useRef, useState } from 'react'
import { Link, useSearchParams } from 'react-router'
import type { WizardPath } from '../../api/types'
import { GamesPicker } from '../../components/wizard/GamesPicker'
import { StepChoice, type Choice } from '../../components/wizard/StepChoice'
import { COPY } from '../../config/copy'
import { answered, initial, legRange, stepsFor, type Answers } from './wizardSteps'
import { WizardResult } from './WizardResult'

const choices = <T extends string>(opts: Record<T, { label: string; body: string }>): Choice<T>[] =>
  (Object.keys(opts) as T[]).map((value) => ({ value, ...opts[value] }))

export default function WizardFlow({ path }: { path: WizardPath }) {
  const w = COPY.wizard
  const [params, setParams] = useSearchParams()
  const [a, setA] = useState<Answers>(() => initial(path))
  const [seed, setSeed] = useState(0)

  const steps = stepsFor(path, a.risk)
  const firstOpen = steps.findIndex((s) => !answered(s, a))
  const requested = Math.max(0, Number(params.get('step')) || 0)
  // Never show a step whose earlier questions are unanswered (e.g. after a refresh).
  const index = Math.min(requested, firstOpen === -1 ? steps.length - 1 : firstOpen, steps.length - 1)
  const step = steps[index]
  const questions = steps.filter((s) => s !== 'result')

  // Each step is a new screen: start at the top and put focus on its heading.
  const container = useRef<HTMLDivElement>(null)
  const firstRender = useRef(true)
  useEffect(() => {
    if (firstRender.current) {
      firstRender.current = false
      return
    }
    window.scrollTo(0, 0)
    const h1 = container.current?.querySelector('h1')
    if (h1) {
      h1.setAttribute('tabindex', '-1')
      h1.focus({ preventScroll: true })
    }
  }, [step])

  function goTo(i: number) {
    setParams({ step: String(i) })
  }

  function answer(patch: Partial<Answers>) {
    const next = { ...a, ...patch }
    setA(next)
    const nextSteps = stepsFor(path, next.risk)
    const here = nextSteps.indexOf(step)
    // If this step dropped out (Bold → Safe), the next question now sits at this index.
    goTo(here === -1 ? index : here + 1)
  }

  function changeAnswers() {
    goTo(path === 'beginner' ? 1 : 0)
  }

  return (
    <div className="td-wizard" ref={container}>
      {step !== 'result' && (
        <div className="d-flex justify-content-between align-items-center mb-3">
          {index === 0 || (path === 'beginner' && index === 1) ? (
            <Link to="/wizard" className="btn btn-sm btn-link px-0">
              ← {w.back}
            </Link>
          ) : (
            <button type="button" className="btn btn-sm btn-link px-0" onClick={() => goTo(index - 1)}>
              ← {w.back}
            </button>
          )}
          <span className="small text-body-secondary td-num" aria-live="polite">
            {w.stepOf(questions.indexOf(step) + 1, questions.length)}
          </span>
        </div>
      )}

      {step !== 'result' && (
        <div className="td-progress mb-4" aria-hidden="true">
          {questions.map((q, i) => (
            <span key={q} className={i <= questions.indexOf(step) ? 'is-done' : ''} />
          ))}
        </div>
      )}

      {step === 'stake' && (
        <section className="td-surface p-3 p-md-4" aria-labelledby="stake-title">
          <h1 id="stake-title" className="h3 mb-3">
            {w.stake.title}
          </h1>
          <ul className="td-points mb-4">
            {w.stake.points.map((p) => (
              <li key={p}>{p}</li>
            ))}
          </ul>
          <button type="button" className="btn btn-primary w-100" onClick={() => answer({ stakeOk: true })}>
            {w.stake.ok}
          </button>
        </section>
      )}

      {step === 'risk' && (
        <StepChoice
          title={path === 'beginner' ? w.risk.beginnerTitle : w.risk.title}
          intro={w.risk.intro}
          choices={choices(path === 'beginner' ? w.risk.beginnerOptions : w.risk.options)}
          value={a.risk}
          onSelect={(risk) => answer({ risk, boldOk: path === 'beginner' ? false : true })}
        />
      )}

      {step === 'bold' && (
        <section className="td-surface p-3 p-md-4 td-bold" aria-labelledby="bold-title">
          <h1 id="bold-title" className="h3 mb-3">
            {w.bold.title}
          </h1>
          <ul className="td-points mb-4">
            {w.bold.points.map((p) => (
              <li key={p}>{p}</li>
            ))}
          </ul>
          <div className="d-grid gap-2">
            <button type="button" className="btn btn-primary" onClick={() => answer({ risk: 'safe', boldOk: false })}>
              {w.bold.safer}
            </button>
            <button type="button" className="btn btn-outline-secondary" onClick={() => answer({ boldOk: true })}>
              {w.bold.keep}
            </button>
          </div>
        </section>
      )}

      {step === 'focus' && (
        <StepChoice title={w.focus.title} choices={choices(w.focus.options)} value={a.focus} onSelect={(focus) => answer({ focus })} />
      )}

      {step === 'games' && (
        <GamesPicker
          title={path === 'beginner' ? w.games.beginnerTitle : w.games.title}
          help={path === 'beginner' ? w.games.beginnerHelp : w.games.help}
          note={path === 'guided' ? w.games.tradeoff : undefined}
          {...legRange(path)}
          value={a.legs}
          label={w.games.label}
          onSelect={(legs) => answer({ legs })}
        />
      )}

      {step === 'when' && (
        <StepChoice title={w.when.title} choices={choices(w.when.options)} value={a.when} onSelect={(when) => answer({ when })} />
      )}

      {step !== 'result' && step !== 'stake' && step !== 'bold' && answered(step, a) && (
        <button type="button" className="btn btn-primary w-100 mt-3" onClick={() => goTo(index + 1)}>
          {w.continue}
        </button>
      )}

      {step === 'result' && a.risk && a.focus && a.legs && a.when && (
        <WizardResult
          request={{ path, risk: a.risk, focus: a.focus, legs: a.legs, when: a.when, seed }}
          onRegenerate={() => setSeed((s) => s + 1)}
          onChangeAnswers={changeAnswers}
        />
      )}
    </div>
  )
}
