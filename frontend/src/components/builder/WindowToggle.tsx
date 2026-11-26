import type { FixtureWindow } from '../../api/types'
import { COPY } from '../../config/copy'

const OPTIONS: FixtureWindow[] = ['today', 'weekend', 'upcoming']

export function WindowToggle({ value, onChange }: { value: FixtureWindow; onChange: (w: FixtureWindow) => void }) {
  return (
    <div className="td-segmented" role="group" aria-label="When">
      {OPTIONS.map((w) => (
        <button key={w} type="button" aria-pressed={w === value} onClick={() => onChange(w)}>
          {COPY.builder.windows[w]}
        </button>
      ))}
    </div>
  )
}
