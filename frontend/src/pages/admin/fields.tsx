import { MIN_PASSWORD } from '../../api/admin'
import { passwordProblem } from './validation'

/** A labelled text input for the admin forms. */
export function Field(props: {
  id: string
  label: string
  value: string
  onChange: (v: string) => void
  type?: 'text' | 'password'
  autoComplete: string
  help?: string
  invalid?: string | false
}) {
  return (
    <div className="mb-3">
      <label htmlFor={props.id} className="form-label">
        {props.label}
      </label>
      <input
        id={props.id}
        type={props.type ?? 'text'}
        className={`form-control${props.invalid ? ' is-invalid' : ''}`}
        autoComplete={props.autoComplete}
        autoCapitalize="none"
        spellCheck={false}
        value={props.value}
        aria-describedby={props.help || props.invalid ? `${props.id}-help` : undefined}
        onChange={(e) => props.onChange(e.target.value)}
      />
      {(props.invalid || props.help) && (
        <div id={`${props.id}-help`} className={props.invalid ? 'invalid-feedback' : 'form-text'}>
          {props.invalid || props.help}
        </div>
      )}
    </div>
  )
}

/** New password plus confirmation, showing problems only once something is typed. */
export function NewPasswordFields(props: {
  idPrefix: string
  password: string
  confirm: string
  onPassword: (v: string) => void
  onConfirm: (v: string) => void
  label?: string
}) {
  const problem = passwordProblem(props.password, props.confirm)
  return (
    <>
      <Field
        id={`${props.idPrefix}-password`}
        label={props.label ?? 'Password'}
        type="password"
        autoComplete="new-password"
        value={props.password}
        onChange={props.onPassword}
        help={`At least ${MIN_PASSWORD} characters. A short sentence is easy to remember.`}
        invalid={props.password !== '' && problem?.password}
      />
      <Field
        id={`${props.idPrefix}-confirm`}
        label="Same password again"
        type="password"
        autoComplete="new-password"
        value={props.confirm}
        onChange={props.onConfirm}
        invalid={props.confirm !== '' && problem?.confirm}
      />
    </>
  )
}
