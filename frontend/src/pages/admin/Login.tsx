import { useState, type FormEvent } from 'react'
import { Link, Navigate, useNavigate, useSearchParams } from 'react-router'
import { useAdminMe, useLogin, useSetupStatus } from '../../api/admin'
import { IconLogo } from '../../components/Icons'
import { RouteEffects } from '../../components/RouteEffects'
import { useNoIndex } from './useNoIndex'

/** Only go back to admin pages after signing in, never to another site. */
function safeNext(next: string | null): string {
  return next && next.startsWith('/admin') && !next.startsWith('/admin/login') ? next : '/admin'
}

export default function AdminLogin() {
  useNoIndex()
  const [params] = useSearchParams()
  const next = safeNext(params.get('next'))
  const navigate = useNavigate()
  const me = useAdminMe()
  const login = useLogin()
  const setup = useSetupStatus()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')

  if (me.isSuccess) return <Navigate to={next} replace />

  const submit = (e: FormEvent) => {
    e.preventDefault()
    login.mutate(
      { username, password },
      {
        onSuccess: () => navigate(next, { replace: true }),
        onError: () => setPassword(''),
      },
    )
  }

  return (
    <div className="td-admin-login">
      <RouteEffects mainId="login-main" />
      <main id="login-main" tabIndex={-1} className="td-surface p-4">
        <div className="d-flex align-items-center gap-2 mb-3">
          <IconLogo width={32} height={32} style={{ color: 'var(--td-brand)' }} />
          <div>
            <h1 className="h5 mb-0">Ticket Doctor admin</h1>
            <p className="small text-body-secondary mb-0">Sign in to enter results and check the rules.</p>
          </div>
        </div>

        {setup.data?.open && (
          <div className="alert alert-info py-2 small" role="status">
            No admin account exists yet. <Link to="/admin/setup">Set up the first admin</Link>.
          </div>
        )}
        <form onSubmit={submit} noValidate>
          {login.isError && (
            <div className="alert alert-danger py-2 small" role="alert">
              {login.error.message}
            </div>
          )}
          <div className="mb-3">
            <label htmlFor="admin-username" className="form-label">
              Username
            </label>
            <input
              id="admin-username"
              className="form-control"
              autoComplete="username"
              autoCapitalize="none"
              spellCheck={false}
              required
              value={username}
              onChange={(e) => setUsername(e.target.value)}
            />
          </div>
          <div className="mb-3">
            <label htmlFor="admin-password" className="form-label">
              Password
            </label>
            <input
              id="admin-password"
              type="password"
              className="form-control"
              autoComplete="current-password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </div>
          <button type="submit" className="btn btn-primary w-100" disabled={login.isPending || !username || !password}>
            {login.isPending ? 'Signing in…' : 'Sign in'}
          </button>
        </form>
        <p className="small text-body-secondary mt-3 mb-0">
          Need an account? Ask an admin to add you. <Link to="/">Back to the site</Link>
        </p>
      </main>
    </div>
  )
}
