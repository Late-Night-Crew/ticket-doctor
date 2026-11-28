import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api, post } from './client'
import type {
  AcceptSuggestionResponse,
  AnalyseRequest,
  AnalyseResponse,
  Fixture,
  FixtureWindow,
  League,
  ResultsResponse,
  SavedLeg,
  SavedTicket,
  WizardRequest,
  WizardResponse,
} from './types'

const HOUR = 60 * 60 * 1000

export function useLeagues() {
  return useQuery({
    queryKey: ['leagues'],
    queryFn: () => api<League[]>('/leagues'),
    staleTime: 24 * HOUR,
  })
}

export function useFixtures(leagueId: string | null, when: FixtureWindow = 'upcoming') {
  return useQuery({
    queryKey: ['fixtures', leagueId, when],
    queryFn: () => api<Fixture[]>(`/leagues/${leagueId}/fixtures?when=${when}`),
    enabled: !!leagueId,
    staleTime: HOUR,
  })
}

/**
 * Analyse a ticket. A query keyed on the ticket's contents, so the same ticket
 * is never sent twice (each analysis is logged server-side, FR-7.1) and undoing
 * a swap reuses the earlier result. Previous data stays visible while updating.
 */
export function useAnalysis(req: AnalyseRequest | null) {
  return useQuery({
    queryKey: ['analyse', req],
    queryFn: () => post<AnalyseResponse>('/tickets/analyse', req),
    enabled: !!req && req.legs.length > 0,
    staleTime: Infinity,
    retry: false,
    placeholderData: keepPreviousData,
  })
}

export function useAcceptSuggestion() {
  return useMutation({
    mutationFn: (suggestionId: string) =>
      post<AcceptSuggestionResponse>(`/suggestions/${suggestionId}/accept`, {}),
  })
}

/** Generate a wizard ticket. Keyed on the answers and seed, so Regenerate (new seed) gets a new ticket. */
export function useWizardTicket(req: WizardRequest | null) {
  return useQuery({
    queryKey: ['wizard', req],
    queryFn: () => post<WizardResponse>('/wizard', req),
    enabled: !!req,
    staleTime: Infinity,
    retry: false,
  })
}

export function useResults(filters: { league?: string; market?: string } = {}) {
  const qs = new URLSearchParams()
  if (filters.league) qs.set('league', filters.league)
  if (filters.market) qs.set('market', filters.market)
  return useQuery({
    queryKey: ['results', filters],
    queryFn: () => api<ResultsResponse>(`/results?${qs}`),
    staleTime: HOUR,
  })
}

/* ---------- Saved tickets (V2) ---------- */

export function useSaveTicket() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (ticketId: string) => post<SavedTicket>('/saved', { ticket_id: ticketId }),
    onSuccess: (t) => {
      qc.setQueryData(['saved', t.id], t)
      qc.invalidateQueries({ queryKey: ['saved-list'] })
    },
  })
}

/** A game that has kicked off but has no result yet: its score could arrive any minute. */
export const awaitingResult = (l: SavedLeg, now = Date.now()) =>
  !l.score && l.match_status !== 'postponed' && new Date(l.fixture.kickoff).getTime() <= now

/** How often an open saved ticket looks again while a game awaits its result (A10). */
export const SAVED_POLL_MS = 2 * 60 * 1000

export function useSavedTicket(id: string | undefined) {
  return useQuery({
    queryKey: ['saved', id],
    queryFn: () => api<SavedTicket>(`/saved/${encodeURIComponent(id!)}`),
    enabled: !!id,
    retry: false,
    staleTime: 5 * 60 * 1000,
    // Scores arrive every 15 minutes on match days: look again while one is due.
    refetchInterval: (q) => (q.state.data?.legs?.some((l) => awaitingResult(l)) ? SAVED_POLL_MS : false),
  })
}

export function useSavedList(ids: string[]) {
  return useQuery({
    queryKey: ['saved-list', ids],
    queryFn: () => api<SavedTicket[]>(`/saved?ids=${ids.map(encodeURIComponent).join(',')}`),
    enabled: ids.length > 0,
    staleTime: 5 * 60 * 1000,
    // Lists carry no legs, so poll more gently while any ticket is still open.
    refetchInterval: (q) => (q.state.data?.some((t) => t.status === 'pending') ? 5 * 60 * 1000 : false),
  })
}

export function useDeleteSaved() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ticketId }: { id: string; ticketId: string }) =>
      api<void>(`/saved/${encodeURIComponent(id)}`, { method: 'DELETE', body: JSON.stringify({ ticket_id: ticketId }) }),
    onSuccess: (_, { id }) => {
      qc.removeQueries({ queryKey: ['saved', id] })
      qc.invalidateQueries({ queryKey: ['saved-list'] })
    },
  })
}
