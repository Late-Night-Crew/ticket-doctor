import { Link } from 'react-router'
import { COPY } from '../config/copy'
import { IconChart, IconTicket, IconWand } from '../components/Icons'

export default function Home() {
  const c = COPY.home
  return (
    <>
      <section className="td-hero">
        <div className="td-eyebrow mb-2">{COPY.tagline}</div>
        <h1 className="mb-3">{c.title}</h1>
        <p className="lead mb-4">{c.lead}</p>

        <div className="row g-3">
          <div className="col-12 col-md-6">
            <Link to="/ticket" className="td-surface td-path-card h-100">
              <IconTicket width={28} height={28} className="mb-2" style={{ color: 'var(--td-brand)' }} />
              <h2>{c.ctaCheck}</h2>
              <p className="mb-0 text-body-secondary small">
                You already have picks in mind. Add them and see where the ticket is weakest.
              </p>
            </Link>
          </div>
          <div className="col-12 col-md-6">
            <Link to="/wizard" className="td-surface td-path-card h-100">
              <IconWand width={28} height={28} className="mb-2" style={{ color: 'var(--td-brand)' }} />
              <h2>{c.ctaWizard}</h2>
              <p className="mb-0 text-body-secondary small">
                Answer a few quick questions. New to betting? There is a step-by-step path for you.
              </p>
            </Link>
          </div>
        </div>
      </section>

      <section className="mt-4">
        <h2 className="h5 mb-3">{c.howTitle}</h2>
        <ol className="td-steps">
          {c.steps.map((s) => (
            <li key={s.title} className="td-surface">
              <strong>{s.title}</strong>
              <span className="text-body-secondary small">{s.body}</span>
            </li>
          ))}
        </ol>
      </section>

      <section className="mt-4 td-surface p-3 d-flex gap-3 align-items-start">
        <IconChart width={28} height={28} className="flex-none" style={{ color: 'var(--td-brand)' }} />
        <div>
          <h2 className="h6 mb-1">{c.honestTitle}</h2>
          <p className="mb-2 small text-body-secondary">{c.honestBody}</p>
          <Link to="/results" className="small fw-semibold">
            See the track record →
          </Link>
        </div>
      </section>
    </>
  )
}
