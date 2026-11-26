import type { League } from '../../api/types'

interface Props {
  leagues: League[]
  value: string | null
  onChange: (id: string) => void
}

export function LeaguePicker({ leagues, value, onChange }: Props) {
  return (
    <div className="td-pills" role="group" aria-label="League">
      {leagues.map((l) => (
        <button
          key={l.id}
          type="button"
          className="td-pill"
          aria-pressed={l.id === value}
          onClick={() => onChange(l.id)}
        >
          {l.name}
        </button>
      ))}
    </div>
  )
}
