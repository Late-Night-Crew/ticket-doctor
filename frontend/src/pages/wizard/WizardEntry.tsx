import { Link } from 'react-router'
import { IconTicket, IconWand } from '../../components/Icons'
import { PageHeader } from '../../components/PageHeader'
import { COPY } from '../../config/copy'

/** Two paths into the wizard (FR-8.1). */
export default function WizardEntry() {
  const w = COPY.wizard
  return (
    <>
      <PageHeader eyebrow={w.eyebrow} title={w.entryTitle}>
        {w.entryLead}
      </PageHeader>
      <div className="row g-3">
        <div className="col-12 col-md-6">
          <Link to="beginner" className="td-surface td-path-card h-100">
            <IconTicket width={28} height={28} className="mb-2" style={{ color: 'var(--td-brand)' }} />
            <h2>{w.paths.beginner.title}</h2>
            <p className="mb-0 text-body-secondary small">{w.paths.beginner.body}</p>
          </Link>
        </div>
        <div className="col-12 col-md-6">
          <Link to="guided" className="td-surface td-path-card h-100">
            <IconWand width={28} height={28} className="mb-2" style={{ color: 'var(--td-brand)' }} />
            <h2>{w.paths.guided.title}</h2>
            <p className="mb-0 text-body-secondary small">{w.paths.guided.body}</p>
          </Link>
        </div>
      </div>
    </>
  )
}
