import { useId, useState, type ReactNode } from 'react'
import Modal from 'react-bootstrap/Modal'
import { glossaryEntry } from '../config/glossary'

/** Tap a market name to see a short explanation (FR-8.10). */
export function GlossaryTerm({ id, children }: { id: string; children: ReactNode }) {
  const [open, setOpen] = useState(false)
  const titleId = useId()
  const entry = glossaryEntry(id)
  if (!entry) return <>{children}</>
  return (
    <>
      <button type="button" className="td-link-button" onClick={() => setOpen(true)}>
        {children}
      </button>
      <Modal show={open} onHide={() => setOpen(false)} centered aria-labelledby={titleId}>
        <Modal.Header closeButton>
          <Modal.Title as="h2" className="h5" id={titleId}>
            {entry.term}
          </Modal.Title>
        </Modal.Header>
        <Modal.Body>
          <p className="mb-0">{entry.body}</p>
        </Modal.Body>
      </Modal>
    </>
  )
}
