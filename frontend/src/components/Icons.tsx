/** Small inline icon set (no icon font, to keep the bundle light). */
import type { SVGProps } from 'react'

type P = SVGProps<SVGSVGElement>
const base = { viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', strokeWidth: 2, strokeLinecap: 'round', strokeLinejoin: 'round', 'aria-hidden': true } as const

export const IconHome = (p: P) => (<svg {...base} {...p}><path d="M3 11 12 4l9 7" /><path d="M5 10v10h14V10" /></svg>)
export const IconTicket = (p: P) => (<svg {...base} {...p}><path d="M3 8a2 2 0 0 0 2-2h14a2 2 0 0 0 2 2v2a2 2 0 0 0 0 4v2a2 2 0 0 0-2 2H5a2 2 0 0 0-2-2v-2a2 2 0 0 0 0-4z" /><path d="M12 9v6M9 12h6" /></svg>)
export const IconWand = (p: P) => (<svg {...base} {...p}><path d="m4 20 11-11" /><path d="m14 4 1 2 2 1-2 1-1 2-1-2-2-1 2-1z" /><path d="m19 12 .5 1 1 .5-1 .5-.5 1-.5-1-1-.5 1-.5z" /></svg>)
export const IconChart = (p: P) => (<svg {...base} {...p}><path d="M4 20V10M10 20V4M16 20v-7M22 20H2" /></svg>)
export const IconInfo = (p: P) => (<svg {...base} {...p}><circle cx="12" cy="12" r="9" /><path d="M12 11v5M12 8h.01" /></svg>)
export const IconCheck = (p: P) => (<svg {...base} {...p}><path d="m5 12 5 5 9-10" /></svg>)
export const IconAlert = (p: P) => (<svg {...base} {...p}><path d="M12 4 2.5 20h19z" /><path d="M12 10v4M12 17h.01" /></svg>)
export const IconDot = (p: P) => (<svg {...base} {...p}><circle cx="12" cy="12" r="5" fill="currentColor" /></svg>)
export const IconSun = (p: P) => (<svg {...base} {...p}><circle cx="12" cy="12" r="4" /><path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4" /></svg>)
export const IconMoon = (p: P) => (<svg {...base} {...p}><path d="M20 14.5A8 8 0 0 1 9.5 4 8 8 0 1 0 20 14.5z" /></svg>)
export const IconSwap = (p: P) => (<svg {...base} {...p}><path d="M7 4 3 8l4 4M3 8h14M17 20l4-4-4-4M21 16H7" /></svg>)
export const IconLogo = (p: P) => (
  <svg viewBox="0 0 32 32" aria-hidden="true" {...p}>
    <path fill="currentColor" d="M4 7a3 3 0 0 1 3-3h18a3 3 0 0 1 3 3v5a4 4 0 0 0 0 8v5a3 3 0 0 1-3 3H7a3 3 0 0 1-3-3v-5a4 4 0 0 0 0-8z" />
    <path fill="var(--td-surface, #fff)" d="M14 10h4v4h4v4h-4v4h-4v-4h-4v-4h4z" />
  </svg>
)
