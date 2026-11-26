import { useId } from 'react'

interface Props {
  title: string
  help: string
  note?: string
  min: number
  max: number
  value: number | null
  label: (n: number) => string
  onSelect: (n: number) => void
}

/** Pick an exact number of games with one tap. */
export function GamesPicker({ title, help, note, min, max, value, label, onSelect }: Props) {
  const id = useId()
  const options = Array.from({ length: max - min + 1 }, (_, i) => min + i)
  return (
    <fieldset aria-labelledby={id} aria-describedby={`${id}-help`}>
      <legend id={id} className="mb-2">
        <h1 className="h3 mb-0">{title}</h1>
      </legend>
      <p id={`${id}-help`} className="text-body-secondary mb-3">
        {help}
      </p>
      <div className="td-games" style={{ ['--td-games-cols' as string]: Math.min(options.length, 5) }}>
        {options.map((n) => (
          <button key={n} type="button" className="td-games__n td-num" aria-pressed={n === value} aria-label={label(n)} onClick={() => onSelect(n)}>
            {n}
          </button>
        ))}
      </div>
      {note && <p className="small text-body-secondary mt-3 mb-0">{note}</p>}
    </fieldset>
  )
}
