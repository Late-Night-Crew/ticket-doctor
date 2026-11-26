import { useEffect, useRef } from 'react'
import { useLocation } from 'react-router'
import { titleFor } from '../lib/titles'

/**
 * On navigation: set the document title, scroll to the top, and move focus to
 * the main content so keyboard and screen-reader users start at the new page.
 */
export function RouteEffects({ mainId }: { mainId: string }) {
  const { pathname } = useLocation()
  const first = useRef(true)

  useEffect(() => {
    document.title = titleFor(pathname)
    if (first.current) {
      first.current = false
      return
    }
    window.scrollTo(0, 0)
    document.getElementById(mainId)?.focus({ preventScroll: true })
  }, [pathname, mainId])

  return null
}
