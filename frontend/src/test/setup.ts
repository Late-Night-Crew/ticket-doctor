import '@testing-library/jest-dom/vitest'

// jsdom lacks matchMedia (used by react-bootstrap and the theme toggle).
if (typeof window !== 'undefined' && !window.matchMedia) {
  window.matchMedia = (query: string) =>
    ({
      matches: false,
      media: query,
      onchange: null,
      addListener: () => {},
      removeListener: () => {},
      addEventListener: () => {},
      removeEventListener: () => {},
      dispatchEvent: () => false,
    }) as MediaQueryList
}

// Integration tests go through the mock API with simulated latency; allow for slow CI machines.
import { configure } from '@testing-library/react'
configure({ asyncUtilTimeout: 3000 })
