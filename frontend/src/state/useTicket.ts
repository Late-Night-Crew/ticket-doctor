import { useContext } from 'react'
import { TicketContext, type TicketContextValue } from './ticketContextObject'

export function useTicket(): TicketContextValue {
  const ctx = useContext(TicketContext)
  if (!ctx) throw new Error('useTicket must be used inside <TicketProvider>')
  return ctx
}
