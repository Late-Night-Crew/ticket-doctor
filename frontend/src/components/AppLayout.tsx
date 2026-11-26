import { Suspense } from 'react'
import { Link, NavLink, Outlet, useLocation } from 'react-router'
import { COPY } from '../config/copy'
import { useTicket } from '../state/useTicket'
import { ErrorBoundary } from './ErrorBoundary'
import { Footer } from './Footer'
import { IconChart, IconHome, IconLogo, IconTicket, IconWand } from './Icons'
import { RouteEffects } from './RouteEffects'
import { ThemeToggle } from './ThemeToggle'

const NAV = [
  { to: '/', label: COPY.nav.home, icon: IconHome, end: true },
  { to: '/ticket', label: COPY.nav.build, icon: IconTicket, end: false },
  { to: '/wizard', label: COPY.nav.wizard, icon: IconWand, end: false },
  { to: '/results', label: COPY.nav.results, icon: IconChart, end: false },
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
        {NAV.map(({ to, label, icon: Icon, end }) => (
          <NavLink key={to} to={to} end={end}>
            <span className="position-relative">
              <Icon />
              {to === '/ticket' && legCount > 0 && (
                <span className="badge rounded-pill text-bg-primary td-bottomnav-count td-num">{legCount}</span>
              )}
            </span>
            {label}
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
