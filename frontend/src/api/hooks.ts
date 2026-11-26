import { keepPreviousData, useMutation, useQuery } from '@tanstack/react-query'
import { api, post } from './client'
import type {
  AcceptSuggestionResponse,
  AnalyseRequest,
  AnalyseResponse,
  Fixture,
  FixtureWindow,
  League,
  ResultsResponse,
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
