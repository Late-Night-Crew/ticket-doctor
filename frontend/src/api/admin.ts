/**
 * Admin API (B6). Kept apart from the public hooks so it only loads with the
 * admin pages. The session is an HttpOnly cookie the browser sends by itself:
 * nothing here ever sees or stores the token.
 */
import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api, ApiError, post, put } from './client'

export interface Admin {
  id: string
  username: string
  role: 'admin'
  /** The sign-in before this one; null on the first. */
  last_login_at: string | null
}

export interface ImportRun {
  id: string
  started_at: string
  finished_at: string | null
  status: 'running' | 'succeeded' | 'failed'
  summary: ImportSummary
  error: string | null
}

/**
 * The parts of a run summary the dashboard shows: a full import
 * (backend services.ImportSummary) or a quick score check (services.ScoresSummary).
 */
export interface ImportSummary {
  leagues?: {
    league: string
    inserted?: number
    updated?: number
    /** Score checks: games given a full-time score. */
    results?: number
    unmatched?: string[]
    /** Where the results file changed a quick or hand-entered score. */
    corrected?: string[]
    error?: string
    fixtures?: { inserted: number; updated: number; results?: number; unmatched?: string[]; error?: string }
  }[]
}

export interface Dashboard {
  matches: { upcoming: number; awaiting_result: number; finished: number }
  tickets: { total: number; last_7_days: number; legs: number }
  suggestions: { shown: number; accepted: number }
  imports: ImportRun[]
  /** The latest quick score checks (every 15 minutes on match days). */
  score_checks: ImportRun[]
}

export type MatchView = 'awaiting' | 'recent'

export interface AdminMatch {
  id: string
  league_id: string
  kickoff: string
  status: 'scheduled' | 'postponed' | 'finished'
  home: string
  away: string
  home_goals: number | null
  away_goals: number | null
  source: string
}

export interface Threshold {
  key: string
  label: string
  description: string
  value: number
  default: number
  unit: 'percent' | 'count'
}

const ME = ['admin', 'me'] as const

/** The signed-in admin; the query fails with a 401 ApiError when signed out. */
export function useAdminMe() {
  return useQuery({
    queryKey: ME,
    queryFn: () => api<Admin>('/admin/me'),
    retry: false,
    staleTime: 5 * 60 * 1000,
  })
}

export function useLogin() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: { username: string; password: string }) =>
      post<{ admin: Admin; expires_at: string }>('/admin/login', body),
    onSuccess: (s) => qc.setQueryData(ME, s.admin),
  })
}

/**
 * Sign out, then run `after` (e.g. go to the sign-in page). Every admin answer
 * is dropped afterwards, so nothing is left in memory; dropping it first would
 * make the admin pages bounce to sign-in on their own.
 */
export function useLogout(after: () => void) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => post<void>('/admin/logout', {}),
    onSettled: () => {
      after()
      qc.removeQueries({ queryKey: ['admin'] })
    },
  })
}

export function useDashboard() {
  return useQuery({ queryKey: ['admin', 'dashboard'], queryFn: () => api<Dashboard>('/admin/dashboard') })
}

export function useAdminMatches(view: MatchView, league: string) {
  const q = new URLSearchParams({ view })
  if (league) q.set('league', league)
  return useQuery({
    queryKey: ['admin', 'matches', view, league],
    queryFn: () => api<AdminMatch[]>(`/admin/matches?${q}`),
  })
}

export function useEnterResult() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, home, away }: { id: string; home: number; away: number }) =>
      put<void>(`/admin/matches/${id}/result`, { home_goals: home, away_goals: away }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin', 'matches'] })
      qc.invalidateQueries({ queryKey: ['admin', 'dashboard'] })
      // The public track record may change too.
      qc.invalidateQueries({ queryKey: ['results'] })
    },
  })
}

export function useThresholds() {
  return useQuery({
    queryKey: ['admin', 'thresholds'],
    queryFn: () => api<Threshold[]>('/admin/thresholds'),
    staleTime: Infinity,
  })
}

export const isSignedOut = (e: unknown) => e instanceof ApiError && e.status === 401

/* ---------- first-admin setup and managing admins ---------- */

export interface AdminAccount extends Admin {
  disabled: boolean
  created_at: string
}

/** Whether /admin/setup can create the first account (a setup code is set and no account exists). */
export function useSetupStatus() {
  return useQuery({
    queryKey: ['admin-setup'],
    queryFn: () => api<{ open: boolean }>('/admin/setup'),
    retry: false,
  })
}

export function useSetup() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: { code: string; username: string; password: string }) =>
      post<{ admin: Admin; expires_at: string }>('/admin/setup', body),
    onSuccess: (s) => {
      qc.setQueryData(ME, s.admin)
      qc.setQueryData(['admin-setup'], { open: false })
    },
  })
}

export function useAdminAccounts() {
  return useQuery({ queryKey: ['admin', 'users'], queryFn: () => api<AdminAccount[]>('/admin/users') })
}

export function useCreateAdmin() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: { username: string; password: string }) => post<Admin>('/admin/users', body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin', 'users'] }),
  })
}

export function useSetDisabled() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ username, disabled }: { username: string; disabled: boolean }) =>
      put<void>(`/admin/users/${encodeURIComponent(username)}/disabled`, { disabled }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin', 'users'] }),
  })
}

/** Change your own password; other sessions end, this one carries on with a new cookie. */
export function useChangePassword() {
  return useMutation({
    mutationFn: (body: { current_password: string; new_password: string }) =>
      put<{ admin: Admin; expires_at: string }>('/admin/me/password', body),
  })
}

export const MIN_PASSWORD = 12

/* ---------- calibration ---------- */

export type CalibrationSource = 'backtest' | 'tickets'

export interface CalibrationBin {
  from: number
  to: number
  n: number
  predicted: number
  actual: number
}

export interface Calibration {
  source: CalibrationSource
  n: number
  matches: number
  skipped_low_confidence: number
  brier: number
  bins: CalibrationBin[]
  bands: { band: 'low' | 'medium' | 'high'; n: number; predicted: number; actual: number }[]
  markets: { market: string; n: number; predicted: number; actual: number; brier: number; base_brier: number; skill: number }[]
}

export interface CalibrationQuery {
  source: CalibrationSource
  league: string
  market: string
  kind: 'picks' | 'suggestions'
  low: boolean
}

export function useCalibration(q: CalibrationQuery) {
  const p = new URLSearchParams({ source: q.source })
  if (q.league) p.set('league', q.league)
  if (q.market) p.set('market', q.market)
  if (q.source === 'tickets') p.set('kind', q.kind)
  if (q.source === 'backtest' && q.low) p.set('low', '1')
  return useQuery({
    queryKey: ['admin', 'calibration', p.toString()],
    queryFn: () => api<Calibration>(`/admin/calibration?${p}`),
    placeholderData: keepPreviousData,
    staleTime: 10 * 60 * 1000,
  })
}
