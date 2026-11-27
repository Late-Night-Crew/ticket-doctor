import { COPY } from '../config/copy'

/** Responsible-gambling footer: 18+ notice and help links (RG-4). */
export function Footer() {
  return (
    <footer className="td-footer">
      <div className="td-container">
        <div className="d-flex gap-3 align-items-start mb-3">
          <span className="td-age" aria-hidden="true">18+</span>
          <div>
            <p className="mb-1 fw-semibold text-body">{COPY.footer.ageNotice}</p>
            <p className="mb-0">
              {COPY.footer.helpTitle}{' '}
              {COPY.footer.helpLinks.map((l, i) => (
                <span key={l.href}>
                  <a href={l.href} target="_blank" rel="noopener noreferrer">
                    {l.label}
                  </a>
                  {i < COPY.footer.helpLinks.length - 1 && ' · '}
                </span>
              ))}
            </p>
            <p className="mb-0">
              {COPY.footer.localHelp.before}
              <a href={COPY.footer.localHelp.href} target="_blank" rel="noopener noreferrer">
                {COPY.footer.localHelp.label}
              </a>
              {COPY.footer.localHelp.after}
            </p>
          </div>
        </div>
        <p className="mb-0 small">{COPY.footer.noStake}</p>
      </div>
    </footer>
  )
}
