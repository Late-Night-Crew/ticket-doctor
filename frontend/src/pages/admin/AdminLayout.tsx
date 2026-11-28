import { Suspense, useState } from 'react'
import Offcanvas from 'react-bootstrap/Offcanvas'
import { Link, Navigate, NavLink, Outlet, useLocation, useNavigate } from 'react-router'
import { isSignedOut, useAdminMe, useLogout } from '../../api/admin'
import { ErrorBoundary } from '../../components/ErrorBoundary'
import { IconGrid, IconLogo, IconLogout, IconMenu, IconPencil, IconSliders, IconTarget, IconUsers } from '../../components/Icons'
import { RouteEffects } from '../../components/RouteEffects'
import { ThemeToggle } from '../../components/ThemeToggle'
import { PageSpinner } from '../../components/AppLayout'
import { useNoIndex } from './useNoIndex'

const NAV = [
  { to: '/admin', label: 'Dashboard', icon: IconGrid, end: true },
  { to: '/admin/results', label: 'Enter results', icon: IconPencil, end: false },
  { to: '/admin/calibration', label: 'Calibration', icon: IconTarget, end: false },
  { to: '/admin/thresholds', label: 'Rule thresholds', icon: IconSliders, end: false },
  { to: '/admin/admins', label: 'Admins', icon: IconUsers, end: false },
]

/**
 * The admin app shell (B6): its own layout, apart from the public one, with a
 * sidebar on wide screens and a slide-out menu on phones. Every page inside
 * needs a signed-in admin; otherwise it goes to the sign-in page and comes
 * back afterwards.
 */
export default function AdminLayout() {
  useNoIndex()
  const me = useAdminMe()
  const navigate = useNavigate()
  const logout = useLogout(() => navigate('/admin/login', { replace: true }))
  const { pathname, search } = useLocation()
  const [menuOpen, setMenuOpen] = useState(false)

  // While signing out, wait for the sign-in page rather than redirecting with ?next.
  if (me.isPending || !logout.isIdle) return <PageSpinner />
  if (isSignedOut(me.error)) {
    return <Navigate to={`/admin/login?next=${encodeURIComponent(pathname + search)}`} replace />
  }
  if (me.isError) {
    return (
      <div className="td-container py-5">
        <div className="alert alert-danger" role="alert">
          Couldn't reach the server: {me.error.message}
        </div>
      </div>
    )
  }

  const signOut = () => logout.mutate()

  const nav = (
    <nav className="td-admin-nav" aria-label="Admin">
      {NAV.map(({ to, label, icon: Icon, end }) => (
        <NavLink key={to} to={to} end={end} onClick={() => setMenuOpen(false)}>
          <Icon />
          {label}
        </NavLink>
      ))}
      <hr />
      <Link to="/">Back to the site</Link>
    </nav>
  )

  return (
    <div className="td-admin">
      <RouteEffects mainId="admin-main" />
      <a href="#admin-main" className="visually-hidden-focusable btn btn-primary position-absolute m-2">
        Skip to content
      </a>

      <aside className="td-admin-sidebar d-none d-md-flex">
        <Brand />
        {nav}
      </aside>

      <div className="td-admin-body">
        <header className="td-admin-topbar">
          <button
            type="button"
            className="btn btn-link d-md-none p-0 me-2 td-admin-menu"
            aria-label="Open menu"
            onClick={() => setMenuOpen(true)}
          >
            <IconMenu />
          </button>
          <span className="d-md-none">
            <Brand />
          </span>
          <div className="ms-auto d-flex align-items-center gap-2">
            <span className="small text-body-secondary d-none d-sm-inline">
              Signed in as <strong className="text-body">{me.data.username}</strong>
            </span>
            <ThemeToggle />
            <button
              type="button"
              className="btn btn-outline-secondary btn-sm text-nowrap td-admin-signout"
              aria-label="Sign out"
              title="Sign out"
              onClick={signOut}
              disabled={logout.isPending}
            >
              <IconLogout width={18} height={18} />
              <span className="d-none d-sm-inline ms-1">Sign out</span>
            </button>
          </div>
        </header>

        <main id="admin-main" className="td-admin-main" tabIndex={-1}>
          <ErrorBoundary key={pathname}>
            <Suspense fallback={<PageSpinner />}>
              <Outlet />
            </Suspense>
          </ErrorBoundary>
        </main>
      </div>

      <Offcanvas show={menuOpen} onHide={() => setMenuOpen(false)} className="td-admin-offcanvas">
        <Offcanvas.Header closeButton>
          <Offcanvas.Title as="div">
            <Brand />
          </Offcanvas.Title>
        </Offcanvas.Header>
        <Offcanvas.Body>{nav}</Offcanvas.Body>
      </Offcanvas>
    </div>
  )
}

function Brand() {
  return (
    <Link to="/admin" className="td-admin-brand">
      <IconLogo width={26} height={26} style={{ color: 'var(--td-brand)' }} />
      <span>
        Ticket Doctor <span className="td-admin-tag">Admin</span>
      </span>
    </Link>
  )
}
