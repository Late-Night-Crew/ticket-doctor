import { useId } from 'react'

export interface Choice<T extends string> {
  value: T
  label: string
  body: string
}

interface Props<T extends string> {
  title: string
  intro?: string
  choices: Choice<T>[]
  value: T | null
  onSelect: (v: T) => void
}

/** A question answered with one tap (FR-8.2). */
export function StepChoice<T extends string>({ title, intro, choices, value, onSelect }: Props<T>) {
  const id = useId()
  return (
    <fieldset aria-labelledby={id}>
      <legend id={id} className="mb-3">
        <h1 className="h3 mb-0">{title}</h1>
      </legend>
      {intro && <p className="text-body-secondary mt-n2 mb-3">{intro}</p>}
      <div className="d-grid gap-2">
        {choices.map((c) => (
          <button
            key={c.value}
            type="button"
            className="td-choice"
            aria-pressed={c.value === value}
            onClick={() => onSelect(c.value)}
          >
            <span className="td-choice__label">{c.label}</span>
            <span className="td-choice__body">{c.body}</span>
          </button>
        ))}
      </div>
    </fieldset>
  )
}
