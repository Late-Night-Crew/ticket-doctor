import { COPY } from '../config/copy'
import { IconInfo } from './Icons'

/** Shown on every analysis page (RG-1). */
export function Disclaimer({ className = '' }: { className?: string }) {
  return (
    <p className={`td-disclaimer mb-0 ${className}`} role="note">
      <IconInfo />
      <span>{COPY.disclaimer}</span>
    </p>
  )
}
