/** 0.256 → "26%" */
export function pct(p: number, digits = 0): string {
  return `${(p * 100).toFixed(digits)}%`
}

/** 0.17 → 6, for "about 1 in 6" (RG-2). */
export function oneIn(p: number): number {
  if (p <= 0) return Infinity
  return Math.max(1, Math.round(1 / p))
}

export function decimalOdds(o: number): string {
  return o.toFixed(2)
}

export function kickoffLabel(iso: string, now = new Date()): string {
  const d = new Date(iso)
  const time = d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' })
  const sameDay = d.toDateString() === now.toDateString()
  if (sameDay) return `Today ${time}`
  const day = d.toLocaleDateString(undefined, { weekday: 'short', day: 'numeric', month: 'short' })
  return `${day} ${time}`
}

/** "Today", "Tomorrow" or "Sat 29 Nov", for grouping fixtures by day. */
export function dayLabel(iso: string, now = new Date()): string {
  const d = new Date(iso)
  const start = (x: Date) => new Date(x.getFullYear(), x.getMonth(), x.getDate()).getTime()
  const diff = Math.round((start(d) - start(now)) / 86400000)
  if (diff === 0) return 'Today'
  if (diff === 1) return 'Tomorrow'
  return d.toLocaleDateString(undefined, { weekday: 'short', day: 'numeric', month: 'short' })
}

export function timeLabel(iso: string): string {
  return new Date(iso).toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' })
}
