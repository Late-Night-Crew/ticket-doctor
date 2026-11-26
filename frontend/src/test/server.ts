import { setupServer } from 'msw/node'
import { handlers } from '../mocks/handlers'

/** Mock API for component tests: the same handlers the dev app uses. */
export const server = setupServer(...handlers)
