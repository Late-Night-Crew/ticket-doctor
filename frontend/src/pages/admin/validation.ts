import { MIN_PASSWORD } from '../../api/admin'

export const USERNAME_RE = /^[a-z0-9._-]{3,32}$/
export const USERNAME_HELP = '3–32 characters: lowercase letters, digits, dots, dashes or underscores.'

/** Problems with a new password and its confirmation, or null when both are fine. */
export function passwordProblem(password: string, confirm: string): { password?: string; confirm?: string } | null {
  if ([...password].length < MIN_PASSWORD) return { password: `Use at least ${MIN_PASSWORD} characters.` }
  if (new TextEncoder().encode(password).length > 72) return { password: 'Use at most 72 bytes.' }
  if (confirm !== password) return { confirm: "The passwords don't match." }
  return null
}
