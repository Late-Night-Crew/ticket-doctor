import { useState } from 'react'
import { COPY } from '../../config/copy'
import { savedLink } from '../../lib/savedStore'

/** The private link in a read-only box, with a copy button. */
export function CopyLink({ id }: { id: string }) {
  const c = COPY.saved
  const [copied, setCopied] = useState(false)
  const link = savedLink(id)
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(link)
      setCopied(true)
      setTimeout(() => setCopied(false), 2500)
    } catch {
      /* clipboard blocked: the link is selectable in the box */
    }
  }
  return (
    <div className="input-group input-group-sm">
      <input
        className="form-control td-num"
        readOnly
        value={link}
        aria-label={c.copyLinkTitle}
        onFocus={(e) => e.currentTarget.select()}
      />
      <button type="button" className="btn btn-outline-primary" onClick={copy}>
        {copied ? c.copied : c.copyLink}
      </button>
      <span className="visually-hidden" role="status">
        {copied ? c.copied : ''}
      </span>
    </div>
  )
}
