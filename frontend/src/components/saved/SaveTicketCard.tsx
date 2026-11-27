import { useState } from 'react'
import { Link } from 'react-router'
import { useSaveTicket } from '../../api/hooks'
import { COPY } from '../../config/copy'
import { rememberSaved, savedFor } from '../../lib/savedStore'
import { CopyLink } from './CopyLink'

/**
 * Save the analysed ticket (FR-9.1). Saving takes a snapshot of the last
 * analysis, so it waits until any update has finished. Saving again replaces
 * the snapshot and keeps the link.
 */
export function SaveTicketCard({ ticketId, updating, changed }: { ticketId: string; updating: boolean; changed: string }) {
  const c = COPY.saved
  const save = useSaveTicket()
  // What was last saved from this ticket (from this device's list), and the
  // ticket contents at that moment, to know whether there is anything new.
  const [savedId, setSavedId] = useState(() => savedFor(ticketId)?.id ?? null)
  const [savedVersion, setSavedVersion] = useState<string | null>(null)

  const upToDate = savedId !== null && savedVersion === changed
  const doSave = () =>
    save.mutate(ticketId, {
      onSuccess: (t) => {
        rememberSaved({ id: t.id, ticketId, savedAt: new Date().toISOString() })
        setSavedId(t.id)
        setSavedVersion(changed)
      },
    })

  return (
    <section className="td-surface p-3 mt-3" aria-labelledby="save-title">
      <h2 id="save-title" className="h6 mb-1">
        {c.saveTitle}
      </h2>
      {savedId ? (
        <>
          <p className="small text-body-secondary mb-2">{c.savedNote}</p>
          <CopyLink id={savedId} />
          <div className="d-flex flex-wrap gap-2 mt-2">
            <Link to={`/t/${savedId}`} className="btn btn-sm btn-primary">
              {c.open}
            </Link>
            {!upToDate && (
              <button type="button" className="btn btn-sm btn-outline-primary" onClick={doSave} disabled={updating || save.isPending}>
                {save.isPending ? c.saving : c.resave}
              </button>
            )}
          </div>
        </>
      ) : (
        <>
          <p className="small text-body-secondary mb-2">{c.saveBody}</p>
          <button type="button" className="btn btn-primary w-100" onClick={doSave} disabled={updating || save.isPending}>
            {save.isPending ? c.saving : c.save}
          </button>
        </>
      )}
      {updating && <p className="small text-body-secondary mt-2 mb-0">{c.waitForUpdate}</p>}
      {save.isError && (
        <p className="small text-danger mt-2 mb-0" role="alert">
          {c.saveError}
        </p>
      )}
    </section>
  )
}
