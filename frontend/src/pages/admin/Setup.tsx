import { useState, type FormEvent } from 'react'
import { Link, Navigate, useNavigate } from 'react-router'
import { useSetup, useSetupStatus } from '../../api/admin'
import { PageSpinner } from '../../components/AppLayout'
import { IconLogo } from '../../components/Icons'
import { RouteEffects } from '../../components/RouteEffects'
import { Field, NewPasswordFields } from './fields'
import { passwordProblem, USERNAME_HELP, USERNAME_RE } from './validation'
import { useNoIndex } from './useNoIndex'

/**
 * Creates the first admin account from the browser. The server only allows it
 * while there are no accounts, and only with the setup code from Render
 * (TD_ADMIN_SETUP_CODE), so nobody else can claim it first.
 */
export default function AdminSetup() {
  useNoIndex()
  const status = useSetupStatus()
  const setup = useSetup()
  const navigate = useNavigate()
  const [code, setCode] = useState('')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')

  if (status.isPending) return <PageSpinner />
  if (setup.isSuccess) return <Navigate to="/admin" replace />

  const name = username.trim().toLowerCase()
  const ready = code.trim() !== '' && USERNAME_RE.test(name) && !passwordProblem(password, confirm) && !setup.isPending

  const submit = (e: FormEvent) => {
    e.preventDefault()
    if (ready) {
      setup.mutate({ code: code.trim(), username: name, password }, { onSuccess: () => navigate('/admin', { replace: true }) })
    }
  }

  return (
    <div className="td-admin-login">
      <RouteEffects mainId="setup-main" />
      <main id="setup-main" tabIndex={-1} className="td-surface p-4">
        <div className="d-flex align-items-center gap-2 mb-3">
          <IconLogo width={32} height={32} style={{ color: 'var(--td-brand)' }} />
          <div>
            <h1 className="h5 mb-0">Set up the first admin</h1>
            <p className="small text-body-secondary mb-0">This works once, while no admin account exists.</p>
          </div>
        </div>

        {!status.data?.open ? (
          <>
            <div className="alert alert-info small" role="status">
              {status.isError
                ? `Couldn't check: ${status.error.message}`
                : 'Setup is closed: either an admin account already exists, or no setup code is set on the server.'}
            </div>
            <Link to="/admin/login" className="btn btn-primary w-100">
              Go to sign in
            </Link>
          </>
        ) : (
          <form onSubmit={submit} noValidate>
            {setup.isError && (
              <div className="alert alert-danger py-2 small" role="alert">
                {setup.error.message}
              </div>
            )}
            <Field
              id="setup-code"
              label="Setup code"
              type="password"
              autoComplete="off"
              value={code}
              onChange={setCode}
              help="The value of TD_ADMIN_SETUP_CODE in Render (ticket-doctor-api → Environment)."
            />
            <Field
              id="setup-username"
              label="Username"
              autoComplete="username"
              value={username}
              onChange={setUsername}
              help={USERNAME_HELP}
              invalid={username !== '' && !USERNAME_RE.test(name) && USERNAME_HELP}
            />
            <NewPasswordFields idPrefix="setup" password={password} confirm={confirm} onPassword={setPassword} onConfirm={setConfirm} />
            <button type="submit" className="btn btn-primary w-100" disabled={!ready}>
              {setup.isPending ? 'Creating…' : 'Create admin and sign in'}
            </button>
          </form>
        )}
      </main>
    </div>
  )
}
