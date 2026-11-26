import { useEffect, useMemo, useReducer, type ReactNode } from 'react'
import { RULES } from '../config/rules'
import { TicketContext, type TicketContextValue } from './ticketContextObject'
import { loadInitial, TICKET_STORAGE_KEY, ticketReducer } from './ticketReducer'

export function TicketProvider({ children }: { children: ReactNode }) {
  const [ticket, dispatch] = useReducer(ticketReducer, undefined, loadInitial)

  // Persist for the session while the user edits (FR-3.5).
  useEffect(() => {
    try {
      sessionStorage.setItem(TICKET_STORAGE_KEY, JSON.stringify(ticket))
    } catch {
      /* ignore */
    }
  }, [ticket])

  const value = useMemo<TicketContextValue>(
    () => ({
      ticket,
      dispatch,
      isFull: ticket.legs.length >= RULES.maxLegs,
      hasMatch: (id) => ticket.legs.some((l) => l.fixture.id === id),
    }),
    [ticket],
  )

  return <TicketContext.Provider value={value}>{children}</TicketContext.Provider>
}
