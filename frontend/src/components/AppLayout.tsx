import { Suspense } from 'react'
import { Link, NavLink, Outlet, useLocation } from 'react-router'
import { COPY } from '../config/copy'
import { useTicket } from '../state/useTicket'
import { ErrorBoundary } from './ErrorBoundary'
import { Footer } from './Footer'
import { IconBookmark, IconChart, IconHome, IconLogo, IconTicket, IconWand } from './Icons'
import { RouteEffects } from './RouteEffects'
import { ThemeToggle } from './ThemeToggle'

const NAV = [
  { to: '/', label: COPY.nav.home, short: COPY.nav.short.home, icon: IconHome, end: true },
  { to: '/ticket', label: COPY.nav.build, short: COPY.nav.short.build, icon: IconTicket, end: false },
  { to: '/wizard', label: COPY.nav.wizard, short: COPY.nav.short.wizard, icon: IconWand, end: false },
  { to: '/results', label: COPY.nav.results, short: COPY.nav.short.results, icon: IconChart, end: false },
  { to: '/tickets', label: COPY.nav.saved, short: COPY.nav.short.saved, icon: IconBookmark, end: false },
]

export function AppLayout() {
  const { ticket } = useTicket()
  const { pathname } = useLocation()
  const legCount = ticket.legs.length

  return (
    <div className="td-app">
      <RouteEffects mainId="main" />
      <a href="#main" className="visually-hidden-focusable btn btn-primary position-absolute m-2">
        Skip to content
      </a>

      <header className="navbar td-topbar">
        <div className="td-container d-flex align-items-center w-100">
          <Link to="/" className="navbar-brand">
            <IconLogo width={28} height={28} style={{ color: 'var(--td-brand)' }} />
            {COPY.appName}
          </Link>
          <nav className="d-none d-md-flex ms-auto gap-1" aria-label="Main">
            {NAV.slice(1).map((n) => (
              <NavLink key={n.to} to={n.to} end={n.end} className="nav-link">
                {n.label}
                {n.to === '/ticket' && legCount > 0 && (
                  <span className="badge rounded-pill text-bg-primary ms-1 td-num">{legCount}</span>
                )}
              </NavLink>
            ))}
          </nav>
          <div className="ms-auto ms-md-2">
            <ThemeToggle />
          </div>
        </div>
      </header>

      <main id="main" className="td-main" tabIndex={-1}>
        <div className="td-container">
          {/* Keyed by path so an error on one page clears when you navigate away. */}
          <ErrorBoundary key={pathname}>
            <Suspense fallback={<PageSpinner />}>
              <Outlet />
            </Suspense>
          </ErrorBoundary>
        </div>
      </main>

      <Footer />

      <nav className="td-bottomnav" aria-label="Main">
        {NAV.map(({ to, label, short, icon: Icon, end }) => (
          // Short visible label; the full one is what screen readers announce.
          <NavLink
            key={to}
            to={to}
            end={end}
            aria-label={to === '/ticket' && legCount > 0 ? `${label}, ${COPY.nav.legsOnTicket(legCount)}` : label}
          >
            <span className="position-relative">
              <Icon />
              {to === '/ticket' && legCount > 0 && (
                <span className="badge rounded-pill text-bg-primary td-bottomnav-count td-num">{legCount}</span>
              )}
            </span>
            {short}
          </NavLink>
        ))}
      </nav>
    </div>
  )
}

export function PageSpinner() {
  return (
    <div className="d-flex justify-content-center py-5">
      <div className="spinner-border text-primary" role="status">
        <span className="visually-hidden">Loading…</span>
      </div>
    </div>
  )
}
