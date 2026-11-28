import { useState } from 'react'
import type { CalibrationBin } from '../../api/admin'
import { pct } from '../../lib/format'

const W = 320
const H = 300
const PAD = { top: 12, right: 12, bottom: 40, left: 44 }
const PW = W - PAD.left - PAD.right
const PH = H - PAD.top - PAD.bottom
const x = (v: number) => PAD.left + v * PW
const y = (v: number) => PAD.top + (1 - v) * PH
const TICKS = [0, 0.25, 0.5, 0.75, 1]

/**
 * Reliability chart: each dot is a group of estimates, placed at what we said
 * (x) and what happened (y). On the diagonal means the estimates were right;
 * below it means we were too confident, above it too cautious. One series, so
 * no legend; every value is also in the table under the chart.
 */
export function ReliabilityChart({ bins, label }: { bins: CalibrationBin[]; label: string }) {
  const points = bins.filter((b) => b.n > 0)
  const [hover, setHover] = useState<CalibrationBin | null>(null)

  return (
    <figure className="td-relchart mb-0">
      <svg viewBox={`0 0 ${W} ${H}`} role="img" aria-label={label} className="w-100 h-auto">
        {/* Recessive grid and axes: solid hairlines */}
        {TICKS.map((t) => (
          <g key={t}>
            <line x1={x(0)} x2={x(1)} y1={y(t)} y2={y(t)} className="td-relchart__grid" />
            <line x1={x(t)} x2={x(t)} y1={y(0)} y2={y(1)} className="td-relchart__grid" />
            <text x={PAD.left - 6} y={y(t)} className="td-relchart__tick" textAnchor="end" dominantBaseline="middle">
              {pct(t)}
            </text>
            <text x={x(t)} y={y(0) + 14} className="td-relchart__tick" textAnchor="middle">
              {pct(t)}
            </text>
          </g>
        ))}
        <text x={x(0.5)} y={H - 4} className="td-relchart__axis" textAnchor="middle">
          What we said
        </text>
        <text transform={`translate(11 ${y(0.5)}) rotate(-90)`} className="td-relchart__axis" textAnchor="middle">
          What happened
        </text>

        {/* Perfect calibration (explained in the card's subtitle) */}
        <line x1={x(0)} y1={y(0)} x2={x(1)} y2={y(1)} className="td-relchart__ideal" />

        {points.length > 1 && (
          <polyline points={points.map((b) => `${x(b.predicted)},${y(b.actual)}`).join(' ')} className="td-relchart__line" />
        )}
        {points.map((b) => (
          <g key={b.from}>
            <circle cx={x(b.predicted)} cy={y(b.actual)} r={5} className="td-relchart__dot" />
            {/* 24px hit area, focusable, same readout as hover */}
            <circle
              cx={x(b.predicted)}
              cy={y(b.actual)}
              r={12}
              className="td-relchart__hit"
              tabIndex={0}
              aria-label={`We said ${pct(b.from)}–${pct(b.to)} (average ${pct(b.predicted, 1)}): happened ${pct(b.actual, 1)} of ${b.n}`}
              onPointerEnter={() => setHover(b)}
              onPointerLeave={() => setHover(null)}
              onFocus={() => setHover(b)}
              onBlur={() => setHover(null)}
            />
          </g>
        ))}
      </svg>
      {hover && (
        <div
          className="td-relchart__tip"
          style={{ left: `${(x(hover.predicted) / W) * 100}%`, top: `${(y(hover.actual) / H) * 100}%` }}
          role="status"
        >
          <strong>{pct(hover.actual, 1)}</strong> happened
          <div>
            we said {pct(hover.predicted, 1)} · {hover.n.toLocaleString()} picks
          </div>
        </div>
      )}
    </figure>
  )
}
