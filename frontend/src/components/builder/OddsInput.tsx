import { useId, useState } from 'react'
import { oddsToInput, parseOdds } from '../../lib/odds'

interface Props {
  value: number | null
  /** Called with a valid value (or null) when the user commits. */
  onCommit: (v: number | null) => void
  label: string
  help?: string
  size?: 'sm'
  hideLabel?: boolean
  placeholder?: string
  /** Report validity while typing, e.g. to disable a submit button. */
  onValidChange?: (valid: boolean, value: number | null) => void
}

export function OddsInput({ value, onCommit, label, help, size, hideLabel, placeholder = 'e.g. 1.85', onValidChange }: Props) {
  const id = useId()
  const [text, setText] = useState(oddsToInput(value))
  const [error, setError] = useState<string | null>(null)

  // Sync from outside changes, but never reformat what the user is typing.
  const [prevValue, setPrevValue] = useState(value)
  if (value !== prevValue) {
    setPrevValue(value)
    const r = parseOdds(text)
    if (!(r.ok && r.value === value)) setText(oddsToInput(value))
  }

  function change(next: string) {
    setText(next)
    const r = parseOdds(next)
    if (r.ok) setError(null)
    onValidChange?.(r.ok, r.ok ? r.value : null)
  }

  function commit() {
    const r = parseOdds(text)
    if (!r.ok) {
      setError(r.error)
      return
    }
    setError(null)
    setText(oddsToInput(r.value))
    if (r.value !== value) onCommit(r.value)
  }

  return (
    <div>
      <label htmlFor={id} className={hideLabel ? 'visually-hidden' : 'form-label fw-semibold small mb-1'}>
        {label}
      </label>
      <input
        id={id}
        type="text"
        inputMode="decimal"
        autoComplete="off"
        className={`form-control td-num ${size === 'sm' ? 'form-control-sm' : ''} ${error ? 'is-invalid' : ''}`}
        placeholder={placeholder}
        value={text}
        onChange={(e) => change(e.target.value)}
        onBlur={commit}
        onKeyDown={(e) => e.key === 'Enter' && commit()}
        aria-invalid={!!error}
        aria-describedby={help || error ? `${id}-help` : undefined}
      />
      {(error || help) && (
        <div id={`${id}-help`} className={error ? 'invalid-feedback d-block' : 'form-text'}>
          {error ?? help}
        </div>
      )}
    </div>
  )
}
