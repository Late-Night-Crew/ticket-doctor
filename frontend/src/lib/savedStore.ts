/**
 * The tickets this browser has saved (V2, FR-9.8). There are no accounts, so
 * this list in localStorage is what "My tickets" shows. Each entry keeps the
 * ticket's own id, which is what proves ownership for deleting; the shared
 * link only has the saved id. Clearing browser data loses the list (the links
 * still work).
 */
export interface SavedEntry {
  /** The saved ticket's link id. */
  id: string
  /** The ticket's own id (never shared). */
  ticketId: string
  savedAt: string
}

const KEY = 'td.saved'
const MAX = 50 // the API answers at most 50 ids at once

export function loadSaved(): SavedEntry[] {
  try {
    const raw = JSON.parse(localStorage.getItem(KEY) ?? '[]') as unknown
    return Array.isArray(raw)
      ? raw.filter((e): e is SavedEntry => typeof e?.id === 'string' && typeof e?.ticketId === 'string')
      : []
  } catch {
    return []
  }
}

function store(list: SavedEntry[]) {
  try {
    localStorage.setItem(KEY, JSON.stringify(list.slice(0, MAX)))
  } catch {
    /* storage full or blocked: the link still works */
  }
}

/** Newest first; saving the same ticket again moves it to the top. */
export function rememberSaved(entry: SavedEntry) {
  store([entry, ...loadSaved().filter((e) => e.id !== entry.id)])
}

export function forgetSaved(id: string) {
  store(loadSaved().filter((e) => e.id !== id))
}

/** Drop entries the server no longer has (deleted, or kept past 90 days). */
export function keepOnly(ids: Set<string>) {
  const list = loadSaved()
  if (list.some((e) => !ids.has(e.id))) store(list.filter((e) => ids.has(e.id)))
}

export const savedFor = (ticketId: string) => loadSaved().find((e) => e.ticketId === ticketId)
export const ownerOf = (id: string) => loadSaved().find((e) => e.id === id)

/** The private link to a saved ticket. */
export const savedLink = (id: string) => `${window.location.origin}/t/${id}`
