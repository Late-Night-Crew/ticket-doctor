import { useState, type FormEvent } from 'react'
import {
  useAdminAccounts,
  useAdminMe,
  useChangePassword,
  useCreateAdmin,
  useSetDisabled,
  type AdminAccount,
} from '../../api/admin'
import { PageSpinner } from '../../components/AppLayout'
import { PageHeader } from '../../components/PageHeader'
import { Field, NewPasswordFields } from './fields'
import { passwordProblem, USERNAME_HELP, USERNAME_RE } from './validation'

const when = (iso: string | null) =>
  iso ? new Date(iso).toLocaleString(undefined, { day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' }) : 'never'

/** Admin accounts: who can sign in, add someone, and change your own password. */
export default function Admins() {
  const me = useAdminMe()
  const accounts = useAdminAccounts()

  return (
    <>
      <PageHeader eyebrow="Admin" title="Admins">
        Everyone here can enter results and add other admins. Disable an account to sign it out at once and stop it
        signing in; you can turn it back on later.
      </PageHeader>

      <section className="td-surface p-0 overflow-hidden mb-4" aria-labelledby="admins-list-title">
        <h2 id="admins-list-title" className="h6 px-3 pt-3 mb-2">
          Accounts
        </h2>
        {accounts.isPending ? (
          <PageSpinner />
        ) : accounts.isError ? (
          <div className="alert alert-danger m-3">Couldn't load the accounts: {accounts.error.message}</div>
        ) : (
          <div className="table-responsive">
            <table className="table align-middle mb-0">
              <thead>
                <tr>
                  <th scope="col">Username</th>
                  <th scope="col">Last sign-in</th>
                  <th scope="col" className="text-end">
                    <span className="visually-hidden">Actions</span>
                  </th>
                </tr>
              </thead>
              <tbody>
                {accounts.data.map((a) => (
                  <AccountRow key={a.id} account={a} isMe={a.username === me.data?.username} />
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <div className="row g-4">
        <div className="col-lg-6">
          <AddAdmin />
        </div>
        <div className="col-lg-6">
          <ChangePassword />
        </div>
      </div>
    </>
  )
}

function AccountRow({ account: a, isMe }: { account: AdminAccount; isMe: boolean }) {
  const toggle = useSetDisabled()
  return (
    <tr>
      <td>
        <span className="fw-semibold">{a.username}</span>
        {isMe && <span className="badge text-bg-light border ms-2">you</span>}
        {a.disabled && <span className="badge text-bg-secondary ms-2">disabled</span>}
        {toggle.isError && <div className="small text-danger">{toggle.error.message}</div>}
      </td>
      <td className="small text-body-secondary text-nowrap">{when(a.last_login_at)}</td>
      <td className="text-end">
        {!isMe && (
          <button
            type="button"
            className={`btn btn-sm ${a.disabled ? 'btn-outline-primary' : 'btn-outline-danger'}`}
            disabled={toggle.isPending}
            aria-label={`${a.disabled ? 'Enable' : 'Disable'} ${a.username}`}
            onClick={() => toggle.mutate({ username: a.username, disabled: !a.disabled })}
          >
            {a.disabled ? 'Enable' : 'Disable'}
          </button>
        )}
      </td>
    </tr>
  )
}

function AddAdmin() {
  const create = useCreateAdmin()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [added, setAdded] = useState<string | null>(null)

  const name = username.trim().toLowerCase()
  const ready = USERNAME_RE.test(name) && !passwordProblem(password, confirm) && !create.isPending

  const submit = (e: FormEvent) => {
    e.preventDefault()
    if (!ready) return
    create.mutate(
      { username: name, password },
      {
        onSuccess: (a) => {
          setAdded(a.username)
          setUsername('')
          setPassword('')
          setConfirm('')
        },
      },
    )
  }

  return (
    <section className="td-surface p-3 h-100" aria-labelledby="add-admin-title">
      <h2 id="add-admin-title" className="h6">
        Add an admin
      </h2>
      <p className="small text-body-secondary">
        Give them the password in person or over a private message, and ask them to change it after signing in.
      </p>
      <form onSubmit={submit} noValidate>
        {create.isError && (
          <div className="alert alert-danger py-2 small" role="alert">
            {create.error.message}
          </div>
        )}
        {added && !create.isError && (
          <div className="alert alert-success py-2 small" role="status">
            Added {added}. They can sign in at /admin.
          </div>
        )}
        <Field
          id="new-admin-username"
          label="Username"
          autoComplete="off"
          value={username}
          onChange={(v) => {
            setUsername(v)
            setAdded(null)
          }}
          help={USERNAME_HELP}
          invalid={username !== '' && !USERNAME_RE.test(name) && USERNAME_HELP}
        />
        <NewPasswordFields idPrefix="new-admin" password={password} confirm={confirm} onPassword={setPassword} onConfirm={setConfirm} />
        <button type="submit" className="btn btn-primary" disabled={!ready}>
          {create.isPending ? 'Adding…' : 'Add admin'}
        </button>
      </form>
    </section>
  )
}

function ChangePassword() {
  const change = useChangePassword()
  const [current, setCurrent] = useState('')
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')

  const ready = current !== '' && !passwordProblem(password, confirm) && !change.isPending

  const submit = (e: FormEvent) => {
    e.preventDefault()
    if (!ready) return
    change.mutate(
      { current_password: current, new_password: password },
      {
        onSuccess: () => {
          setCurrent('')
          setPassword('')
          setConfirm('')
        },
      },
    )
  }

  return (
    <section className="td-surface p-3 h-100" aria-labelledby="change-password-title">
      <h2 id="change-password-title" className="h6">
        Change your password
      </h2>
      <p className="small text-body-secondary">This signs you out everywhere else; you stay signed in here.</p>
      <form onSubmit={submit} noValidate>
        {change.isError && (
          <div className="alert alert-danger py-2 small" role="alert">
            {change.error.message}
          </div>
        )}
        {change.isSuccess && (
          <div className="alert alert-success py-2 small" role="status">
            Password changed.
          </div>
        )}
        <Field
          id="current-password"
          label="Current password"
          type="password"
          autoComplete="current-password"
          value={current}
          onChange={setCurrent}
        />
        <NewPasswordFields
          idPrefix="change"
          label="New password"
          password={password}
          confirm={confirm}
          onPassword={setPassword}
          onConfirm={setConfirm}
        />
        <button type="submit" className="btn btn-primary" disabled={!ready}>
          {change.isPending ? 'Saving…' : 'Change password'}
        </button>
      </form>
    </section>
  )
}
