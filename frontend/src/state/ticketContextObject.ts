import { createContext } from 'react'
import type { TicketAction, TicketState } from './ticketReducer'

export interface TicketContextValue {
  ticket: TicketState
  dispatch: React.Dispatch<TicketAction>
  isFull: boolean
  hasMatch: (matchId: string) => boolean
}

export const TicketContext = createContext<TicketContextValue | null>(null)
